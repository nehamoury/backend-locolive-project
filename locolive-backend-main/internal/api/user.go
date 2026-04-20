package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/user"
	"privacy-social-backend/internal/token"
)

type createUserRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,alphanum"`
	FullName string `json:"full_name" binding:"required"`
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
	CreatedAt         time.Time `json:"created_at"`
}

type searchUserResponse struct {
	ID         uuid.UUID `json:"id"`
	Username   string    `json:"username"`
	FullName   string    `json:"full_name"`
	AvatarUrl  string    `json:"avatar_url"`
	IsVerified bool      `json:"is_verified"`
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
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.user.CreateUser(ctx, user.CreateUserParams{
		Phone:    req.Phone,
		Email:    req.Email,
		Username: req.Username,
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

	// Generate Tokens for Auto-Login
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, user.ID, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user.Username, user.ID, server.config.RefreshTokenDuration)
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

	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}

	ctx.JSON(http.StatusCreated, rsp)

	// Broadcast activity to admins
	server.hub.BroadcastActivity("user_created", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"fullName": user.FullName,
	})
}


type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	result, err := server.user.LoginUser(ctx, user.LoginUserParams{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: ctx.Request.UserAgent(),
		ClientIP:  ctx.ClientIP(),
	})
	if err != nil {
		if err.Error() == "user not found" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "incorrect password" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := loginUserResponse{
		SessionID:             result.SessionID,
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
		User:                  newUserResponse(result.User),
	}
	ctx.JSON(http.StatusOK, rsp)
}

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

	users, err := server.user.SearchUsers(ctx, query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Initialize as empty array to avoid null in JSON
	rsp := make([]searchUserResponse, 0, len(users))
	for _, u := range users {
		// Ensure avatar_url is a relative path starting with /
		avatarUrl := u.AvatarUrl.String
		if avatarUrl != "" && !strings.HasPrefix(avatarUrl, "http") && !strings.HasPrefix(avatarUrl, "/") {
			avatarUrl = "/" + avatarUrl
		}

		rsp = append(rsp, searchUserResponse{
			ID:         u.ID,
			Username:   u.Username,
			FullName:   u.FullName,
			AvatarUrl:  avatarUrl,
			IsVerified: u.IsVerified,
		})
	}

	ctx.JSON(http.StatusOK, rsp)
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

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func (server *Server) updateUserPassword(ctx *gin.Context) {
	var req updatePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err := server.user.UpdatePassword(ctx, payload.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err.Error() == "incorrect current password" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
