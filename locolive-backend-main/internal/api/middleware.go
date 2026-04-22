package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"privacy-social-backend/internal/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// authMiddleware creates a gin middleware for authorization
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
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
		// This prevents tokens from leaking into logs via normal GET requests
		if accessToken == "" && strings.HasPrefix(ctx.Request.URL.Path, "/api/ws") {
			accessToken = ctx.Query("token")
		}

		if accessToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Verify Token and handle errors safely (no raw leaks)
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			msg := "invalid or expired token"
			if errors.Is(err, token.ErrExpiredToken) {
				msg = "token has expired"
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
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
				// In development, allow localhost for flexibility
				if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") || origin == frontendURL {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

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

		// Polished Content-Security-Policy (CSP)
		// Allows external assets like Mapbox, Google Fonts, and local dev assets.
		csp := strings.Join([]string{
			"default-src 'self'",
			"img-src 'self' data: https: blob: http://localhost:8081", // Allow local uploads
			"script-src 'self' 'unsafe-inline' https://api.mapbox.com",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://api.mapbox.com",
			"font-src 'self' https://fonts.gstatic.com",
			"connect-src 'self' https://api.mapbox.com https://*.tiles.mapbox.com ws://localhost:8081 http://localhost:8081",
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
