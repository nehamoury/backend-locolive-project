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

// ─── Request / Response DTOs ──────────────────────────────────────────────────

type createReelRequest struct {
	VideoURL      string  `json:"video_url"       binding:"required"`
	Caption       string  `json:"caption"`
	IsAiGenerated bool    `json:"is_ai_generated"`
	LocationName  string  `json:"location_name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	HasLocation   bool    `json:"has_location"`
}

type reelCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

type reelResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	VideoURL       string    `json:"video_url"`
	Caption        *string   `json:"caption"`
	IsAiGenerated  bool      `json:"is_ai_generated"`
	LocationName   *string   `json:"location_name"`
	Geohash        *string   `json:"geohash"`
	Lat            float64   `json:"lat"`
	Lng            float64   `json:"lng"`
	LikesCount     int32     `json:"likes_count"`
	CommentsCount  int32     `json:"comments_count"`
	SharesCount    int32     `json:"shares_count"`
	SavesCount     int32     `json:"saves_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Username       string    `json:"username,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	DistanceMeters *float64  `json:"distance_meters,omitempty"`
	IsLiked        bool      `json:"is_liked"`
	IsSaved        bool      `json:"is_saved"`
	ConnectionStatus string    `json:"connection_status,omitempty"`
}

type reelCommentResponse struct {
	ID        uuid.UUID `json:"id"`
	ReelID    uuid.UUID `json:"reel_id"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Mappers ─────────────────────────────────────────────────────────────────

func nullStrToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func toReelResponseFromCreate(r db.CreateReelRow) reelResponse {
	return reelResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		VideoURL:      r.VideoUrl,
		Caption:       nullStrToPtr(r.Caption),
		IsAiGenerated: r.IsAiGenerated,
		LocationName:  nullStrToPtr(r.LocationName),
		Geohash:       nullStrToPtr(r.Geohash),
		Lat:           r.Lat,
		Lng:           r.Lng,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		SharesCount:   r.SharesCount,
		SavesCount:    r.SavesCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toReelResponseFromGet(r db.GetReelRow) reelResponse {
	return reelResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		VideoURL:      r.VideoUrl,
		Caption:       nullStrToPtr(r.Caption),
		IsAiGenerated: r.IsAiGenerated,
		LocationName:  nullStrToPtr(r.LocationName),
		Geohash:       nullStrToPtr(r.Geohash),
		Lat:           r.Lat,
		Lng:           r.Lng,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		SharesCount:   r.SharesCount,
		SavesCount:    r.SavesCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toReelResponseFromFeed(r db.ListReelsFeedRow) reelResponse {
	rsp := reelResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		VideoURL:      r.VideoUrl,
		Caption:       nullStrToPtr(r.Caption),
		IsAiGenerated: r.IsAiGenerated,
		LocationName:  nullStrToPtr(r.LocationName),
		Geohash:       nullStrToPtr(r.Geohash),
		Lat:           r.Lat,
		Lng:           r.Lng,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		SharesCount:   r.SharesCount,
		SavesCount:    r.SavesCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Username:      r.Username,
		IsLiked:       r.IsLiked,
		IsSaved:       r.IsSaved,
		ConnectionStatus: fmt.Sprintf("%v", r.ConnectionStatus),
	}
	if r.AvatarUrl.Valid {
		rsp.AvatarURL = &r.AvatarUrl.String
	}
	return rsp
}

func toReelResponseFromNearby(r db.ListNearbyReelsRow) reelResponse {
	rsp := reelResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		VideoURL:      r.VideoUrl,
		Caption:       nullStrToPtr(r.Caption),
		IsAiGenerated: r.IsAiGenerated,
		LocationName:  nullStrToPtr(r.LocationName),
		Geohash:       nullStrToPtr(r.Geohash),
		Lat:           r.Lat,
		Lng:           r.Lng,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		SharesCount:   r.SharesCount,
		SavesCount:    r.SavesCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Username:      r.Username,
		IsLiked:       r.IsLiked,
		IsSaved:       r.IsSaved,
		ConnectionStatus: fmt.Sprintf("%v", r.ConnectionStatus),
	}
	if r.AvatarUrl.Valid {
		rsp.AvatarURL = &r.AvatarUrl.String
	}
	if d, ok := r.DistanceMeters.(float64); ok {
		rsp.DistanceMeters = &d
	}
	return rsp
}

func toReelResponseFromUserReels(r db.ListUserReelsRow) reelResponse {
	rsp := reelResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		VideoURL:      r.VideoUrl,
		Caption:       nullStrToPtr(r.Caption),
		IsAiGenerated: r.IsAiGenerated,
		LocationName:  nullStrToPtr(r.LocationName),
		Geohash:       nullStrToPtr(r.Geohash),
		Lat:           r.Lat,
		Lng:           r.Lng,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		SharesCount:   r.SharesCount,
		SavesCount:    r.SavesCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Username:      r.Username,
		IsLiked:       r.IsLiked,
		IsSaved:       r.IsSaved,
		ConnectionStatus: fmt.Sprintf("%v", r.ConnectionStatus),
	}
	if r.AvatarUrl.Valid {
		rsp.AvatarURL = &r.AvatarUrl.String
	}
	return rsp
}

func toReelCommentResponse(c db.ReelComment) reelCommentResponse {
	return reelCommentResponse{
		ID:        c.ID,
		ReelID:    c.ReelID,
		UserID:    c.UserID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}

func toReelCommentResponseFromList(c db.ListReelCommentsRow) reelCommentResponse {
	rsp := reelCommentResponse{
		ID:        c.ID,
		ReelID:    c.ReelID,
		UserID:    c.UserID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		Username:  c.Username,
	}
	if c.AvatarUrl.Valid {
		rsp.AvatarURL = &c.AvatarUrl.String
	}
	return rsp
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// createReel creates a new reel.
func (server *Server) createReel(ctx *gin.Context) {
	var req createReelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Determine geom if location is provided
	var geom interface{}
	if req.HasLocation {
		// PostGIS WKT format: POINT(lng lat)
		geom = fmt.Sprintf("SRID=4326;POINT(%f %f)", req.Longitude, req.Latitude)
	}

	reel, err := server.store.CreateReel(ctx, db.CreateReelParams{
		UserID:        authPayload.UserID,
		VideoUrl:      req.VideoURL,
		Caption:       sql.NullString{String: req.Caption, Valid: req.Caption != ""},
		IsAiGenerated: req.IsAiGenerated,
		LocationName:  sql.NullString{String: req.LocationName, Valid: req.LocationName != ""},
		Geom:          geom,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Real-time notification for nearby users
	if req.HasLocation {
		nearbyUsers, _ := server.location.GetNearbyUsers(ctx, authPayload.UserID, req.Latitude, req.Longitude, 5.0)
		targetUserIDs := make([]uuid.UUID, 0, len(nearbyUsers))
		for _, u := range nearbyUsers {
			if id, err := uuid.Parse(u.Name); err == nil {
				targetUserIDs = append(targetUserIDs, id)
			}
		}
		if len(targetUserIDs) > 0 {
			server.hub.BroadcastNewReelNearby(reel, targetUserIDs)
		}
	}

	ctx.JSON(http.StatusCreated, toReelResponseFromCreate(reel))
}

// getReelsFeed returns a paginated feed of reels.
func (server *Server) getReelsFeed(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 30 {
		pageSize = 10
	}

	reels, err := server.store.ListReelsFeed(ctx, db.ListReelsFeedParams{
		UserID: authPayload.UserID,
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]reelResponse, len(reels))
	for i, r := range reels {
		rsp[i] = toReelResponseFromFeed(r)
	}

	ctx.JSON(http.StatusOK, gin.H{"reels": rsp, "page": page, "page_size": pageSize})
}

// getNearbyReels returns reels within a certain radius.
func (server *Server) getNearbyReels(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid latitude")))
		return
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid longitude")))
		return
	}

	radius, _ := strconv.ParseFloat(ctx.DefaultQuery("radius", "5000"), 64) // Default 5km
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	reels, err := server.store.ListNearbyReels(ctx, db.ListNearbyReelsParams{
		Lat:      lat,
		Lng:      lng,
		ViewerID: authPayload.UserID,
		Radius:   radius,
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]reelResponse, len(reels))
	for i, r := range reels {
		rsp[i] = toReelResponseFromNearby(r)
	}

	ctx.JSON(http.StatusOK, gin.H{"reels": rsp, "page": page, "page_size": pageSize})
}

// getUserReels returns paginated reels for a specific user with privacy checks
func (server *Server) getUserReels(ctx *gin.Context) {
	targetUserID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid user id")))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Privacy checks: ghost mode, blocked users
	targetUser, err := server.store.GetUserByID(ctx, targetUserID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("user not found")))
		return
	}

	// Check if viewing user is blocked
	if targetUserID != authPayload.UserID {
		blocked, _ := server.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
			BlockerID: targetUserID,
			BlockedID: authPayload.UserID,
		})
		if blocked {
			ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("user has blocked you")))
			return
		}
	}

	// If target user is in ghost mode and not viewing own reels, deny access
	if targetUser.IsGhostMode && targetUserID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("user is in ghost mode")))
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

	reels, err := server.store.ListUserReels(ctx, db.ListUserReelsParams{
		UserID:   targetUserID,
		ViewerID: authPayload.UserID,
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]reelResponse, len(reels))
	for i, r := range reels {
		rsp[i] = toReelResponseFromUserReels(r)
	}

	ctx.JSON(http.StatusOK, gin.H{"reels": rsp, "page": page, "page_size": pageSize})
}

// likeReel likes a reel.
func (server *Server) likeReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	_, err = server.store.LikeReel(ctx, db.LikeReelParams{
		ReelID: reelID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		// Conflict is fine
		ctx.JSON(http.StatusOK, gin.H{"message": "liked"})
		return
	}

	_ = server.store.IncrementReelLikes(ctx, reelID)

	// WebSocket event to owner
	reel, err := server.store.GetReel(ctx, reelID)
	if err == nil {
		// Get liker username
		liker, _ := server.store.GetUserByID(ctx, authPayload.UserID)
		server.hub.BroadcastReelLiked(reelID, reel.UserID, authPayload.UserID, liker.Username)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "liked"})
}

// unlikeReel unlikes a reel.
func (server *Server) unlikeReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if err := server.store.UnlikeReel(ctx, db.UnlikeReelParams{
		ReelID: reelID,
		UserID: authPayload.UserID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.DecrementReelLikes(ctx, reelID)
	ctx.JSON(http.StatusOK, gin.H{"message": "unliked"})
}

// addReelComment adds a comment to a reel.
func (server *Server) addReelComment(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req reelCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	comment, err := server.store.CreateReelComment(ctx, db.CreateReelCommentParams{
		ReelID:  reelID,
		UserID:  authPayload.UserID,
		Content: req.Content,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.IncrementReelComments(ctx, reelID)

	// WebSocket event to owner
	reel, err := server.store.GetReel(ctx, reelID)
	if err == nil {
		commenter, _ := server.store.GetUserByID(ctx, authPayload.UserID)
		server.hub.BroadcastReelCommented(reelID, reel.UserID, authPayload.UserID, commenter.Username, comment)
	}

	ctx.JSON(http.StatusCreated, toReelCommentResponse(comment))
}

// listReelComments returns comments for a reel.
func (server *Server) listReelComments(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	comments, err := server.store.ListReelComments(ctx, reelID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]reelCommentResponse, len(comments))
	for i, c := range comments {
		rsp[i] = toReelCommentResponseFromList(c)
	}

	ctx.JSON(http.StatusOK, rsp)
}

// saveReel saves a reel to bookmarks.
func (server *Server) saveReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	_, err = server.store.SaveReel(ctx, db.SaveReelParams{
		ReelID: reelID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	_ = server.store.IncrementReelSaves(ctx, reelID)
	ctx.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// unsaveReel removes a reel from bookmarks.
func (server *Server) unsaveReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if err := server.store.UnsaveReel(ctx, db.UnsaveReelParams{
		ReelID: reelID,
		UserID: authPayload.UserID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_ = server.store.DecrementReelSaves(ctx, reelID)
	ctx.JSON(http.StatusOK, gin.H{"message": "unsaved"})
}

// shareReel increments the share counter and broadcasts activity.
func (server *Server) shareReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.store.IncrementReelShares(ctx, reelID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Wait, is there a broadcast for sharing a reel? Reusing existing pattern if so, otherwise just success
	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// deleteReelComment deletes a comment the user owns.
func (server *Server) deleteReelComment(ctx *gin.Context) {
	commentID, err := uuid.Parse(ctx.Param("commentId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	reelID, err := server.store.DeleteReelComment(ctx, db.DeleteReelCommentParams{
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

	_ = server.store.DecrementReelComments(ctx, reelID)
	ctx.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// deleteReel deletes a reel the user owns.
func (server *Server) deleteReel(ctx *gin.Context) {
	reelID, err := uuid.Parse(ctx.Param("id"))
	if (err != nil) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid reel id")))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err = server.store.DeleteReel(ctx, db.DeleteReelParams{
		ID:     reelID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "reel deleted"})
}
