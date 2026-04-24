package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/username"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
)

// checkUsernameRequest represents a request to check username availability
type checkUsernameRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
}

// checkUsernameResponse represents the availability check response
type checkUsernameResponse struct {
	Available   bool     `json:"available"`
	Username    string   `json:"username"`
	Suggestions []string `json:"suggestions,omitempty"`
	Message     string   `json:"message,omitempty"`
}

// validateUsernameRequest represents a username validation request
type validateUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}

// validateUsernameResponse represents the validation response
type validateUsernameResponse struct {
	Valid        bool   `json:"valid"`
	Normalized   string `json:"normalized"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// suggestUsernamesRequest represents a username suggestion request
type suggestUsernamesRequest struct {
	Base   string `json:"base" form:"base" binding:"required"`
	Count  int    `json:"count" form:"count"`
}

// suggestUsernamesResponse represents the suggestion response
type suggestUsernamesResponse struct {
	Suggestions []string `json:"suggestions"`
}

// reserveUsernameRequest represents a username reservation request
type reserveUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}

// reserveUsernameResponse represents the reservation response
type reserveUsernameResponse struct {
	Reserved  bool   `json:"reserved"`
	ExpiresIn int    `json:"expires_in_seconds"`
	Message   string `json:"message"`
}

// changeUsernameRequest represents a username change request
type changeUsernameRequest struct {
	NewUsername string `json:"new_username" binding:"required"`
	Password    string `json:"password" binding:"required"` // Require password confirmation
}

// changeUsernameResponse represents the change response
type changeUsernameResponse struct {
	Success  bool   `json:"success"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// checkUsername handles GET /api/users/check-username
// Public endpoint to check username availability
func (server *Server) checkUsername(ctx *gin.Context) {
	var req checkUsernameRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Clean up the username input
	username := strings.TrimSpace(req.Username)
	username = strings.TrimPrefix(username, "@")

	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	// Check availability
	available, suggestions, err := server.usernameService.CheckAvailability(ctx, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	resp := checkUsernameResponse{
		Available: available,
		Username:  username,
	}

	if !available {
		resp.Suggestions = suggestions
		resp.Message = "This username is not available"
	} else {
		resp.Message = "This username is available"
	}

	ctx.JSON(http.StatusOK, resp)
}

// validateUsername handles POST /api/users/validate-username
// Validates username format without checking availability
func (server *Server) validateUsername(ctx *gin.Context) {
	var req validateUsernameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	validation := server.usernameService.ValidateUsername(ctx, req.Username)

	ctx.JSON(http.StatusOK, validateUsernameResponse{
		Valid:        validation.IsValid,
		Normalized:   validation.Normalized,
		ErrorCode:    validation.ErrorCode,
		ErrorMessage: validation.ErrorMessage,
	})
}

// suggestUsernames handles GET /api/users/suggest-usernames
// Generates username suggestions based on a base string
func (server *Server) suggestUsernames(ctx *gin.Context) {
	var req suggestUsernamesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default count
	count := req.Count
	if count <= 0 || count > 10 {
		count = 5
	}

	suggestions := server.usernameService.GenerateSuggestions(req.Base, count)

	ctx.JSON(http.StatusOK, suggestUsernamesResponse{
		Suggestions: suggestions,
	})
}

// reserveUsername handles POST /api/users/reserve-username (Protected)
// Temporarily reserves a username during registration flow
func (server *Server) reserveUsername(ctx *gin.Context) {
	var req reserveUsernameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get user ID from auth context (if authenticated)
	var userID uuid.UUID
	if authPayload, exists := ctx.Get(authorizationPayloadKey); exists {
		if payload, ok := authPayload.(*token.Payload); ok {
			userID = payload.UserID
		}
	}

	// If not authenticated, use session ID or IP as identifier
	if userID == uuid.Nil {
		// For anonymous reservations, we could use IP + session
		// For now, require authentication for reservations
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Validate username first
	validation := server.usernameService.ValidateUsername(ctx, req.Username)
	if !validation.IsValid {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": validation.ErrorMessage,
			"code":  validation.ErrorCode,
		})
		return
	}

	// Check availability
	available, suggestions, err := server.usernameService.CheckAvailability(ctx, validation.Normalized)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if !available {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":       "username is not available",
			"suggestions": suggestions,
		})
		return
	}

	// Reserve the username for 5 minutes
	err = server.usernameService.ReserveUsername(ctx, validation.Normalized, userID, 5*time.Minute)
	if err != nil {
		if err == username.ErrUsernameTaken {
			ctx.JSON(http.StatusConflict, gin.H{"error": "username is already reserved"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, reserveUsernameResponse{
		Reserved:  true,
		ExpiresIn: 300, // 5 minutes in seconds
		Message:   "Username reserved for 5 minutes",
	})
}

// changeUsername handles PUT /api/users/change-username (Protected)
// Allows authenticated users to change their username
func (server *Server) changeUsername(ctx *gin.Context) {
	var req changeUsernameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get authenticated user
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Verify password first
	user, err := server.store.GetUserByID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}

	// Change username
	updatedUser, err := server.usernameService.ChangeUsername(ctx, authPayload.UserID, req.NewUsername)
	if err != nil {
		switch err {
		case username.ErrUsernameTaken:
			ctx.JSON(http.StatusConflict, gin.H{"error": "username is already taken"})
		case username.ErrUsernameReserved:
			ctx.JSON(http.StatusForbidden, gin.H{"error": "this username is reserved"})
		case username.ErrUsernameInvalid, username.ErrUsernameTooShort, username.ErrUsernameTooLong:
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	ctx.JSON(http.StatusOK, changeUsernameResponse{
		Success:  true,
		Username: updatedUser.Username,
		Message:  "Username changed successfully",
	})
}

// Admin endpoints for reserved username management

// listReservedUsernames handles GET /api/admin/reserved-usernames (Admin only)
func (server *Server) listReservedUsernames(ctx *gin.Context) {
	names, err := server.store.GetReservedUsernames(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"reserved_usernames": names,
		"count":              len(names),
	})
}

// addReservedUsername handles POST /api/admin/reserved-usernames (Admin only)
func (server *Server) addReservedUsername(ctx *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Reason   string `json:"reason"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if req.Reason == "" {
		req.Reason = "admin_reserved"
	}

	err := server.store.AddReservedUsername(ctx, db.AddReservedUsernameParams{
		Username: req.Username,
		Reason:   req.Reason,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Refresh cache
	_ = server.usernameService.RefreshReservedCache(ctx)

	ctx.JSON(http.StatusCreated, gin.H{
		"message":  "Username reserved successfully",
		"username": req.Username,
	})
}

// removeReservedUsername handles DELETE /api/admin/reserved-usernames/:username (Admin only)
func (server *Server) removeReservedUsername(ctx *gin.Context) {
	username := ctx.Param("username")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	err := server.store.RemoveReservedUsername(ctx, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Refresh cache
	_ = server.usernameService.RefreshReservedCache(ctx)

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "Username removed from reserved list",
		"username": username,
	})
}
