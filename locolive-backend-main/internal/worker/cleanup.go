package worker

import (
	"context"
	"time"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/notification"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type CleanupWorker struct {
	store        repository.Store
	notification *notification.NotificationService
}

func NewCleanupWorker(store repository.Store, notification *notification.NotificationService) *CleanupWorker {
	return &CleanupWorker{
		store:        store,
		notification: notification,
	}
}

// sendPushNotificationToUser fetches all FCM tokens for a user and sends a push notification
func (worker *CleanupWorker) sendPushNotificationToUser(ctx context.Context, userID uuid.UUID, title, body string, data map[string]string) {
	if worker.notification == nil {
		return
	}

	tokens, err := worker.store.GetUserFCMTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}

	// Send to all registered devices for this user
	invalidTokens, err := worker.notification.SendMulticastNotification(ctx, tokens, title, body, data)
	if err != nil {
		log.Error().Err(err).Msg("failed to send multicast push notification")
		return
	}

	// Clean up invalid tokens
	if len(invalidTokens) > 0 {
		for _, token := range invalidTokens {
			log.Info().Str("token", token).Str("user_id", userID.String()).Msg("removing invalid FCM token")
			_ = worker.store.RemoveFCMToken(ctx, db.RemoveFCMTokenParams{
				UserID: userID,
				Token:  token,
			})
		}
	}
}


func (worker *CleanupWorker) Start() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for {
			<-ticker.C
			log.Info().Msg("Running cleanup worker...")
			worker.cleanup()
		}
	}()
}

func (worker *CleanupWorker) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err := worker.store.DeleteExpiredLocations(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete expired locations")
	} else {
		log.Info().Msg("Expired locations deleted")
	}

	// Cleanup expired stories
	err = worker.store.DeleteExpiredStories(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete expired stories")
	} else {
		log.Info().Msg("Expired stories deleted")
	}

	// Cleanup old messages (30+ days)
	err = worker.store.DeleteOldMessages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete old messages")
	} else {
		log.Info().Msg("Old messages deleted")
	}

	// Cleanup expired messages (Secret Mode disappearing messages)
	err = worker.store.DeleteExpiredMessages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete expired messages")
	} else {
		log.Info().Msg("Expired messages deleted")
	}

	// Cleanup old notifications (30+ days)
	err = worker.store.DeleteOldNotifications(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete old notifications")
	} else {
		log.Info().Msg("Old notifications deleted")
	}

	// Cleanup expired password resets
	err = worker.store.DeleteExpiredPasswordResets(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete expired password resets")
	} else {
		log.Info().Msg("Expired password resets deleted")
	}
}
