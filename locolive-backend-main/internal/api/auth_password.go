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
	Email string `json:"email" binding:"required"` // This field can now be email or username
}

func (server *Server) forgotPassword(ctx *gin.Context) {
	var req forgotPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var user db.User
	var err error

	// 1. Try finding by Email (case-insensitive handled by DB LOWER() query)
	user, err = server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// 2. Fallback: Try finding by Username (case-insensitive handled by DB LOWER() query)
			user, err = server.store.GetUserByUsername(ctx, req.Email)
			if err != nil {
				if err == sql.ErrNoRows {
					// Security: Do not reveal user existence
					ctx.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
					return
				}
				ctx.JSON(http.StatusInternalServerError, errorResponse(err))
				return
			}
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	// 3. Ensure user has an email address
	if !user.Email.Valid || user.Email.String == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "This account does not have an email address associated with it. Please contact support."})
		return
	}

	// 4. Generate Token
	resetToken := util.RandomString(32)
	expiresAt := time.Now().Add(15 * time.Minute)

	// 5. Save to DB
	_, err = server.store.CreatePasswordReset(ctx, db.CreatePasswordResetParams{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 6. Send Email to the REGISTERED email
	err = server.mailer.SendResetEmail(user.Email.String, resetToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
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
