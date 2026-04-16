package api

import (
	"database/sql"
	"net/http"
	"time"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/util"

	"github.com/gin-gonic/gin"
)

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (server *Server) forgotPassword(ctx *gin.Context) {
	var req forgotPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUserByEmail(ctx, sql.NullString{String: req.Email, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			// Do not reveal email existence for security
			ctx.JSON(http.StatusOK, gin.H{"message": "If this email exists, a reset link has been sent."})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Generate Token
	resetToken := util.RandomString(32)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Save to DB
	_, err = server.store.CreatePasswordReset(ctx, db.CreatePasswordResetParams{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Send Email
	err = server.mailer.SendResetEmail(req.Email, resetToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "If this email exists, a reset link has been sent."})
}

type verifyResetTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func (server *Server) verifyResetToken(ctx *gin.Context) {
	var req verifyResetTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, err := server.store.GetPasswordResetByToken(ctx, req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "token is valid"})
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (server *Server) resetPassword(ctx *gin.Context) {
	var req resetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Find token
	reset, err := server.store.GetPasswordResetByToken(ctx, req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Hash new password
	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Update user password
	err = server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           reset.UserID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// One-time use: Delete all tokens for this user
	err = server.store.DeleteUserPasswordResets(ctx, reset.UserID)
	if err != nil {
		// Log error but don't fail for user
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
