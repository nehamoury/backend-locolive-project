package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
	"github.com/rs/zerolog/log"
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

type verifyPhoneRequest struct {
	Code string `json:"code" binding:"required"`
}

func (server *Server) verifyPhone(ctx *gin.Context) {
	var req verifyPhoneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get user ID from token
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	log.Info().Str("user_id", authPayload.UserID.String()).Str("code", req.Code).Msg("Attempting phone verification")

	user, err := server.user.VerifyPhone(ctx, authPayload.UserID, req.Code)
	if err != nil {
		log.Error().Err(err).Str("user_id", authPayload.UserID.String()).Msg("Phone verification failed in service")
		
		// If it's a validation error (like incorrect code), return 400 instead of 500
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Phone verified successfully",
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

func (server *Server) resendPhoneVerification(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.user.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if user.IsPhoneVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Phone is already verified"})
		return
	}

	// ─── RESEND COOLDOWN CHECK (30 seconds) ───────────────────────────────────
	cooldownKey := "otp_cooldown:" + authPayload.UserID.String()
	ttl, err := server.redis.TTL(ctx, cooldownKey).Result()
	if err == nil && ttl > 0 {
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Please wait before requesting a new OTP",
			"retry_after": int(ttl.Seconds()) + 1,
		})
		return
	}

	// ─── PER-USER DAILY OTP LIMIT (5 per day) ─────────────────────────────────
	dailyKey := "otp_daily:" + authPayload.UserID.String()
	dailyCount, err := server.redis.Get(ctx, dailyKey).Int()
	if err == nil && dailyCount >= 5 {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Daily OTP limit reached. Try again tomorrow."})
		return
	}

	// Generate new OTP
	code := util.RandomDigitString(6)
	_, err = server.store.CreatePhoneVerification(ctx, db.CreatePhoneVerificationParams{
		UserID:    user.ID,
		Phone:     user.Phone,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Send SMS
	err = server.smsProvider.SendOTP(user.Phone, code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Set cooldown (30 seconds)
	server.redis.Set(ctx, cooldownKey, "1", 30*time.Second)

	// Increment daily counter (24 hour expiry)
	pipe := server.redis.Pipeline()
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, 24*time.Hour)
	_, _ = pipe.Exec(ctx)

	ctx.JSON(http.StatusOK, gin.H{"message": "Verification OTP sent"})
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
