package api

import (
	"database/sql"
	"net/http"
	"time"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type groupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func mapGroupResponse(g db.Group) groupResponse {
	return groupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description.String,
		CreatedBy:   g.CreatedBy,
		CreatedAt:   g.CreatedAt,
	}
}

type createGroupRequest struct {
	Name        string      `json:"name" binding:"required"`
	Description string      `json:"description"`
	MemberIDs   []uuid.UUID `json:"member_ids"` // Initial members
}

func (server *Server) createGroup(ctx *gin.Context) {
	var req createGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Create Group
	group, err := server.store.CreateGroup(ctx, db.CreateGroupParams{
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		CreatedBy:   authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Add Creator as Admin
	_, err = server.store.AddGroupMember(ctx, db.AddGroupMemberParams{
		GroupID: group.ID,
		UserID:  authPayload.UserID,
		Role:    "admin",
	})
	if err != nil {
		// Rollback desirable but skipping for simple impl
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Add other members
	for _, memberID := range req.MemberIDs {
		// Verify connection? (Optional but good practice)
		// ...
		server.store.AddGroupMember(ctx, db.AddGroupMemberParams{
			GroupID: group.ID,
			UserID:  memberID,
			Role:    "member",
		})
	}

	ctx.JSON(http.StatusCreated, mapGroupResponse(group))
}

func (server *Server) getMyGroups(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	groups, err := server.store.GetUserGroups(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]groupResponse, len(groups))
	for i, g := range groups {
		rsp[i] = mapGroupResponse(g)
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) getGroupByID(ctx *gin.Context) {
	groupID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Verify membership
	isMember, err := server.store.CheckGroupMembership(ctx, db.CheckGroupMembershipParams{
		GroupID: groupID,
		UserID:  authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if !isMember {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied: you are not a member of this group"})
		return
	}

	group, err := server.store.GetGroupByID(ctx, groupID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": mapGroupResponse(group)})
}

func (server *Server) getGroupMessages(ctx *gin.Context) {
	groupID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Verify membership
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	isMember, err := server.store.CheckGroupMembership(ctx, db.CheckGroupMembershipParams{
		GroupID: groupID,
		UserID:  authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if !isMember {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied: you are not a member of this group"})
		return
	}

	msgs, err := server.store.GetGroupMessages(ctx, uuid.NullUUID{UUID: groupID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, msgs)
}

func (server *Server) getGroupMembers(ctx *gin.Context) {
	groupID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Verify membership
	isMember, err := server.store.CheckGroupMembership(ctx, db.CheckGroupMembershipParams{
		GroupID: groupID,
		UserID:  authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if !isMember {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied: you are not a member of this group"})
		return
	}

	members, err := server.store.GetGroupMembers(ctx, groupID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	type memberResponse struct {
		UserID    uuid.UUID `json:"user_id"`
		Role      string    `json:"role"`
		JoinedAt  time.Time `json:"joined_at"`
		Username  string    `json:"username"`
		AvatarUrl string    `json:"avatar_url"`
	}

	rsp := make([]memberResponse, len(members))
	for i, m := range members {
		rsp[i] = memberResponse{
			UserID:    m.UserID,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
			Username:  m.Username,
			AvatarUrl: m.AvatarUrl.String,
		}
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) leaveGroup(ctx *gin.Context) {
	groupID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Verify membership
	isMember, err := server.store.CheckGroupMembership(ctx, db.CheckGroupMembershipParams{
		GroupID: groupID,
		UserID:  authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if !isMember {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "you are not a member of this group"})
		return
	}

	// Remove member
	err = server.store.RemoveGroupMember(ctx, db.RemoveGroupMemberParams{
		GroupID: groupID,
		UserID:  authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "successfully left the group"})
}
