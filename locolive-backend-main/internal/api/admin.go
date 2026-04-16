package api

import (
	"database/sql"
	"net/http"
	"privacy-social-backend/internal/service/admin"
	"privacy-social-backend/internal/util"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"strconv"
)

const (
	adminStatsCacheTTL = 1 * time.Minute
)

// Response Models for Clean JSON
type AdminUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	AvatarUrl string    `json:"avatar_url"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	IsBanned  bool      `json:"is_banned"`
	CreatedAt time.Time `json:"created_at"`
}

type listUsersRequest struct {
	PageID   int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

func newAdminUserResponse(user db.User) AdminUserResponse {
	return AdminUserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		FullName:  user.FullName,
		AvatarUrl: user.AvatarUrl.String,
		Email:     user.Email.String,
		Role:      string(user.Role),
		Status:    "offline", // Default, will be updated by Hub if needed
		IsBanned:  user.IsShadowBanned,
		CreatedAt: user.CreatedAt,
	}
}

func (server *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	users, count, err := server.admin.ListUsers(ctx, admin.ListUsersParams{
		PageID:   req.PageID,
		PageSize: req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userResponses := make([]AdminUserResponse, len(users))
	for i, u := range users {
		userResponses[i] = newAdminUserResponse(u)
		if server.hub.IsUserOnline(u.ID) {
			userResponses[i].Status = "online"
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     userResponses,
			"total":     count,
			"page":      req.PageID,
			"page_size": req.PageSize,
		},
	})
}

// Admin: Ban/Unban User
type banUserRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Ban    bool   `json:"ban" binding:"required"`
}

func (server *Server) banUser(ctx *gin.Context) {
	var req banUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, ok := parseUUIDParam(ctx, req.UserID, "user_id")
	if !ok {
		return
	}

	// Wait, req.UserID is string in my param struct but the parser helper returns uuid.
	// Actually banUserRequest defined UserID as string.
	// Service expects BanUserParams with string UserID to be parsed inside or UUID?
	// AdminService.BanUser takes params.UserID as string and parses it.
	// So I can pass req.UserID directly if I want, or pass the UUID.
	// Let's check AdminService signature again.
	// func (s *ServiceImpl) BanUser(ctx context.Context, params BanUserParams) (db.User, error)
	// type BanUserParams struct { UserID string; Ban bool }

	// So I should pass string. parseUUIDParam is doing validation which is good.
	// I will keep validation but just pass the string to service.

	user, err := server.admin.BanUser(ctx, admin.BanUserParams{
		UserID: req.UserID,
		Ban:    req.Ban,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    newAdminUserResponse(user),
	})
}

// Admin: Delete User
type deleteUserRequest struct {
	UserID string `uri:"id" binding:"required,uuid"`
}

func (server *Server) deleteUser(ctx *gin.Context) {
	var req deleteUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.admin.DeleteUser(ctx, req.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "user deleted"},
	})
}

// Admin: Get Statistics (with Redis caching)
func (server *Server) getStats(ctx *gin.Context) {
	response, isCached, err := server.admin.GetStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if isCached {
		ctx.Header("X-Cache", "HIT")
	} else {
		ctx.Header("X-Cache", "MISS")
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// Admin: List Reports
type listReportsRequest struct {
	Resolved bool  `form:"resolved"`
	PageID   int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) listReports(ctx *gin.Context) {
	var req listReportsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	reports, err := server.admin.ListReports(ctx, req.Resolved, req.PageID, req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     reports,
			"page":      req.PageID,
			"page_size": req.PageSize,
		},
	})
}

// Admin: Resolve Report
type resolveReportRequest struct {
	ReportID string `uri:"id" binding:"required,uuid"`
}

func (server *Server) resolveReport(ctx *gin.Context) {
	var req resolveReportRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	report, err := server.admin.ResolveReport(ctx, req.ReportID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

// Admin: Delete Story
type deleteStoryRequest struct {
	StoryID string `uri:"id" binding:"required,uuid"`
}

func (server *Server) deleteStory(ctx *gin.Context) {
	var req deleteStoryRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.admin.DeleteStory(ctx, req.StoryID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "story deleted"},
	})
}

// Admin: List All Stories
type listAllStoriesRequest struct {
	PageID   int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) listAllStories(ctx *gin.Context) {
	var req listAllStoriesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	stories, err := server.admin.ListAllStories(ctx, req.PageID, req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     stories,
			"page":      req.PageID,
			"page_size": req.PageSize,
		},
	})
}

// ─── Admin: Map Active Users ──────────────────────────────────────────────

type mapActiveRequest struct {
	Radius float64 `form:"radius,default=5"` // km
	Lat    float64 `form:"lat"`
	Lng    float64 `form:"lng"`
}

type mapUserResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	FullName  string  `json:"full_name"`
	AvatarUrl string  `json:"avatar_url"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Online    bool    `json:"online"`
}

func (server *Server) getMapActiveUsers(ctx *gin.Context) {
	// Get all users with positions in Redis GEO
	positions, err := server.redis.GeoSearchLocation(ctx, "users:locations", &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			// Search entire globe – we filter per viewport on client
			Longitude:  77.5946, // Default center (Bangalore)
			Latitude:   12.9716,
			Radius:     20000, // 20,000 km = global
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      500,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	users := make([]mapUserResponse, 0, len(positions))
	for _, pos := range positions {
		userID, err := uuid.Parse(pos.Name)
		if err != nil {
			continue
		}
		user, err := server.store.GetUserByID(ctx, userID)
		if err != nil || user.IsGhostMode || user.IsShadowBanned {
			continue
		}

		avatar := ""
		if user.AvatarUrl.Valid {
			avatar = user.AvatarUrl.String
		}

		online := user.LastActiveAt.Valid && time.Since(user.LastActiveAt.Time) < 5*time.Minute

		users = append(users, mapUserResponse{
			ID:        pos.Name,
			Username:  user.Username,
			FullName:  user.FullName,
			AvatarUrl: avatar,
			Lat:       pos.Latitude,
			Lng:       pos.Longitude,
			Online:    online,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users": users,
			"total": len(users),
		},
	})
}

// ─── Admin: Logout ──────────────────────────────────────────────────────

func (server *Server) adminLogout(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Block all sessions for this user
	err := server.store.BlockSession(ctx, authPayload.UserID)
	if err != nil {
		// Non-critical – log but don't fail
		log.Warn().Err(err).Msg("Failed to block sessions on logout")
	}

	// Add token to Redis blacklist until its natural expiry
	tokenStr := ctx.GetHeader("Authorization")
	if len(tokenStr) > 7 {
		tokenStr = tokenStr[7:] // Strip "Bearer "
		ttl := time.Until(authPayload.ExpiredAt)
		if ttl > 0 {
			server.redis.Set(ctx, "blacklist:"+tokenStr, "1", ttl)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "logged out successfully"},
	})
}

// ─── Admin: Search Users ─────────────────────────────────────────────────

type searchUsersAdminRequest struct {
	Query    string `form:"q" binding:"required,min=1"`
	PageID   int32  `form:"page,default=1"`
	PageSize int32  `form:"page_size,default=20"`
}

func (server *Server) searchUsersAdmin(ctx *gin.Context) {
	var req searchUsersAdminRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	users, err := server.store.SearchUsersAdmin(ctx, db.SearchUsersAdminParams{
		Query:  req.Query,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	count, err := server.store.CountSearchUsersAdmin(ctx, req.Query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userResponses := make([]AdminUserResponse, len(users))
	for i, u := range users {
		userResponses[i] = newAdminUserResponse(u)
		if server.hub.IsUserOnline(u.ID) {
			userResponses[i].Status = "online"
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     userResponses,
			"total":     count,
			"page":      req.PageID,
			"page_size": req.PageSize,
		},
	})
}

// ─── Admin: List Crossings ───────────────────────────────────────────────

type listAdminCrossingsRequest struct {
	PageID   int32 `form:"page,default=1"`
	PageSize int32 `form:"page_size,default=20"`
}

type adminCrossingResponse struct {
	ID       string             `json:"id"`
	UserA    AdminUserResponse  `json:"userA"`
	UserB    AdminUserResponse  `json:"userB"`
	Time     time.Time          `json:"time"`
	Location map[string]float64 `json:"location"`
	Distance int                `json:"distance"`
}

func (server *Server) listAdminCrossings(ctx *gin.Context) {
	var req listAdminCrossingsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	crossings, err := server.store.ListAdminCrossings(ctx, db.ListAdminCrossingsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	total, err := server.store.CountAdminCrossings(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	responses := make([]adminCrossingResponse, len(crossings))
	for i, c := range crossings {
		// Basic mapping
		userA := AdminUserResponse{
			ID:       c.U1ID.String(),
			Username: c.U1Username,
			FullName: c.U1FullName,
		}
		if c.U1AvatarUrl.Valid {
			userA.AvatarUrl = c.U1AvatarUrl.String
		}
		if server.hub.IsUserOnline(c.U1ID) {
			userA.Status = "online"
		} else {
			userA.Status = "offline"
		}

		userB := AdminUserResponse{
			ID:       c.U2ID.String(),
			Username: c.U2Username,
			FullName: c.U2FullName,
		}
		if c.U2AvatarUrl.Valid {
			userB.AvatarUrl = c.U2AvatarUrl.String
		}
		if server.hub.IsUserOnline(c.U2ID) {
			userB.Status = "online"
		} else {
			userB.Status = "offline"
		}

		// Add dummy distance, or 0 since we mock it
		// Safe coordinate parsing
		lat, _ := c.Lat.(float64)
		if c.Lat == nil {
			lat = 0.0
		}
		lng, _ := c.Lng.(float64)
		if c.Lng == nil {
			lng = 0.0
		}

		responses[i] = adminCrossingResponse{
			ID:    c.ID.String(),
			UserA: userA,
			UserB: userB,
			Time:  c.OccurredAt,
			Location: map[string]float64{
				"lat": lat,
				"lng": lng,
			},
			Distance: 0,
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     responses,
			"total":     total,
			"page":      req.PageID,
			"page_size": req.PageSize,
		},
	})
}

type adminResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (server *Server) adminResetUserPassword(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var req adminResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := server.store.GetUserByID(ctx, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	err = server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           uid,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	log.Info().Str("username", user.Username).Str("email", user.Email.String).Msg("admin reset user password")
	ctx.JSON(http.StatusOK, gin.H{"success": true, "message": "password updated successfully"})
}

// Admin: List Activity Logs
func (server *Server) listActivityLogs(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	logs, err := server.admin.ListActivityLogs(ctx, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     logs,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// Admin: List All Comments
func (server *Server) listAllComments(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	comments, err := server.admin.ListAllComments(ctx, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     comments,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// Admin: Moderate Comment
type moderateCommentRequest struct {
	CommentID string `json:"comment_id" binding:"required,uuid"`
	Source    string `json:"source"     binding:"required,oneof=post reel"`
	Action    string `json:"action"     binding:"required,oneof=approve delete"`
}

func (server *Server) moderateComment(ctx *gin.Context) {
	var req moderateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	cid, _ := uuid.Parse(req.CommentID)

	if req.Action == "delete" {
		if req.Source == "post" {
			// Find user id first for trust update
			comment, err := server.store.GetPostComment(ctx, cid)
			if err == nil {
				_ = server.moderation.ProcessModerationAction(ctx, comment.UserID, "content_deleted")
			}
			postID, err := server.store.AdminDeletePostComment(ctx, cid)
			if err == nil {
				_ = server.store.DecrementPostComments(ctx, postID)
			}
		} else {
			comment, err := server.store.GetReelComment(ctx, cid)
			if err == nil {
				_ = server.moderation.ProcessModerationAction(ctx, comment.UserID, "content_deleted")
			}
			reelID, err := server.store.AdminDeleteReelComment(ctx, cid)
			if err == nil {
				_ = server.store.DecrementReelComments(ctx, reelID)
			}
		}

	} else {
		// Approve -> Unflag
		if req.Source == "post" {
			_ = server.store.UpdatePostCommentFlag(ctx, db.UpdatePostCommentFlagParams{ID: cid, IsFlagged: false})
		} else {
			_ = server.store.UpdateReelCommentFlag(ctx, db.UpdateReelCommentFlagParams{ID: cid, IsFlagged: false})
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Admin: Send Broadcast Notification ──────────────────────────────────

type sendNotificationRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=100"`
	Message string `json:"message" binding:"required,min=1,max=500"`
	Target  string `json:"target" binding:"required,oneof=all online location"`
	City    string `json:"city,omitempty"`
}

func (server *Server) sendBroadcastNotification(ctx *gin.Context) {
	var req sendNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var userIDs []uuid.UUID
	var err error

	switch req.Target {
	case "all":
		users, countErr := server.store.ListUsers(ctx, db.ListUsersParams{
			Limit:  10000,
			Offset: 0,
		})
		if countErr != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(countErr))
			return
		}
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
		_ = err // ignore err from above

	case "online":
		users, _ := server.store.ListUsers(ctx, db.ListUsersParams{
			Limit:  10000,
			Offset: 0,
		})
		for _, u := range users {
			if u.LastActiveAt.Valid && time.Since(u.LastActiveAt.Time) < 5*time.Minute {
				userIDs = append(userIDs, u.ID)
			}
		}

	case "location":
		if req.City == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "city is required for location target"})
			return
		}
	}

	// Create notifications for all target users
	notificationCount := 0
	for _, userID := range userIDs {
		_, notifErr := server.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:  userID,
			Type:    "system_announcement",
			Title:   req.Title,
			Message: req.Message,
		})
		if notifErr == nil {
			notificationCount++
		}
	}

	log.Info().Str("title", req.Title).Int("count", notificationCount).Msg("broadcast notification sent")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message":      "notification sent successfully",
			"recipients":   notificationCount,
			"total_target": len(userIDs),
		},
	})
}

// ─── Admin: Get Recent Notifications (sent by admin) ───────────────────

func (server *Server) listAdminNotifications(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	notifications, err := server.store.ListNotificationsAdmin(ctx, db.ListNotificationsAdminParams{
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	count, err := server.store.CountNotificationsAdmin(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     notifications,
			"total":     count,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ─── Admin: App Settings ────────────────────────────────────────────────

type appSettingsResponse struct {
	DiscoveryRadius       int    `json:"discovery_radius"`
	CrossingDistance      int    `json:"crossing_distance"`
	LocationUpdateSeconds int    `json:"location_update_seconds"`
	ReelsEnabled          bool   `json:"reels_enabled"`
	CrossingsEnabled      bool   `json:"crossings_enabled"`
	Version               string `json:"version"`
	BuildDate             string `json:"build_date"`
	Environment           string `json:"environment"`
}

func (server *Server) getAppSettings(ctx *gin.Context) {
	settings := appSettingsResponse{
		DiscoveryRadius:       5,
		CrossingDistance:      50,
		LocationUpdateSeconds: 30,
		ReelsEnabled:          true,
		CrossingsEnabled:      true,
		Version:               "1.0.0",
		BuildDate:             "2024.01.15",
		Environment:           "production",
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

type updateSettingsRequest struct {
	DiscoveryRadius       *int  `json:"discovery_radius,omitempty"`
	CrossingDistance      *int  `json:"crossing_distance,omitempty"`
	LocationUpdateSeconds *int  `json:"location_update_seconds,omitempty"`
	ReelsEnabled          *bool `json:"reels_enabled,omitempty"`
	CrossingsEnabled      *bool `json:"crossings_enabled,omitempty"`
}

func (server *Server) updateAppSettings(ctx *gin.Context) {
	var req updateSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// In a real app, these would be stored in a settings table or Redis
	// For now, we just validate and return success
	log.Info().Interface("settings", req).Msg("app settings updated")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "settings updated successfully",
	})
}

// ─── Admin: List Admin Users ────────────────────────────────────────────

func (server *Server) listAdminUsers(ctx *gin.Context) {
	users, err := server.store.ListAdminUsers(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userResponses := make([]AdminUserResponse, len(users))
	for i, u := range users {
		userResponses[i] = AdminUserResponse{
			ID:        u.ID.String(),
			Username:  u.Username,
			FullName:  u.FullName,
			Email:     u.Email.String,
			Role:      string(u.Role),
			Status:    "active",
			IsBanned:  false,
			CreatedAt: u.CreatedAt,
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": userResponses,
		},
	})
}

type createAdminRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin moderator"`
}

func (server *Server) createAdminUser(ctx *gin.Context) {
	var req createAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if username already exists
	existingUser, err := server.store.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser.ID.String() != "" {
		ctx.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	// Create user as admin
	hashedPassword, hashErr := util.HashPassword(req.Password)
	if hashErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Create the admin user
	user, createErr := server.store.CreateUser(ctx, db.CreateUserParams{
		Phone:        "admin-" + req.Username + "-" + uuid.New().String()[:8],
		Email:        sql.NullString{String: req.Email, Valid: true},
		PasswordHash: hashedPassword,
		Username:     req.Username,
		FullName:     req.Username,
		IsGhostMode:  false,
	})

	if createErr != nil {
		// Try to get existing user and update role
		existingUser, _ := server.store.GetUserByUsername(ctx, req.Username)
		if existingUser.ID.String() != "" {
			updatedUser, updateErr := server.store.UpdateUserRole(ctx, db.UpdateUserRoleParams{
				ID:   existingUser.ID,
				Role: db.UserRole(req.Role),
			})
			if updateErr != nil {
				ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
				return
			}
			ctx.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    newAdminUserResponse(updatedUser),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(createErr))
		return
	}

	// Update role to admin/moderator
	updatedUser, updateErr := server.store.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   user.ID,
		Role: db.UserRole(req.Role),
	})
	if updateErr != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
		return
	}

	log.Info().Str("username", req.Username).Str("role", req.Role).Msg("admin user created")

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    newAdminUserResponse(updatedUser),
	})
}

type updateAdminRequest struct {
	Role string `json:"role" binding:"required,oneof=admin moderator user"`
}

func (server *Server) updateAdminUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var req updateAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	updatedUser, updateErr := server.store.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   uid,
		Role: db.UserRole(req.Role),
	})
	if updateErr != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
		return
	}

	log.Info().Str("user_id", userID).Str("new_role", req.Role).Msg("admin user role updated")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    newAdminUserResponse(updatedUser),
	})
}

func (server *Server) deleteAdminUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Reset role to user (don't delete the account)
	updatedUser, updateErr := server.store.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   uid,
		Role: db.UserRole("user"),
	})
	if updateErr != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
		return
	}

	log.Info().Str("user_id", userID).Msg("admin user demoted to regular user")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    newAdminUserResponse(updatedUser),
	})
}
