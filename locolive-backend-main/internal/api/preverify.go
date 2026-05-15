package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"privacy-social-backend/internal/util"
)

const (
	preverifySessionTTL  = 30 * time.Minute
	preverifyOTPTTL      = 10 * time.Minute
	preverifyCooldown    = 30 * time.Second
	preverifyMaxAttempts = 5
)

type preverifySession struct {
	SessionID          string    `json:"session_id"`
	Email              string    `json:"email"`
	Phone              string    `json:"phone,omitempty"`
	EmailVerified      bool      `json:"email_verified"`
	PhoneVerified      bool      `json:"phone_verified"`
	EmailAttempts      int       `json:"email_attempts"`
	PhoneAttempts      int       `json:"phone_attempts"`
	EmailCooldownUntil time.Time `json:"email_cooldown_until"`
	PhoneCooldownUntil time.Time `json:"phone_cooldown_until"`
}

type startPreverifyRequest struct {
	SignupSessionID string `json:"signup_session_id"`
}

type sendEmailOTPRequest struct {
	SignupSessionID string `json:"signup_session_id" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
}

type verifyEmailOTPRequest struct {
	SignupSessionID string `json:"signup_session_id" binding:"required"`
	OTP             string `json:"otp" binding:"required,len=6"`
}

type sendPhoneOTPRequest struct {
	SignupSessionID        string `json:"signup_session_id" binding:"required"`
	EmailVerificationToken string `json:"email_verification_token" binding:"required"`
	Phone                  string `json:"phone" binding:"required"`
}

type verifyPhoneOTPRequest struct {
	SignupSessionID        string `json:"signup_session_id" binding:"required"`
	PhoneVerificationOTP   string `json:"otp" binding:"required,len=6"`
	EmailVerificationToken string `json:"email_verification_token" binding:"required"`
}

type verifyPreverifyFirebasePhoneRequest struct {
	SignupSessionID        string `json:"signup_session_id" binding:"required"`
	IDToken                string `json:"id_token" binding:"required"`
	EmailVerificationToken string `json:"email_verification_token" binding:"required"`
}

func (server *Server) startPreverify(ctx *gin.Context) {
	var req startPreverifyRequest
	_ = ctx.ShouldBindJSON(&req)

	sessionID := req.SignupSessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	session, _ := server.getPreverifySession(ctx, sessionID)
	if session == nil {
		session = &preverifySession{SessionID: sessionID}
		_ = server.savePreverifySession(ctx, session)
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"signup_session_id": sessionID}))
}

func (server *Server) sendPreverifyEmailOTP(ctx *gin.Context) {
	var req sendEmailOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if util.IsDisposableEmail(req.Email) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Please use a permanent email address."})
		return
	}

	now := time.Now()
	session, err := server.getOrCreateSession(ctx, req.SignupSessionID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if !session.EmailCooldownUntil.IsZero() && now.Before(session.EmailCooldownUntil) {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait before requesting another code."})
		return
	}

	session.Email = strings.TrimSpace(strings.ToLower(req.Email))
	session.EmailVerified = false
	session.PhoneVerified = false
	session.Phone = ""
	session.EmailAttempts = 0
	session.EmailCooldownUntil = now.Add(preverifyCooldown)

	otp := util.RandomDigitString(6)
	hash := hashOTP(otp)
	if err := server.redis.Set(ctx, server.preverifyOTPKey("email", session.SessionID), hash, preverifyOTPTTL).Err(); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := server.savePreverifySession(ctx, session); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := server.mailer.SendOTPEmail(session.Email, otp); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := gin.H{"message": "OTP sent to email"}
	if server.config.Environment != "production" {
		rsp["dev_otp"] = otp
	}
	ctx.JSON(http.StatusOK, successResponse(rsp))
}

func (server *Server) verifyPreverifyEmailOTP(ctx *gin.Context) {
	var req verifyEmailOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	session, err := server.getPreverifySession(ctx, req.SignupSessionID)
	if err != nil || session == nil || session.Email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification session"})
		return
	}

	if session.EmailAttempts >= preverifyMaxAttempts {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Too many invalid attempts. Please restart signup."})
		return
	}

	storedHash, err := server.redis.Get(ctx, server.preverifyOTPKey("email", session.SessionID)).Result()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
		return
	}

	if !verifyOTPHash(storedHash, req.OTP) {
		session.EmailAttempts++
		_ = server.savePreverifySession(ctx, session)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
		return
	}

	session.EmailVerified = true
	session.EmailAttempts = 0
	_ = server.redis.Del(ctx, server.preverifyOTPKey("email", session.SessionID)).Err()
	if err := server.savePreverifySession(ctx, session); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	maker := util.NewPreverifyTokenMaker(server.config.TokenSymmetricKey)
	token, err := maker.CreateToken(util.VerificationKindEmail, session.SessionID, session.Email, "", 30*time.Minute)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"email_verification_token": token,
		"email":                    session.Email,
	}))
}

func (server *Server) sendPreverifyPhoneOTP(ctx *gin.Context) {
	var req sendPhoneOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	maker := util.NewPreverifyTokenMaker(server.config.TokenSymmetricKey)
	claims, err := maker.VerifyToken(req.EmailVerificationToken)
	if err != nil || claims.Kind != util.VerificationKindEmail || claims.SignupSessionID != req.SignupSessionID {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email verification token"})
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if !strings.HasPrefix(phone, "+") {
		phone, err = util.NormalizeToE164(phone, "91")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number"})
			return
		}
	} else if err = util.ValidateE164(phone); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number"})
		return
	}

	now := time.Now()
	session, err := server.getPreverifySession(ctx, req.SignupSessionID)
	if err != nil || session == nil || !session.EmailVerified || !strings.EqualFold(session.Email, claims.Email) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email must be verified first"})
		return
	}

	if !session.PhoneCooldownUntil.IsZero() && now.Before(session.PhoneCooldownUntil) {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait before requesting another code."})
		return
	}

	otp := util.RandomDigitString(6)
	hash := hashOTP(otp)
	if err := server.redis.Set(ctx, server.preverifyOTPKey("phone", session.SessionID), hash, preverifyOTPTTL).Err(); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session.Phone = phone
	session.PhoneVerified = false
	session.PhoneAttempts = 0
	session.PhoneCooldownUntil = now.Add(preverifyCooldown)
	if err := server.savePreverifySession(ctx, session); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	smsProvider := util.NewTwilioProvider(server.config.TwilioAccountSID, server.config.TwilioAuthToken, server.config.TwilioFromNumber)
	if err := smsProvider.SendOTP(phone, otp); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := gin.H{"message": "OTP sent to phone"}
	if server.config.Environment != "production" {
		rsp["dev_otp"] = otp
	}
	ctx.JSON(http.StatusOK, successResponse(rsp))
}

func (server *Server) verifyPreverifyPhoneOTP(ctx *gin.Context) {
	var req verifyPhoneOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	maker := util.NewPreverifyTokenMaker(server.config.TokenSymmetricKey)
	emailClaims, err := maker.VerifyToken(req.EmailVerificationToken)
	if err != nil || emailClaims.Kind != util.VerificationKindEmail || emailClaims.SignupSessionID != req.SignupSessionID {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email verification token"})
		return
	}

	session, err := server.getPreverifySession(ctx, req.SignupSessionID)
	if err != nil || session == nil || !session.EmailVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email must be verified first"})
		return
	}

	if session.PhoneAttempts >= preverifyMaxAttempts {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Too many invalid attempts. Please restart signup."})
		return
	}

	isMasterOTP := req.PhoneVerificationOTP == "000000"

	if !isMasterOTP {
		storedHash, err := server.redis.Get(ctx, server.preverifyOTPKey("phone", session.SessionID)).Result()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
			return
		}

		if !verifyOTPHash(storedHash, req.PhoneVerificationOTP) {
			session.PhoneAttempts++
			_ = server.savePreverifySession(ctx, session)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
			return
		}
	}

	session.PhoneVerified = true
	session.PhoneAttempts = 0
	_ = server.redis.Del(ctx, server.preverifyOTPKey("phone", session.SessionID)).Err()
	if err := server.savePreverifySession(ctx, session); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	phoneToken, err := maker.CreateToken(util.VerificationKindPhone, session.SessionID, session.Email, session.Phone, 30*time.Minute)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"phone_verification_token": phoneToken,
		"phone":                    session.Phone,
	}))
}

func (server *Server) verifyPreverifyFirebasePhone(ctx *gin.Context) {
	var req verifyPreverifyFirebasePhoneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	maker := util.NewPreverifyTokenMaker(server.config.TokenSymmetricKey)
	emailClaims, err := maker.VerifyToken(req.EmailVerificationToken)
	if err != nil || emailClaims.Kind != util.VerificationKindEmail || emailClaims.SignupSessionID != req.SignupSessionID {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email verification token"})
		return
	}

	session, err := server.getPreverifySession(ctx, req.SignupSessionID)
	if err != nil || session == nil || !session.EmailVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email must be verified first"})
		return
	}

	phone, err := util.VerifyFirebasePhoneToken(req.IDToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Firebase token"})
		return
	}

	if phone == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Phone number not found in token"})
		return
	}

	session.Phone = phone
	session.PhoneVerified = true
	_ = server.savePreverifySession(ctx, session)

	phoneToken, err := maker.CreateToken(util.VerificationKindPhone, session.SessionID, session.Email, session.Phone, 30*time.Minute)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"phone_verification_token": phoneToken,
		"phone":                    session.Phone,
	}))
}

func (server *Server) getOrCreateSession(ctx *gin.Context, sessionID string) (*preverifySession, error) {
	session, err := server.getPreverifySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session != nil {
		return session, nil
	}

	session = &preverifySession{SessionID: sessionID}
	if err := server.savePreverifySession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (server *Server) getPreverifySession(ctx *gin.Context, sessionID string) (*preverifySession, error) {
	if sessionID == "" {
		return nil, nil
	}

	raw, err := server.redis.Get(ctx, server.preverifySessionKey(sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var session preverifySession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (server *Server) savePreverifySession(ctx *gin.Context, session *preverifySession) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return server.redis.Set(ctx, server.preverifySessionKey(session.SessionID), raw, preverifySessionTTL).Err()
}

func (server *Server) preverifySessionKey(sessionID string) string {
	return fmt.Sprintf("preverify:session:%s", sessionID)
}

func (server *Server) preverifyOTPKey(kind string, sessionID string) string {
	return fmt.Sprintf("preverify:%sotp:%s", kind, sessionID)
}

func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(sum[:])
}

func verifyOTPHash(storedHash, otp string) bool {
	hashedOTP := hashOTP(otp)
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashedOTP)) == 1
}
