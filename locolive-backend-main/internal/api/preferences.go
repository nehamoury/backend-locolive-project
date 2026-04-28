package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"privacy-social-backend/internal/repository/db"
)

type updateUserPreferencesRequest struct {
	Theme                string `json:"theme" binding:"required,oneof=light dark system"`
	Language             string `json:"language" binding:"required,min=2,max=10"`
	ContentFilterEnabled *bool  `json:"content_filter_enabled" binding:"required"`
}

func (server *Server) updatePreferences(ctx *gin.Context) {
	var req updateUserPreferencesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	prefs, err := server.store.UpsertUserPreferences(ctx, db.UpsertUserPreferencesParams{
		UserID:               authPayload.UserID,
		Theme:                req.Theme,
		Language:             req.Language,
		ContentFilterEnabled: *req.ContentFilterEnabled,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(prefs))
}

func (server *Server) getPreferences(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	prefs, err := server.store.GetUserPreferences(ctx, authPayload.UserID)
	if err != nil {
		// Return defaults if not found
		ctx.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"theme":                  "light",
			"language":               "en",
			"content_filter_enabled": true,
		}))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(prefs))
}

type updateNotificationSettingsRequest struct {
	EmailNotifications    *bool `json:"email_notifications" binding:"required"`
	PushNotifications     *bool `json:"push_notifications" binding:"required"`
	MarketingEmails       *bool `json:"marketing_emails" binding:"required"`
	ActivityNotifications *bool `json:"activity_notifications" binding:"required"`
}

func (server *Server) updateNotificationSettings(ctx *gin.Context) {
	var req updateNotificationSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	settings, err := server.store.UpsertNotificationSettings(ctx, db.UpsertNotificationSettingsParams{
		UserID:                authPayload.UserID,
		EmailNotifications:    *req.EmailNotifications,
		PushNotifications:     *req.PushNotifications,
		MarketingEmails:       *req.MarketingEmails,
		ActivityNotifications: *req.ActivityNotifications,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(settings))
}

func (server *Server) getNotificationSettings(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	settings, err := server.store.GetNotificationSettings(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"email_notifications":    true,
			"push_notifications":     true,
			"marketing_emails":       false,
			"activity_notifications": true,
		}))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(settings))
}
