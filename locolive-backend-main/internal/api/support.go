package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"privacy-social-backend/internal/repository/db"
)

type createSupportTicketRequest struct {
	Subject     string `json:"subject" binding:"required,min=5,max=255"`
	Description string `json:"description" binding:"required,min=20"`
	Priority    string `json:"priority" binding:"required,oneof=low medium high urgent"`
}

func (server *Server) createSupportTicket(ctx *gin.Context) {
	var req createSupportTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := getAuthPayload(ctx)

	ticket, err := server.store.CreateSupportTicket(ctx, db.CreateSupportTicketParams{
		UserID:      authPayload.UserID,
		Subject:     req.Subject,
		Description: req.Description,
		Priority:    req.Priority,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, successResponse(ticket))
}

func (server *Server) listMySupportTickets(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	tickets, err := server.store.GetUserSupportTickets(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(tickets))
}
