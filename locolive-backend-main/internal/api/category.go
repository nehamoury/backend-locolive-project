package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
)

// ─── Request / Response DTOs ──────────────────────────────────────────────────

type createCategoryRequest struct {
	Name     string `json:"name" binding:"required"`
	Slug     string `json:"slug" binding:"required"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
	IsActive bool   `json:"is_active"`
}

type updateCategoryRequest struct {
	Name     *string `json:"name"`
	Slug     *string `json:"slug"`
	Icon     *string `json:"icon"`
	Color    *string `json:"color"`
	IsActive *bool   `json:"is_active"`
}

// listCategories returns all active categories
func (server *Server) listCategories(ctx *gin.Context) {
	categories, err := server.store.ListCategories(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, successResponse(categories))
}

// listTrendingCategories returns top categories by stats
func (server *Server) listTrendingCategories(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	categories, err := server.store.ListTrendingCategories(ctx, db.ListTrendingCategoriesParams{
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, successResponse(categories))
}

// ─── Admin APIs ──────────────────────────────────────────────────────────────

func (server *Server) adminListCategories(ctx *gin.Context) {
	// In a real scenario, this might return inactive ones too if a query exists
	categories, err := server.store.ListCategories(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, successResponse(categories))
}

func (server *Server) adminCreateCategory(ctx *gin.Context) {
	var req createCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	cat, err := server.store.CreateCategory(ctx, db.CreateCategoryParams{
		Name:     req.Name,
		Slug:     req.Slug,
		Icon:     sql.NullString{String: req.Icon, Valid: req.Icon != ""},
		Color:    sql.NullString{String: req.Color, Valid: req.Color != ""},
		IsActive: req.IsActive,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusCreated, successResponse(cat))
}

func (server *Server) adminUpdateCategory(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.UpdateCategoryParams{
		ID: categoryID,
	}
	if req.Name != nil {
		arg.Name = sql.NullString{String: *req.Name, Valid: true}
	}
	if req.Slug != nil {
		arg.Slug = sql.NullString{String: *req.Slug, Valid: true}
	}
	if req.Icon != nil {
		arg.Icon = sql.NullString{String: *req.Icon, Valid: true}
	}
	if req.Color != nil {
		arg.Color = sql.NullString{String: *req.Color, Valid: true}
	}
	if req.IsActive != nil {
		arg.IsActive = sql.NullBool{Bool: *req.IsActive, Valid: true}
	}

	cat, err := server.store.UpdateCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, successResponse(cat))
}

func (server *Server) adminDeleteCategory(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.store.DeleteCategory(ctx, categoryID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "category deleted"})
}
