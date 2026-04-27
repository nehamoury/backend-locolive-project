package privacy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
)

// AccessDeniedReason represents why access was denied
type AccessDeniedReason string

const (
	ReasonAllowed   AccessDeniedReason = ""
	ReasonBlocked   AccessDeniedReason = "blocked"
	ReasonPanicMode AccessDeniedReason = "panic_mode"
	ReasonGhostMode AccessDeniedReason = "ghost_mode"
	ReasonPrivate   AccessDeniedReason = "private"
	ReasonHidden    AccessDeniedReason = "hidden"
	ReasonDeleted   AccessDeniedReason = "deleted"
	ReasonBanned    AccessDeniedReason = "banned"
)

// VisibilityState represents the user's activity-based visibility
type VisibilityState string

const (
	VisibilityActive VisibilityState = "active"
	VisibilityFading VisibilityState = "fading"
	VisibilityHidden VisibilityState = "hidden"
)

// AccessResult contains the result of an access check
type AccessResult struct {
	Allowed bool
	Reason  AccessDeniedReason
}

// --- Cache configuration ---

const (
	blockCachePrefix   = "privacy:blocked:"
	stateCachePrefix   = "privacy:state:"
	blockCacheTTL      = 5 * time.Minute
	stateCacheTTL      = 2 * time.Minute
)

// cachedPrivacyState is a serializable subset of the privacy state for Redis caching.
type cachedPrivacyState struct {
	IsGhostMode  bool       `json:"g"`
	PanicMode    bool       `json:"p"`
	IsPrivate    bool       `json:"pr"`
	LastActiveAt *time.Time `json:"la,omitempty"`
}

// Service provides centralized privacy and access control logic.
// All access decisions in the application must go through this service.
//
// Priority order (CRITICAL — evaluated in this exact sequence):
//  1. Block System (highest priority)
//  2. Panic Mode
//  3. Ghost Mode
//  4. Private Account
//  5. Visibility Engine (last_active_at)
//  6. Default (Public — allowed)
//
// NOTE: deleted_at and is_shadow_banned are handled at the DB query layer,
// NOT in this rule engine. Queries should include WHERE deleted_at IS NULL.
type Service struct {
	store repository.Store
	redis *redis.Client
}

// NewService creates a new Privacy Service
func NewService(store repository.Store, rdb *redis.Client) *Service {
	return &Service{
		store: store,
		redis: rdb,
	}
}

// CanUserAccess is the SINGLE SOURCE OF TRUTH for all access decisions.
// viewerID = the user trying to access content
// targetID = the user whose content is being accessed
func (s *Service) CanUserAccess(ctx context.Context, viewerID, targetID uuid.UUID) AccessResult {
	// Self-access is always allowed
	if viewerID == targetID {
		return AccessResult{Allowed: true}
	}

	// 1. BLOCK CHECK (Highest Priority) — bidirectional, cached
	blocked, err := s.isBlockedCached(ctx, viewerID, targetID)
	if err != nil {
		log.Error().Err(err).Msg("privacy: block check failed")
		return AccessResult{Allowed: false, Reason: ReasonBlocked}
	}
	if blocked {
		return AccessResult{Allowed: false, Reason: ReasonBlocked}
	}

	// 2-5. Get target user's privacy state (cached)
	state, err := s.getPrivacyStateCached(ctx, targetID)
	if err != nil {
		log.Error().Err(err).Msg("privacy: failed to get user privacy state")
		return AccessResult{Allowed: false, Reason: ReasonDeleted}
	}

	// 2. PANIC MODE CHECK
	if state.PanicMode {
		return AccessResult{Allowed: false, Reason: ReasonPanicMode}
	}

	// 3. GHOST MODE CHECK
	if state.IsGhostMode {
		return AccessResult{Allowed: false, Reason: ReasonGhostMode}
	}

	// 4. PRIVATE ACCOUNT CHECK
	if state.IsPrivate {
		isFollower, err := s.isFollower(ctx, viewerID, targetID)
		if err != nil {
			log.Error().Err(err).Msg("privacy: follower check failed")
			return AccessResult{Allowed: false, Reason: ReasonPrivate}
		}
		if !isFollower {
			return AccessResult{Allowed: false, Reason: ReasonPrivate}
		}
	}

	// 5. VISIBILITY ENGINE (last_active_at)
	visibility := s.getVisibilityFromState(state)
	if visibility == VisibilityHidden {
		return AccessResult{Allowed: false, Reason: ReasonHidden}
	}

	// 6. DEFAULT — Public access allowed
	return AccessResult{Allowed: true}
}

// CanViewProfile is a lenient variant for profile viewing.
// It still blocks for blocked/panic users, but returns partial results
// for private/ghost/hidden users (the API layer blanks out sensitive fields).
func (s *Service) CanViewProfile(ctx context.Context, viewerID, targetID uuid.UUID) AccessResult {
	if viewerID == targetID {
		return AccessResult{Allowed: true}
	}

	// 1. Block check (bidirectional, cached)
	blocked, err := s.isBlockedCached(ctx, viewerID, targetID)
	if err != nil {
		return AccessResult{Allowed: false, Reason: ReasonBlocked}
	}
	if blocked {
		return AccessResult{Allowed: false, Reason: ReasonBlocked}
	}

	state, err := s.getPrivacyStateCached(ctx, targetID)
	if err != nil {
		return AccessResult{Allowed: false, Reason: ReasonDeleted}
	}

	// 2. Panic mode — completely invisible
	if state.PanicMode {
		return AccessResult{Allowed: false, Reason: ReasonPanicMode}
	}

	// For profile viewing: ghost/private/hidden users CAN be found,
	// but their content will be restricted. This is handled by the API layer.
	if state.IsPrivate {
		isFollower, err := s.isFollower(ctx, viewerID, targetID)
		if err != nil || !isFollower {
			// Allowed=true but Reason=private tells the API to blank out sensitive data
			return AccessResult{Allowed: true, Reason: ReasonPrivate}
		}
	}

	return AccessResult{Allowed: true}
}

// GetVisibilityState returns the current visibility state of a user
func (s *Service) GetVisibilityState(lastActiveAt sql.NullTime) VisibilityState {
	if !lastActiveAt.Valid {
		return VisibilityHidden
	}
	return s.getVisibilityFromTime(lastActiveAt.Time)
}

// --- Cache Invalidation (MUST be called when state changes) ---

// InvalidateUserCache clears the cached privacy state for a user.
// Call after: panic toggle, ghost toggle, privacy toggle, profile update.
func (s *Service) InvalidateUserCache(ctx context.Context, userID uuid.UUID) {
	key := stateCachePrefix + userID.String()
	s.redis.Del(ctx, key)
}

// InvalidateBlockCache clears the cached block relationship between two users.
// Call after: block, unblock.
func (s *Service) InvalidateBlockCache(ctx context.Context, uid1, uid2 uuid.UUID) {
	key1 := blockCacheKey(uid1, uid2)
	key2 := blockCacheKey(uid2, uid1)
	s.redis.Del(ctx, key1, key2)
}

// --- Internal: Cached lookups ---

func (s *Service) isBlockedCached(ctx context.Context, viewerID, targetID uuid.UUID) (bool, error) {
	// Check cache: viewer→target direction
	key := blockCacheKey(viewerID, targetID)
	cached, err := s.redis.Get(ctx, key).Result()
	if err == nil {
		return cached == "1", nil
	}

	// Cache miss — query DB (both directions in one check)
	blocked, err := s.isBlockedDB(ctx, viewerID, targetID)
	if err != nil {
		return false, err
	}

	// Cache the result for both directions
	val := "0"
	if blocked {
		val = "1"
	}
	s.redis.Set(ctx, key, val, blockCacheTTL)
	s.redis.Set(ctx, blockCacheKey(targetID, viewerID), val, blockCacheTTL)

	return blocked, nil
}

func (s *Service) getPrivacyStateCached(ctx context.Context, userID uuid.UUID) (*cachedPrivacyState, error) {
	key := stateCachePrefix + userID.String()

	// Try cache
	cached, err := s.redis.Get(ctx, key).Result()
	if err == nil {
		var state cachedPrivacyState
		if json.Unmarshal([]byte(cached), &state) == nil {
			return &state, nil
		}
	}

	// Cache miss — query DB
	dbState, err := s.store.GetUserPrivacyState(ctx, userID)
	if err != nil {
		return nil, err
	}

	state := &cachedPrivacyState{
		IsGhostMode: dbState.IsGhostMode,
		PanicMode:   dbState.PanicMode,
		IsPrivate:   dbState.IsPrivate,
	}
	if dbState.LastActiveAt.Valid {
		t := dbState.LastActiveAt.Time
		state.LastActiveAt = &t
	}

	// Cache for short TTL
	if data, err := json.Marshal(state); err == nil {
		s.redis.Set(ctx, key, data, stateCacheTTL)
	}

	return state, nil
}

// --- Internal: DB-level lookups ---

func (s *Service) isBlockedDB(ctx context.Context, viewerID, targetID uuid.UUID) (bool, error) {
	// Direction 1: target blocked viewer
	blocked1, err := s.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
		BlockerID: targetID,
		BlockedID: viewerID,
	})
	if err != nil {
		return false, err
	}
	if blocked1 {
		return true, nil
	}

	// Direction 2: viewer blocked target
	blocked2, err := s.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
		BlockerID: viewerID,
		BlockedID: targetID,
	})
	if err != nil {
		return false, err
	}
	return blocked2, nil
}

func (s *Service) isFollower(ctx context.Context, viewerID, targetID uuid.UUID) (bool, error) {
	conn, err := s.store.GetConnection(ctx, db.GetConnectionParams{
		RequesterID: viewerID,
		TargetID:    targetID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return conn.Status == "accepted", nil
}

func (s *Service) getVisibilityFromState(state *cachedPrivacyState) VisibilityState {
	if state.LastActiveAt == nil {
		return VisibilityHidden
	}
	return s.getVisibilityFromTime(*state.LastActiveAt)
}

func (s *Service) getVisibilityFromTime(t time.Time) VisibilityState {
	since := time.Since(t)
	switch {
	case since < 24*time.Hour:
		return VisibilityActive
	case since < 3*24*time.Hour:
		return VisibilityFading
	default:
		return VisibilityHidden
	}
}

func blockCacheKey(a, b uuid.UUID) string {
	return fmt.Sprintf("%s%s:%s", blockCachePrefix, a.String(), b.String())
}
