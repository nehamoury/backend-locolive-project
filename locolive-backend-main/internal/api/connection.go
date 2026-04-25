package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

type friendResponse struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	FullName     string     `json:"full_name"`
	AvatarUrl    string     `json:"avatar_url"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

type connectionResponse struct {
	RequesterID   uuid.UUID        `json:"requester_id"`
	TargetID      uuid.UUID        `json:"target_id"`
	Status        string           `json:"status"`
	CreatedAt     time.Time        `json:"created_at"`
	Username      string           `json:"username"`
	FullName      string           `json:"full_name"`
	AvatarUrl     string           `json:"avatar_url"`
	MutualFriends []friendResponse `json:"mutual_friends"`
	MutualCount   int64            `json:"mutual_count"`
}

func (server *Server) listConnections(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	connections, err := server.store.ListConnections(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]friendResponse, len(connections))
	for i, c := range connections {
		var lastActive *time.Time
		if c.LastActiveAt.Valid {
			lastActive = &c.LastActiveAt.Time
		}
		rsp[i] = friendResponse{
			ID:           c.ID,
			Username:     c.Username,
			FullName:     c.FullName,
			AvatarUrl:    c.AvatarUrl.String,
			LastActiveAt: lastActive,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) listFollowers(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// In Locolive, an accepted connection is bi-directional.
	query := `
		SELECT u.id, u.username, u.full_name, u.avatar_url, u.last_active_at
		FROM connections c
		JOIN users u ON (u.id = c.requester_id OR u.id = c.target_id)
		WHERE (c.requester_id = $1 OR c.target_id = $1)
		  AND c.status = 'accepted'
		  AND u.id != $1
	`
	rows, err := server.store.GetDB().QueryContext(ctx, query, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	defer rows.Close()

	var rsp []friendResponse
	for rows.Next() {
		var c friendResponse
		var lastActive sql.NullTime
		var avatarUrl sql.NullString
		if err := rows.Scan(&c.ID, &c.Username, &c.FullName, &avatarUrl, &lastActive); err != nil {
			continue
		}
		if lastActive.Valid {
			c.LastActiveAt = &lastActive.Time
		}
		c.AvatarUrl = avatarUrl.String
		rsp = append(rsp, c)
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) listFollowing(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// In Locolive, an accepted connection is bi-directional.
	query := `
		SELECT u.id, u.username, u.full_name, u.avatar_url, u.last_active_at
		FROM connections c
		JOIN users u ON (u.id = c.requester_id OR u.id = c.target_id)
		WHERE (c.requester_id = $1 OR c.target_id = $1)
		  AND c.status = 'accepted'
		  AND u.id != $1
	`
	rows, err := server.store.GetDB().QueryContext(ctx, query, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	defer rows.Close()

	var rsp []friendResponse
	for rows.Next() {
		var c friendResponse
		var lastActive sql.NullTime
		var avatarUrl sql.NullString
		if err := rows.Scan(&c.ID, &c.Username, &c.FullName, &avatarUrl, &lastActive); err != nil {
			continue
		}
		if lastActive.Valid {
			c.LastActiveAt = &lastActive.Time
		}
		c.AvatarUrl = avatarUrl.String
		rsp = append(rsp, c)
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) listPendingRequests(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	requests, err := server.store.ListPendingRequests(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]connectionResponse, len(requests))
	for i, r := range requests {
		var mutualFriends []friendResponse
		if mfBytes, ok := r.MutualFriends.([]byte); ok && len(mfBytes) > 0 {
			if err := json.Unmarshal(mfBytes, &mutualFriends); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal mutual friends")
			}
		}

		rsp[i] = connectionResponse{
			RequesterID:   r.RequesterID,
			TargetID:      r.TargetID,
			Status:        string(r.Status),
			CreatedAt:     r.CreatedAt,
			Username:      r.Username,
			FullName:      r.FullName,
			AvatarUrl:     r.AvatarUrl.String,
			MutualFriends: mutualFriends,
			MutualCount:   r.MutualCount,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) listSentRequests(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	requests, err := server.store.ListSentConnectionRequests(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]connectionResponse, len(requests))
	for i, r := range requests {
		rsp[i] = connectionResponse{
			RequesterID: r.RequesterID,
			TargetID:    r.TargetID,
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt,
			Username:    r.Username,
			FullName:    r.FullName,
			AvatarUrl:   r.AvatarUrl.String,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

type connectionRequest struct {
	TargetUserID string `json:"target_user_id" binding:"required,uuid"`
}

func (server *Server) sendConnectionRequest(ctx *gin.Context) {
	var req connectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	targetID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if targetID == authPayload.UserID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot connect with yourself"})
		return
	}

	// 1. Check for an existing connection/request
	existing, err := server.store.GetConnection(ctx, db.GetConnectionParams{
		RequesterID: authPayload.UserID,
		TargetID:    targetID,
	})

	if err == nil {
		if existing.Status == "accepted" {
			ctx.JSON(http.StatusOK, gin.H{"message": "already connected", "is_match": false})
			return
		}

		// If the target user was the requester, this is a mutual match!
		if existing.RequesterID == targetID && existing.Status == "pending" {
			conn, err := server.store.UpdateConnectionStatus(ctx, db.UpdateConnectionStatusParams{
				RequesterID: targetID,
				TargetID:    authPayload.UserID,
				Status:      "accepted",
			})
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, errorResponse(err))
				return
			}

			// Broadcast real-time match event
			msg, _ := json.Marshal(gin.H{
				"type": "connection_accepted",
				"payload": gin.H{
					"requester_id": targetID,
					"target_id":    authPayload.UserID,
				},
			})
			server.hub.SendToUser(targetID, msg)
			server.hub.SendToUser(authPayload.UserID, msg)

			ctx.JSON(http.StatusOK, gin.H{
				"status":   conn.Status,
				"is_match": true,
			})
			return
		}

		// If I was the requester, it's just a duplicate
		if existing.RequesterID == authPayload.UserID && existing.Status == "pending" {
			ctx.JSON(http.StatusOK, gin.H{"message": "connection request already sent", "is_match": false})
			return
		}
	}

	// 2. Normal Request Logic
	count, err := server.store.CountConnectionRequestsToday(ctx, authPayload.UserID)
	if err == nil && count >= 20 {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "daily connection request limit reached (20/day)"})
		return
	}

	conn, err := server.store.CreateConnectionRequest(ctx, db.CreateConnectionRequestParams{
		RequesterID: authPayload.UserID,
		TargetID:    targetID,
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusOK, gin.H{"message": "connection request already sent", "is_match": false})
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Auto-accept if the target user is public
	targetUser, err := server.store.GetUserByID(ctx, targetID)
	if err == nil && !targetUser.IsPrivate {
		conn, err = server.store.UpdateConnectionStatus(ctx, db.UpdateConnectionStatusParams{
			RequesterID: authPayload.UserID,
			TargetID:    targetID,
			Status:      "accepted",
		})
		
		if err == nil {
			// Notify about the accepted connection
			server.createNotificationWithSound(ctx, targetID, "connection_accepted", "connection",
				"New Connection", "Someone started following you!",
				map[string]uuid.UUID{"user": authPayload.UserID})
			
			// Invalidate profile caches
			server.redis.Del(ctx, "profile:"+authPayload.UserID.String())
			server.redis.Del(ctx, "profile:"+targetID.String())

			ctx.JSON(http.StatusCreated, gin.H{
				"status":   conn.Status,
				"is_match": false,
			})
			return
		}
	}

	// Create notification for pending request
	server.createNotificationWithSound(ctx, targetID, "connection_request", "connection", 
		"New Connection Request", "Someone wants to connect!", 
		map[string]uuid.UUID{"user": authPayload.UserID})

	ctx.JSON(http.StatusCreated, gin.H{
		"status":   conn.Status,
		"is_match": false,
	})
}

type updateConnectionRequest struct {
	RequesterID string `json:"requester_id" binding:"required,uuid"`
	Status      string `json:"status" binding:"required,oneof=accepted blocked"`
}

func (server *Server) updateConnection(ctx *gin.Context) {
	var req updateConnectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	requesterID, ok := parseUUIDParam(ctx, req.RequesterID, "requester_id")
	if !ok {
		return
	}
	authPayload := getAuthPayload(ctx)

	conn, err := server.store.UpdateConnectionStatus(ctx, db.UpdateConnectionStatusParams{
		RequesterID: requesterID,
		TargetID:    authPayload.UserID, // I am the target accepting the request
		Status:      db.ConnectionStatus(req.Status),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "connection request not found or already handled"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Remove the 'connection_request' notification as it's now handled
	err = server.store.DeleteConnectionRequestNotifications(ctx, db.DeleteConnectionRequestNotificationsParams{
		UserID:        authPayload.UserID,
		RelatedUserID: uuid.NullUUID{UUID: requesterID, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete connection request notification")
	}

	// Create notification if connection was accepted
	if req.Status == "accepted" {
		accepter, err := server.store.GetUserByID(ctx, authPayload.UserID)
		if err == nil {
			_, err = server.createNotificationWithSound(ctx, requesterID, "connection_accepted", "connection",
				"Connection Accepted", fmt.Sprintf("%s accepted your connection request", accepter.Username),
				map[string]uuid.UUID{"user": authPayload.UserID})
			if err != nil {
				log.Error().Err(err).Msg("failed to create connection accepted notification")
			}
		}

		server.redis.Del(ctx, "profile:"+authPayload.UserID.String())
		server.redis.Del(ctx, "profile:"+requesterID.String())

		// Update Trust Scores for both parties
		go server.user.UpdateTrustScore(context.Background(), authPayload.UserID)
		go server.user.UpdateTrustScore(context.Background(), requesterID)

		// Real-time WebSocket notification to both parties
		msg, _ := json.Marshal(gin.H{
			"type": "connection_accepted",
			"payload": gin.H{
				"requester_id":       requesterID,
				"target_id":          authPayload.UserID,
				"requester_username": "the other user", // Simplified, frontend can refetch
				"target_username":    accepter.Username,
			},
		})
		server.hub.SendToUser(requesterID, msg)
		server.hub.SendToUser(authPayload.UserID, msg)
	}

	ctx.JSON(http.StatusOK, conn)
}

func (server *Server) deleteConnection(ctx *gin.Context) {
	targetUserIDStr := ctx.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err = server.store.DeleteConnection(ctx, db.DeleteConnectionParams{
		RequesterID: authPayload.UserID,
		TargetID:    targetUserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate profile caches for both users so connection count updates instantly
	server.redis.Del(ctx, "profile:"+authPayload.UserID.String())
	server.redis.Del(ctx, "profile:"+targetUserID.String())

	ctx.JSON(http.StatusOK, gin.H{"message": "connection deleted"})
}

type suggestedConnectionResponse struct {
	ID            uuid.UUID        `json:"id"`
	Username      string           `json:"username"`
	FullName      string           `json:"full_name"`
	AvatarUrl     string           `json:"avatar_url"`
	MutualFriends []friendResponse `json:"mutual_friends"`
	MutualCount   int64            `json:"mutual_count"`
	Bio           string           `json:"bio"`
	Distance      float64          `json:"distance_km"`
	IsVerified    bool             `json:"is_verified"`
	LastActiveAt  *time.Time       `json:"last_active_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

func (server *Server) getSuggestedConnections(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// 1. Fetch Mutual Friend Suggestions
	suggestions, err := server.store.GetSuggestedConnections(ctx, db.GetSuggestedConnectionsParams{
		RequesterID: authPayload.UserID,
		Limit:       12,
	})

	var rsp []suggestedConnectionResponse

	if err == nil {
		for _, s := range suggestions {
			var lastActive *time.Time
			if s.LastActiveAt.Valid {
				lastActive = &s.LastActiveAt.Time
			}

			var mutualFriends []friendResponse
			if mfBytes, ok := s.MutualFriends.([]byte); ok && len(mfBytes) > 0 {
				if err := json.Unmarshal(mfBytes, &mutualFriends); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal mutual friends")
				}
			}

			rsp = append(rsp, suggestedConnectionResponse{
				ID:            s.ID,
				Username:      s.Username,
				FullName:      s.FullName,
				AvatarUrl:     s.AvatarUrl.String,
				MutualFriends: mutualFriends,
				MutualCount:   s.MutualCount,
				Bio:           s.Bio.String,
				IsVerified:    s.IsVerified,
				LastActiveAt:  lastActive,
				CreatedAt:     s.CreatedAt,
			})
		}
	}

	// 2. Proximity Fallback (Option B)
	// If we have few mutual suggestions, add nearby users who are NOT yet connected
	if len(rsp) < 6 {
		// Fetch lat/lng from Redis
		pos, err := server.redis.GeoPos(ctx, "users:locations", authPayload.UserID.String()).Result()
		if err == nil && len(pos) > 0 && pos[0] != nil {
			lng := pos[0].Longitude
			lat := pos[0].Latitude
			radius := 50.0 // 50km
			matches, _ := server.redis.GeoRadius(ctx, "users:locations", lng, lat, &redis.GeoRadiusQuery{
				Radius:      radius * 1000,
				Unit:        "m",
				Sort:        "ASC",
				WithDist:    true,
			}).Result()

			for _, match := range matches {
				uid, _ := uuid.Parse(match.Name)
				if uid == authPayload.UserID {
					continue
				}

				// Check if already in response
				exists := false
				for _, r := range rsp {
					if r.ID == uid {
						exists = true
						break
					}
				}
				if exists {
					continue
				}

				// Check connection status
				_, err := server.store.GetConnection(ctx, db.GetConnectionParams{
					RequesterID: authPayload.UserID,
					TargetID:    uid,
				})
				if err == nil {
					continue // Already requested/connected
				}

				// Get user details
				u, err := server.store.GetUserByID(ctx, uid)
				if err == nil && !u.IsShadowBanned {
					rsp = append(rsp, suggestedConnectionResponse{
						ID:          u.ID,
						Username:    u.Username,
						FullName:    u.FullName,
						AvatarUrl:   u.AvatarUrl.String,
						MutualCount: 0,
						Bio:         u.Bio.String,
						Distance:    match.Dist / 1000.0,
					})
				}

				if len(rsp) >= 12 {
					break
				}
			}
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}
