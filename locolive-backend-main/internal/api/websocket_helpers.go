package api

import (
	"context"
	"encoding/json"
	"privacy-social-backend/internal/realtime"

	"github.com/google/uuid"
)

// sendWSNotification sends a WebSocket notification to a user
func (server *Server) sendWSNotification(userID uuid.UUID, msgType string, payload interface{}) {
	wsMsg := realtime.WSMessage{
		Type:    msgType,
		Payload: payload,
	}
	wsMsgBytes, _ := json.Marshal(wsMsg)
	server.hub.SendToUser(userID, wsMsgBytes)

	// Also send Push Notification via FCM
	go server.sendPushNotificationToUser(
		context.Background(),
		userID,
		"Locolive Notification",
		msgType, // Simple message for now
		map[string]string{
			"type": msgType,
		},
	)
}
