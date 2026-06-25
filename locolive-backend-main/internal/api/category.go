package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
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

// ─── Public Category Detail APIs ────────────────────────────────────────────

func (server *Server) getCategoryByID(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid category id")))
		return
	}

	cat, err := server.store.GetCategory(ctx, categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("category not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{
		"id":         cat.ID,
		"name":       cat.Name,
		"slug":       cat.Slug,
		"icon":       cat.Icon.String,
		"color":      cat.Color.String,
		"is_active":  cat.IsActive,
		"created_at": cat.CreatedAt,
	}))
}

func (server *Server) getCategoryPosts(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	posts, err := server.store.ListCategoryPosts(ctx, db.ListCategoryPostsParams{
		ViewerID:   authPayload.UserID,
		CategoryID: uuid.NullUUID{UUID: categoryID, Valid: true},
		Off:        int32((page - 1) * pageSize),
		Lim:        int32(pageSize),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(posts))
	for i, p := range posts {
		rsp[i] = gin.H{
			"id":              p.ID,
			"caption":         p.Caption.String,
			"body_text":       p.BodyText.String,
			"media_url":       p.MediaUrl,
			"media_type":      p.MediaType,
			"user_id":         p.UserID,
			"username":        p.Username,
			"avatar_url":      p.AvatarUrl.String,
			"likes_count":     p.LikesCount,
			"comments_count":  p.CommentsCount,
			"shares_count":    p.SharesCount,
			"created_at":      p.CreatedAt,
			"liked_by_viewer": p.LikedByViewer,
			"is_saved":        p.IsSaved,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"posts": rsp, "page": page, "page_size": pageSize}))
}

func (server *Server) getCategoryReels(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "12"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}

	reels, err := server.store.ListCategoryReels(ctx, db.ListCategoryReelsParams{
		Limit:      int32(pageSize),
		Offset:     int32((page - 1) * pageSize),
		ViewerID:   authPayload.UserID,
		CategoryID: uuid.NullUUID{UUID: categoryID, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(reels))
	for i, r := range reels {
		rsp[i] = gin.H{
			"id":           r.ID,
			"video_url":    r.VideoUrl,
			"caption":      r.Caption.String,
			"user_id":      r.UserID,
			"username":     r.Username,
			"avatar_url":   r.AvatarUrl.String,
			"likes_count":  r.LikesCount,
			"created_at":   r.CreatedAt,
			"is_liked":     r.IsLiked,
			"is_saved":     r.IsSaved,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"reels": rsp, "page": page, "page_size": pageSize}))
}

func (server *Server) getCategoryCreators(ctx *gin.Context) {
	categoryID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	creators, err := server.store.ListCategoryCreators(ctx, db.ListCategoryCreatorsParams{
		Limit:      int32(pageSize),
		Offset:     int32((page - 1) * pageSize),
		CategoryID: uuid.NullUUID{UUID: categoryID, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := make([]gin.H, len(creators))
	for i, c := range creators {
		rsp[i] = gin.H{
			"id":            c.ID,
			"username":      c.Username,
			"full_name":     c.FullName,
			"avatar_url":    c.AvatarUrl.String,
			"is_verified":   c.IsVerified,
			"bio":           c.Bio.String,
			"content_count": c.ContentCount,
		}
	}

	ctx.JSON(http.StatusOK, successResponse(gin.H{"creators": rsp, "page": page, "page_size": pageSize}))
}
