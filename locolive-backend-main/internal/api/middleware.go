package api

import (
	"database/sql"
	"errors"
	"fmt"
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
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		// Check for query parameter (for WebSockets)
		if len(authorizationHeader) == 0 {
			tokenParam := ctx.Query("token")
			if len(tokenParam) > 0 {
				fmt.Printf("[DEBUG] authMiddleware: found token in query param\n")
				authorizationHeader = "Bearer " + tokenParam
			}
		}

		if len(authorizationHeader) == 0 {
			fmt.Printf("[DEBUG] authMiddleware: no authorization header or token param\n")
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

var ErrNotAdmin = errors.New("user is not an admin")

// adminMiddleware verifies that the user has admin role
func adminMiddleware(server *Server) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

		// Get user from database to check role
		user, err := server.store.GetUserByID(ctx, authPayload.UserID)
		if err != nil {
			if err == sql.ErrNoRows {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrNotAdmin))
				return
			}
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		// Check if user is admin or moderator
		if user.Role != "admin" && user.Role != "moderator" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(ErrNotAdmin))
			return
		}

		ctx.Next()
	}
}

// corsMiddleware handles the CORS middleware
func corsMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allowed origins for CORS
		allowedOrigins := map[string]bool{
			"http://localhost:5173": true, // Vite dev server
			"http://localhost:3000": true, // Alternative dev port
			"http://localhost:8080": true, // Backend serving frontend
			"http://127.0.0.1:5173": true, // Localhost alternative
			"http://127.0.0.1:3000": true, // Localhost alternative
			"http://127.0.0.1:8080": true, // Localhost alternative
		}

		// Add production frontend URL if configured
		if frontendURL != "" {
			allowedOrigins[frontendURL] = true
		}

		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// For requests without origin (same-origin requests or non-browser clients)
			// Use the frontend URL if configured, otherwise omit header
			if frontendURL != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", frontendURL)
			}
			// If no origin and no frontendURL configured, don't set header
			// Browser will handle same-origin requests normally
		} else {
			// For disallowed origins: don't set Access-Control-Allow-Origin
			// This will cause browser to block the request (correct security behavior)
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

// securityHeadersMiddleware adds essential security headers to all responses
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		// Referrer policy
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// CSP (IMPORTANT) - Prevents XSS by controlling resource loading
		c.Writer.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'")

		// XSS protection (legacy browsers)
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// Permissions policy (disable unused browser features)
		c.Writer.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(self)")

		// HSTS (only HTTPS in production)
		if gin.Mode() == gin.ReleaseMode {
			c.Writer.Header().Set("Strict-Transport-Security",
				"max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}
