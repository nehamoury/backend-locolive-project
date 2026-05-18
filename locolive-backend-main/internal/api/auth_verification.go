package api

import (
	"fmt"
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
		errMsg := err.Error()
		if errMsg == "invalid or expired verification token" || errMsg == "verification token has expired" {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
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
	token := util.RandomDigitString(6)
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
	err = server.mailer.SendOTPEmail(user.Email.String, token)
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

type verifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

func (server *Server) verifyEmailOTP(ctx *gin.Context) {
	var req verifyOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Look up OTP in email_verifications table
	verification, err := server.store.GetEmailVerificationByOTP(ctx, db.GetEmailVerificationByOTPParams{
		Token: req.OTP,
		Email: req.Email,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid or expired verification code")))
		return
	}

	// Mark both email and phone verified → activates the user
	user, err := server.store.VerifyEmailWithOTP(ctx, verification.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Clean up used token
	_ = server.store.DeleteEmailVerification(ctx, verification.UserID)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account verified successfully",
		"user":    dbToUserResponse(user),
	})
}

type dbUserResponse struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	FullName          string `json:"full_name"`
	Email             string `json:"email"`
	IsEmailVerified   bool   `json:"is_email_verified"`
	IsPhoneVerified   bool   `json:"is_phone_verified"`
	IsActive          bool   `json:"is_active"`
	IsProfileComplete bool   `json:"is_profile_complete"`
}

func dbToUserResponse(u db.User) dbUserResponse {
	return dbUserResponse{
		ID:                u.ID.String(),
		Username:          u.Username,
		FullName:          u.FullName,
		Email:             u.Email.String,
		IsEmailVerified:   u.IsEmailVerified,
		IsPhoneVerified:   u.IsPhoneVerified,
		IsActive:          u.IsActive,
		IsProfileComplete: u.IsProfileComplete,
	}
}
