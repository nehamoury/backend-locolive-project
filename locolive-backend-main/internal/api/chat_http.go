package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"privacy-social-backend/internal/realtime"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const chatCacheTTL = 10 * time.Minute

// checkConnection verifies that two users can interact based on the centralized privacy rule engine
func (server *Server) checkConnection(ctx context.Context, viewerID, targetID uuid.UUID) error {
	// Use the CENTRAL RULE ENGINE for all access decisions
	// This automatically handles: Block > Panic > Ghost > Private
	result := server.privacy.CanUserAccess(ctx, viewerID, targetID)

	if !result.Allowed {
		switch result.Reason {
		case "blocked", "panic_mode", "ghost_mode":
			return sql.ErrNoRows // Return 404 behavior (invisible)
		case "private":
			return fmt.Errorf("private account") // Handled as 403 by caller
		default:
			return sql.ErrNoRows
		}
	}

	return nil
}

// API to get chat history
func (server *Server) getChatHistory(ctx *gin.Context) {
	targetIDStr := ctx.Query("user_id")
	targetID, ok := parseUUIDParam(ctx, targetIDStr, "user_id")
	if !ok {
		return
	}
	authPayload := getAuthPayload(ctx)

	// Check for mutual connection
	access := server.privacy.CanUserAccess(ctx, authPayload.UserID, targetID)
	if !access.Allowed {
		reason := "You must be connected to this user to chat."
		switch access.Reason {
		case "blocked":
			reason = "This user has blocked you or you have blocked them."
		case "panic_mode", "ghost_mode":
			reason = "This user is currently unavailable due to their privacy settings."
		case "private":
			reason = "You must be an accepted follower to message this private account."
		case "hidden":
			reason = "This user is currently inactive."
		}
		ctx.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}


	// Create cache key with sorted IDs for consistency
	ids := []string{authPayload.UserID.String(), targetID.String()}
	sort.Strings(ids)
	cacheKey := "messages:" + ids[0] + ":" + ids[1]

	// Try Redis cache first
	cachedData, err := server.redis.Get(context.Background(), cacheKey).Result()
	if err == nil && cachedData != "" {
		ctx.Header("X-Cache", "HIT")
		ctx.Data(http.StatusOK, "application/json", []byte(cachedData))
		return
	}

	msgs, err := server.store.ListMessages(ctx, db.ListMessagesParams{
		SenderID:   authPayload.UserID,
		ReceiverID: uuid.NullUUID{UUID: targetID, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Map to response struct to ensure Reactions are valid JSON, not Base64
	type MessageResponse struct {
		ID         uuid.UUID       `json:"id"`
		SenderID   uuid.UUID       `json:"sender_id"`
		ReceiverID *uuid.UUID      `json:"receiver_id"`
		GroupID    *uuid.UUID      `json:"group_id"`
		Content    string          `json:"content"`
		IsRead     bool            `json:"is_read"`
		CreatedAt  time.Time       `json:"created_at"`
		ReadAt     sql.NullTime    `json:"read_at"`
		ExpiresAt  sql.NullTime    `json:"expires_at"`
		MediaUrl   *string         `json:"media_url"`
		MediaType  *string         `json:"media_type"`
		Reactions  json.RawMessage `json:"reactions"`
	}

	responseMsgs := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		var reactionsJSON json.RawMessage
		if m.Reactions != nil {
			switch v := m.Reactions.(type) {
			case []byte:
				reactionsJSON = json.RawMessage(v)
			case string:
				reactionsJSON = json.RawMessage(v)
			default:
				reactionsJSON = []byte("[]")
			}
		} else {
			reactionsJSON = []byte("[]")
		}

		var receiverID *uuid.UUID
		if m.ReceiverID.Valid {
			id := m.ReceiverID.UUID
			receiverID = &id
		}

		var groupID *uuid.UUID
		if m.GroupID.Valid {
			id := m.GroupID.UUID
			groupID = &id
		}

		responseMsgs[i] = MessageResponse{
			ID:         m.ID,
			SenderID:   m.SenderID,
			ReceiverID: receiverID,
			GroupID:    groupID,
			Content:    m.Content,
			IsRead:     m.IsRead,
			CreatedAt:  m.CreatedAt,
			ReadAt:     m.ReadAt,
			ExpiresAt:  m.ExpiresAt,
			MediaUrl:   nullStringToStrPtr(m.MediaUrl),
			MediaType:  nullStringToStrPtr(m.MediaType),
			Reactions:  reactionsJSON,
		}
	}

	// Cache the result
	responseJSON, _ := json.Marshal(responseMsgs)
	server.redis.Set(context.Background(), cacheKey, responseJSON, chatCacheTTL)

	ctx.Header("X-Cache", "MISS")
	ctx.Data(http.StatusOK, "application/json", responseJSON)
}

// REST API helper to send a message
type sendMessageRequest struct {
	ReceiverID       *uuid.UUID `json:"receiver_id"`
	GroupID          *uuid.UUID `json:"group_id"`
	Content          string     `json:"content"` // Not required if media is present
	MediaUrl         string     `json:"media_url"`
	MediaType        string     `json:"media_type"`
	ExpiresInSeconds int64      `json:"expires_in_seconds"` // Optional
}

func (server *Server) sendMessage(ctx *gin.Context) {
	var req sendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fmt.Printf("ERROR: sendMessage JSON bind failed: %v\n", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	fmt.Printf("DEBUG: Back-end received sendMessage request: %+v\n", req)

	authPayload := getAuthPayload(ctx)

	// Validation: Must have either ReceiverID OR GroupID, not both (for now)
	if req.ReceiverID == nil && req.GroupID == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "recipient (user or group) is required"})
		return
	}

	var receiverID uuid.NullUUID
	var groupID uuid.NullUUID

	if req.ReceiverID != nil {
		receiverID = uuid.NullUUID{UUID: *req.ReceiverID, Valid: true}
		
		// Centralized Privacy Check: Block, Panic, Ghost, etc.
		access := server.privacy.CanUserAccess(ctx, authPayload.UserID, *req.ReceiverID)
		if !access.Allowed {
			reason := "Access denied"
			switch access.Reason {
			case "blocked":
				reason = "You cannot message this user."
			case "panic_mode", "ghost_mode":
				reason = "User is currently unavailable."
			case "private":
				reason = "You must be connected to this user to send messages."
			}
			ctx.JSON(http.StatusForbidden, gin.H{"error": reason})
			return
		}
	}

	if req.GroupID != nil {
		groupID = uuid.NullUUID{UUID: *req.GroupID, Valid: true}
		// Check membership
		isMember, err := server.store.CheckGroupMembership(ctx, db.CheckGroupMembershipParams{
			GroupID: *req.GroupID,
			UserID:  authPayload.UserID,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		if !isMember {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this group to send messages."})
			return
		}
	}

	// Handle expiry - DEFAULT TO 24 HOURS (Snapchat-style)
	var expiresAt sql.NullTime
	if req.ExpiresInSeconds > 0 {
		// Custom expiry time provided
		expiresAt = sql.NullTime{
			Time:  time.Now().UTC().Add(time.Duration(req.ExpiresInSeconds) * time.Second),
			Valid: true,
		}
	} else {
		// DEFAULT: All messages expire after 24 hours
		expiresAt = sql.NullTime{
			Time:  time.Now().UTC().Add(24 * time.Hour),
			Valid: true,
		}
	}

	msg, err := server.store.CreateMessage(ctx, db.CreateMessageParams{
		SenderID:   authPayload.UserID,
		ReceiverID: receiverID,
		GroupID:    groupID,
		Content:    req.Content,
		MediaUrl:   toNullString(req.MediaUrl),
		MediaType:  toNullString(req.MediaType),
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if receiverID.Valid {
		// Invalidate cache for this conversation (1:1)
		server.invalidateConversationCache(authPayload.UserID, receiverID.UUID)
		server.incrementUnreadCount(receiverID.UUID)

		wsMsg := realtime.WSMessage{
			Type:      "new_message",
			SubType:   "message",
			Sound:     "chat_pop.wav",
			Payload:   msg,
			SenderID:  authPayload.UserID,
			CreatedAt: msg.CreatedAt,
		}
		data, _ := json.Marshal(wsMsg)
		server.hub.SendToUser(receiverID.UUID, data)
		
		// Send Push Notification via FCM
		go server.sendPushNotificationToUser(
			context.Background(),
			receiverID.UUID,
			fmt.Sprintf("New message from %s", authPayload.Username),
			req.Content,
			map[string]string{
				"type": "new_message",
				"sender_id": authPayload.UserID.String(),
			},
		)
	} else if groupID.Valid {
		// Group Logic
		// 1. Notify All Members
		members, err := server.store.GetGroupMembers(ctx, groupID.UUID)
		if err == nil {
			wsMsg := realtime.WSMessage{
				Type:      "new_group_message",
				SubType:   "message",
				Sound:     "chat_pop.wav",
				Payload:   msg,
				SenderID:  authPayload.UserID,
				CreatedAt: msg.CreatedAt,
			}
			wsMsgBytes, _ := json.Marshal(wsMsg)
			
			for _, m := range members {
				// Don't send back to self (already handled below by echo)
				if m.UserID != authPayload.UserID {
					server.hub.SendToUser(m.UserID, wsMsgBytes)
					
					// Send Push Notification (async)
					go server.sendPushNotificationToUser(
						context.Background(),
						m.UserID,
						"New message in group",
						req.Content,
						map[string]string{
							"type": "new_group_message",
							"group_id": groupID.UUID.String(),
						},
					)
				}
			}
		}
	}

	// Also send to SENDER so their client can update the messages list
	// Wait, for groups, sender is also a member.
	// For 1:1, sender needs update.

	wsMsg := realtime.WSMessage{
		Type:      "new_message",
		SubType:   "message",
		Sound:     "", // No sound for self-echo usually, but keeping struct consistent
		Payload:   msg,
		SenderID:  authPayload.UserID,
		CreatedAt: msg.CreatedAt,
	}
	wsMsgBytes, _ := json.Marshal(wsMsg)
	server.hub.SendToUser(authPayload.UserID, wsMsgBytes) // Always echo back?

	ctx.JSON(http.StatusCreated, msg)
}

// deleteMessage allows a user to unsend/delete their own message
func (server *Server) deleteMessage(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, ok := parseUUIDParam(ctx, messageIDStr, "message_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	// Get the message first to find the receiver for cache invalidation
	msg, err := server.store.GetMessage(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify sender owns the message
	if msg.SenderID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own messages"})
		return
	}

	// Delete the message
	err = server.store.DeleteMessage(ctx, db.DeleteMessageParams{
		ID:       messageID,
		SenderID: authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache and Notify
	if msg.ReceiverID.Valid {
		server.invalidateConversationCache(msg.SenderID, msg.ReceiverID.UUID)
		server.sendWSNotification(msg.ReceiverID.UUID, "message_deleted", gin.H{"message_id": messageID})
	}
	// TODO: Handle Group deletion notification

	ctx.JSON(http.StatusOK, gin.H{"message": "Message deleted"})
}

// editMessageRequest defines the request body for editing a message
type editMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// editMessage allows a user to edit their own message
func (server *Server) editMessage(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, ok := parseUUIDParam(ctx, messageIDStr, "message_id")
	if !ok {
		return
	}

	var req editMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	// Get the message first to find the receiver for cache invalidation
	originalMsg, err := server.store.GetMessage(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify sender owns the message
	if originalMsg.SenderID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own messages"})
		return
	}

	// Update the message
	updatedMsg, err := server.store.UpdateMessage(ctx, db.UpdateMessageParams{
		ID:        messageID,
		SenderID:  authPayload.UserID,
		Content:   req.Content,
		MediaUrl:  originalMsg.MediaUrl,  // Keep original media
		MediaType: originalMsg.MediaType, // Keep original type
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache and Notify
	if originalMsg.ReceiverID.Valid {
		server.invalidateConversationCache(originalMsg.SenderID, originalMsg.ReceiverID.UUID)
		server.sendWSNotification(originalMsg.ReceiverID.UUID, "message_edited", updatedMsg)
	}
	// TODO: Handle Group edit notification

	ctx.JSON(http.StatusOK, updatedMsg)
}

// saveMessage prevents a message from expiring (sets expires_at to NULL)
func (server *Server) saveMessage(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, ok := parseUUIDParam(ctx, messageIDStr, "message_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	// Get the message first
	msg, err := server.store.GetMessage(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify user is part of the conversation (sender or receiver)
	isParticipant := (msg.SenderID == authPayload.UserID)
	if msg.ReceiverID.Valid && msg.ReceiverID.UUID == authPayload.UserID {
		isParticipant = true
	} else if msg.GroupID.Valid {
		// Assume participant if group msg? Or check membership?
		// For MVP, if it's a group msg, anyone can save? Or only members.
		isParticipant = true // Simplified
	}

	if !isParticipant {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "You can only save messages from your own conversations"})
		return
	}

	// Save the message (set expires_at to NULL)
	savedMsg, err := server.store.SaveMessage(ctx, messageID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache
	if msg.ReceiverID.Valid {
		server.invalidateConversationCache(msg.SenderID, msg.ReceiverID.UUID)
		server.sendWSNotification(msg.ReceiverID.UUID, "message_saved", gin.H{"message_id": messageID, "saved_by": authPayload.UserID})
	} else if msg.GroupID.Valid {
		// Group notification logic
	}

	// Notify the other user (logic adjusted for null receiver)
	// otherUserID := msg.SenderID
	// if msg.SenderID == authPayload.UserID && msg.ReceiverID.Valid {
	// 	otherUserID = msg.ReceiverID.UUID
	// }
	// server.sendWSNotification(otherUserID, "message_saved", gin.H{"message_id": messageID, "saved_by": authPayload.UserID})

	ctx.JSON(http.StatusOK, gin.H{"message": "Message saved successfully", "data": savedMsg})
}

// markConversationRead marks all messages from a user as read
func (server *Server) markConversationRead(ctx *gin.Context) {
	senderIDStr := ctx.Param("userId")
	senderID, ok := parseUUIDParam(ctx, senderIDStr, "user_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	err := server.store.MarkConversationRead(ctx, db.MarkConversationReadParams{
		ReceiverID: uuid.NullUUID{UUID: authPayload.UserID, Valid: true},
		SenderID:   senderID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache
	server.invalidateConversationCache(authPayload.UserID, senderID)

	// Clear Unread Count Cache for Reader (User)
	server.invalidateUnreadCountCache(authPayload.UserID)

	// Notify sender that their messages were read
	wsMsg := realtime.WSMessage{
		Type: "messages_read",
		Payload: gin.H{
			"reader_id": authPayload.UserID,
			"sender_id": senderID,
		},
	}
	wsMsgBytes, _ := json.Marshal(wsMsg)
	server.hub.SendToUser(senderID, wsMsgBytes)

	// Notify Self (Reader) to update badges on other devices
	server.hub.SendToUser(authPayload.UserID, wsMsgBytes)

	ctx.JSON(http.StatusOK, gin.H{"message": "Conversation marked as read"})
}

// Reaction request body
type reactionRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

// addReaction adds a reaction to a message
func (server *Server) addReaction(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req reactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Get the message to find the other user
	msg, err := server.store.GetMessage(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	reaction, err := server.store.CreateMessageReaction(ctx, db.CreateMessageReactionParams{
		MessageID: messageID,
		UserID:    authPayload.UserID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache
	if msg.ReceiverID.Valid {
		ids := []string{msg.SenderID.String(), msg.ReceiverID.UUID.String()}
		sort.Strings(ids)
		cacheKey := "messages:" + ids[0] + ":" + ids[1]
		server.redis.Del(context.Background(), cacheKey)
	}

	// Notify the other user
	var otherUserID uuid.UUID
	shouldNotify := false

	if msg.SenderID == authPayload.UserID {
		if msg.ReceiverID.Valid {
			otherUserID = msg.ReceiverID.UUID
			shouldNotify = true
		}
	} else {
		otherUserID = msg.SenderID // If I'm receiver, notify sender
		shouldNotify = true
	}

	if shouldNotify {
		wsMsg := realtime.WSMessage{
			Type: "reaction_added",
			Payload: gin.H{
				"message_id": messageID,
				"user_id":    authPayload.UserID,
				"emoji":      req.Emoji,
			},
		}
		wsMsgBytes, _ := json.Marshal(wsMsg)
		server.hub.SendToUser(otherUserID, wsMsgBytes)
	}

	ctx.JSON(http.StatusCreated, reaction)
}

// removeReaction removes a reaction from a message
func (server *Server) removeReaction(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req reactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Get the message to find the other user
	msg, err := server.store.GetMessage(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = server.store.DeleteMessageReaction(ctx, db.DeleteMessageReactionParams{
		MessageID: messageID,
		UserID:    authPayload.UserID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Notify the other user
	var otherUserID uuid.UUID
	shouldNotify := false

	if msg.SenderID == authPayload.UserID {
		if msg.ReceiverID.Valid {
			otherUserID = msg.ReceiverID.UUID
			shouldNotify = true
		}
	} else {
		otherUserID = msg.SenderID
		shouldNotify = true
	}

	if shouldNotify {
		wsMsg := realtime.WSMessage{
			Type: "reaction_removed",
			Payload: gin.H{
				"message_id": messageID,
				"user_id":    authPayload.UserID,
				"emoji":      req.Emoji,
			},
		}
		wsMsgBytes, _ := json.Marshal(wsMsg)
		server.hub.SendToUser(otherUserID, wsMsgBytes)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}

// getMessageReactions gets all reactions for a message
func (server *Server) getMessageReactions(ctx *gin.Context) {
	messageIDStr := ctx.Param("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	reactions, err := server.store.GetMessageReactions(ctx, messageID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, reactions)
}

// getUnreadMessageCount returns the total number of unread messages for the user
func (server *Server) getUnreadMessageCount(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Try Redis first
	cacheKey := "unread_count:" + authPayload.UserID.String()
	cachedCount, err := server.redis.Get(context.Background(), cacheKey).Int64()
	if err == nil {
		ctx.Header("X-Cache", "HIT")
		ctx.JSON(http.StatusOK, gin.H{"count": cachedCount})
		return
	}

	// Fix: NullUUID
	count, err := server.store.GetUnreadMessageCount(ctx, uuid.NullUUID{UUID: authPayload.UserID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	// Cache the result
	server.redis.Set(context.Background(), cacheKey, count, 30*time.Minute)

	ctx.Header("X-Cache", "MISS")
	ctx.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// getConversationList returns list of conversations sorted by most recent message
func (server *Server) getConversationList(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	// Fix: NullUUID
	conversations, err := server.store.GetConversationList(ctx, uuid.NullUUID{UUID: authPayload.UserID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Convert to response format
	type ConversationResponse struct {
		ID            uuid.UUID `json:"id"`
		Username      string    `json:"username"`
		FullName      string    `json:"full_name"`
		AvatarUrl     string    `json:"avatar_url"`
		LastMessage   string    `json:"last_message"`
		LastMessageAt time.Time `json:"last_message_at"`
		LastSenderID  uuid.UUID `json:"last_sender_id"`
		UnreadCount   int64     `json:"unread_count"`
		IsBlocked     bool      `json:"is_blocked"`
	}

	response := make([]ConversationResponse, len(conversations))
	for i, conv := range conversations {
		unreadCount := int64(0)
		if count, ok := conv.UnreadCount.(int64); ok {
			unreadCount = count
		}

		// Check if blocked
		isBlocked, _ := server.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
			BlockerID: authPayload.UserID,
			BlockedID: conv.ID,
		})

		response[i] = ConversationResponse{
			ID:            conv.ID,
			Username:      conv.Username,
			FullName:      conv.FullName,
			AvatarUrl:     conv.AvatarUrl.String,
			LastMessage:   conv.LastMessage,
			LastMessageAt: conv.LastMessageAt,
			LastSenderID:  conv.LastSenderID,
			UnreadCount:   unreadCount,
			IsBlocked:     isBlocked,
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// deleteConversation deletes all messages between the authenticated user and another user
func (server *Server) deleteConversation(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, ok := parseUUIDParam(ctx, userIDStr, "user_id")
	if !ok {
		return
	}

	authPayload := getAuthPayload(ctx)

	// Delete all messages in the conversation
	err := server.store.DeleteConversation(ctx, db.DeleteConversationParams{
		SenderID:   authPayload.UserID,
		ReceiverID: uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate cache
	server.invalidateConversationCache(authPayload.UserID, userID)
	server.invalidateUnreadCountCache(authPayload.UserID)

	ctx.JSON(http.StatusOK, gin.H{"message": "conversation deleted"})
}

// getIcebreakers returns a list of suggested messages to start a conversation
func (server *Server) getIcebreakers(ctx *gin.Context) {
	targetIDStr := ctx.Query("user_id")
	targetID, ok := parseUUIDParam(ctx, targetIDStr, "user_id")
	if !ok {
		// If no user_id, return generic ones
		ctx.JSON(http.StatusOK, gin.H{
			"icebreakers": []string{
				"Hey there! 👋",
				"Hi! How is your day going?",
				"Hello! I'd love to connect.",
				"Hey! What's up?",
			},
		})
		return
	}

	authPayload := getAuthPayload(ctx)
	
	var icebreakers []string

	// 1. Check for recent crossing
	crossing, err := server.store.GetLatestCrossingBetweenUsers(ctx, db.GetLatestCrossingBetweenUsersParams{
		UserID1: authPayload.UserID,
		UserID2: targetID,
	})

	if err == nil {
		// We have a crossing!
		icebreakers = append(icebreakers, "Hey! I saw we crossed paths recently. How are you? 👋")
		
		// If it was very recent (last 1 hour)
		if time.Since(crossing.OccurredAt) < 1*time.Hour {
			icebreakers = append(icebreakers, "Hi! Just saw you nearby. Hope you're having a good time! ✨")
		}
	}

	// 2. Add generic/fun ones
	icebreakers = append(icebreakers, []string{
		"Hey! I'm new here, just saying hi! 👋",
		"Hi! Love your profile! How are you?",
		"Hey! Do you live around here too? 📍",
		"Hello! Hope you're having an awesome day! 🌟",
	}...)

	// Shuffle or limit? Let's just return all for now (frontend can pick)
	ctx.JSON(http.StatusOK, gin.H{
		"icebreakers": icebreakers,
	})
}
