package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type clientLogRequest struct {
	Level     string                 `json:"level" binding:"required,oneof=info warn error"`
	Message   string                 `json:"message" binding:"required"`
	Stack     string                 `json:"stack"`
	Component string                 `json:"component"`
	URL       string                 `json:"url"`
	UserID    string                 `json:"user_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// logClientError handles logs sent from the frontend (Observability)
func (server *Server) logClientError(ctx *gin.Context) {
	var req clientLogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	requestID := ctx.GetString("request_id")
	clientIP := ctx.ClientIP()

	event := log.Info()
	switch req.Level {
	case "warn":
		event = log.Warn()
	case "error":
		event = log.Error()
	}

	event.
		Str("type", "client_log").
		Str("client_request_id", requestID).
		Str("client_ip", clientIP).
		Str("component", req.Component).
		Str("url", req.URL).
		Str("user_id", req.UserID).
		Interface("metadata", req.Metadata).
		Time("server_time", time.Now()).
		Msg(req.Message)

	if req.Stack != "" && req.Level == "error" {
		log.Debug().Str("stack", req.Stack).Msg("Client Error Stack Trace")
	}

	ctx.JSON(http.StatusOK, successResponse(nil))
}
