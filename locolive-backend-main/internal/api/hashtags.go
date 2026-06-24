package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

type hashtagResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	UsageCount  int32     `json:"usage_count"`
	ReelsCount  int32     `json:"reels_count"`
	LastUsedAt  time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func toHashtagResponse(h db.Hashtag) hashtagResponse {
	return hashtagResponse{
		ID:         h.ID,
		Name:       h.Name,
		UsageCount: h.UsageCount,
		ReelsCount: h.ReelsCount,
		LastUsedAt: h.LastUsedAt,
		CreatedAt:  h.CreatedAt,
	}
}

// searchHashtags searches for hashtags by prefix
func (server *Server) searchHashtags(ctx *gin.Context) {
	query := ctx.Query("q")
	if query == "" {
		ctx.JSON(http.StatusOK, successResponse([]hashtagResponse{}))
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	hashtags, err := server.store.SearchHashtags(ctx, db.SearchHashtagsParams{
		Column1: sql.NullString{String: query, Valid: true},
		Limit:   int32(limit),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]hashtagResponse, len(hashtags))
	for i, h := range hashtags {
		rsp[i] = toHashtagResponse(h)
	}

	ctx.JSON(http.StatusOK, successResponse(rsp))
}



// getTrendingHashtags returns top trending hashtags
func (server *Server) getTrendingHashtags(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	hashtags, err := server.store.GetTrendingHashtags(ctx, int32(limit))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(hashtags))
	for i, h := range hashtags {
		rsp[i] = gin.H{
			"name":  h.Name,
			"count": h.UsageCount,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(rsp))
}

// getReelsByHashtag returns reels tagged with a specific hashtag
func (server *Server) getReelsByHashtag(ctx *gin.Context) {
	hashtagName := ctx.Param("name")
	if hashtagName == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("hashtag name is required")))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	reels, err := server.store.ListReelsByHashtag(ctx, db.ListReelsByHashtagParams{
		HashtagName: hashtagName,
		ViewerID:    authPayload.UserID,
		Limit:       int32(pageSize),
		Offset:      int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]reelResponse, len(reels))
	for i, r := range reels {
		rsp[i] = reelResponse{
			ID:               r.ID,
			UserID:           r.UserID,
			VideoURL:         r.VideoUrl,
			Caption:          nullStrToPtr(r.Caption),
			IsAiGenerated:    r.IsAiGenerated,
			LocationName:     nullStrToPtr(r.LocationName),
			Geohash:          nullStrToPtr(r.Geohash),
			Lat:              r.Lat,
			Lng:              r.Lng,
			LikesCount:       r.LikesCount,
			CommentsCount:    r.CommentsCount,
			SharesCount:      r.SharesCount,
			SavesCount:       r.SavesCount,
			CreatedAt:        r.CreatedAt,
			UpdatedAt:        r.UpdatedAt,
			Username:         r.Username,
			IsLiked:          r.IsLiked,
			IsSaved:          r.IsSaved,
			ConnectionStatus: fmt.Sprintf("%v", r.ConnectionStatus),
		}
		if r.AvatarUrl.Valid {
			rsp[i].AvatarURL = &r.AvatarUrl.String
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"reels": rsp, "page": page, "page_size": pageSize}))
}

// getHashtagByName returns details and counts for a specific hashtag
func (server *Server) getHashtagByName(ctx *gin.Context) {
	hashtagName := ctx.Param("name")
	if hashtagName == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("hashtag name is required")))
		return
	}

	h, err := server.store.GetHashtagByName(ctx, hashtagName)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("hashtag not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"name":        h.Name,
		"posts_count": h.UsageCount - h.ReelsCount,
		"reels_count": h.ReelsCount,
		"total_count": h.UsageCount,
	}))
}

// getPostsByHashtag returns posts tagged with a specific hashtag
func (server *Server) getPostsByHashtag(ctx *gin.Context) {
	hashtagName := ctx.Param("name")
	if hashtagName == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("hashtag name is required")))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	posts, err := server.store.ListPostsByHashtag(ctx, db.ListPostsByHashtagParams{
		HashtagName: hashtagName,
		ViewerID:    authPayload.UserID,
		Lim:         int32(pageSize),
		Off:         int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(posts))
	for i, p := range posts {
		var catMap interface{} = nil
		if p.CategoryID.Valid {
			catMap = gin.H{
				"id":   p.CategoryID.UUID.String(),
				"name": p.CategoryName.String,
				"icon": p.CategoryIcon.String,
			}
		}

		rsp[i] = gin.H{
			"id":              p.ID,
			"caption":         p.Caption.String,
			"body_text":       p.BodyText.String,
			"hashtags":        p.Hashtags,
			"category":        catMap,
			"user_id":         p.UserID,
			"media_url":       p.MediaUrl,
			"media_type":      p.MediaType,
			"likes_count":     p.LikesCount,
			"comments_count":  p.CommentsCount,
			"shares_count":    p.SharesCount,
			"created_at":      p.CreatedAt,
			"username":        p.Username,
			"avatar_url":      p.AvatarUrl.String,
			"liked_by_viewer": p.LikedByViewer,
			"is_saved":        p.IsSaved,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}
