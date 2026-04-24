package story

import (
	"context"
	"database/sql"
	"errors"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/realtime"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/safety"
)

type CreateStoryParams struct {
	UserID       uuid.UUID
	MediaURL     string
	MediaType    string
	Latitude     float64
	Longitude    float64
	Caption      string
	IsAnonymous  bool
	ShowLocation bool
}

type GetFeedParams struct {
	UserID    uuid.UUID
	Latitude  float64
	Longitude float64
}

type Service interface {
	CreateStory(ctx context.Context, params CreateStoryParams) (*db.CreateStoryRow, error)
	GetFeed(ctx context.Context, params GetFeedParams) ([]db.GetStoriesWithinRadiusRow, string, float64, error)
	GetMapStories(ctx context.Context, params GetFeedParams) ([]db.GetStoriesWithinRadiusRow, error)
	GetMyStories(ctx context.Context, userID uuid.UUID) ([]db.GetActiveStoriesByUserIDRow, error)
	DeleteStory(ctx context.Context, storyID uuid.UUID, userID uuid.UUID) error
}

type ServiceImpl struct {
	store  repository.Store
	redis  *redis.Client
	safety *safety.Monitor
	hub    *realtime.Hub
}

func NewService(store repository.Store, rdb *redis.Client, safety *safety.Monitor, hub *realtime.Hub) Service {
	return &ServiceImpl{
		store:  store,
		redis:  rdb,
		safety: safety,
		hub:    hub,
	}
}

func (s *ServiceImpl) CreateStory(ctx context.Context, req CreateStoryParams) (*db.CreateStoryRow, error) {
	hash := geohash.Encode(req.Latitude, req.Longitude)

	// Safety Check: Fake GPS
	val := s.safety.ValidateUserMovement(ctx, req.UserID.String(), req.Latitude, req.Longitude)
	if !val.Allowed {
		if val.ShouldBan {
			log.Warn().
				Str("user_id", req.UserID.String()).
				Float64("lat", req.Latitude).
				Float64("lng", req.Longitude).
				Msg("Fake GPS detected (Dev Bypass: User not banned)")
		}
	}

	user, err := s.store.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	expiryDuration := 24 * time.Hour
	isPremium := false
	if user.IsPremium.Valid && user.IsPremium.Bool {
		expiryDuration = 48 * time.Hour
		isPremium = true
	}
	expiresAt := time.Now().UTC().Add(expiryDuration)

	var captionNull sql.NullString
	if req.Caption != "" {
		captionNull = sql.NullString{String: req.Caption, Valid: true}
	}

	story, err := s.store.CreateStory(ctx, db.CreateStoryParams{
		UserID:       req.UserID,
		MediaUrl:     req.MediaURL,
		MediaType:    req.MediaType,
		Caption:      captionNull,
		Geohash:      hash,
		Lng:          req.Longitude,
		Lat:          req.Latitude,
		IsAnonymous:  req.IsAnonymous,
		ShowLocation: req.ShowLocation,
		IsPremium:    sql.NullBool{Bool: isPremium, Valid: true},
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return nil, err
	}

	_, err = s.store.UpdateUserActivity(ctx, req.UserID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update user activity")
	}

	// Gamification: Increment daily stories posted
	s.store.IncrementDailyStats(ctx, db.IncrementDailyStatsParams{
		UserID:        req.UserID,
		Date:          time.Now().UTC(),
		StoriesPosted: int32(1),
	})

	userGeohash := hash
	if len(userGeohash) > 5 {
		userGeohash = userGeohash[:5]
	}
	s.invalidateFeedCache(ctx, userGeohash)

	// Notify nearby users about the new post
	go s.notifyNearbyUsers(context.Background(), story)

	return &story, nil
}

func (s *ServiceImpl) notifyNearbyUsers(ctx context.Context, story db.CreateStoryRow) {
	// Radius for nearby post alert: 2km
	const alertRadius = 2.0 // in km for GeoSearch
	const userLocationsKey = "users:locations"

	matches, err := s.redis.GeoSearchLocation(ctx, userLocationsKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  story.Lng.(float64),
			Latitude:   story.Lat.(float64),
			Radius:     alertRadius,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
	}).Result()

	if err != nil {
		log.Error().Err(err).Msg("failed to find nearby users for story alert using GeoSearch")
		return
	}

	for _, match := range matches {
		targetIDStr := match.Name
		if targetIDStr == story.UserID.String() {
			continue
		}

		targetID, err := uuid.Parse(targetIDStr)
		if err != nil {
			continue
		}

		// Deduplication: 5 minutes per user for ANY nearby story alert
		dedupKey := fmt.Sprintf("alert_dedup:nearby_story:%s", targetID.String())
		val, _ := s.redis.SetNX(ctx, dedupKey, "1", 5*time.Minute).Result()
		if !val {
			continue // Already notified recently
		}

		// Create DB notification
		notif, err := s.store.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:         targetID,
			Type:           "nearby_story",
			SubType:        sql.NullString{String: "nearby_story", Valid: true},
			Sound:          sql.NullString{String: "soft_ping.wav", Valid: true},
			Title:          "Nearby Activity!",
			Message:        "Someone just posted a new story near you",
			RelatedStoryID: uuid.NullUUID{UUID: story.ID, Valid: true},
		})
		if err != nil {
			continue
		}

		// Push WebSocket alert
		if s.hub != nil {
			msg := realtime.WSMessage{
				Type:    "new_nearby_post",
				SubType: "nearby_story",
				Sound:   "soft_ping.wav",
				Payload: notif,
			}
			data, _ := json.Marshal(msg)
			s.hub.SendToUser(targetID, data)
		}
	}
}

func (s *ServiceImpl) GetFeed(ctx context.Context, params GetFeedParams) ([]db.GetStoriesWithinRadiusRow, string, float64, error) {
	radii := []float64{5000.0, 10000.0, 20000.0, 50000.0}
	var stories []db.GetStoriesWithinRadiusRow
	var err error
	var currentRadius float64

	for _, radius := range radii {
		currentRadius = radius
		stories, err = s.store.GetStoriesWithinRadius(ctx, db.GetStoriesWithinRadiusParams{
			Lng:          params.Longitude,
			Lat:          params.Latitude,
			RadiusMeters: radius,
			UserID:       params.UserID,
		})
		if err != nil {
			return nil, "", 0, err
		}

		if len(stories) > 0 {
			break
		}
	}

	sort.Slice(stories, func(i, j int) bool {
		return stories[i].CreatedAt.After(stories[j].CreatedAt)
	})

	var message string
	if len(stories) > 0 {
		if currentRadius <= 5000.0 {
			message = "Stories found nearby"
		} else {
			message = "Expanded search to find nearby stories"
		}
	} else {
		message = "No stories found within 20km. Be the first to share!"
	}

	return stories, message, currentRadius, nil
}

func (s *ServiceImpl) GetMapStories(ctx context.Context, params GetFeedParams) ([]db.GetStoriesWithinRadiusRow, error) {
	// For map, we use a larger radius or specific bounds. 
	// To reuse GetStoriesWithinRadius for now, we use a large radius.
	// Ideally we'd use GetStoriesInBounds, but let's stick to radius for consistency with Feed for now.
	const mapRadius = 100000.0 // 100km for map view

	stories, err := s.store.GetStoriesWithinRadius(ctx, db.GetStoriesWithinRadiusParams{
		Lng:          params.Longitude,
		Lat:          params.Latitude,
		RadiusMeters: mapRadius,
		UserID:       params.UserID,
	})
	if err != nil {
		return nil, err
	}

	connections, err := s.store.ListConnections(ctx, params.UserID)
	if err != nil {
		return nil, err
	}

	connectionMap := make(map[uuid.UUID]bool)
	for _, c := range connections {
		connectionMap[c.ID] = true
	}

	sort.Slice(stories, func(i, j int) bool {
		return stories[i].CreatedAt.After(stories[j].CreatedAt)
	})

	return stories, nil
}

func (s *ServiceImpl) GetMyStories(ctx context.Context, userID uuid.UUID) ([]db.GetActiveStoriesByUserIDRow, error) {
	stories, err := s.store.GetActiveStoriesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return stories, nil
}

func (s *ServiceImpl) DeleteStory(ctx context.Context, storyID uuid.UUID, userID uuid.UUID) error {
	story, err := s.store.GetStoryByID(ctx, storyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("story not found")
		}
		return err
	}

	if story.UserID != userID {
		return errors.New("you can only delete your own stories")
	}

	err = s.store.DeleteStory(ctx, storyID)
	if err != nil {
		return err
	}

	userGeohash := story.Geohash
	if len(userGeohash) > 5 {
		userGeohash = userGeohash[:5]
	}
	s.invalidateFeedCache(ctx, userGeohash)

	return nil
}

func (s *ServiceImpl) invalidateFeedCache(ctx context.Context, geohash string) {
	cacheKey := "feed:" + geohash
	s.redis.Del(ctx, cacheKey)
}
