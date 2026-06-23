package api

import (
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
		Column1: sqlcArgString(query),
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

func sqlcArgString(s string) string {
	return s
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

	rsp := make([]hashtagResponse, len(hashtags))
	for i, h := range hashtags {
		rsp[i] = toHashtagResponse(h)
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
