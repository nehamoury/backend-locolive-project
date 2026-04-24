package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"privacy-social-backend/internal/repository/db"
	"github.com/rs/zerolog/log"
	"github.com/sqlc-dev/pqtype"
)

// getStreak retrieves the current streak for the authenticated user
func (server *Server) getStreak(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	streak, err := server.store.GetUserStreak(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusOK, gin.H{
				"current_streak": 0,
				"longest_streak": 0,
				"message":        "Start your streak today! 🔥",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, streak)
}

// getDailyStats retrieves daily statistics for the user
func (server *Server) getDailyStats(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	// Default to last 7 days
	days := 7
	startDate := time.Now().AddDate(0, 0, -days)

	stats, err := server.store.GetDailyStats(ctx, db.GetDailyStatsParams{
		UserID: authPayload.UserID,
		Date:   startDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// listBadges lists all badges earned by the user and all available badges
func (server *Server) listBadges(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	earnedBadges, err := server.store.GetUserBadges(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	allBadges, err := server.store.ListAllBadges(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"earned_badges":    earnedBadges,
		"available_badges": allBadges,
	})
}

type updatePreferencesRequest struct {
	PushEnabled     bool `json:"push_enabled"`
	EmailEnabled    bool `json:"email_enabled"`
	CrossingAlerts  bool `json:"crossing_alerts"`
	MessageAlerts   bool `json:"message_alerts"`
	StoryAlerts     bool `json:"story_alerts"`
}

// updateNotificationPreferences updates user notification settings
func (server *Server) updateNotificationPreferences(ctx *gin.Context) {
	var req updatePreferencesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	prefs, err := server.store.UpdateNotificationPreferences(ctx, db.UpdateNotificationPreferencesParams{
		UserID:         authPayload.UserID,
		PushEnabled:    sql.NullBool{Bool: req.PushEnabled, Valid: true},
		EmailEnabled:   sql.NullBool{Bool: req.EmailEnabled, Valid: true},
		CrossingAlerts: sql.NullBool{Bool: req.CrossingAlerts, Valid: true},
		MessageAlerts:  sql.NullBool{Bool: req.MessageAlerts, Valid: true},
		StoryAlerts:    sql.NullBool{Bool: req.StoryAlerts, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, prefs)
}

// getNotificationPreferences retrieves user notification settings
func (server *Server) getNotificationPreferences(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	prefs, err := server.store.GetNotificationPreferences(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			ctx.JSON(http.StatusOK, gin.H{
				"push_enabled":    true,
				"email_enabled":   true,
				"crossing_alerts": true,
				"message_alerts":  true,
				"story_alerts":    true,
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, prefs)
}

// updateStreakLogic is an internal helper to update a user's streak
func (server *Server) updateStreakLogic(userID uuid.UUID) {
	ctx := context.Background()
	
	streak, err := server.store.GetUserStreak(ctx, userID)
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Create first streak
			server.store.UpdateUserStreak(ctx, db.UpdateUserStreakParams{
				UserID:            userID,
				CurrentStreak:     int32(1),
				LongestStreak:    int32(1),
				LastActivityDate: sql.NullTime{Time: todayDate, Valid: true},
			})
			return
		}
		log.Error().Err(err).Msg("failed to get user streak")
		return
	}

	lastDate := streak.LastActivityDate
	
	if lastDate.Valid && lastDate.Time.Equal(todayDate) {
		// Already updated today
		return
	}

	yesterday := todayDate.AddDate(0, 0, -1)
	newCurrent := int32(1)
	
	if lastDate.Valid && lastDate.Time.Equal(yesterday) {
		newCurrent = streak.CurrentStreak + 1
	}

	newLongest := streak.LongestStreak
	if newCurrent > newLongest {
		newLongest = newCurrent
	}

	server.store.UpdateUserStreak(ctx, db.UpdateUserStreakParams{
		UserID:            userID,
		CurrentStreak:     newCurrent,
		LongestStreak:    newLongest,
		LastActivityDate: sql.NullTime{Time: todayDate, Valid: true},
	})

	// Milestone notifications
	if newCurrent > 0 && newCurrent % 7 == 0 {
		server.createInternalNotification(userID, db.NotificationTypeStreak, "🔥 Streak Milestone!", 
			"You've reached a milestone! Keep going!")
	}
}

// trackEventLogic is an internal helper to track engagement events
func (server *Server) trackEventLogic(userID uuid.UUID, eventType string, data interface{}) {
	ctx := context.Background()
	
	var eventData pqtype.NullRawMessage
	if data != nil {
		bytes, _ := json.Marshal(data)
		eventData = pqtype.NullRawMessage{RawMessage: json.RawMessage(bytes), Valid: true}
	}

	server.store.CreateEngagementEvent(ctx, db.CreateEngagementEventParams{
		UserID:    userID,
		EventType: eventType,
		EventData: eventData,
	})
	
	log.Info().Str("event", eventType).Str("user_id", userID.String()).Msg("Tracking engagement event")
}

// createInternalNotification is a helper to create notifications without an API request
func (server *Server) createInternalNotification(userID uuid.UUID, nType db.NotificationType, title, message string) {
	ctx := context.Background()
	subType := string(nType)
	server.createNotificationWithSound(ctx, userID, nType, subType, title, message, nil)
}
