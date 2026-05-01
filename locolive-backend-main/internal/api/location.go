package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mmcloughlin/geohash"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

const (
	locationPrecision = 7                // +/- 76m approx
	bucketDuration    = 10 * time.Minute // 10 min time buckets
	locationTTL       = 24 * time.Hour
)

type updateLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
}

func (server *Server) updateLocation(ctx *gin.Context) {
	var req updateLocationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	userIDStr := authPayload.UserID.String()

	log.Debug().
		Str("user_id", userIDStr).
		Float64("lat", req.Latitude).Float64("lng", req.Longitude).
		Msg("[LocationPing] Incoming location update")

	// Ghost Mode Logic
	user, userErr := server.store.GetUserByID(ctx, authPayload.UserID)
	if userErr == nil && user.IsGhostMode {
		if user.GhostModeExpiresAt.Valid && time.Now().After(user.GhostModeExpiresAt.Time) {
			server.store.ToggleGhostMode(ctx, db.ToggleGhostModeParams{
				ID:                 authPayload.UserID,
				IsGhostMode:        false,
				GhostModeExpiresAt: sql.NullTime{},
			})
		} else {
			log.Debug().Str("user_id", userIDStr).Msg("[LocationPing] Skipped — ghost mode active")
			ctx.JSON(http.StatusOK, gin.H{"status": "ghost"})
			return
		}
	}

	hash := geohash.Encode(req.Latitude, req.Longitude)
	if len(hash) > locationPrecision {
		hash = hash[:locationPrecision]
	}

	// Safety Check: Fake GPS
	val := server.safety.ValidateUserMovement(ctx, userIDStr, req.Latitude, req.Longitude)
	if !val.Allowed {
		if val.ShouldBan {
			server.store.BanUser(ctx, db.BanUserParams{
				ID:             authPayload.UserID,
				IsShadowBanned: true,
			})
			log.Warn().Str("user_id", userIDStr).Msg("[LocationPing] User shadow-banned for fake GPS")
		}
		log.Debug().Str("user_id", userIDStr).Msg("[LocationPing] Skipped — safety check failed")
		ctx.JSON(http.StatusOK, gin.H{"status": "updated"})
		return
	}

	now := time.Now().UTC()
	bucketTime := now.Truncate(bucketDuration)
	expiresAt := now.Add(locationTTL)

	_, err := server.store.CreateLocation(ctx, db.CreateLocationParams{
		UserID:     authPayload.UserID,
		Geohash:    hash,
		Lng:        req.Longitude,
		Lat:        req.Latitude,
		TimeBucket: bucketTime,
		ExpiresAt:  expiresAt,
	})

	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("[LocationPing] Failed to persist location to DB")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_, err = server.store.UpdateUserActivity(ctx, authPayload.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("[LocationPing] Failed to update user activity")
	}

	// Redis GEO Update & Crossing Detection & Nearby Broadcast
	if err := server.location.UpdateUserLocation(ctx, authPayload.UserID, req.Latitude, req.Longitude); err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("[LocationPing] Redis location service error")
	}

	log.Debug().Str("user_id", userIDStr).Msg("[LocationPing] Complete")
	ctx.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (server *Server) getHeatmap(ctx *gin.Context) {
	data, err := server.store.GetHeatmapData(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	type heatmapPoint struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Weight    int64   `json:"weight"`
	}

	rsp := make([]heatmapPoint, len(data))
	for i, d := range data {
		lat, _ := d.Latitude.(float64)
		lng, _ := d.Longitude.(float64)
		rsp[i] = heatmapPoint{
			Latitude:  lat,
			Longitude: lng,
			Weight:    d.Weight,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getNearbyUsersRequest struct {
	Latitude  float64 `form:"lat" binding:"required,min=-90,max=90"`
	Longitude float64 `form:"lng" binding:"required,min=-180,max=180"`
	Radius    float64 `form:"radius,default=5"` // in km (Default 5km as per req)
}

type nearbyUserResponse struct {
	UserID    string  `json:"id"`
	Distance  float64 `json:"distance_km"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
	Username  string  `json:"username"`
	FullName  string  `json:"full_name"`
	AvatarUrl string  `json:"avatar_url"`
	Bio       string  `json:"bio"`
	Online    bool    `json:"online"`
}

func (server *Server) getNearbyUsers(ctx *gin.Context) {
	var req getNearbyUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	userIDStr := authPayload.UserID.String()

	log.Debug().
		Str("user_id", userIDStr).
		Float64("lat", req.Latitude).Float64("lng", req.Longitude).
		Float64("radius_km", req.Radius).
		Msg("[GetNearbyUsers] Request received")

	// Fetch from Redis via Service
	matches, err := server.location.GetNearbyUsers(ctx, authPayload.UserID, req.Latitude, req.Longitude, req.Radius)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Float64("lat", req.Latitude).Float64("lng", req.Longitude).Msg("[GetNearbyUsers] Failed to get nearby users")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch nearby users", "details": err.Error()})
		return
	}

	log.Debug().
		Str("user_id", userIDStr).
		Int("redis_matches", len(matches)).
		Msg("[GetNearbyUsers] Redis matches returned")

	seen := make(map[string]bool)
	rsp := make([]nearbyUserResponse, 0, len(matches))
	for _, match := range matches {
		if seen[match.Name] {
			continue
		}

		targetID, err := uuid.Parse(match.Name)
		if err != nil {
			continue
		}

		// CENTRAL PRIVACY CHECK: Filter out blocked users
		result := server.privacy.CanViewProfile(ctx, authPayload.UserID, targetID)
		if !result.Allowed {
			continue // Blocked, Panic, or Ghost (Invisible)
		}

		user, err := server.store.GetUserByID(ctx, targetID)

		avatar := ""
		if user.AvatarUrl.Valid {
			avatar = user.AvatarUrl.String
		}
		bio := ""
		if user.Bio.Valid {
			bio = user.Bio.String
		}

		// Calculate online status (active in last 5 mins)
		online := false
		if user.LastActiveAt.Valid && time.Since(user.LastActiveAt.Time) < 5*time.Minute {
			online = true
		}

		rsp = append(rsp, nearbyUserResponse{
			UserID:    match.Name,
			Distance:  match.Dist,
			Latitude:  match.Latitude,
			Longitude: match.Longitude,
			Username:  user.Username,
			FullName:  user.FullName,
			AvatarUrl: avatar,
			Bio:       bio,
			Online:    online,
		})
		seen[match.Name] = true
	}

	log.Info().
		Str("user_id", userIDStr).
		Int("response_count", len(rsp)).
		Msg("[GetNearbyUsers] Returning nearby users")

	ctx.JSON(http.StatusOK, rsp)
}
