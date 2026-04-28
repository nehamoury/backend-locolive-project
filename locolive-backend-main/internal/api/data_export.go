package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (server *Server) requestDataExport(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	// Check if there's already a pending job
	latest, err := server.store.GetLatestDataExportJob(ctx, authPayload.UserID)
	if err == nil && (latest.Status == "pending" || latest.Status == "processing") {
		ctx.JSON(http.StatusConflict, gin.H{"error": "a data export is already in progress"})
		return
	}

	// Create job in DB
	job, err := server.store.CreateDataExportJob(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Push to Redis Queue for worker
	err = server.redis.RPush(ctx, "queue:data_export", job.ID.String()).Err()
	if err != nil {
		// Log and continue, maybe the worker can recover or we'll retry
		fmt.Printf("Failed to queue job: %v\n", err)
	}

	ctx.JSON(http.StatusAccepted, successResponse(gin.H{
		"message": "Data export requested. You will be notified when it's ready.",
		"job_id":  job.ID,
	}))
}

func (server *Server) getDataExportStatus(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	job, err := server.store.GetLatestDataExportJob(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no export requests found"})
		return
	}

	ctx.JSON(http.StatusOK, successResponse(job))
}
