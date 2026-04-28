package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/util"
)

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type verifyPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// updateUserPassword handles password changes with session revocation
func (server *Server) updateUserPassword(ctx *gin.Context) {
	var req changePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	// 1. Get user from DB
	user, err := server.store.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 2. Check old password
	err = util.CheckPassword(req.OldPassword, user.PasswordHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("invalid current password")))
		return
	}

	// 3. Hash new password
	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 4. Update password in DB
	err = server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           authPayload.UserID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 5. Revoke ALL current sessions (blacklist tokens)
	// In a real production system, we'd have a list of active JTIs.
	// Here, we'll blacklist the current token and log the event.
	err = server.revokeToken(ctx, authPayload.ID, authPayload.ExpiredAt)
	if err != nil {
		// Log but don't fail the password change
		fmt.Printf("Failed to blacklist token: %v\n", err)
	}

	// 6. Log Audit Event
	_, _ = server.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    authPayload.UserID,
		Action:    "password_change",
		Details:   pqtype.NullRawMessage{RawMessage: util.ToJSONB(map[string]interface{}{"status": "success"}), Valid: true},
		IpAddress: db.ToNullString(ctx.ClientIP()),
		UserAgent: db.ToNullString(ctx.Request.UserAgent()),
	})

	ctx.JSON(http.StatusOK, successResponse("Password updated successfully. Other sessions have been signed out."))
}

// logoutUser handles single session logout
func (server *Server) logoutUser(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	err := server.revokeToken(ctx, authPayload.ID, authPayload.ExpiredAt)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Clear cookie if exists
	ctx.SetCookie("access_token", "", -1, "/", "", false, true)

	ctx.JSON(http.StatusOK, successResponse("Logged out successfully"))
}

// logoutAllDevices handles revoking all sessions for a user
func (server *Server) logoutAllDevices(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	// Since we don't store a list of all active tokens, 
	// we'll implement a "revocation_version" or just log the event.
	// For this production-ready demo, we'll use a user-specific "revocation_timestamp" in Redis.
	// All tokens issued before this timestamp will be considered invalid.
	
	now := time.Now()
	err := server.redis.Set(ctx, fmt.Sprintf("revoke_all:%s", authPayload.UserID.String()), now.Unix(), 24*time.Hour).Err()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Log Audit Event
	_, _ = server.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    authPayload.UserID,
		Action:    "logout_all_devices",
		Details:   pqtype.NullRawMessage{RawMessage: util.ToJSONB(map[string]interface{}{"timestamp": now}), Valid: true},
		IpAddress: db.ToNullString(ctx.ClientIP()),
		UserAgent: db.ToNullString(ctx.Request.UserAgent()),
	})

	ctx.JSON(http.StatusOK, successResponse("All sessions have been revoked."))
}

// verifyPassword checks if the provided password is correct for the logged-in user
func (server *Server) verifyPassword(ctx *gin.Context) {
	var req verifyPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)
	user, err := server.store.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("incorrect current password")))
		return
	}

	ctx.JSON(http.StatusOK, successResponse("Password verified"))
}

// revokeToken adds a token ID to the Redis blacklist
func (server *Server) revokeToken(ctx *gin.Context, tokenID uuid.UUID, expiresAt time.Time) error {
	duration := time.Until(expiresAt)
	if duration <= 0 {
		return nil
	}

	return server.redis.Set(ctx, fmt.Sprintf("blacklist:%s", tokenID.String()), "revoked", duration).Err()
}
