package worker

import (
	"context"
	"log"
	"time"

	"privacy-social-backend/internal/repository/db"

	"github.com/google/uuid"
)

func (worker *CleanupWorker) StartCrossingDetector() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			<-ticker.C
			log.Println("Running crossing detection...")
			worker.detectCrossings()
		}
	}()
}

func (worker *CleanupWorker) detectCrossings() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Look back at the last 10-15 minutes
	now := time.Now().UTC()
	maxTime := now
	minTime := now.Add(-15 * time.Minute)

	crossings, err := worker.store.FindPotentialCrossings(ctx, db.FindPotentialCrossingsParams{
		MinTime: minTime,
		MaxTime: maxTime,
	})
	if err != nil {
		log.Printf("failed to find crossings: %v", err)
		return
	}

	for _, c := range crossings {
		// Daily Limit Check (Max 50 crossings/day)
		count, err := worker.store.CountCrossingsToday(ctx, c.User1)
		if err == nil && count >= 50 {
			// Limit exceeded, skip silently
			continue
		}

		// Create Crossing Record
		crossing, err := worker.store.CreateCrossing(ctx, db.CreateCrossingParams{
			UserID1:        c.User1,
			UserID2:        c.User2,
			LocationCenter: c.Location,
			OccurredAt:     c.TimeBucket,
		})
		if err != nil {
			// Duplicate likely, skip
			continue
		}

		// Connection-aware notifications
		conn, err := worker.store.GetConnection(ctx, db.GetConnectionParams{
			RequesterID: c.User1,
			TargetID:    c.User2,
		})
		isConnected := err == nil && conn.Status == "accepted"
		title := "Path Crossed!"
		message := "You crossed paths with someone nearby"
		if isConnected {
			title = "Crossed Again!"
			message = "You crossed paths with a connection again!"
		} else {
			title = "Connection Suggestion!"
			message = "Send a connection request to someone you crossed paths with!"
		}

		// Notification for user1
		_, err = worker.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:            c.User1,
			Type:              "crossing_detected",
			Title:             title,
			Message:           message,
			RelatedUserID:     uuid.NullUUID{UUID: c.User2, Valid: true},
			RelatedCrossingID: uuid.NullUUID{UUID: crossing.ID, Valid: true},
		})
		if err != nil {
			log.Printf("failed to create crossing notification for user1: %v", err)
		}

		// Notification for user2
		_, err = worker.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:            c.User2,
			Type:              "crossing_detected",
			Title:             title,
			Message:           message,
			RelatedUserID:     uuid.NullUUID{UUID: c.User1, Valid: true},
			RelatedCrossingID: uuid.NullUUID{UUID: crossing.ID, Valid: true},
		})
		if err != nil {
			log.Printf("failed to create crossing notification for user2: %v", err)
		}
	}

	log.Printf("Processed %d potential crossings", len(crossings))
}
