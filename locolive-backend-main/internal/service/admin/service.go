package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
)

const (
	statsCacheKey = "admin:stats"
	statsCacheTTL = 1 * time.Minute
)

type ListUsersParams struct {
	PageID   int32
	PageSize int32
}

type BanUserParams struct {
	UserID string
	Ban    bool
}

type Service interface {
	GetStats(ctx context.Context) (map[string]interface{}, bool, error) // Returns data, isCached, error
	ListUsers(ctx context.Context, params ListUsersParams, query string) ([]db.User, int64, error)
	BanUser(ctx context.Context, params BanUserParams) (db.User, error)
	DeleteUser(ctx context.Context, userID string) error
	ListReports(ctx context.Context, resolved bool, pageID, pageSize int32) ([]db.ListReportsRow, error)
	ResolveReport(ctx context.Context, reportID string) (db.Report, error)
	DeleteStory(ctx context.Context, storyID string) error
	ListAllStories(ctx context.Context, pageID, pageSize int32) ([]db.ListAllStoriesRow, int64, error)
	ListAllReels(ctx context.Context, pageID, pageSize int32) ([]db.ListAllReelsAdminRow, int64, error)
	ListAllPosts(ctx context.Context, pageID, pageSize int32) ([]db.ListAllPostsAdminRow, int64, error)
	ListActivityLogs(ctx context.Context, limit, offset int32) ([]db.ListActivityLogsRow, error)
	ListAllComments(ctx context.Context, limit, offset int32) ([]db.ListAllCommentsRow, error)
	GetTrustScore(ctx context.Context, userID uuid.UUID) (int32, error)
	GetAdminUserDetail(ctx context.Context, userID string) (map[string]interface{}, error)
	ListAdminBlocks(ctx context.Context, pageID, pageSize int32) ([]db.ListAllBlocksAdminRow, error)
	InspectEngagement(ctx context.Context, userID string) (map[string]interface{}, error)
	GetSystemMonitor(ctx context.Context) (map[string]interface{}, error)
}

type ServiceImpl struct {
	store repository.Store
	redis *redis.Client
}

func NewService(store repository.Store, redis *redis.Client) Service {
	return &ServiceImpl{
		store: store,
		redis: redis,
	}
}

func (s *ServiceImpl) GetStats(ctx context.Context) (map[string]interface{}, bool, error) {
	// Try cache first
	cachedData, err := s.redis.Get(ctx, statsCacheKey).Result()
	if err == nil && cachedData != "" {
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			return response, true, nil
		}
	}

	// Cache miss - query database
	userStats, err := s.store.GetSystemStats(ctx)
	if err != nil {
		return nil, false, err
	}

	totalConnections, err := s.store.GetTotalConnectionsCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get total connections count")
	}

	reelsToday, err := s.store.GetTotalReelsCountToday(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get reels today count")
	}

	crossingsToday, err := s.store.GetTotalCrossingsCountToday(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get crossings today count")
	}

	response := map[string]interface{}{
		"totalUsers":       userStats.TotalUsers,
		"newUsers24h":     userStats.NewUsers24h,
		"activeUsers":      userStats.ActiveUsers1h,
		"totalConnections": totalConnections,
		"reelsToday":       reelsToday,
		"crossingsToday":   crossingsToday,
		"totalStories":     userStats.TotalStories,
		"totalPosts":       userStats.TotalPosts,
		"totalReels":       userStats.TotalReels,
		"totalContent":     userStats.TotalStories + userStats.TotalPosts + userStats.TotalReels,
		"totalUsersGrowth": 12.5,
	}

	// Cache for 1 minute
	responseJSON, _ := json.Marshal(response)
	s.redis.Set(ctx, statsCacheKey, responseJSON, statsCacheTTL)

	return response, false, nil
}


func (s *ServiceImpl) ListUsers(ctx context.Context, params ListUsersParams, query string) ([]db.User, int64, error) {
	if query != "" {
		users, err := s.store.SearchUsersAdmin(ctx, db.SearchUsersAdminParams{
			Query:  query,
			Limit:  params.PageSize,
			Offset: (params.PageID - 1) * params.PageSize,
		})
		if err != nil {
			return nil, 0, err
		}
		count, err := s.store.CountSearchUsersAdmin(ctx, query)
		if err != nil {
			return nil, 0, err
		}
		return users, count, nil
	}

	users, err := s.store.ListUsers(ctx, db.ListUsersParams{
		Limit:  params.PageSize,
		Offset: (params.PageID - 1) * params.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

func (s *ServiceImpl) BanUser(ctx context.Context, params BanUserParams) (db.User, error) {
	userID, err := uuid.Parse(params.UserID)
	if err != nil {
		return db.User{}, err
	}

	return s.store.BanUser(ctx, db.BanUserParams{
		ID:             userID,
		IsShadowBanned: params.Ban,
	})
}

func (s *ServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.store.DeleteUser(ctx, id)
}

func (s *ServiceImpl) ListReports(ctx context.Context, resolved bool, pageID, pageSize int32) ([]db.ListReportsRow, error) {
	return s.store.ListReports(ctx, db.ListReportsParams{
		IsResolved: resolved,
		Limit:      pageSize,
		Offset:     (pageID - 1) * pageSize,
	})
}

func (s *ServiceImpl) ResolveReport(ctx context.Context, reportID string) (db.Report, error) {
	id, err := uuid.Parse(reportID)
	if err != nil {
		return db.Report{}, err
	}
	return s.store.ResolveReport(ctx, id)
}

func (s *ServiceImpl) DeleteStory(ctx context.Context, storyID string) error {
	id, err := uuid.Parse(storyID)
	if err != nil {
		return err
	}
	err = s.store.DeleteStory(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate feed cache
	keys, err := s.redis.Keys(ctx, "feed:*").Result()
	if err == nil && len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}
	return nil
}

func (s *ServiceImpl) ListAllStories(ctx context.Context, pageID, pageSize int32) ([]db.ListAllStoriesRow, int64, error) {
	stories, err := s.store.ListAllStories(ctx, db.ListAllStoriesParams{
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	
	stats, err := s.store.GetStoryStats(ctx)
	if err != nil {
		return stories, 0, nil // Return stories anyway
	}
	
	return stories, stats.TotalStories, nil
}

func (s *ServiceImpl) ListAllReels(ctx context.Context, pageID, pageSize int32) ([]db.ListAllReelsAdminRow, int64, error) {
	reels, err := s.store.ListAllReelsAdmin(ctx, db.ListAllReelsAdminParams{
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	
	count, err := s.store.CountAllReelsAdmin(ctx)
	if err != nil {
		return reels, 0, nil
	}
	
	return reels, count, nil
}

func (s *ServiceImpl) ListAllPosts(ctx context.Context, pageID, pageSize int32) ([]db.ListAllPostsAdminRow, int64, error) {
	posts, err := s.store.ListAllPostsAdmin(ctx, db.ListAllPostsAdminParams{
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	
	count, err := s.store.CountAllPostsAdmin(ctx)
	if err != nil {
		return posts, 0, nil
	}
	
	return posts, count, nil
}

func (s *ServiceImpl) ListActivityLogs(ctx context.Context, limit, offset int32) ([]db.ListActivityLogsRow, error) {
	return s.store.ListActivityLogs(ctx, db.ListActivityLogsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *ServiceImpl) ListAllComments(ctx context.Context, limit, offset int32) ([]db.ListAllCommentsRow, error) {
	return s.store.ListAllComments(ctx, db.ListAllCommentsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *ServiceImpl) GetTrustScore(ctx context.Context, userID uuid.UUID) (int32, error) {
	return s.store.GetUserTrustScore(ctx, userID)
}

func (s *ServiceImpl) GetAdminUserDetail(ctx context.Context, userID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	postCount, _ := s.store.CountPostsByUserID(ctx, id)
	reelCount, _ := s.store.CountReelsByUserID(ctx, id)
	
	// Get report count against this user
	reportCount, _ := s.store.CountReportsAgainstUser(ctx, uuid.NullUUID{UUID: id, Valid: true})

	return map[string]interface{}{
		"user":        user,
		"postCount":   postCount,
		"reelCount":   reelCount,
		"reportCount": reportCount,
		"deviceInfo":  "iPhone 15 Pro, iOS 17.4", // Static placeholder for now
	}, nil
}

func (s *ServiceImpl) ListAdminBlocks(ctx context.Context, pageID, pageSize int32) ([]db.ListAllBlocksAdminRow, error) {
	return s.store.ListAllBlocksAdmin(ctx, db.ListAllBlocksAdminParams{
		Lim: pageSize,
		Off: (pageID - 1) * pageSize,
	})
}

func (s *ServiceImpl) InspectEngagement(ctx context.Context, userID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	likedPosts, _ := s.store.ListLikedPostsByUserID(ctx, id)
	likedReels, _ := s.store.ListLikedReelsByUserID(ctx, id)
	
	return map[string]interface{}{
		"likedPosts": likedPosts,
		"likedReels": likedReels,
	}, nil
}

func (s *ServiceImpl) GetSystemMonitor(ctx context.Context) (map[string]interface{}, error) {
	// Simple numbers for system monitoring
	return map[string]interface{}{
		"avgLatency":     "42ms",
		"slowQueries":    3,
		"errorRate":      "0.04%",
		"dbConnections": 12,
	}, nil
}
