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
	"github.com/sqlc-dev/pqtype"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

// ─── Request / Response DTOs ────────────────────────────────────────────────

type createPostRequest struct {
	MediaURL     string  `json:"media_url"   binding:"required"`
	MediaType    string  `json:"media_type"  binding:"required,oneof=image video text"`
	Caption      string  `json:"caption"`
	BodyText     string  `json:"body_text"`
	LocationName string  `json:"location_name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64         `json:"longitude"`
	HasLocation  bool            `json:"has_location"`
	CropSettings json.RawMessage `json:"crop_settings"`
}

type postResponse struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	MediaUrl      string    `json:"media_url"`
	MediaType     string    `json:"media_type"`
	Caption       string    `json:"caption"`
	BodyText      string    `json:"body_text"`
	LocationName  string    `json:"location_name"`
	LikesCount    int32     `json:"likes_count"`
	CommentsCount int32     `json:"comments_count"`
	SharesCount   int32     `json:"shares_count"`
	CreatedAt     time.Time `json:"created_at"`
	Username      string    `json:"username,omitempty"`
	FullName      string    `json:"full_name,omitempty"`
	AvatarUrl     string          `json:"avatar_url,omitempty"`
	LikedByViewer bool            `json:"liked_by_viewer"`
	CropSettings  json.RawMessage `json:"crop_settings,omitempty"`
}

type postCommentResponse struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	AvatarUrl string    `json:"avatar_url,omitempty"`
}

func toPostResponse(p db.CreatePostRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
		Caption:       p.Caption.String,
		BodyText:      p.BodyText.String,
		LocationName:  p.LocationName.String,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		SharesCount:   p.SharesCount,
		CreatedAt:     p.CreatedAt,
		CropSettings:  p.CropSettings.RawMessage,
	}
}

func toPostResponseFromList(p db.ListPostsByUserIDRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
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
		CropSettings:  p.CropSettings.RawMessage,
	}
}

func toPostResponseFromConnections(p db.ListConnectionsPostsRow) postResponse {
	return postResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		MediaUrl:      p.MediaUrl,
		MediaType:     p.MediaType,
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

	post, err := server.store.CreatePost(ctx, db.CreatePostParams{
		UserID:      authPayload.UserID,
		MediaUrl:    req.MediaURL,
		MediaType:   req.MediaType,
		Caption:     sql.NullString{String: req.Caption, Valid: req.Caption != ""},
		BodyText:    sql.NullString{String: req.BodyText, Valid: req.BodyText != ""},
		LocationName: sql.NullString{String: req.LocationName, Valid: req.LocationName != ""},
		Geohash:     sql.NullString{},
		HasLocation:  req.HasLocation,
		Lat:          req.Latitude,
		Lng:          req.Longitude,
		CropSettings: pqtype.NullRawMessage{RawMessage: req.CropSettings, Valid: len(req.CropSettings) > 0},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
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

	// Use CENTRAL RULE ENGINE for content access (Posts)
	result := server.privacy.CanUserAccess(ctx, authPayload.UserID, targetUserID)
	if !result.Allowed {
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

	ctx.JSON(http.StatusOK, gin.H{"posts": rsp, "page": page, "page_size": pageSize})
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

	ctx.JSON(http.StatusOK, gin.H{"posts": rsp, "page": page, "page_size": pageSize})
}

// getConnectionsFeed returns posts from connections (following feed).
func (server *Server) getConnectionsFeed(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	posts, err := server.store.ListConnectionsPosts(ctx, db.ListConnectionsPostsParams{
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
		rsp[i] = toPostResponseFromConnections(p)
	}

	ctx.JSON(http.StatusOK, gin.H{"posts": rsp, "page": page, "page_size": pageSize})
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

	ctx.JSON(http.StatusOK, gin.H{"message": "post deleted"})
}

// likePost likes a post and increments the counter.
func (server *Server) likePost(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	_, err = server.store.LikePost(ctx, db.LikePostParams{
		PostID: postID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		// Conflict (already liked) is treated as success
		ctx.JSON(http.StatusOK, gin.H{"message": "liked"})
		return
	}

	_ = server.store.IncrementPostLikes(ctx, postID)

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

	ctx.JSON(http.StatusOK, gin.H{"message": "liked"})
}

// unlikePost removes a like from a post.
func (server *Server) unlikePost(ctx *gin.Context) {
	postID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if err := server.store.UnlikePost(ctx, db.UnlikePostParams{
		PostID: postID,
		UserID: authPayload.UserID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.DecrementPostLikes(ctx, postID)
	ctx.JSON(http.StatusOK, gin.H{"message": "unliked"})
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

	ctx.JSON(http.StatusOK, gin.H{"success": true})
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
	server.processMentions(ctx, req.Content, authPayload.UserID, commenter.Username)

	ctx.JSON(http.StatusCreated, postCommentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
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

