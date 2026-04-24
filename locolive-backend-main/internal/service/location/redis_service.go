package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/realtime"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
)

const (
	userLocationsKey  = "users:locations"
	crossingKeyPrefix = "crossing:"
	lastPosKeyPrefix  = "user:lastpos:"

	crossingTTL          = 10 * time.Minute
	crossingRadiusMeters = 50.0
	nearbyRadiusKm       = 5.0
	minMovementMeters    = 20.0 // Skip updates if user moved less than this
)

type RedisLocationService struct {
	redis *redis.Client
	store repository.Store
	hub   *realtime.Hub
}

func NewRedisLocationService(redis *redis.Client, store repository.Store, hub *realtime.Hub) *RedisLocationService {
	return &RedisLocationService{
		redis: redis,
		store: store,
		hub:   hub,
	}
}

// UpdateUserLocation updates user position in Redis, broadcasts nearby updates, and detects crossings
func (s *RedisLocationService) UpdateUserLocation(ctx context.Context, userID uuid.UUID, lat, lng float64) error {
	userIDStr := userID.String()

	// ── Step 0: Distance-based optimization ─────────────────────────────
	// Skip heavy processing if user barely moved (<20m)
	significantMove := true
	lastPosKey := lastPosKeyPrefix + userIDStr
	lastPosData, err := s.redis.Get(ctx, lastPosKey).Result()
	if err == nil && lastPosData != "" {
		var lastPos [2]float64
		if json.Unmarshal([]byte(lastPosData), &lastPos) == nil {
			dist := haversineMeters(lastPos[0], lastPos[1], lat, lng)
			if dist < minMovementMeters {
				significantMove = false
				log.Debug().
					Str("user_id", userIDStr).
					Float64("distance_m", dist).
					Msg("[Location] Skip — movement below threshold")
			}
		}
	}

	// ── Step 1: Always update GEO index (keeps TTL alive) ───────────────
	err = s.redis.GeoAdd(ctx, userLocationsKey, &redis.GeoLocation{
		Name:      userIDStr,
		Longitude: lng,
		Latitude:  lat,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to update geo location: %w", err)
	}
	log.Debug().
		Str("user_id", userIDStr).
		Float64("lat", lat).Float64("lng", lng).
		Msg("[Location] Redis GEOADD complete")

	// Save last position for future delta checks
	posBytes, _ := json.Marshal([2]float64{lat, lng})
	s.redis.Set(ctx, lastPosKey, string(posBytes), 30*time.Minute)

	if !significantMove {
		return nil // GEO updated but no broadcasts needed
	}

	// ── Step 2: Fetch nearby users within discovery radius (5km) ────────
	nearbyMatches, err := s.redis.GeoSearchLocation(ctx, userLocationsKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     nearbyRadiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      100,
		},
		WithDist:  true,
		WithCoord: true,
	}).Result()
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("[Location] Failed GEOSEARCH for nearby users")
		return nil
	}

	log.Info().
		Str("user_id", userIDStr).
		Int("nearby_count", len(nearbyMatches)-1). // -1 to exclude self
		Msg("[Location] Nearby users found")

	// ── Step 3: Broadcast nearby_user_update to all nearby users ────────
	s.broadcastNearbyUpdates(ctx, userID, lat, lng, nearbyMatches)

	// ── Step 4: Detect user_left_radius ─────────────────────────────────
	s.detectLeftRadius(ctx, userID, nearbyMatches)

	// ── Step 5: Crossing detection (50m) ────────────────────────────────
	crossingMatches, err := s.redis.GeoSearchLocation(ctx, userLocationsKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     crossingRadiusMeters,
			RadiusUnit: "m",
			Sort:       "ASC",
		},
		WithDist:  true,
		WithCoord: true,
	}).Result()
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("[Location] Failed GEOSEARCH for crossings")
		return nil
	}

	if len(crossingMatches) > 1 {
		log.Info().
			Str("user_id", userIDStr).
			Int("crossing_candidates", len(crossingMatches)-1).
			Msg("[Location] Crossing candidates detected")
	}

	s.processCrossings(ctx, userID, crossingMatches)

	return nil
}

// broadcastNearbyUpdates sends a nearby_user_update WS event to all users within range
func (s *RedisLocationService) broadcastNearbyUpdates(ctx context.Context, movedUserID uuid.UUID, lat, lng float64, matches []redis.GeoLocation) {
	movedUserIDStr := movedUserID.String()

	// Get the moving user's profile for the broadcast payload
	movedUser, err := s.store.GetUserByID(ctx, movedUserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", movedUserIDStr).Msg("[Nearby] Failed to get moved user profile")
		return
	}
	if movedUser.IsGhostMode || movedUser.IsShadowBanned {
		return
	}

	avatar := ""
	if movedUser.AvatarUrl.Valid {
		avatar = movedUser.AvatarUrl.String
	}
	bio := ""
	if movedUser.Bio.Valid {
		bio = movedUser.Bio.String
	}
	online := movedUser.LastActiveAt.Valid && time.Since(movedUser.LastActiveAt.Time) < 5*time.Minute

	for _, match := range matches {
		if match.Name == movedUserIDStr {
			continue
		}

		targetID, err := uuid.Parse(match.Name)
		if err != nil {
			continue
		}

		valid, err := s.validateCrossingPrivacy(ctx, movedUserID, targetID)
		if err != nil || !valid {
			continue
		}

		// Send to the nearby user: "this user is near you"
		payload := map[string]interface{}{
			"id":         movedUserIDStr,
			"username":   movedUser.Username,
			"full_name":  movedUser.FullName,
			"avatar_url": avatar,
			"bio":        bio,
			"lat":        lat,
			"lng":        lng,
			"distance":   match.Dist,
			"online":     online,
		}

		s.hub.SendToUser(targetID, s.formatAlert("nearby_user_update", payload))

		log.Debug().
			Str("moved_user", movedUserIDStr).
			Str("notified_user", match.Name).
			Float64("distance_km", match.Dist).
			Msg("[Nearby] Sent nearby_user_update")
	}
}

// detectLeftRadius checks which users the moving user was previously near but are no longer within 5km
func (s *RedisLocationService) detectLeftRadius(ctx context.Context, userID uuid.UUID, currentNearby []redis.GeoLocation) {
	userIDStr := userID.String()
	nearbySetKey := "nearby_set:" + userIDStr

	// Build set of current nearby user IDs
	currentIDs := make(map[string]bool)
	for _, m := range currentNearby {
		if m.Name != userIDStr {
			currentIDs[m.Name] = true
		}
	}

	// Get previous nearby set from Redis
	previousIDs, err := s.redis.SMembers(ctx, nearbySetKey).Result()
	if err != nil && err != redis.Nil {
		log.Error().Err(err).Msg("[LeftRadius] Failed to read previous nearby set")
	}

	// Detect users who left radius
	for _, prevID := range previousIDs {
		if !currentIDs[prevID] {
			targetID, err := uuid.Parse(prevID)
			if err != nil {
				continue
			}

			// Notify the moving user that this person left their radius
			s.hub.SendToUser(userID, s.formatAlert("user_left_radius", map[string]interface{}{
				"user_id": prevID,
			}))
			// Notify the other user too
			s.hub.SendToUser(targetID, s.formatAlert("user_left_radius", map[string]interface{}{
				"user_id": userIDStr,
			}))

			log.Debug().
				Str("user_id", userIDStr).
				Str("left_user", prevID).
				Msg("[LeftRadius] User left nearby radius")
		}
	}

	// Update the stored nearby set
	pipe := s.redis.Pipeline()
	pipe.Del(ctx, nearbySetKey)
	if len(currentIDs) > 0 {
		members := make([]interface{}, 0, len(currentIDs))
		for id := range currentIDs {
			members = append(members, id)
		}
		pipe.SAdd(ctx, nearbySetKey, members...)
		pipe.Expire(ctx, nearbySetKey, 10*time.Minute)
	}
	pipe.Exec(ctx)
}

// GetNearbyUsers fetches filtered users within a radius using GEOSEARCH
func (s *RedisLocationService) GetNearbyUsers(ctx context.Context, userID uuid.UUID, lat, lng float64, radiusKm float64) ([]redis.GeoLocation, error) {
	// Validate service dependencies
	if s.redis == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	// Validate input parameters
	if radiusKm <= 0 {
		radiusKm = 5.0 // Default to 5km if invalid
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, fmt.Errorf("invalid coordinates: lat=%f, lng=%f", lat, lng)
	}

	// Check Redis connectivity before query
	if err := s.redis.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("[GetNearby] Redis ping failed")
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	matches, err := s.redis.GeoSearchLocation(ctx, userLocationsKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      50,
		},
		WithDist:  true,
		WithCoord: true,
	}).Result()
	if err != nil {
		log.Error().Err(err).Float64("radius_km", radiusKm).Float64("lat", lat).Float64("lng", lng).Msg("[GetNearby] Redis GEOSEARCH failed")
		return nil, fmt.Errorf("failed to search nearby users: %w", err)
	}

	log.Debug().
		Str("user_id", userID.String()).
		Float64("lat", lat).Float64("lng", lng).
		Float64("radius_km", radiusKm).
		Int("raw_matches", len(matches)).
		Msg("[GetNearby] Redis GEOSEARCH result")

	var filtered []redis.GeoLocation
	for _, match := range matches {
		if match.Name == userID.String() {
			continue
		}

		targetID, err := uuid.Parse(match.Name)
		if err != nil {
			continue
		}

		valid, err := s.validateCrossingPrivacy(ctx, userID, targetID)
		if err != nil {
			log.Debug().Str("target_id", targetID.String()).Err(err).Msg("[GetNearby] Privacy check failed, skipping")
			continue
		}
		if !valid {
			continue
		}

		filtered = append(filtered, match)
	}

	log.Debug().
		Str("user_id", userID.String()).
		Int("filtered_count", len(filtered)).
		Msg("[GetNearby] Filtered nearby users")

	return filtered, nil
}

func (s *RedisLocationService) processCrossings(ctx context.Context, userID uuid.UUID, matches []redis.GeoLocation) {
	for _, match := range matches {
		targetUserIDStr := match.Name
		if targetUserIDStr == userID.String() {
			continue
		}

		targetUserID, err := uuid.Parse(targetUserIDStr)
		if err != nil {
			continue
		}

		u1, u2 := userID, targetUserID
		if u1.String() > u2.String() {
			u1, u2 = u2, u1
		}

		// De-duplication: skip if same pair crossed within TTL (10 min)
		dedupKey := fmt.Sprintf("%s%s:%s", crossingKeyPrefix, u1.String(), u2.String())
		exists, err := s.redis.Exists(ctx, dedupKey).Result()
		if err == nil && exists > 0 {
			log.Debug().
				Str("u1", u1.String()).Str("u2", u2.String()).
				Msg("[Crossing] Skipped — dedup active")
			continue
		}

		valid, err := s.validateCrossingPrivacy(ctx, userID, targetUserID)
		if err != nil || !valid {
			continue
		}

		centerHash := geohash.Encode(match.Latitude, match.Longitude)

		crossing, err := s.store.CreateCrossing(ctx, db.CreateCrossingParams{
			UserID1:        u1,
			UserID2:        u2,
			LocationCenter: centerHash,
			OccurredAt:     time.Now().UTC().Truncate(10 * time.Minute),
		})
		if err != nil {
			log.Error().Err(err).Msg("[Crossing] Failed to persist crossing")
			continue
		}

		log.Info().
			Str("u1", u1.String()).Str("u2", u2.String()).
			Float64("distance_m", match.Dist).
			Msg("[Crossing] New crossing created")

		// Connection-aware notifications
		conn, err := s.store.GetConnection(ctx, db.GetConnectionParams{
			RequesterID: userID,
			TargetID:    targetUserID,
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
		notif1, err := s.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:            userID,
			Type:              "crossing_detected",
			SubType:           sql.NullString{String: "crossing", Valid: true},
			Sound:             sql.NullString{String: "soft_ping.wav", Valid: true},
			Title:             title,
			Message:           message,
			RelatedUserID:     uuid.NullUUID{UUID: targetUserID, Valid: true},
			RelatedCrossingID: uuid.NullUUID{UUID: crossing.ID, Valid: true},
		})
		if err != nil {
			log.Error().Err(err).Msg("[Crossing] Failed to create notification for user1")
		} else if s.hub != nil {
			s.hub.SendToUser(userID, s.formatAlert("crossing_detected", notif1))
		}

		// Notification for user2
		notif2, err := s.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:            targetUserID,
			Type:              "crossing_detected",
			SubType:           sql.NullString{String: "crossing", Valid: true},
			Sound:             sql.NullString{String: "soft_ping.wav", Valid: true},
			Title:             title,
			Message:           message,
			RelatedUserID:     uuid.NullUUID{UUID: userID, Valid: true},
			RelatedCrossingID: uuid.NullUUID{UUID: crossing.ID, Valid: true},
		})
		if err != nil {
			log.Error().Err(err).Msg("[Crossing] Failed to create notification for user2")
		} else if s.hub != nil {
			s.hub.SendToUser(targetUserID, s.formatAlert("crossing_detected", notif2))
		}

		s.invalidateCrossingsCache(ctx, userID)
		s.invalidateCrossingsCache(ctx, targetUserID)

		s.redis.Set(ctx, dedupKey, "1", crossingTTL)
	}
}

func (s *RedisLocationService) validateCrossingPrivacy(ctx context.Context, u1, u2 uuid.UUID) (bool, error) {
	blocked, err := s.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
		BlockerID: u1,
		BlockedID: u2,
	})
	if err != nil {
		log.Error().Err(err).Str("u1", u1.String()).Str("u2", u2.String()).Msg("[Privacy] block check failed")
		return false, fmt.Errorf("block check failed: %w", err)
	}
	if blocked {
		log.Debug().Str("u1", u1.String()).Str("u2", u2.String()).Msg("[Privacy] u1 blocked u2")
		return false, nil
	}

	blockedReverse, err := s.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
		BlockerID: u2,
		BlockedID: u1,
	})
	if err != nil {
		log.Error().Err(err).Str("u1", u1.String()).Str("u2", u2.String()).Msg("[Privacy] block check reverse failed")
		return false, fmt.Errorf("block check reverse failed: %w", err)
	}
	if blockedReverse {
		log.Debug().Str("u1", u1.String()).Str("u2", u2.String()).Msg("[Privacy] u2 blocked u1")
		return false, nil
	}

	user1, err := s.store.GetUserByID(ctx, u1)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug().Str("u1", u1.String()).Msg("[Privacy] u1 not found in DB, skipping")
			return false, nil
		}
		log.Error().Err(err).Str("u1", u1.String()).Msg("[Privacy] get user1 failed")
		return false, fmt.Errorf("get user1 failed: %w", err)
	}
	if user1.IsGhostMode {
		log.Debug().Str("u1", u1.String()).Msg("[Privacy] u1 is in ghost mode")
		return false, nil
	}
	if user1.IsShadowBanned {
		log.Debug().Str("u1", u1.String()).Msg("[Privacy] u1 is shadowbanned")
		return false, nil
	}

	user2, err := s.store.GetUserByID(ctx, u2)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug().Str("u2", u2.String()).Msg("[Privacy] u2 not found in DB, skipping")
			return false, nil
		}
		log.Error().Err(err).Str("u2", u2.String()).Msg("[Privacy] get user2 failed")
		return false, fmt.Errorf("get user2 failed: %w", err)
	}
	if user2.IsGhostMode {
		log.Debug().Str("u2", u2.String()).Msg("[Privacy] u2 is in ghost mode")
		return false, nil
	}
	if user2.IsShadowBanned {
		log.Debug().Str("u2", u2.String()).Msg("[Privacy] u2 is shadowbanned")
		return false, nil
	}

	log.Debug().Str("u1", u1.String()).Str("u2", u2.String()).Msg("[Privacy] passed")
	return true, nil
}

func (s *RedisLocationService) formatAlert(msgType string, payload interface{}) []byte {
	sound := ""
	switch msgType {
	case "nearby_user_update":
		sound = "soft_ping.wav"
	case "crossing_detected":
		sound = "soft_ping.wav"
	}

	wsMsg := realtime.WSMessage{
		Type:    msgType,
		SubType: "location",
		Sound:   sound,
		Payload: payload,
	}
	data, _ := json.Marshal(wsMsg)
	return data
}

// GetUserLocation retrieves the user's current location from Redis GEO store
func (s *RedisLocationService) GetUserLocation(ctx context.Context, userID uuid.UUID) (lat, lng float64, exists bool, err error) {
	positions, err := s.redis.GeoPos(ctx, userLocationsKey, userID.String()).Result()
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to get user position: %w", err)
	}
	if len(positions) == 0 || positions[0] == nil {
		return 0, 0, false, nil
	}
	return positions[0].Latitude, positions[0].Longitude, true, nil
}

func (s *RedisLocationService) invalidateCrossingsCache(ctx context.Context, userID uuid.UUID) {
	cacheKey := "crossings:v3:" + userID.String()
	s.redis.Del(ctx, cacheKey)
}

// haversineMeters calculates the distance between two lat/lng points in meters
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371e3 // Earth radius in meters
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// CalculateDistanceKm calculates distance between two points in kilometers
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	return haversineMeters(lat1, lng1, lat2, lng2) / 1000.0
}
