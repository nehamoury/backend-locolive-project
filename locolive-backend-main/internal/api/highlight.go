package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

// createHighlight creates a new Highlight group (e.g. "Travel 2025").
func (server *Server) createHighlight(ctx *gin.Context) {
	var req struct {
		Title    string `json:"title"    binding:"required,min=1,max=50"`
		CoverURL string `json:"cover_url"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	highlight, err := server.store.CreateHighlight(ctx, db.CreateHighlightParams{
		UserID:   authPayload.UserID,
		Title:    req.Title,
		CoverUrl: sql.NullString{String: req.CoverURL, Valid: req.CoverURL != ""},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, highlight)
}

// getHighlights returns all highlight groups for a user.
func (server *Server) getHighlights(ctx *gin.Context) {
	userIDStr := ctx.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		// Fallback: get own highlights
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
		userID = authPayload.UserID
	}

	highlights, err := server.store.ListHighlightsByUserID(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, highlights)
}

// getMyHighlights returns highlights for the authenticated user.
func (server *Server) getMyHighlights(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	highlights, err := server.store.ListHighlightsByUserID(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, highlights)
}

// getHighlightDetails returns all stories in a specific highlight group.
func (server *Server) getHighlightDetails(ctx *gin.Context) {
	highlightID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	stories, err := server.store.GetHighlightDetails(ctx, highlightID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, stories)
}

// addStoryToHighlight links an archived story to a highlight group.
func (server *Server) addStoryToHighlight(ctx *gin.Context) {
	highlightID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req struct {
		ArchivedStoryID string `json:"archived_story_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	archivedStoryID, err := uuid.Parse(req.ArchivedStoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid archived_story_id"})
		return
	}

	_, err = server.store.AddStoryToHighlight(ctx, db.AddStoryToHighlightParams{
		HighlightID:     highlightID,
		ArchivedStoryID: archivedStoryID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "story added to highlight"})
}

// removeStoryFromHighlight removes an archived story from a highlight.
func (server *Server) removeStoryFromHighlight(ctx *gin.Context) {
	highlightID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	archivedStoryID, err := uuid.Parse(ctx.Param("storyId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := server.store.RemoveStoryFromHighlight(ctx, db.RemoveStoryFromHighlightParams{
		HighlightID:     highlightID,
		ArchivedStoryID: archivedStoryID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "story removed from highlight"})
}

// deleteHighlight deletes an entire highlight group.
func (server *Server) deleteHighlight(ctx *gin.Context) {
	highlightID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if err := server.store.DeleteHighlight(ctx, db.DeleteHighlightParams{
		ID:     highlightID,
		UserID: authPayload.UserID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "highlight deleted"})
}
