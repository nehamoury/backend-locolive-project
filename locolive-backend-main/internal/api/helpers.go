package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/token"
)

// getAuthPayload extracts the authenticated user's payload from the context
func getAuthPayload(ctx *gin.Context) *token.Payload {
	return ctx.MustGet(authorizationPayloadKey).(*token.Payload)
}



// successResponse standardizes successful API responses
func successResponse(data interface{}) gin.H {
	return gin.H{
		"success": true,
		"data":    data,
	}
}

// parseUUIDParam parses a UUID string and returns an error response if invalid
func parseUUIDParam(ctx *gin.Context, value string, paramName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("Invalid %s", paramName)))
		return uuid.Nil, false
	}
	return id, true
}

// nullStringToStrPtr converts a sql.NullString to a *string
func nullStringToStrPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// toNullString converts a string to a sql.NullString
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
