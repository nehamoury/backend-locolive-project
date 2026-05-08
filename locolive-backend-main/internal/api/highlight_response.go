package api

import (
	"time"

	"privacy-social-backend/internal/repository/db"

	"github.com/google/uuid"
)

type HighlightStoryResponse struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	StoryID           uuid.UUID  `json:"story_id"`
	MediaUrl          string     `json:"media_url"`
	MediaType         string     `json:"media_type"`
	Caption           *string    `json:"caption"`
	Geohash           string     `json:"geohash"`
	IsAnonymous       bool       `json:"is_anonymous"`
	ShowLocation      bool       `json:"show_location"`
	OriginalCreatedAt time.Time  `json:"original_created_at"`
	ArchivedAt        *time.Time `json:"archived_at"`
	CreatedAt         *time.Time `json:"created_at"`
	AddedAt           time.Time  `json:"added_at"`
}

type HighlightGroupResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Title      string    `json:"title"`
	CoverUrl   *string   `json:"cover_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	StoryCount int64     `json:"story_count"`
}

func toHighlightStoryResponse(row db.GetHighlightDetailsRow) HighlightStoryResponse {
	resp := HighlightStoryResponse{
		ID:                row.ID,
		UserID:            row.UserID,
		StoryID:           row.StoryID,
		MediaUrl:          row.MediaUrl,
		MediaType:         row.MediaType,
		Geohash:           row.Geohash,
		IsAnonymous:       row.IsAnonymous.Bool,
		ShowLocation:      row.ShowLocation.Bool,
		OriginalCreatedAt: row.OriginalCreatedAt,
		AddedAt:           row.AddedAt,
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.ArchivedAt.Valid {
		resp.ArchivedAt = &row.ArchivedAt.Time
	}

	if row.CreatedAt.Valid {
		resp.CreatedAt = &row.CreatedAt.Time
	}

	return resp
}

func toHighlightGroupResponse(row db.HighlightGroup) HighlightGroupResponse {
	resp := HighlightGroupResponse{
		ID:        row.ID,
		UserID:    row.UserID,
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.CoverUrl.Valid {
		resp.CoverUrl = &row.CoverUrl.String
	}

	return resp
}

func toHighlightGroupResponseFromList(row db.ListHighlightsByUserIDRow) HighlightGroupResponse {
	resp := HighlightGroupResponse{
		ID:         row.ID,
		UserID:     row.UserID,
		Title:      row.Title,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		StoryCount: row.StoryCount,
	}

	if row.CoverUrl.Valid {
		resp.CoverUrl = &row.CoverUrl.String
	}

	return resp
}
