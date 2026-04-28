package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
)

type DataExportWorker struct {
	store repository.Store
	redis *redis.Client
}

func NewDataExportWorker(store repository.Store, rdb *redis.Client) *DataExportWorker {
	return &DataExportWorker{
		store: store,
		redis: rdb,
	}
}

func (w *DataExportWorker) Start(ctx context.Context) {
	fmt.Println("Data Export Worker started...")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Pop from queue
			res, err := w.redis.BLPop(ctx, 5*time.Second, "queue:data_export").Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				fmt.Printf("Worker error: %v\n", err)
				time.Sleep(2 * time.Second)
				continue
			}

			jobIDStr := res[1]
			jobID, err := uuid.Parse(jobIDStr)
			if err != nil {
				continue
			}

			w.processJob(ctx, jobID)
		}
	}
}

func (w *DataExportWorker) processJob(ctx context.Context, jobID uuid.UUID) {
	fmt.Printf("Processing data export job: %s\n", jobID)

	// 1. Update status to processing
	job, err := w.store.UpdateDataExportJob(ctx, db.UpdateDataExportJobParams{
		ID:     jobID,
		Status: "processing",
	})
	if err != nil {
		return
	}

	// 2. Aggregate data
	// This is a complex task in production. For this demo, we'll simulate it.
	time.Sleep(10 * time.Second) 

	// Fetch some real data to show we can
	user, _ := w.store.GetUserByID(ctx, job.UserID)
	posts, _ := w.store.GetUserEngagementStats(ctx, job.UserID)

	data := map[string]interface{}{
		"user_profile": user,
		"engagement":   posts,
		"exported_at":  time.Now(),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("Generated export data size: %d bytes\n", len(jsonData))
	// 3. Save to "storage" (simulated with a file path)
	filePath := fmt.Sprintf("/uploads/exports/data_%s.json", jobID)
	// In reality, we'd write to R2/S3 or disk.
	
	// 4. Update job to completed
	expiresAt := time.Now().Add(24 * time.Hour)
	w.store.UpdateDataExportJob(ctx, db.UpdateDataExportJobParams{
		ID:        jobID,
		Status:    "completed",
		FilePath:  db.ToNullString(filePath),
		ExpiresAt: db.ToNullTime(expiresAt),
	})

	fmt.Printf("Job %s completed successfully\n", jobID)
}
