package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// authMiddleware creates a gin middleware for authorization
func (server *Server) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var accessToken string

		// 1. Try Authorization Header (Preferred)
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
		if len(authorizationHeader) > 0 {
			fields := strings.Fields(authorizationHeader)
			if len(fields) >= 2 && strings.ToLower(fields[0]) == authorizationTypeBearer {
				accessToken = fields[1]
			}
		}

		// 2. Try httpOnly Cookie (Browser Security)
		if accessToken == "" {
			cookie, err := ctx.Cookie("access_token")
			if err == nil {
				accessToken = cookie
			}
		}

		// 3. Try Query Param - STRICTLY LIMITED to WebSockets (/ws/*)
		if accessToken == "" && strings.HasPrefix(ctx.Request.URL.Path, "/api/ws") {
			accessToken = ctx.Query("token")
		}

		if accessToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Verify Token and handle errors safely
		payload, err := server.tokenMaker.VerifyToken(accessToken)
		if err != nil {
			msg := "invalid or expired token"
			if errors.Is(err, token.ErrExpiredToken) {
				msg = "token has expired"
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		// CHECK REDIS BLACKLIST (Individual Token)
		blacklisted, err := server.redis.Get(ctx, fmt.Sprintf("blacklist:%s", payload.ID.String())).Result()
		if err == nil && blacklisted != "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session has been revoked"})
			return
		}

		// CHECK GLOBAL REVOCATION (Logout All Devices)
		revokeTimeStr, err := server.redis.Get(ctx, fmt.Sprintf("revoke_all:%s", payload.UserID.String())).Result()
		if err == nil && revokeTimeStr != "" {
			var revokeTime int64
			fmt.Sscanf(revokeTimeStr, "%d", &revokeTime)
			if payload.IssuedAt.Unix() < revokeTime {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired due to security update"})
				return
			}
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

// adminMiddleware verifies that the user has admin role
// OPTIMIZED: Uses the Role embedded in JWT payload to avoid DB queries on every request.
func adminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, exists := ctx.Get(authorizationPayloadKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		authPayload := payload.(*token.Payload)

		// Check role directly from JWT payload
		if authPayload.Role != "admin" && authPayload.Role != "moderator" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: administrative privileges required"})
			return
		}

		ctx.Next()
	}
}

// corsMiddleware handles the CORS middleware with production-safe logic
func corsMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		isRelease := gin.Mode() == gin.ReleaseMode

		if origin != "" {
			// In production, strictly match the configured frontend URL. DO NOT allow wildcard.
			if isRelease {
				if origin == frontendURL {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					// Disallowed origin
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CORS policy: origin not allowed"})
					return
				}
			} else {
				// In development, allow localhost and any other origin for flexibility
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// securityHeadersMiddleware adds essential security headers and polished CSP
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self)")
		c.Writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")

		// Polished Content-Security-Policy (CSP)
		// Allows external assets like Mapbox, Google Fonts, and local dev assets.
		csp := strings.Join([]string{
			"default-src 'self'",
			"img-src 'self' data: https: blob: *", // Allow all image sources (or narrow down to your media domain)
			"script-src 'self' 'unsafe-inline' https://api.mapbox.com",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://api.mapbox.com",
			"font-src 'self' https://fonts.gstatic.com",
			"connect-src 'self' https://api.mapbox.com https://*.tiles.mapbox.com *", // Allow all connections (or narrow down to your API domain)
			"worker-src 'self' blob:",
		}, "; ")
		c.Writer.Header().Set("Content-Security-Policy", csp)

		// HSTS (Only enabled in production when using HTTPS)
		if gin.Mode() == gin.ReleaseMode && c.Request.TLS != nil {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// privacyCheckMiddleware enforces privacy rules before accessing a target user's resource.
// It assumes the target user ID is in the "id" route parameter.
func (server *Server) privacyCheckMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, exists := ctx.Get(authorizationPayloadKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		authPayload := payload.(*token.Payload)

		targetIDStr := ctx.Param("id")
		if targetIDStr == "" {
			ctx.Next()
			return
		}

		targetID, err := uuid.Parse(targetIDStr)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		// Use the centralized privacy service (lenient check for profiles)
		result := server.privacy.CanViewProfile(ctx, authPayload.UserID, targetID)
		if !result.Allowed {
			server.respondToPrivacyDenial(ctx, result.Reason)
			return
		}

		ctx.Next()
	}
}

// respondToPrivacyDenial maps privacy reasons to standard HTTP status codes.
func (server *Server) respondToPrivacyDenial(ctx *gin.Context, reason interface{}) {
	switch fmt.Sprintf("%v", reason) {
	case "blocked", "panic_mode", "deleted", "hidden":
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
	case "private":
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "this account is private"})
	case "banned":
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: user is banned"})
	default:
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied due to privacy settings"})
	}
}

// rateLimitMiddleware provides Redis-based rate limiting
func (server *Server) rateLimitMiddleware(limit int, duration time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", ctx.FullPath(), ctx.ClientIP())
		
		count, err := server.redis.Incr(ctx, key).Result()
		if err != nil {
			ctx.Next()
			return
		}

		if count == 1 {
			server.redis.Expire(ctx, key, duration)
		}

		if count > int64(limit) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		ctx.Next()
	}
}
