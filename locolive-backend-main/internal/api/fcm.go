package api

import (
	"net/http"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"

	"github.com/gin-gonic/gin"
)

type registerFCMTokenRequest struct {
	Token      string `json:"token" binding:"required"`
	DeviceType string `json:"device_type"`
}

func (server *Server) registerFCMToken(ctx *gin.Context) {
	var req registerFCMTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	deviceType := req.DeviceType
	if deviceType == "" {
		deviceType = "web"
	}

	arg := db.RegisterFCMTokenParams{
		UserID:     authPayload.UserID,
		Token:      req.Token,
		DeviceType: deviceType,
	}

	_, err := server.store.RegisterFCMToken(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (server *Server) removeFCMToken(ctx *gin.Context) {
	var req registerFCMTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	arg := db.RemoveFCMTokenParams{
		UserID: authPayload.UserID,
		Token:  req.Token,
	}

	err := server.store.RemoveFCMToken(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}
