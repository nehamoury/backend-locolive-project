package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"privacy-social-backend/internal/repository/db"
)

type createReportRequest struct {
	TargetID   string `json:"target_id" binding:"required,uuid"`
	TargetType string `json:"target_type" binding:"required,oneof=user post reel story"`
	Reason     string `json:"reason" binding:"required,oneof=spam abuse inappropriate other"`
	Description string `json:"description"`
}

func (server *Server) createReport(ctx *gin.Context) {
	var req createReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)
	targetID, _ := uuid.Parse(req.TargetID)

	// Increment priority if already reported
	_ = server.store.IncrementReportPriority(ctx, uuid.NullUUID{UUID: targetID, Valid: true})

	report, err := server.store.CreateReport(ctx, db.CreateReportParams{
		ReporterID:    authPayload.UserID,
		TargetID:      uuid.NullUUID{UUID: targetID, Valid: true},
		TargetType:    sql.NullString{String: req.TargetType, Valid: true},
		Reason:        db.ReportReason(req.Reason),
		Description:   sql.NullString{String: req.Description, Valid: req.Description != ""},
		PriorityScore: 1,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Log activity & Broadcast to Admin
	details, _ := json.Marshal(map[string]interface{}{
		"target_id":   req.TargetID,
		"target_type": req.TargetType,
		"reason":      req.Reason,
	})
	_, _ = server.store.CreateActivityLog(ctx, db.CreateActivityLogParams{
		UserID:     authPayload.UserID,
		ActionType: "report_created",
		TargetID:   uuid.NullUUID{UUID: report.ID, Valid: true},
		TargetType: sql.NullString{String: "report", Valid: true},
		Details:    pqtype.NullRawMessage{RawMessage: details, Valid: true},
	})
	server.hub.BroadcastActivity("report_created", map[string]interface{}{
		"report_id":   report.ID,
		"target_type": req.TargetType,
		"reason":      req.Reason,
	})

	// Update Trust Score if target is a user
	if req.TargetType == "user" {
		go server.user.UpdateTrustScore(context.Background(), targetID)
	}

	ctx.JSON(http.StatusCreated, report)
}
