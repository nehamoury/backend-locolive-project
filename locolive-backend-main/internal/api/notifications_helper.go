package api

import (
	"context"
	"database/sql"
	"privacy-social-backend/internal/repository/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var notificationSoundMap = map[string]string{
	"badge":           "badge_unlock.wav",
	"streak":          "streak_fire.wav",
	"nudge":           "soft_ping.wav",
	"message":         "chat_pop.wav",
	"gift":            "coin_reward.wav",
	"connection":      "chat_pop.wav",
	"crossing":        "soft_ping.wav",
	"story_reaction":  "chat_pop.wav",
	"nearby_story":    "soft_ping.wav",
	"reel_liked":      "chat_pop.wav",
	"reel_commented":  "chat_pop.wav",
	"story_mention":   "chat_pop.wav",
}

// createNotificationWithSound is a central helper to create persistent notifications with associated sounds
func (server *Server) createNotificationWithSound(
	ctx context.Context, 
	userID uuid.UUID, 
	nType db.NotificationType, 
	subType string, 
	title, 
	message string,
	relatedIDs map[string]uuid.UUID,
) (db.Notification, error) {
	
	sound := notificationSoundMap[subType]
	if sound == "" {
		sound = notificationSoundMap[string(nType)]
	}

	arg := db.CreateNotificationParams{
		UserID:   userID,
		Type:     nType,
		SubType:  sql.NullString{String: subType, Valid: subType != ""},
		Sound:    sql.NullString{String: sound, Valid: sound != ""},
		Title:    title,
		Message:  message,
	}

	if val, ok := relatedIDs["user"]; ok {
		arg.RelatedUserID = uuid.NullUUID{UUID: val, Valid: true}
	}
	if val, ok := relatedIDs["story"]; ok {
		arg.RelatedStoryID = uuid.NullUUID{UUID: val, Valid: true}
	}
	if val, ok := relatedIDs["crossing"]; ok {
		arg.RelatedCrossingID = uuid.NullUUID{UUID: val, Valid: true}
	}

	notif, err := server.store.CreateNotification(ctx, arg)
	if err != nil {
		log.Error().Err(err).Msg("failed to create notification with sound")
		return notif, err
	}

	// AUTOMATICALLY send push notification for all persistent notifications
	go server.sendPushNotificationToUser(
		context.Background(),
		userID,
		title,
		message,
		map[string]string{
			"type":    string(nType),
			"subtype": subType,
			"notif_id": notif.ID.String(),
		},
	)

	return notif, nil
}


// sendPushNotificationToUser fetches all FCM tokens for a user and sends a push notification
func (server *Server) sendPushNotificationToUser(ctx context.Context, userID uuid.UUID, title, body string, data map[string]string) {
	if server.notification == nil {
		log.Debug().Msg("FCM notification service not initialized, skipping push")
		return
	}

	tokens, err := server.store.GetUserFCMTokens(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to fetch FCM tokens for user")
		return
	}
	
	if len(tokens) == 0 {
		return
	}

	// Send to all registered devices for this user
	invalidTokens, err := server.notification.SendMulticastNotification(ctx, tokens, title, body, data)
	if err != nil {
		log.Error().Err(err).Msg("failed to send multicast push notification")
		return
	}

	// Clean up invalid tokens
	if len(invalidTokens) > 0 {
		for _, token := range invalidTokens {
			log.Info().Str("token", token).Str("user_id", userID.String()).Msg("removing invalid FCM token")
			err := server.store.RemoveFCMToken(ctx, db.RemoveFCMTokenParams{
				UserID: userID,
				Token:  token,
			})
			if err != nil {
				log.Error().Err(err).Str("token", token).Msg("failed to remove invalid FCM token")
			}
		}
	}
}

