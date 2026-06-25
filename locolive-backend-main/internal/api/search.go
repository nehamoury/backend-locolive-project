package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

type searchRequest struct {
	Query    string `form:"q"`
	Type     string `form:"type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (server *Server) unifiedSearch(ctx *gin.Context) {
	var req searchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	query := strings.TrimSpace(req.Query)
	if len(query) < 2 {
		ctx.JSON(http.StatusOK, successResponse(gin.H{
			"users":    []interface{}{},
			"posts":    []interface{}{},
			"reels":    []interface{}{},
			"hashtags": []interface{}{},
			"places":   []interface{}{},
		}))
		return
	}

	authPayload, _ := ctx.Get(authorizationPayloadKey)
	var viewerID uuid.UUID
	if authPayload != nil {
		viewerID = authPayload.(*token.Payload).UserID
	}

	type searchResult struct {
		users    []gin.H
		posts    []gin.H
		reels    []gin.H
		hashtags []gin.H
		places   []gin.H
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	res := searchResult{}

	wg.Add(5)
	go func() {
		defer wg.Done()
		users, _ := server.searchUsersInternal(ctx, query, viewerID)
		mu.Lock()
		res.users = users
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		posts, _ := server.searchPostsInternal(ctx, query, viewerID)
		mu.Lock()
		res.posts = posts
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		reels, _ := server.searchReelsInternal(ctx, query, viewerID)
		mu.Lock()
		res.reels = reels
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		hashtags, _ := server.searchHashtagsInternal(ctx, query)
		mu.Lock()
		res.hashtags = hashtags
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		places, _ := server.searchPlacesInternal(ctx, query)
		mu.Lock()
		res.places = places
		mu.Unlock()
	}()
	wg.Wait()

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"users":    res.users,
		"posts":    res.posts,
		"reels":    res.reels,
		"hashtags": res.hashtags,
		"places":   res.places,
	}))
}

func (server *Server) searchPosts(ctx *gin.Context) {
	var req searchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 2 {
		ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": []interface{}{}}))
		return
	}

	page, pageSize := parsePageParams(req.Page, req.PageSize, 12)

	authPayload, _ := ctx.Get(authorizationPayloadKey)
	var viewerID uuid.UUID
	if authPayload != nil {
		viewerID = authPayload.(*token.Payload).UserID
	}

	posts, err := server.store.SearchPosts(ctx, db.SearchPostsParams{
		ViewerID: viewerID,
		Query:    query,
		Off:      int32((page - 1) * pageSize),
		Lim:      int32(pageSize),
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

func (server *Server) searchReels(ctx *gin.Context) {
	var req searchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 2 {
		ctx.JSON(http.StatusOK, successResponse(gin.H{"reels": []interface{}{}}))
		return
	}

	page, pageSize := parsePageParams(req.Page, req.PageSize, 12)

	authPayload, _ := ctx.Get(authorizationPayloadKey)
	var viewerID uuid.UUID
	if authPayload != nil {
		viewerID = authPayload.(*token.Payload).UserID
	}

	reels, err := server.store.SearchReels(ctx, db.SearchReelsParams{
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
		ViewerID: viewerID,
		Query:    query,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(reels))
	for i, r := range reels {
		rsp[i] = gin.H{
			"id":                r.ID,
			"user_id":           r.UserID,
			"video_url":         r.VideoUrl,
			"caption":           r.Caption.String,
			"location_name":     r.LocationName.String,
			"likes_count":       r.LikesCount,
			"comments_count":    r.CommentsCount,
			"shares_count":      r.SharesCount,
			"saves_count":       r.SavesCount,
			"created_at":        r.CreatedAt,
			"username":          r.Username,
			"avatar_url":        r.AvatarUrl.String,
			"is_liked":          r.IsLiked,
			"is_saved":          r.IsSaved,
			"connection_status": fmt.Sprintf("%v", r.ConnectionStatus),
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"reels": rsp, "page": page, "page_size": pageSize}))
}

func (server *Server) searchPlaces(ctx *gin.Context) {
	var req searchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 2 {
		ctx.JSON(http.StatusOK, successResponse(gin.H{"places": []interface{}{}}))
		return
	}

	page, pageSize := parsePageParams(req.Page, req.PageSize, 12)

	places, err := server.store.SearchPlaces(ctx, db.SearchPlacesParams{
		Query: query,
		Off:   int32((page - 1) * pageSize),
		Lim:   int32(pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(places))
	for i, p := range places {
		rsp[i] = gin.H{
			"id":         p.ID,
			"name":       p.Name,
			"slug":       p.Slug,
			"address":    p.Address.String,
			"post_count": p.PostCount,
			"created_at": p.CreatedAt,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"places": rsp, "page": page, "page_size": pageSize}))
}

// ─── Internal helpers (for unified search) ──────────────────────────────────

func (server *Server) searchUsersInternal(ctx *gin.Context, query string, viewerID uuid.UUID) ([]gin.H, error) {
	users, err := server.store.SearchUsers(ctx, db.SearchUsersParams{
		Query: query,
		Off:   0,
		Lim:   3,
	})
	if err != nil {
		return nil, err
	}

	rsp := make([]gin.H, 0, len(users))
	for _, u := range users {
		if viewerID != uuid.Nil && viewerID == u.ID {
			continue
		}
		if viewerID != uuid.Nil {
			result := server.privacy.CanViewProfile(ctx, viewerID, u.ID)
			if !result.Allowed {
				continue
			}
		}
		connStatus := "none"
		if viewerID != uuid.Nil {
			conn, err := server.store.GetConnection(ctx, db.GetConnectionParams{
				RequesterID: viewerID,
				TargetID:    u.ID,
			})
			if err == nil {
				connStatus = string(conn.Status)
			}
		}

		rsp = append(rsp, gin.H{
			"id":                u.ID,
			"username":          u.Username,
			"full_name":         u.FullName,
			"avatar_url":        u.AvatarUrl.String,
			"is_verified":       u.IsVerified,
			"is_private":        u.IsPrivate,
			"connection_status": connStatus,
		})
	}
	return rsp, nil
}

func (server *Server) searchPostsInternal(ctx *gin.Context, query string, viewerID uuid.UUID) ([]gin.H, error) {
	posts, err := server.store.SearchPosts(ctx, db.SearchPostsParams{
		ViewerID: viewerID,
		Query:    query,
		Off:      0,
		Lim:      3,
	})
	if err != nil {
		return nil, err
	}

	rsp := make([]gin.H, len(posts))
	for i, p := range posts {
		rsp[i] = gin.H{
			"id":          p.ID,
			"caption":     p.Caption.String,
			"media_url":   p.MediaUrl,
			"media_type":  p.MediaType,
			"likes_count": p.LikesCount,
			"created_at":  p.CreatedAt,
			"username":    p.Username,
			"avatar_url":  p.AvatarUrl.String,
		}
	}
	return rsp, nil
}

func (server *Server) searchReelsInternal(ctx *gin.Context, query string, viewerID uuid.UUID) ([]gin.H, error) {
	reels, err := server.store.SearchReels(ctx, db.SearchReelsParams{
		Limit:    3,
		Offset:   0,
		ViewerID: viewerID,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}

	rsp := make([]gin.H, len(reels))
	for i, r := range reels {
		rsp[i] = gin.H{
			"id":          r.ID,
			"caption":     r.Caption.String,
			"video_url":   r.VideoUrl,
			"likes_count": r.LikesCount,
			"created_at":  r.CreatedAt,
			"username":    r.Username,
			"avatar_url":  r.AvatarUrl.String,
		}
	}
	return rsp, nil
}

func (server *Server) searchHashtagsInternal(ctx *gin.Context, query string) ([]gin.H, error) {
	hashtags, err := server.store.SearchHashtags(ctx, db.SearchHashtagsParams{
		Column1: sql.NullString{String: query, Valid: true},
		Limit:   5,
		Offset:  0,
	})
	if err != nil {
		return nil, err
	}

	rsp := make([]gin.H, len(hashtags))
	for i, h := range hashtags {
		rsp[i] = gin.H{
			"id":          h.ID,
			"name":        h.Name,
			"usage_count": h.UsageCount,
		}
	}
	return rsp, nil
}

func (server *Server) searchPlacesInternal(ctx *gin.Context, query string) ([]gin.H, error) {
	places, err := server.store.SearchPlaces(ctx, db.SearchPlacesParams{
		Query: query,
		Off:   0,
		Lim:   3,
	})
	if err != nil {
		return nil, err
	}

	rsp := make([]gin.H, len(places))
	for i, p := range places {
		rsp[i] = gin.H{
			"id":         p.ID,
			"name":       p.Name,
			"slug":       p.Slug,
			"post_count": p.PostCount,
		}
	}
	return rsp, nil
}

func parsePageParams(page, pageSize, defaultSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = defaultSize
	}
	return page, pageSize
}
