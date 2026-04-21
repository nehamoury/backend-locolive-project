package api

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// AppError represents a custom application error
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Detail  string `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Predefined application errors
var (
	ErrUserNotFound     = &AppError{Code: http.StatusNotFound, Message: "user not found"}
	ErrInvalidPassword  = &AppError{Code: http.StatusUnauthorized, Message: "invalid credentials"}
	ErrUnauthorized     = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden        = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrInvalidInput     = &AppError{Code: http.StatusBadRequest, Message: "invalid input"}
	ErrDuplicateEntry   = &AppError{Code: http.StatusConflict, Message: "resource already exists"}
	ErrInternalServer   = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrServiceUnavailable = &AppError{Code: http.StatusServiceUnavailable, Message: "service temporarily unavailable"}
)

// isProduction returns true if running in production mode
func isProduction() bool {
	return gin.Mode() == gin.ReleaseMode || os.Getenv("ENVIRONMENT") == "production"
}

// errorResponse creates a safe error response for clients
// Logs full error details internally, returns generic message in production
func errorResponse(err error) gin.H {
	if err == nil {
		return gin.H{"success": false, "error": "unknown error"}
	}

	// Log full error details with stack trace context
	log.Error().
		Str("error", err.Error()).
		Str("type", "api_error").
		Msg("API error occurred")

	// Check if it's a known application error
	var appErr *AppError
	if errors.As(err, &appErr) {
		return gin.H{
			"success": false,
			"error":   appErr.Message,
		}
	}

	// Handle specific error types
	// Database errors - don't leak schema details
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return handleDBError(pqErr)
	}

	// Validation errors - return generic message
	if strings.Contains(err.Error(), "validation") ||
		strings.Contains(err.Error(), "required") ||
		strings.Contains(err.Error(), "binding") {
		return gin.H{
			"success": false,
			"error":   "invalid input data",
		}
	}

	// Authentication errors
	if strings.Contains(strings.ToLower(err.Error()), "password") ||
		strings.Contains(strings.ToLower(err.Error()), "unauthorized") ||
		strings.Contains(strings.ToLower(err.Error()), "token") {
		return gin.H{
			"success": false,
			"error":   "authentication failed",
		}
	}

	// File operation errors
	if strings.Contains(strings.ToLower(err.Error()), "file") ||
		strings.Contains(strings.ToLower(err.Error()), "upload") {
		return gin.H{
			"success": false,
			"error":   "file operation failed",
		}
	}

	// Production: generic error message
	// Development: detailed error (if needed for debugging)
	if isProduction() {
		return gin.H{
			"success": false,
			"error":   "an unexpected error occurred",
		}
	}

	// Development mode: return sanitized error
	return gin.H{
		"success": false,
		"error":   sanitizeErrorMessage(err.Error()),
	}
}

// handleDBError maps database errors to safe messages
func handleDBError(pqErr *pq.Error) gin.H {
	switch pqErr.Code.Name() {
	case "unique_violation":
		return gin.H{
			"success": false,
			"error":   "resource already exists",
		}
	case "foreign_key_violation":
		return gin.H{
			"success": false,
			"error":   "referenced resource not found",
		}
	case "check_violation":
		return gin.H{
			"success": false,
			"error":   "invalid data provided",
		}
	case "not_null_violation":
		return gin.H{
			"success": false,
			"error":   "required field missing",
		}
	case "too_many_connections":
		return gin.H{
			"success": false,
			"error":   "service temporarily unavailable",
		}
	default:
		log.Warn().
			Str("pg_code", string(pqErr.Code)).
			Str("pg_message", pqErr.Message).
			Msg("Unhandled database error code")
		return gin.H{
			"success": false,
			"error":   "database error",
		}
	}
}

// sanitizeErrorMessage removes sensitive info from error messages in dev mode
func sanitizeErrorMessage(msg string) string {
	// Remove file paths
	if idx := strings.Index(msg, ":"); idx > 0 && strings.Contains(msg[:idx], "/") {
		// Likely a file path, keep only the main message
		parts := strings.Split(msg, ": ")
		if len(parts) > 1 {
			msg = parts[len(parts)-1]
		}
	}

	// Remove SQL details
	if strings.Contains(strings.ToLower(msg), "sql") ||
		strings.Contains(strings.ToLower(msg), "select") ||
		strings.Contains(strings.ToLower(msg), "insert") ||
		strings.Contains(strings.ToLower(msg), "update") ||
		strings.Contains(strings.ToLower(msg), "delete") {
		return "database query failed"
	}

	// Remove connection strings
	if strings.Contains(msg, "postgresql://") ||
		strings.Contains(msg, "redis://") {
		return "service connection failed"
	}

	return msg
}

