package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
)

type relationshipRequest struct {
	TargetUserID string `uri:"id" binding:"required,uuid"`
}

// followUser allows a user to follow another user
func (server *Server) followUser(ctx *gin.Context) {
	var req relationshipRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	targetUserID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	if authPayload.UserID == targetUserID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot follow yourself"})
		return
	}

	// Create relationship
	_, err = server.store.CreateRelationship(ctx, db.CreateRelationshipParams{
		UserID:       authPayload.UserID,
		TargetUserID: targetUserID,
		Type:         db.RelationshipTypeFollow,
		Status:       db.RelationshipStatusActive,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate profile cache for both users
	server.redis.Del(ctx, "profile:"+authPayload.UserID.String())
	server.redis.Del(ctx, "profile:"+targetUserID.String())

	// Trigger follow notification
	// We'll wire this up properly later, for now just success
	ctx.JSON(http.StatusOK, successResponse("followed successfully"))
}

// unfollowUser allows a user to unfollow another user
func (server *Server) unfollowUser(ctx *gin.Context) {
	var req relationshipRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	targetUserID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err = server.store.DeleteRelationship(ctx, db.DeleteRelationshipParams{
		UserID:       authPayload.UserID,
		TargetUserID: targetUserID,
		Type:         db.RelationshipTypeFollow,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "relationship not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate profile cache for both users
	server.redis.Del(ctx, "profile:"+authPayload.UserID.String())
	server.redis.Del(ctx, "profile:"+targetUserID.String())

	ctx.JSON(http.StatusOK, successResponse("unfollowed successfully"))
}
