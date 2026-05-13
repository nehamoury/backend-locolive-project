package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
)

type verifyFirebasePhoneRequest struct {
	FirebaseToken string `json:"firebase_token" binding:"required"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.user.VerifyEmail(ctx, req.Token)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
		"user":    user,
	})
}


func (server *Server) resendEmailVerification(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.user.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if user.IsEmailVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
		return
	}

	// Generate new token
	token := util.RandomString(32)
	_, err = server.store.CreateEmailVerification(ctx, db.CreateEmailVerificationParams{
		UserID:    user.ID,
		Email:     user.Email.String,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Send email
	err = server.mailer.SendVerificationEmail(user.Email.String, token)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}


// verifyFirebasePhone verifies a Firebase phone auth token and marks the user's phone as verified
func (server *Server) verifyFirebasePhone(ctx *gin.Context) {
	var req verifyFirebasePhoneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Verify Firebase token and extract phone number
	phone, err := util.VerifyFirebasePhoneToken(req.FirebaseToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Get user from context (must be authenticated)
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.user.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Validate that the phone number matches the user's phone
	if user.Phone != phone {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Phone number does not match your account"})
		return
	}

	// Mark phone as verified
	updatedUser, err := server.user.VerifyPhone(ctx, authPayload.UserID, "firebase_verified")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Phone verified successfully via Firebase",
		"user":    newUserResponse(updatedUser),
	})
}

func (server *Server) testEmail(ctx *gin.Context) {
	to := ctx.Query("to")
	if to == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "to query parameter is required"})
		return
	}

	err := server.mailer.SendVerificationEmail(to, "test-token-123")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Check VPS logs for more info",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Test email sent successfully to " + to,
	})
}
