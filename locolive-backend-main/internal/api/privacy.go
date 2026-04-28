package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/privacy"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
)

// Privacy Settings Handlers

type PrivacySettingResponse struct {
	UserID           uuid.UUID `json:"user_id"`
	WhoCanMessage    string    `json:"who_can_message"`
	WhoCanSeeStories string    `json:"who_can_see_stories"`
	ShowLocation     bool      `json:"show_location"`
}

func newPrivacySettingResponse(p db.PrivacySetting) PrivacySettingResponse {
	return PrivacySettingResponse{
		UserID:           p.UserID,
		WhoCanMessage:    p.WhoCanMessage.String,
		WhoCanSeeStories: p.WhoCanSeeStories.String,
		ShowLocation:     p.ShowLocation.Bool,
	}
}

type updatePrivacySettingsRequest struct {
	WhoCanMessage    string `json:"who_can_message" binding:"oneof=everyone connections nobody"`
	WhoCanSeeStories string `json:"who_can_see_stories" binding:"oneof=everyone connections nobody"`
	ShowLocation     *bool  `json:"show_location" binding:"required"`
}

func (server *Server) updatePrivacySettings(ctx *gin.Context) {
	var req updatePrivacySettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	settings, err := server.store.UpsertPrivacySettings(ctx, db.UpsertPrivacySettingsParams{
		UserID:           payload.UserID,
		WhoCanMessage:    sql.NullString{String: req.WhoCanMessage, Valid: true},
		WhoCanSeeStories: sql.NullString{String: req.WhoCanSeeStories, Valid: true},
		ShowLocation:     sql.NullBool{Bool: *req.ShowLocation, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newPrivacySettingResponse(settings))
}

func (server *Server) getPrivacySettings(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	settings, err := server.store.GetPrivacySettings(ctx, payload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			ctx.JSON(http.StatusOK, PrivacySettingResponse{
				UserID:           payload.UserID,
				WhoCanMessage:    "connections",
				WhoCanSeeStories: "connections",
				ShowLocation:     true,
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newPrivacySettingResponse(settings))
}

// Blocking Handlers

type blockUserRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

func (server *Server) blockUser(ctx *gin.Context) {
	var req blockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	blockID, ok := parseUUIDParam(ctx, req.UserID, "user_id")
	if !ok {
		return
	}

	if payload.UserID == blockID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot block yourself"})
		return
	}

	_, err := server.store.BlockUser(ctx, db.BlockUserParams{
		BlockerID: payload.UserID,
		BlockedID: blockID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate ALL relevant caches
	server.invalidateProfileCache(payload.UserID)
	server.invalidateProfileCache(blockID)
	server.redis.Del(context.Background(), "connections:"+payload.UserID.String())
	server.privacy.InvalidateBlockCache(ctx, payload.UserID, blockID)

	// Audit log
	server.privacy.LogAction(ctx, payload.UserID, privacy.AuditActionUserBlocked,
		map[string]interface{}{"blocked_user_id": blockID.String()},
		ctx.ClientIP(), ctx.Request.UserAgent(),
	)

	// Real-time: notify blocked user to refresh state
	server.hub.BroadcastForceLogout(blockID, "blocked")

	ctx.JSON(http.StatusOK, gin.H{"message": "user blocked"})
}

func (server *Server) unblockUser(ctx *gin.Context) {
	targetIDStr := ctx.Param("id")
	targetID, ok := parseUUIDParam(ctx, targetIDStr, "user_id")
	if !ok {
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err := server.store.UnblockUser(ctx, db.UnblockUserParams{
		BlockerID: payload.UserID,
		BlockedID: targetID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate block cache
	server.privacy.InvalidateBlockCache(ctx, payload.UserID, targetID)

	// Audit log
	server.privacy.LogAction(ctx, payload.UserID, privacy.AuditActionUserUnblocked,
		map[string]interface{}{"unblocked_user_id": targetID.String()},
		ctx.ClientIP(), ctx.Request.UserAgent(),
	)

	ctx.JSON(http.StatusOK, gin.H{"message": "user unblocked"})
}

type BlockedUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	AvatarUrl string    `json:"avatar_url"`
	BlockedAt time.Time `json:"blocked_at"`
}

func (server *Server) getBlockedUsers(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	users, err := server.store.GetBlockedUsers(ctx, payload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]BlockedUserResponse, len(users))
	for i, u := range users {
		rsp[i] = BlockedUserResponse{
			ID:        u.ID,
			Username:  u.Username,
			FullName:  u.FullName,
			AvatarUrl: u.AvatarUrl.String,
			BlockedAt: u.BlockedAt.Time,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

// Location Privacy

type toggleGhostModeRequest struct {
	Enabled  bool `json:"enabled"`
	Duration int  `json:"duration"` // minutes, 0 = indefinite
}

func (server *Server) toggleGhostMode(ctx *gin.Context) {
	var req toggleGhostModeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	var expiresAt sql.NullTime
	if req.Enabled && req.Duration > 0 {
		expiresAt = sql.NullTime{
			Time:  time.Now().Add(time.Duration(req.Duration) * time.Minute),
			Valid: true,
		}
	}

	user, err := server.store.ToggleGhostMode(ctx, db.ToggleGhostModeParams{
		ID:                 payload.UserID,
		IsGhostMode:        req.Enabled,
		GhostModeExpiresAt: expiresAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate privacy cache
	server.privacy.InvalidateUserCache(ctx, payload.UserID)

	// Audit log
	action := privacy.AuditActionGhostEnabled
	if !req.Enabled {
		action = privacy.AuditActionGhostDisabled
	}
	server.privacy.LogAction(ctx, payload.UserID, action,
		map[string]interface{}{"duration_min": req.Duration},
		ctx.ClientIP(), ctx.Request.UserAgent(),
	)

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) panicMode(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// 1. Set panic_mode flag
	_, err := server.store.TogglePanicMode(ctx, db.TogglePanicModeParams{
		ID:        payload.UserID,
		PanicMode: true,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 2. Redis cleanup
	server.redis.ZRem(ctx, "users:locations", payload.UserID.String())
	server.redis.Del(ctx, "profile:"+payload.UserID.String())
	server.privacy.InvalidateUserCache(ctx, payload.UserID)

	// 3. GLOBAL SESSION REVOCATION (Logout all devices)
	now := time.Now()
	server.redis.Set(ctx, fmt.Sprintf("revoke_all:%s", payload.UserID.String()), now.Unix(), 24*time.Hour)

	// 4. Audit log (CRITICAL)
	_, _ = server.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    payload.UserID,
		Action:    "panic_mode_activated",
		Details:   pqtype.NullRawMessage{RawMessage: util.ToJSONB(map[string]interface{}{"status": "emergency"}), Valid: true},
		IpAddress: db.ToNullString(ctx.ClientIP()),
		UserAgent: db.ToNullString(ctx.Request.UserAgent()),
	})

	// 5. Force disconnect WebSockets
	server.hub.BroadcastForceLogout(payload.UserID, "panic_mode")

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Panic protocol complete. All traces scrubbed and sessions revoked.",
		"status":  "scrubbed",
	})
}


type updateAccountPrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

func (server *Server) updateAccountPrivacy(ctx *gin.Context) {
	var req updateAccountPrivacyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.store.GetUserByID(ctx, payload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	oldIsPrivate := user.IsPrivate

	updatedUser, err := server.store.UpdateAccountPrivacy(ctx, db.UpdateAccountPrivacyParams{
		ID:        payload.UserID,
		IsPrivate: req.IsPrivate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_, _ = server.store.LogPrivacyChange(ctx, db.LogPrivacyChangeParams{
		UserID:   payload.UserID,
		OldValue: oldIsPrivate,
		NewValue: req.IsPrivate,
	})

	server.invalidateProfileCache(payload.UserID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":    "Privacy updated",
		"is_private": updatedUser.IsPrivate,
	})
}



// canViewContent checks if viewerID can view content owned by ownerID.
// Delegates to the centralized privacy service.
func (server *Server) canViewContent(ctx *gin.Context, viewerID, ownerID uuid.UUID) (bool, string, error) {
	result := server.privacy.CanViewProfile(ctx, viewerID, ownerID)

	if !result.Allowed {
		return false, string(result.Reason), nil
	}

	// CanViewProfile returns Allowed=true with Reason=private for private accounts
	// that viewer isn't following. The API handler uses this to blank out sensitive data.
	if result.Reason == privacy.ReasonPrivate {
		return false, "private", nil
	}

	return true, "", nil
}

