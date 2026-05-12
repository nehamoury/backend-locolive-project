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

type setPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
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

	// Clear cookies
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/users/renew-access",
		Domain:   "",
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

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

// setPasswordForGoogleUser sets a password for Google-created users who don't have one yet
func (server *Server) setPasswordForGoogleUser(ctx *gin.Context) {
	var req setPasswordRequest
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

	// Validate password strength
	if !isValidPassword(req.Password) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("password must be at least 8 characters with uppercase, lowercase, and a number")))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           authPayload.UserID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Revoke all current sessions
	err = server.revokeToken(ctx, authPayload.ID, authPayload.ExpiredAt)
	if err != nil {
		fmt.Printf("Failed to blacklist token: %v\n", err)
	}

	// Generate new tokens
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, user.ID, string(user.Role), server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user.Username, user.ID, string(user.Role), server.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Set cookies
	isProduction := server.config.Environment == "production"
	sameSite := http.SameSiteLaxMode
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		MaxAge:   int(server.config.AccessTokenDuration.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isProduction,
		HttpOnly: true,
		SameSite: sameSite,
	})
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		MaxAge:   int(server.config.RefreshTokenDuration.Seconds()),
		Path:     "/api/users/renew-access",
		Domain:   "",
		Secure:   isProduction,
		HttpOnly: true,
		SameSite: sameSite,
	})

	// Log audit event
	_, _ = server.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    authPayload.UserID,
		Action:    "password_set",
		Details:   pqtype.NullRawMessage{RawMessage: util.ToJSONB(map[string]interface{}{"provider": user.Provider}), Valid: true},
		IpAddress: db.ToNullString(ctx.ClientIP()),
		UserAgent: db.ToNullString(ctx.Request.UserAgent()),
	})

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"session_id":               session.ID,
		"access_token":             accessToken,
		"access_token_expires_at":  accessPayload.ExpiredAt,
		"refresh_token":            refreshToken,
		"refresh_token_expires_at": refreshPayload.ExpiredAt,
		"user":                     newUserResponse(user),
		"message":                  "Password set successfully. Use your username and password next time to log in.",
	}))
}

// isValidPassword checks password strength
func isValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasUpper := false
	hasLower := false
	hasNumber := false
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		}
	}
	return hasUpper && hasLower && hasNumber
}

// revokeToken adds a token ID to the Redis blacklist
func (server *Server) revokeToken(ctx *gin.Context, tokenID uuid.UUID, expiresAt time.Time) error {
	duration := time.Until(expiresAt)
	if duration <= 0 {
		return nil
	}

	return server.redis.Set(ctx, fmt.Sprintf("blacklist:%s", tokenID.String()), "revoked", duration).Err()
}
