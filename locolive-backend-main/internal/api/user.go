package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"

	"github.com/rs/zerolog/log"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/user"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
	usernameutil "privacy-social-backend/internal/util/username"
)

type createUserRequest struct {
	Phone       string `json:"phone" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Username    string `json:"username" binding:"required,alphanum"`
	FullName    string `json:"full_name" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
	IsGhostMode bool   `json:"is_ghost_mode"`
}

type userResponse struct {
	ID                uuid.UUID `json:"id"`
	Phone             string    `json:"phone"`
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Bio               string    `json:"bio"`
	AvatarUrl         string    `json:"avatar_url"`
	BannerUrl         string    `json:"banner_url"`
	Theme             string    `json:"theme"`
	ProfileVisibility string    `json:"profile_visibility"`
	Email             string    `json:"email"`
	IsGhostMode       bool      `json:"is_ghost_mode"`
	Role              string    `json:"role"`
	Provider          string    `json:"provider"`
	IsProfileComplete bool      `json:"is_profile_complete"`
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsPhoneVerified   bool      `json:"is_phone_verified"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
}

type searchUserResponse struct {
	ID               uuid.UUID `json:"id"`
	Username         string    `json:"username"`
	FullName         string    `json:"full_name"`
	AvatarUrl        string    `json:"avatar_url"`
	IsVerified       bool      `json:"is_verified"`
	IsPrivate        bool      `json:"is_private"`
	ConnectionStatus string    `json:"connection_status"`
	IsBlocked        bool      `json:"is_blocked"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		ID:                user.ID,
		Phone:             user.Phone,
		Username:          user.Username,
		FullName:          user.FullName,
		Bio:               user.Bio.String,
		AvatarUrl:         user.AvatarUrl.String,
		BannerUrl:         user.BannerUrl.String,
		Theme:             user.Theme.String,
		ProfileVisibility: user.ProfileVisibility.String,
		Email:             user.Email.String,
		IsGhostMode:       user.IsGhostMode,
		Role:              string(user.Role),
		Provider:          user.Provider,
		IsProfileComplete: user.IsProfileComplete,
		IsEmailVerified:   user.IsEmailVerified,
		IsPhoneVerified:   user.IsPhoneVerified,
		IsActive:          user.IsActive,
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Block disposable email domains
	if util.IsDisposableEmail(req.Email) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Please use a permanent email address. Temporary email providers are not allowed."})
		return
	}

	// Validate and normalize phone number to E.164 format
	phone := req.Phone
	if !strings.HasPrefix(phone, "+") {
		var err error
		phone, err = util.NormalizeToE164(phone, "91") // Default country code: India
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid phone number: %v", err)})
			return
		}
	} else {
		if err := util.ValidateE164(phone); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Twilio Lookup: detect VoIP/virtual numbers and verify carrier
	if server.config.Environment == "production" {
		lookup := util.NewPhoneLookupProvider(server.config.TwilioAccountSID, server.config.TwilioAuthToken, true)
		if lookup == nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Phone verification service is not configured. Please contact support."})
			return
		}
		lookupResult, err := lookup.ValidateAndCheck(phone)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = lookupResult // carrier info available for logging/audit
	} else {
		// Development: still validate E.164 format but skip Twilio Lookup
		if err := util.ValidateE164(phone); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Update request phone to normalized E.164 format
	req.Phone = phone

	// Normalize and validate username
	req.Username = usernameutil.NormalizeUsername(req.Username)
	if !usernameutil.IsValidUsername(req.Username) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid username format. Must be 3-20 characters, start with a letter, and contain only a-z, 0-9, or underscore."})
		return
	}

	user, err := server.user.CreateUser(ctx, user.CreateUserParams{
		Phone:       req.Phone,
		Email:       req.Email,
		Username:    req.Username,
		FullName:    req.FullName,
		Password:    req.Password,
		IsGhostMode: req.IsGhostMode,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 1. Generate & Send Email Verification
	emailToken := util.RandomString(32)
	_, err = server.store.CreateEmailVerification(ctx, db.CreateEmailVerificationParams{
		UserID:    user.ID,
		Email:     user.Email.String,
		Token:     emailToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err == nil {
		err = server.mailer.SendVerificationEmail(user.Email.String, emailToken)
		if err != nil {
			log.Error().Err(err).Str("email", user.Email.String).Msg("failed to send verification email during signup")
		}
	}

	// Generate Tokens for Auto-Login
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

	// Set Access Token in Cookie
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

	// Set Refresh Token in Cookie
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

	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}

	ctx.JSON(http.StatusCreated, successResponse(rsp))

	// Broadcast activity to admins
	server.hub.BroadcastActivity("user_created", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"fullName": user.FullName,
	})
}

type loginUserRequest struct {
	Identity      string `json:"identity" binding:"required"`
	Password      string `json:"password" binding:"required,min=6"`
	IsAdminPortal bool   `json:"is_admin_portal"`
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type renewAccessTokenResponse struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.Identity = strings.TrimSpace(req.Identity)

	result, err := server.user.LoginUser(ctx, user.LoginUserParams{
		Identity:     req.Identity,
		Password:     req.Password,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIP:     ctx.ClientIP(),
		RequireAdmin: req.IsAdminPortal,
	})
	if err != nil {
		if err.Error() == "user not found" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		if err.Error() == "incorrect password" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		if err.Error() == "access denied: administrators only" {
			ctx.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Access denied. This portal is for administrators only."})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Set Access Token in Cookie
	isProduction := server.config.Environment == "production"
	sameSite := http.SameSiteLaxMode
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    result.AccessToken,
		MaxAge:   int(server.config.AccessTokenDuration.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isProduction,
		HttpOnly: true,
		SameSite: sameSite,
	})

	// Set Refresh Token in Cookie
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    result.RefreshToken,
		MaxAge:   int(server.config.RefreshTokenDuration.Seconds()),
		Path:     "/api/users/renew-access",
		Domain:   "",
		Secure:   isProduction,
		HttpOnly: true,
		SameSite: sameSite,
	})

	rsp := loginUserResponse{
		SessionID:             result.SessionID,
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
		User:                  newUserResponse(result.User),
	}
	ctx.JSON(http.StatusOK, successResponse(rsp))
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	_ = ctx.ShouldBindJSON(&req)

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		cookieToken, err := ctx.Cookie("refresh_token")
		if err == nil {
			refreshToken = cookieToken
		}
	}

	if refreshToken == "" {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("refresh token is required")))
		return
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(refreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("session not found")))
			return
		}
		log.Error().Err(err).Msg("Database error during session retrieval")
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("session verification failed")))
		return
	}

	if session.IsBlocked {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("blocked session")))
		return
	}

	if session.UserID != refreshPayload.UserID {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("incorrect session user")))
		return
	}

	if session.RefreshToken != refreshToken {
		// TOKEN REUSE DETECTED: Possible theft, revoke entire user's sessions
		server.redis.Set(ctx, fmt.Sprintf("revoke_all:%s", session.UserID.String()), time.Now().Unix(), 24*time.Hour)
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("refresh token reuse detected, all sessions revoked")))
		return
	}

	if time.Now().After(session.ExpiresAt) {
		ctx.JSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("expired session")))
		return
	}

	// ─── REFRESH TOKEN ROTATION ───────────────────────────────────────────────
	// 1. Blacklist the OLD refresh token immediately
	if err := server.revokeToken(ctx, refreshPayload.ID, refreshPayload.ExpiredAt); err != nil {
		log.Error().Err(err).Msg("Failed to blacklist old refresh token")
	}

	// 2. Create NEW refresh token with a new JTI
	newRefreshToken, newRefreshPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		refreshPayload.UserID,
		refreshPayload.Role,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 3. Create new session record with the new refresh token
	_, err = server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           newRefreshPayload.ID,
		UserID:       refreshPayload.UserID,
		RefreshToken: newRefreshToken,
		UserAgent:    session.UserAgent,
		ClientIp:     session.ClientIp,
		IsBlocked:    false,
		ExpiresAt:    newRefreshPayload.ExpiredAt,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create rotated session")
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("failed to rotate session")))
		return
	}

	// 4. Create new access token
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		refreshPayload.UserID,
		refreshPayload.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 5. Set both cookies with rotated tokens
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
		Value:    newRefreshToken,
		MaxAge:   int(server.config.RefreshTokenDuration.Seconds()),
		Path:     "/api/users/renew-access",
		Domain:   "",
		Secure:   isProduction,
		HttpOnly: true,
		SameSite: sameSite,
	})

	ctx.JSON(http.StatusOK, successResponse(renewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiredAt,
	}))
}

// logoutUser is now handled in security.go

type searchUsersRequest struct {
	Query string `form:"q"`
}

func (server *Server) searchUsers(ctx *gin.Context) {
	var req searchUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Trim and sanitize query
	query := strings.TrimSpace(req.Query)
	query = strings.TrimPrefix(query, "@")
	if len(query) < 2 {
		ctx.JSON(http.StatusOK, []searchUserResponse{})
		return
	}

	var currentUserID uuid.UUID
	authPayload, authExists := ctx.Get(authorizationPayloadKey)
	if authExists && authPayload != nil {
		currentUserID = authPayload.(*token.Payload).UserID
	}

	users, err := server.user.SearchUsers(ctx, query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Initialize as empty array to avoid null in JSON
	rsp := make([]searchUserResponse, 0, len(users))
	// authPayload and authExists already declared above

	for _, u := range users {
		// 1. Exclude self from search
		if authExists && authPayload != nil && authPayload.(*token.Payload).UserID == u.ID {
			continue
		}

		// 2. CENTRAL PRIVACY CHECK: Filter out blocked users
		if authExists && authPayload != nil {
			result := server.privacy.CanViewProfile(ctx, authPayload.(*token.Payload).UserID, u.ID)
			if !result.Allowed {
				continue // Blocked, Panic, or Ghost (Invisible)
			}
		}

		// Ensure avatar_url is a relative path starting with /
		avatarUrl := u.AvatarUrl.String
		if avatarUrl != "" && !strings.HasPrefix(avatarUrl, "http") && !strings.HasPrefix(avatarUrl, "/") {
			avatarUrl = "/" + avatarUrl
		}

		// Get connection status
		connStatus := "none"
		if authExists {
			conn, err := server.store.GetConnection(ctx, db.GetConnectionParams{
				RequesterID: currentUserID,
				TargetID:    u.ID,
			})
			if err == nil {
				connStatus = string(conn.Status)
			}

			// Check if blocked
			blocked, err := server.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
				BlockerID: currentUserID,
				BlockedID: u.ID,
			})
			if err == nil && blocked {
				connStatus = "blocked"
			}
		}

		rsp = append(rsp, searchUserResponse{
			ID:               u.ID,
			Username:         u.Username,
			FullName:         u.FullName,
			AvatarUrl:        avatarUrl,
			IsVerified:       u.IsVerified,
			IsPrivate:        u.IsPrivate,
			ConnectionStatus: connStatus,
			IsBlocked:        connStatus == "blocked",
		})
	}

	ctx.JSON(http.StatusOK, successResponse(rsp))
}

// completeProfileRequest handles profile completion for Google OAuth users
type completeProfileRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Phone    string `json:"phone" binding:"required"`
}

func (server *Server) completeProfile(ctx *gin.Context) {
	var req completeProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Normalize and validate username
	req.Username = usernameutil.NormalizeUsername(req.Username)
	if !usernameutil.IsValidUsername(req.Username) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid username format. Must be 3-20 characters, start with a letter, and contain only a-z, 0-9, or underscore."})
		return
	}

	// Validate and normalize phone number to E.164 format
	phone := req.Phone
	if !strings.HasPrefix(phone, "+") {
		var err error
		phone, err = util.NormalizeToE164(phone, "91")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid phone number: %v", err)})
			return
		}
	} else {
		if err := util.ValidateE164(phone); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Twilio Lookup: detect VoIP/virtual numbers
	if server.config.Environment == "production" {
		lookup := util.NewPhoneLookupProvider(server.config.TwilioAccountSID, server.config.TwilioAuthToken, true)
		if lookup == nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Phone verification service is not configured. Please contact support."})
			return
		}
		lookupResult, err := lookup.ValidateAndCheck(phone)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = lookupResult
	} else {
		if err := util.ValidateE164(phone); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Update request phone to normalized E.164 format
	req.Phone = phone

	// Get authenticated user from context
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Check if username is already taken by another user
	existingUser, err := server.store.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser.ID != payload.UserID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "username already taken"})
		return
	}

	// Check if phone is already taken by another user
	existingPhone, err := server.store.GetUserByPhone(ctx, req.Phone)
	if err == nil && existingPhone.ID != payload.UserID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "phone number already registered"})
		return
	}

	// Complete the profile
	user, err := server.store.CompleteUserProfile(ctx, db.CompleteUserProfileParams{
		ID:       payload.UserID,
		Username: req.Username,
		Phone:    req.Phone,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, gin.H{"error": "username or phone already exists"})
				return
			}
		}
		log.Error().Err(err).Msg("CompleteUserProfile failed in store")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Generate new tokens for the updated user
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

	rsp := completeProfileResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
		RequiresPhoneVerify:   true,
	}
	ctx.JSON(http.StatusOK, successResponse(rsp))
}

type completeProfileResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
	RequiresPhoneVerify   bool         `json:"requires_phone_verify"`
}

type updateEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (server *Server) updateUserEmail(ctx *gin.Context) {
	var req updateEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	resultUser, err := server.user.UpdateEmail(ctx, user.UpdateEmailParams{
		UserID: payload.UserID,
		Email:  req.Email,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"email": resultUser.Email.String})
}

// deleteAccount handles soft account deletion
func (server *Server) deleteAccount(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	// 1. Soft delete in DB
	err := server.store.SoftDeleteUser(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 2. Global revocation of all sessions
	now := time.Now()
	server.redis.Set(ctx, fmt.Sprintf("revoke_all:%s", authPayload.UserID.String()), now.Unix(), 24*time.Hour)

	// 3. Log Audit Event
	_, _ = server.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    authPayload.UserID,
		Action:    "account_deleted",
		Details:   pqtype.NullRawMessage{RawMessage: util.ToJSONB(map[string]interface{}{"status": "soft_deleted"}), Valid: true},
		IpAddress: db.ToNullString(ctx.ClientIP()),
		UserAgent: db.ToNullString(ctx.Request.UserAgent()),
	})

	// 4. Clear cookies
	ctx.SetCookie("access_token", "", -1, "/", "", false, true)
	ctx.SetCookie("refresh_token", "", -1, "/api/users/renew-access", "", false, true)

	ctx.JSON(http.StatusOK, successResponse("Account has been deactivated. You can restore it within 30 days by logging back in."))
}

// checkEmail handles GET /api/users/check-email
func (server *Server) checkEmail(ctx *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(ctx.Query("email")))
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	u, err := server.store.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err == nil {
		// Only consider it 'taken' if the account is not soft-deleted
		if !u.DeletedAt.Valid {
			ctx.JSON(http.StatusOK, gin.H{"available": false, "message": "Email is unavailable"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"available": true})
}

// checkPhone handles GET /api/users/check-phone
func (server *Server) checkPhone(ctx *gin.Context) {
	phone := ctx.Query("phone")
	if phone == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}

	_, err := server.store.GetUserByPhone(ctx, phone)
	if err == nil {
		ctx.JSON(http.StatusOK, gin.H{"available": false, "message": "Phone number is already registered"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"available": true})
}
