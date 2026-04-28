package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/util"

	"github.com/gin-gonic/gin"
)

type googleLoginRequest struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	Code        string `json:"code"`
}

type googleUser struct {
	Sub           string      `json:"sub"`
	Email         string      `json:"email"`
	EmailVerified interface{} `json:"email_verified"`
	Name          string      `json:"name"`
	Picture       string      `json:"picture"`
}

type googleLoginResponse struct {
	SessionID                 string       `json:"session_id"`
	AccessToken               string       `json:"access_token"`
	AccessTokenExpiresAt      string       `json:"access_token_expires_at"`
	RefreshToken              string       `json:"refresh_token"`
	RefreshTokenExpiresAt     string       `json:"refresh_token_expires_at"`
	User                      userResponse `json:"user"`
	RequiresProfileCompletion bool         `json:"requires_profile_completion"`
}

func (server *Server) googleLogin(ctx *gin.Context) {
	var req googleLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// 1. Verify Google Token or Exchange Code
	var gUser *googleUser
	var err error

	if req.Code != "" {
		gUser, err = server.exchangeGoogleCode(req.Code)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
	} else if req.IDToken != "" {
		gUser, err = verifyGoogleToken(req.IDToken)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
	} else if req.AccessToken != "" {
		gUser, err = fetchGoogleUserinfo(req.AccessToken)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
	} else {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("either id_token, access_token or code is required")))
		return
	}

	var user db.User
	var requiresProfileCompletion bool

	// 2. Check if user exists by Google ID
	existingUser, err := server.store.GetUserByGoogleID(ctx, sql.NullString{String: gUser.Sub, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			// 3. Not found by Google ID, check by Email for account linking
			existingUser, err = server.store.GetUserByEmail(ctx, sql.NullString{String: gUser.Email, Valid: true})
			if err != nil {
				if err == sql.ErrNoRows {
					// 4. Create new Google user (requires profile completion)
					hashedPassword, _ := util.HashPassword(util.RandomString(12))
					user, err = server.store.CreateUser(ctx, db.CreateUserParams{
						Phone:             "google_" + gUser.Sub,
						Email:             sql.NullString{String: gUser.Email, Valid: true},
						PasswordHash:      hashedPassword,
						Username:          util.RandomString(10),
						FullName:          gUser.Name,
						IsGhostMode:       false,
						Provider:          "google",
						IsProfileComplete: false,
					})
					if err != nil {
						ctx.JSON(http.StatusInternalServerError, errorResponse(err))
						return
					}

					// Update Google ID and avatar for the new user
					user, err = server.store.UpdateUserGoogleID(ctx, db.UpdateUserGoogleIDParams{
						ID:       user.ID,
						GoogleID: sql.NullString{String: gUser.Sub, Valid: true},
						Provider: sql.NullString{String: "google", Valid: true},
					})
					if err != nil {
						ctx.JSON(http.StatusInternalServerError, errorResponse(err))
						return
					}

					// Set avatar from Google picture if available
					if gUser.Picture != "" {
						_, _ = server.store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
							ID:        user.ID,
							AvatarUrl: sql.NullString{String: gUser.Picture, Valid: true},
						})
					}

					requiresProfileCompletion = true
				} else {
					ctx.JSON(http.StatusInternalServerError, errorResponse(err))
					return
				}
			} else {
				// User exists by email - LINK accounts automatically
				user, err = server.store.UpdateUserGoogleID(ctx, db.UpdateUserGoogleIDParams{
					ID:       existingUser.ID,
					GoogleID: sql.NullString{String: gUser.Sub, Valid: true},
					Provider: sql.NullString{String: "linked", Valid: true},
				})
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, errorResponse(err))
					return
				}

				// Update avatar from Google if not already set
				if gUser.Picture != "" && !existingUser.AvatarUrl.Valid {
					_, _ = server.store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
						ID:        user.ID,
						AvatarUrl: sql.NullString{String: gUser.Picture, Valid: true},
					})
				}

				requiresProfileCompletion = !user.IsProfileComplete
			}
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	} else {
		// Found by Google ID
		user = existingUser
		requiresProfileCompletion = !user.IsProfileComplete
	}

	// 5. Generate Tokens
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
	ctx.SetCookie(
		"access_token",
		accessToken,
		int(server.config.AccessTokenDuration.Seconds()),
		"/",
		"",
		isProduction,
		true,
	)
	ctx.SetCookie(
		"refresh_token",
		refreshToken,
		int(server.config.RefreshTokenDuration.Seconds()),
		"/api/users/renew_access",
		"",
		isProduction,
		true,
	)

	rsp := googleLoginResponse{
		SessionID:                 session.ID.String(),
		AccessToken:               accessToken,
		AccessTokenExpiresAt:      accessPayload.ExpiredAt.String(),
		RefreshToken:              refreshToken,
		RefreshTokenExpiresAt:     refreshPayload.ExpiredAt.String(),
		User:                      newUserResponse(user),
		RequiresProfileCompletion: requiresProfileCompletion,
	}
	ctx.JSON(http.StatusOK, rsp)
}

// GoogleCallback handles the redirect from Google and forwards it to Expo Go
func (server *Server) googleCallback(ctx *gin.Context) {
	expoUrl := server.config.ExpoRedirectURL
	if expoUrl == "" {
		expoUrl = "exp://127.0.0.1:8081/--/google-auth"
	}

	location := fmt.Sprintf("%s?%s", expoUrl, ctx.Request.URL.RawQuery)
	ctx.Redirect(http.StatusFound, location)
}

func (server *Server) exchangeGoogleCode(code string) (*googleUser, error) {
	tokenEndpoint := "https://oauth2.googleapis.com/token"

	resp, err := http.PostForm(tokenEndpoint,
		map[string][]string{
			"code":          {code},
			"client_id":     {server.config.GoogleClientID},
			"client_secret": {server.config.GoogleClientSecret},
			"redirect_uri":  {"postmessage"},
			"grant_type":    {"authorization_code"},
		})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorBody)
		return nil, fmt.Errorf("failed to exchange code: %v, body: %v", resp.Status, errorBody)
	}

	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return verifyGoogleToken(tokenResp.IDToken)
}

func verifyGoogleToken(token string) (*googleUser, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=%s", token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token")
	}

	var gUser googleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, err
	}

	verified := false
	switch v := gUser.EmailVerified.(type) {
	case bool:
		verified = v
	case string:
		verified = (v == "true")
	}

	if !verified {
		return nil, fmt.Errorf("email not verified")
	}

	return &gUser, nil
}

func fetchGoogleUserinfo(accessToken string) (*googleUser, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/userinfo?access_token=%s", accessToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch userinfo: %s", resp.Status)
	}

	var gUser googleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, err
	}

	if gUser.Email == "" {
		return nil, fmt.Errorf("email not found in userinfo")
	}

	verified := false
	switch v := gUser.EmailVerified.(type) {
	case bool:
		verified = v
	case string:
		verified = (v == "true")
	}

	if !verified {
		return nil, fmt.Errorf("email not verified")
	}

	return &gUser, nil
}
