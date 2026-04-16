package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"privacy-social-backend/internal/realtime"
	"privacy-social-backend/internal/token"
)

// activityWebSocket handles WebSocket connections for real-time admin activities
func (server *Server) activityWebSocket(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// User must be admin or moderator
	user, err := server.store.GetUserByID(ctx, authPayload.UserID)
	if err != nil || (user.Role != "admin" && user.Role != "moderator") {
		ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(ErrNotAdmin))
		return
	}

	// Upgrade HTTP to WS
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %v", err)
		return
	}

	client := &realtime.Client{
		Hub:      server.hub,
		UserID:   authPayload.UserID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Username: authPayload.Username,
		IsAdmin:  true,
	}

	server.hub.Register <- client

	// Start pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}
