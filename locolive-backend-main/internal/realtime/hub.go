package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Hub maintains the set of active clients and broadcasts messages to the
type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	mutex      sync.RWMutex
	redis      *redis.Client
	adminPool  map[*Client]bool
}

const (
	streamKey        = "locolive:stream:routing"
	adminActivityKey = "locolive:admin:activity"
)

var soundMap = map[string]string{
	"badge":          "badge_unlock.wav",
	"streak":         "streak_fire.wav",
	"nudge":          "soft_ping.wav",
	"message":        "chat_pop.wav",
	"gift":           "coin_reward.wav",
	"reel_liked":     "chat_pop.wav", // Reusing for now
	"reel_commented": "chat_pop.wav", // Reusing for now
	"new_reel_nearby": "soft_ping.wav",
	"force_logout":    "system_alert.wav",
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]map[*Client]bool),
		adminPool:  make(map[*Client]bool),
		redis:      rdb,
	}
}

func (h *Hub) Run() {
	// Start consuming Redis Stream messages
	go h.listenRedisStream()
	// Start consuming Admin Activity messages
	go h.listenAdminActivity()

	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			if client.IsAdmin {
				h.adminPool[client] = true
			}
			if _, ok := h.clients[client.UserID]; !ok {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mutex.Unlock()
			log.Info().Str("username", client.Username).Bool("is_admin", client.IsAdmin).Msg("Client registered")

			// Broadcast activity to admins
			if !client.IsAdmin {
				h.BroadcastActivity("user_online", map[string]interface{}{
					"user_id":  client.UserID,
					"username": client.Username,
				})
			}

		case client := <-h.Unregister:
			h.mutex.Lock()
			if client.IsAdmin {
				delete(h.adminPool, client)
			}
			if userClients, ok := h.clients[client.UserID]; ok {
				if _, ok := userClients[client]; ok {
					delete(userClients, client)
					close(client.Send)
					if len(userClients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mutex.Unlock()
			log.Info().Str("username", client.Username).Msg("Client unregistered")
		}
	}
}


// listenRedisStream pumps messages from Redis Stream to local clients
func (h *Hub) listenRedisStream() {
	// Start reading from the end of the stream ($)
	lastID := "$"

	for {
		// Block for up to 2 seconds waiting for new messages
		streams, err := h.redis.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamKey, lastID},
			Count:   10,
			Block:   2000 * time.Millisecond,
		}).Result()

		if err == redis.Nil {
			continue // No new messages
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed to read from Redis Stream")
			time.Sleep(5 * time.Second) // Backoff on error
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				lastID = msg.ID

				targetUserIDStr, ok := msg.Values["target_user_id"].(string)
				if !ok {
					continue
				}
				payload, ok := msg.Values["payload"].(string)
				if !ok {
					continue
				}

				userID, err := uuid.Parse(targetUserIDStr)
				if err != nil {
					continue
				}

				h.broadcastToLocal(userID, []byte(payload))
			}
		}
	}
}

// broadcastToLocal sends a message ONLY to locally connected clients
func (h *Hub) broadcastToLocal(userID uuid.UUID, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(clients, client)
			}
		}
	}
}

// SendToUser writes a message to the Redis Stream.
// This ensures that ANY server instance holding the user's connection receives it.
func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	// Add message to the stream
	// We use "*" to let Redis generate the ID
	// We define fields "target_user_id" and "payload"
	err := h.redis.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"target_user_id": userID.String(),
			"payload":        string(message),
		},
		// Optional: Cap the stream approx length to prevent infinite growth
		MaxLen: 100000,
		Approx: true,
	}).Err()

	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to publish message to Redis Stream")
	}
}

// IsUserOnline checks if a user has any active connections (Local check only for now)
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// ─── Reels Events ────────────────────────────────────────────────────────────

func (h *Hub) BroadcastReelLiked(reelID uuid.UUID, ownerID uuid.UUID, likerID uuid.UUID, likerUsername string) {
	msg := WSMessage{
		Type:    "reel_liked",
		SubType: "reel",
		Sound:   soundMap["reel_liked"],
		Payload: map[string]interface{}{
			"reel_id":  reelID,
			"user_id":  likerID,
			"username": likerUsername,
		},
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	h.SendToUser(ownerID, data)
}

func (h *Hub) BroadcastReelCommented(reelID uuid.UUID, ownerID uuid.UUID, commenterID uuid.UUID, commenterUsername string, comment interface{}) {
	msg := WSMessage{
		Type:    "reel_commented",
		SubType: "reel",
		Sound:   soundMap["reel_commented"],
		Payload: map[string]interface{}{
			"reel_id":  reelID,
			"user_id":  commenterID,
			"username": commenterUsername,
			"comment":  comment,
		},
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	h.SendToUser(ownerID, data)
}

func (h *Hub) BroadcastNewReelNearby(reel interface{}, targetUserIDs []uuid.UUID) {
	msg := WSMessage{
		Type:    "new_reel_nearby",
		SubType: "reel",
		Sound:   soundMap["new_reel_nearby"],
		Payload: reel,
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	for _, id := range targetUserIDs {
		h.SendToUser(id, data)
	}
}

func (h *Hub) BroadcastForceLogout(userID uuid.UUID, reason string) {
	msg := WSMessage{
		Type:    "force_logout",
		SubType: "system",
		Sound:   soundMap["force_logout"],
		Payload: map[string]interface{}{
			"reason": reason,
		},
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	h.SendToUser(userID, data)
}

func (h *Hub) DisconnectUser(userID uuid.UUID) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			delete(clients, client)
			close(client.Send)
		}
		delete(h.clients, userID)
	}
}

// ─── Admin Broadcasting ──────────────────────────────────────────────────

// listenAdminActivity listens for admin activity notifications via Redis Pub/Sub
func (h *Hub) listenAdminActivity() {
	pubsub := h.redis.Subscribe(context.Background(), adminActivityKey)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		h.broadcastToAdminsLocal([]byte(msg.Payload))
	}
}

// broadcastToAdminsLocal sends a message to all locally connected admin clients
func (h *Hub) broadcastToAdminsLocal(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.adminPool {
		select {
		case client.Send <- message:
		default:
			// Buffer full, handled by Unregister in WritePump or just drop
		}
	}
}

// BroadcastActivity sends a system activity to all admin instances via Redis
func (h *Hub) BroadcastActivity(eventType string, payload interface{}) {
	msg := map[string]interface{}{
		"type":    "activity",
		"payload": map[string]interface{}{
			"type":      eventType,
			"payload":   payload,
			"timestamp": time.Now().UTC(),
		},
	}
	data, _ := json.Marshal(msg)
	h.redis.Publish(context.Background(), adminActivityKey, data)
}
