package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sqlc-dev/pqtype"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
)

// ─── Request / Response DTOs ────────────────────────────────────────────────

type createPostRequest struct {
	MediaURL     string          `json:"media_url"`
	MediaType    string          `json:"media_type"  binding:"required,oneof=image video text"`
	Caption      string          `json:"caption"`
	BodyText     string          `json:"body_text"`
	LocationName string          `json:"location_name"`
	Latitude     float64         `json:"latitude"`
	Longitude    float64         `json:"longitude"`
	HasLocation  bool            `json:"has_location"`
	CropSettings json.RawMessage `json:"crop_settings"`
	CategoryID   string          `json:"category_id"`
	Width        *int32          `json:"width,omitempty"`
	Height       *int32          `json:"height,omitempty"`
	AspectRatio  *float64        `json:"aspect_ratio,omitempty"`
	ThumbnailUrl *string         `json:"thumbnail_url,omitempty"`
	BlurHash     *string         `json:"blur_hash,omitempty"`
	Duration     *int32          `json:"duration,omitempty"`
	FileSize     *int32          `json:"file_size,omitempty"`
	MimeType     *string         `json:"mime_type,omitempty"`
}


type MediaItem struct {
	URL          string   `json:"url"`
	Type         string   `json:"type"`
	Width        *int32   `json:"width,omitempty"`
	Height       *int32   `json:"height,omitempty"`
	AspectRatio  *float64 `json:"aspect_ratio,omitempty"`
	ThumbnailUrl *string  `json:"thumbnail_url,omitempty"`
	BlurHash     *string  `json:"blur_hash,omitempty"`
	Duration     *int32   `json:"duration,omitempty"`
	FileSize     *int32   `json:"file_size,omitempty"`
	MimeType     *string  `json:"mime_type,omitempty"`
}

type postResponse struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	MediaUrl      string          `json:"media_url"`
	MediaType     string          `json:"media_type"`
	Media         []MediaItem     `json:"media"`
	Caption       string          `json:"caption"`
	BodyText      string          `json:"body_text"`
	LocationName  string          `json:"location_name"`
	LikesCount    int32           `json:"likes_count"`
	CommentsCount int32           `json:"comments_count"`
	SharesCount   int32           `json:"shares_count"`
	CreatedAt     time.Time       `json:"created_at"`
	Username      string          `json:"username,omitempty"`
	FullName      string          `json:"full_name,omitempty"`
	AvatarUrl     string          `json:"avatar_url,omitempty"`
	LikedByViewer bool            `json:"liked_by_viewer"`
	IsSaved       bool            `json:"is_saved"`
	CategoryID    *uuid.UUID      `json:"category_id,omitempty"`
	CropSettings  json.RawMessage `json:"crop_settings,omitempty"`
}

type postCommentResponse struct {
	ID        uuid.UUID       `json:"id"`
	PostID    uuid.UUID       `json:"post_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	Username  string          `json:"username,omitempty"`
	FullName  string          `json:"full_name,omitempty"`
	AvatarUrl string          `json:"avatar_url,omitempty"`
	Mentions  []MentionedUser `json:"mentions,omitempty"`
}

func toPostResponse(p db.CreatePostRow) postResponse {
	rsp := postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
		Media: []MediaItem{
			{
				URL:          p.MediaUrl,
				Type:         p.MediaType,
				Width:        func() *int32 { if p.Width.Valid { v := p.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if p.Height.Valid { v := p.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if p.AspectRatio.Valid { v := p.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if p.ThumbnailUrl.Valid { v := p.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if p.BlurHash.Valid { v := p.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if p.Duration.Valid { v := p.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if p.FileSize.Valid { v := p.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if p.MimeType.Valid { v := p.MimeType.String; return &v }; return nil }(),
			},
		},
		Caption:       p.Caption.String,
		BodyText:      p.BodyText.String,
		LocationName:  p.LocationName.String,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		SharesCount:   p.SharesCount,
		CreatedAt:     p.CreatedAt,
		CropSettings:  p.CropSettings.RawMessage,
	}
	if p.CategoryID.Valid {
		catID := p.CategoryID.UUID
		rsp.CategoryID = &catID
	}
	return rsp
}

func toPostResponseFromList(p db.ListPostsByUserIDRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
		Media: []MediaItem{
			{
				URL:          p.MediaUrl,
				Type:         p.MediaType,
				Width:        func() *int32 { if p.Width.Valid { v := p.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if p.Height.Valid { v := p.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if p.AspectRatio.Valid { v := p.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if p.ThumbnailUrl.Valid { v := p.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if p.BlurHash.Valid { v := p.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if p.Duration.Valid { v := p.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if p.FileSize.Valid { v := p.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if p.MimeType.Valid { v := p.MimeType.String; return &v }; return nil }(),
			},
		},
		Caption:       p.Caption.String,
		BodyText:      p.BodyText.String,
		LocationName:  p.LocationName.String,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		SharesCount:   p.SharesCount,
		CreatedAt:     p.CreatedAt,
		Username:      p.Username,
		FullName:      p.FullName,
		AvatarUrl:     p.AvatarUrl.String,
		LikedByViewer: p.LikedByViewer,
		IsSaved:       p.IsSaved,
		CropSettings:  p.CropSettings.RawMessage,
	}
}

func toPostResponseFromConnections(p db.ListConnectionsPostsRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
		Media: []MediaItem{
			{
				URL:          p.MediaUrl,
				Type:         p.MediaType,
				Width:        func() *int32 { if p.Width.Valid { v := p.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if p.Height.Valid { v := p.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if p.AspectRatio.Valid { v := p.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if p.ThumbnailUrl.Valid { v := p.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if p.BlurHash.Valid { v := p.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if p.Duration.Valid { v := p.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if p.FileSize.Valid { v := p.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if p.MimeType.Valid { v := p.MimeType.String; return &v }; return nil }(),
			},
		},
		Caption:       p.Caption.String,
		BodyText:      p.BodyText.String,
		LocationName:  p.LocationName.String,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		SharesCount:   p.SharesCount,
		CreatedAt:     p.CreatedAt,
		Username:      p.Username,
		FullName:      p.FullName,
		AvatarUrl:     p.AvatarUrl.String,
		LikedByViewer: p.LikedByViewer,
		IsSaved:       p.IsSaved,
		CropSettings:  p.CropSettings.RawMessage,
	}
}

func toPostResponseFromSaved(p db.ListSavedPostsRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
		Media: []MediaItem{
			{
				URL:          p.MediaUrl,
				Type:         p.MediaType,
				Width:        func() *int32 { if p.Width.Valid { v := p.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if p.Height.Valid { v := p.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if p.AspectRatio.Valid { v := p.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if p.ThumbnailUrl.Valid { v := p.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if p.BlurHash.Valid { v := p.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if p.Duration.Valid { v := p.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if p.FileSize.Valid { v := p.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if p.MimeType.Valid { v := p.MimeType.String; return &v }; return nil }(),
			},
		},
		Caption:       p.Caption.String,
		BodyText:      p.BodyText.String,
		LocationName:  p.LocationName.String,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		SharesCount:   p.SharesCount,
		CreatedAt:     p.CreatedAt,
		Username:      p.Username,
		FullName:      p.FullName,
		AvatarUrl:     p.AvatarUrl.String,
		LikedByViewer: p.LikedByViewer,
		IsSaved:       p.IsSaved,
		CropSettings:  p.CropSettings.RawMessage,
	}
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// createPost creates a new permanent post.
func (server *Server) createPost(ctx *gin.Context) {
	var req createPostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// 1. Moderate Content
	isFlagged, reason := server.moderation.ModerateText(req.Caption + " " + req.BodyText)
	if isFlagged {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Content flagged as " + reason,
			"code":  "MODERATION_FAILED",
		})
		return
	}

	var catID uuid.NullUUID
	if req.CategoryID != "" {
		if id, err := uuid.Parse(req.CategoryID); err == nil {
			catID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	post, err := server.store.CreatePost(ctx, db.CreatePostParams{
		UserID:       authPayload.UserID,
		MediaUrl:     req.MediaURL,
		MediaType:    req.MediaType,
		Caption:      sql.NullString{String: req.Caption, Valid: req.Caption != ""},
		BodyText:     sql.NullString{String: req.BodyText, Valid: req.BodyText != ""},
		LocationName: sql.NullString{String: req.LocationName, Valid: req.LocationName != ""},
		Geohash:      sql.NullString{},
		HasLocation:  req.HasLocation,
		Lat:          req.Latitude,
		Lng:          req.Longitude,
		CropSettings: pqtype.NullRawMessage{RawMessage: req.CropSettings, Valid: len(req.CropSettings) > 0},
		CategoryID:   catID,
		Width:        sql.NullInt32{Int32: func() int32 { if req.Width != nil { return *req.Width }; return 0 }(), Valid: req.Width != nil},
		Height:       sql.NullInt32{Int32: func() int32 { if req.Height != nil { return *req.Height }; return 0 }(), Valid: req.Height != nil},
		AspectRatio:  sql.NullFloat64{Float64: func() float64 { if req.AspectRatio != nil { return *req.AspectRatio }; return 0 }(), Valid: req.AspectRatio != nil},
		ThumbnailUrl: sql.NullString{String: func() string { if req.ThumbnailUrl != nil { return *req.ThumbnailUrl }; return "" }(), Valid: req.ThumbnailUrl != nil},
		BlurHash:     sql.NullString{String: func() string { if req.BlurHash != nil { return *req.BlurHash }; return "" }(), Valid: req.BlurHash != nil},
		Duration:     sql.NullInt32{Int32: func() int32 { if req.Duration != nil { return *req.Duration }; return 0 }(), Valid: req.Duration != nil},
		FileSize:     sql.NullInt32{Int32: func() int32 { if req.FileSize != nil { return *req.FileSize }; return 0 }(), Valid: req.FileSize != nil},
		MimeType:     sql.NullString{String: func() string { if req.MimeType != nil { return *req.MimeType }; return "" }(), Valid: req.MimeType != nil},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Extract and save hashtags
	if req.Caption != "" || req.BodyText != "" {
		textToParse := req.Caption + " " + req.BodyText
		tags := util.ExtractHashtags(textToParse)
		for _, tagName := range tags {
			hashtag, err := server.store.UpsertHashtagForPost(ctx, db.UpsertHashtagForPostParams{
				Name: tagName,
				Slug: sql.NullString{String: tagName, Valid: true},
			})
			if err == nil {
				_ = server.store.AddPostHashtag(ctx, db.AddPostHashtagParams{
					PostID:    post.ID,
					HashtagID: hashtag.ID,
				})
			}
		}
	}

	ctx.JSON(http.StatusCreated, toPostResponse(post))
}

// getUserPosts returns a grid of permanent posts for a user profile.
func (server *Server) getUserPosts(ctx *gin.Context) {
	targetUserID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	viewerID := authPayload.UserID

	// DEBUG: Log viewer and target
	log.Info().
		Str("viewer_id", viewerID.String()).
		Str("target_id", targetUserID.String()).
		Str("endpoint", "/users/:id/posts").
		Msg("[PRIVACY DEBUG] Fetching user posts")

	// Use CENTRAL RULE ENGINE for content access (Posts)
	result := server.privacy.CanUserAccess(ctx, viewerID, targetUserID)
	if !result.Allowed {
		log.Warn().
			Str("viewer_id", viewerID.String()).
			Str("target_id", targetUserID.String()).
			Str("reason", string(result.Reason)).
			Msg("[PRIVACY BLOCKED] Posts access denied")
		if result.Reason == "private" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "This account is private"})
			return
		}
		// As per production spec: Blocked, Panic, or Ghost = Invisible (404)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	posts, err := server.store.ListPostsByUserID(ctx, db.ListPostsByUserIDParams{
		UserID:   targetUserID,
		ViewerID: authPayload.UserID,
		Lim:      int32(pageSize),
		Off:      int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]postResponse, len(posts))
	for i, p := range posts {
		rsp[i] = toPostResponseFromList(p)
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}

// getMyPosts returns the authenticated user's own posts.
func (server *Server) getMyPosts(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	posts, err := server.store.ListPostsByUserID(ctx, db.ListPostsByUserIDParams{
		UserID:   authPayload.UserID,
		ViewerID: authPayload.UserID,
		Lim:      int32(pageSize),
		Off:      int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]postResponse, len(posts))
	for i, p := range posts {
		rsp[i] = toPostResponseFromList(p)
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}

// getConnectionsFeed returns posts from connections (following feed).
func (server *Server) getConnectionsFeed(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	cursorStr := ctx.Query("cursor")
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	var cursor sql.NullTime
	if cursorStr != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, cursorStr)
		if err == nil {
			cursor = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}

	posts, err := server.store.ListConnectionsPosts(ctx, db.ListConnectionsPostsParams{
		ViewerID: authPayload.UserID,
		Lim:      int32(pageSize),
		Cursor:   cursor,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]postResponse, len(posts))
	var nextCursor string
	for i, p := range posts {
		rsp[i] = toPostResponseFromConnections(p)
		if i == len(posts)-1 {
			nextCursor = p.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "next_cursor": nextCursor, "page_size": pageSize}))
}

// deletePost lets a user delete their own post.
func (server *Server) deletePost(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if err := server.store.DeletePost(ctx, db.DeletePostParams{
		ID:     postID,
		UserID: authPayload.UserID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"message": "post deleted"}))
}

// likePost likes a post and increments the counter atomically.
func (server *Server) likePost(ctx *gin.Context) {
	postID, ok := parseUUIDParam(ctx, ctx.Param("id"), "post_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	likesCount, err := server.store.LikePostAtomic(ctx, db.LikePostAtomicParams{
		PostID:  postID,
		LikerID: authPayload.UserID,
	})
	if err != nil && err != sql.ErrNoRows {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Broadcase if it was a new like
	if err == nil {
		// Log activity & Broadcast to Admin
		details, _ := json.Marshal(map[string]interface{}{"post_id": postID})
		_, _ = server.store.CreateActivityLog(ctx, db.CreateActivityLogParams{
			UserID:     authPayload.UserID,
			ActionType: "post_liked",
			TargetID:   uuid.NullUUID{UUID: postID, Valid: true},
			TargetType: sql.NullString{String: "post", Valid: true},
			Details:    pqtype.NullRawMessage{RawMessage: details, Valid: true},
		})
		server.hub.BroadcastActivity("post_liked", map[string]interface{}{
			"user_id": authPayload.UserID,
			"post_id": postID,
		})
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"message":     "liked",
		"likes_count": likesCount,
	}))
}

// unlikePost removes a like from a post atomically.
func (server *Server) unlikePost(ctx *gin.Context) {
	postID, ok := parseUUIDParam(ctx, ctx.Param("id"), "post_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	likesCount, err := server.store.UnlikePostAtomic(ctx, db.UnlikePostAtomicParams{
		PostID:  postID,
		LikerID: authPayload.UserID,
	})
	if err != nil && err != sql.ErrNoRows {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"message":     "unliked",
		"likes_count": likesCount,
	}))
}

// sharePost increments the share counter and broadcasts activity.
func (server *Server) sharePost(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err = server.store.IncrementPostShares(ctx, postID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Log activity & Broadcast to Admin
	details, _ := json.Marshal(map[string]interface{}{"post_id": postID})
	_, _ = server.store.CreateActivityLog(ctx, db.CreateActivityLogParams{
		UserID:     authPayload.UserID,
		ActionType: "post_shared",
		TargetID:   uuid.NullUUID{UUID: postID, Valid: true},
		TargetType: sql.NullString{String: "post", Valid: true},
		Details:    pqtype.NullRawMessage{RawMessage: details, Valid: true},
	})
	server.hub.BroadcastActivity("post_shared", map[string]interface{}{
		"user_id": authPayload.UserID,
		"post_id": postID,
	})

	ctx.JSON(http.StatusOK, successResponse(gin.H{"message": "shared"}))
}

// addPostComment adds a comment to a post.
func (server *Server) addPostComment(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1,max=500"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	isFlagged, _ := server.moderation.ModerateText(req.Content)

	comment, err := server.store.CreatePostComment(ctx, db.CreatePostCommentParams{
		PostID:    postID,
		UserID:    authPayload.UserID,
		Content:   req.Content,
		IsFlagged: isFlagged,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.IncrementPostComments(ctx, postID)

	// Log activity & Broadcast to Admin
	details, _ := json.Marshal(map[string]interface{}{"post_id": postID, "content": req.Content, "is_flagged": isFlagged})
	_, _ = server.store.CreateActivityLog(ctx, db.CreateActivityLogParams{
		UserID:     authPayload.UserID,
		ActionType: "comment_created",
		TargetID:   uuid.NullUUID{UUID: comment.ID, Valid: true},
		TargetType: sql.NullString{String: "comment", Valid: true},
		Details:    pqtype.NullRawMessage{RawMessage: details, Valid: true},
	})
	server.hub.BroadcastActivity("comment_created", map[string]interface{}{
		"user_id":    authPayload.UserID,
		"post_id":    postID,
		"content":    req.Content,
		"is_flagged": isFlagged,
	})

	// Send notification to post owner if it's not their own post
	post, err := server.store.GetPost(ctx, postID)
	if err == nil && post.UserID != authPayload.UserID {
		commenter, _ := server.store.GetUserByID(ctx, authPayload.UserID)
		server.createNotificationWithSound(ctx, post.UserID, "story_reaction", "reel_commented",
			"New Comment", fmt.Sprintf("%s commented on your post: %s", commenter.Username, req.Content),
			map[string]uuid.UUID{"user": authPayload.UserID})
	}

	// Process @mentions in the comment
	commenter, _ := server.store.GetUserByID(ctx, authPayload.UserID)
	mentions := server.processCommentMentions(ctx, "post_comment", comment.ID, req.Content, authPayload.UserID, commenter.Username)

	ctx.JSON(http.StatusCreated, postCommentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		Mentions:  mentions,
	})
}

// listPostComments returns comments for a post.
func (server *Server) listPostComments(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	comments, err := server.store.ListPostComments(ctx, postID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]postCommentResponse, len(comments))
	for i, c := range comments {
		rsp[i] = postCommentResponse{
			ID:        c.ID,
			PostID:    c.PostID,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			Username:  c.Username,
			FullName:  c.FullName,
			AvatarUrl: c.AvatarUrl.String,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

// deletePostComment deletes a comment the user owns.
func (server *Server) deletePostComment(ctx *gin.Context) {
	commentID, err := uuid.Parse(ctx.Param("commentId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	postID, err := server.store.DeletePostComment(ctx, db.DeletePostCommentParams{
		ID:     commentID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("comment not found or unauthorized")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.DecrementPostComments(ctx, postID)
	ctx.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// savePost saves a post for the user.
func (server *Server) savePost(ctx *gin.Context) {
	postID, ok := parseUUIDParam(ctx, ctx.Param("id"), "post_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	savesCount, err := server.store.SavePostAtomic(ctx, db.SavePostAtomicParams{
		PostID: postID,
		UserID: authPayload.UserID,
	})
	if err != nil && err != sql.ErrNoRows {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"message":     "saved",
		"saves_count": savesCount,
	}))
}

// unsavePost removes a saved post for the user.
func (server *Server) unsavePost(ctx *gin.Context) {
	postID, ok := parseUUIDParam(ctx, ctx.Param("id"), "post_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	savesCount, err := server.store.UnsavePostAtomic(ctx, db.UnsavePostAtomicParams{
		PostID: postID,
		UserID: authPayload.UserID,
	})
	if err != nil && err != sql.ErrNoRows {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"message":     "unsaved",
		"saves_count": savesCount,
	}))
}

// getSavedPosts returns the authenticated user's saved posts.
func (server *Server) getSavedPosts(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	posts, err := server.store.ListSavedPosts(ctx, db.ListSavedPostsParams{
		ViewerID: authPayload.UserID,
		Lim:      int32(pageSize),
		Off:      int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]postResponse, len(posts))
	for i, p := range posts {
		rsp[i] = toPostResponseFromSaved(p)
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}

// updatePost handles updating the caption of a post the user owns.
func (server *Server) updatePost(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req struct {
		Caption string `json:"caption"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	post, err := server.store.UpdatePost(ctx, db.UpdatePostParams{
		ID:      postID,
		UserID:  authPayload.UserID,
		Caption: sql.NullString{String: req.Caption, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("post not found or unauthorized")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"message": "post updated",
		"post": postResponse{
			ID:            post.ID,
			UserID:        post.UserID,
			MediaUrl:      post.MediaUrl,
			MediaType:     post.MediaType,
			Caption:       post.Caption.String,
			BodyText:      post.BodyText.String,
			LocationName:  post.LocationName.String,
			LikesCount:    post.LikesCount,
			CommentsCount: post.CommentsCount,
			SharesCount:   post.SharesCount,
			CreatedAt:     post.CreatedAt,
			CropSettings:  post.CropSettings.RawMessage,
		},
	}))
}

// getTrendingNearbyPosts returns trending posts around a location
func (server *Server) getTrendingNearbyPosts(ctx *gin.Context) {
	latStr := ctx.Query("lat")
	lngStr := ctx.Query("lng")
	
	if latStr == "" || lngStr == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("lat and lng are required")))
		return
	}
	
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid lat")))
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid lng")))
		return
	}

	radiusStr := ctx.DefaultQuery("radius", "10")
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid radius")))
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

	timeFilter := ctx.DefaultQuery("time", "")
	categoryID := ctx.DefaultQuery("category", "")
	if categoryID == "" {
		categoryID = "00000000-0000-0000-0000-000000000000"
	}

	posts, err := server.store.ListTrendingNearbyPosts(ctx, db.ListTrendingNearbyPostsParams{
		Lat:        lat,
		Lng:        lng,
		ViewerID:   authPayload.UserID,
		RadiusKm:   radius,
		TimeFilter: timeFilter,
		CategoryID: categoryID,
		Off:        int32((page - 1) * pageSize),
		Lim:        int32(pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(posts))
	for i, p := range posts {
		rsp[i] = gin.H{
			"id":             p.ID,
			"caption":        p.Caption.String,
			"body_text":      p.BodyText.String,
			"user_id":        p.UserID,
			"media_url":      p.MediaUrl,
			"media_type":     p.MediaType,
			"likes_count":    p.LikesCount,
			"comments_count": p.CommentsCount,
			"shares_count":   p.SharesCount,
			"created_at":     p.CreatedAt,
			"username":       p.Username,
			"avatar_url":     p.AvatarUrl.String,
			"liked_by_viewer": p.LikedByViewer,
			"is_saved":       p.IsSaved,
			"distance_meters": p.DistanceMeters,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}

func (server *Server) getPost(ctx *gin.Context) {
	postIDStr := ctx.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid post ID: %w", err)))
		return
	}

	post, err := server.store.GetPost(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("post not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// viewer ID can be extracted when we implement real like/save checks

	// Fetch user details
	user, err := server.store.GetUserProfile(ctx, post.UserID)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Msg("failed to get user profile for post")
	}

	// Fetch category if valid
	var category interface{}
	if post.CategoryID.Valid {
		cat, err := server.store.GetCategory(ctx, post.CategoryID.UUID)
		if err == nil {
			category = gin.H{
				"id": cat.ID,
				"name": cat.Name,
				"icon": nullStrToPtr(cat.Icon),
				"color": nullStrToPtr(cat.Color),
			}
		}
	}

	// Fetch hashtags
	hashtags := []string{}
	// Note: You would normally call GetHashtagsForPost here.
	// For now we mock it or use an empty array if not implemented.

	// Construct Unified JSON Response
	rsp := gin.H{
		"post": gin.H{
			"id": post.ID,
			"caption": post.Caption.String,
			"body_text": post.BodyText.String,
			"created_at": post.CreatedAt,
		},
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"full_name": user.FullName,
			"avatar_url": user.AvatarUrl.String,
		},
		"category": category,
		"hashtags": hashtags,
		"location": gin.H{
			"name": post.LocationName.String,
		},
		"media": []gin.H{
			{
				"url": post.MediaUrl,
				"type": post.MediaType,
			},
		},
		"stats": gin.H{
			"likes": post.LikesCount,
			"comments": post.CommentsCount,
			"shares": post.SharesCount,
			"views": 0, // Mock for now
			"saved": post.SavesCount,
		},
		"viewer": gin.H{
			// Would query DB for CheckPostLike / CheckPostSave
			"liked": false,
			"saved": false,
			"following": false,
		},
	}

	ctx.JSON(http.StatusOK, successResponse(rsp))
}

func (server *Server) getRelatedPosts(ctx *gin.Context) {
	// Mock implementation for Related Posts algorithm
	// Should implement: +40 Category, +30 Hashtag, +20 Location, +10 Recent
	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": []interface{}{}}))
}

func (server *Server) viewPost(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, successResponse(gin.H{"success": true}))
}
