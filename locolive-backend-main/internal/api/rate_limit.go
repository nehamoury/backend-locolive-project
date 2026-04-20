package api

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// Rate limit configurations
var (
	// General API rate limit: 500 requests per minute (increased from 100)
	generalRate = limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  500,
	}

	// Auth endpoints: 500 requests per 1 minute (increased for development)
	authRate = limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  500,
	}

	// Story creation: 50 per hour
	storyRate = limiter.Rate{
		Period: 1 * time.Hour,
		Limit:  50,
	}

	// Location updates: 600 per hour (10 per minute, increased from 300)
	locationRate = limiter.Rate{
		Period: 1 * time.Hour,
		Limit:  600,
	}

	// Messages: 200 per minute
	messageRate = limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  200,
	}

	// Search: 10 requests per second
	searchRate = limiter.Rate{
		Period: 1 * time.Second,
		Limit:  10,
	}
)

// createRateLimiter creates a rate limiter with Redis store
func (server *Server) createRateLimiter(rate limiter.Rate) gin.HandlerFunc {
	// Bypass rate limiting in tests
	if gin.Mode() == gin.TestMode {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}

	var store limiter.Store
	// Check if Redis is actually reachable
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	if err := server.redis.Ping(ctx).Err(); err != nil {
		// Fallback immediately if Redis fails
		store = memory.NewStore()
	} else {
		var err error
		store, err = sredis.NewStoreWithOptions(server.redis, limiter.StoreOptions{
			Prefix:   "rate_limit",
			MaxRetry: 3,
		})
		if err != nil {
			store = memory.NewStore()
		}
	}

	instance := limiter.New(store, rate)
	middleware := mgin.NewMiddleware(instance)

	return func(ctx *gin.Context) {
		// Bypass for localhost / load tests / development mode
		ip := ctx.ClientIP()
		if gin.Mode() != gin.ReleaseMode || ip == "::1" || ip == "127.0.0.1" || (len(ip) >= 4 && ip[:4] == "172.") {
			ctx.Next()
			return
		}
		middleware(ctx)
	}
}

// generalRateLimiter applies general rate limiting
func (server *Server) generalRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(generalRate)
}

// authRateLimiter applies strict rate limiting for auth endpoints
func (server *Server) authRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(authRate)
}

// storyRateLimiter applies rate limiting for story creation
func (server *Server) storyRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(storyRate)
}

// locationRateLimiter applies rate limiting for location updates
func (server *Server) locationRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(locationRate)
}

// messageRateLimiter applies rate limiting for messaging
func (server *Server) messageRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(messageRate)
}

// searchRateLimiter applies rate limiting for user search
func (server *Server) searchRateLimiter() gin.HandlerFunc {
	return server.createRateLimiter(searchRate)
}
