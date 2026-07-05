package api

import (
	"encoding/json"
	"time"

	"privacy-social-backend/internal/repository/db"

	"github.com/google/uuid"
)

// StoryResponse is the DTO for story API responses
type StoryResponse struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	MediaURL     string          `json:"media_url"`
	MediaType    string          `json:"media_type"`
	Media         []MediaItem     `json:"media"`
	ThumbnailURL *string         `json:"thumbnail_url"`
	Caption      *string         `json:"caption"`
	Geohash      string          `json:"geohash"`
	Visibility   string          `json:"visibility"`
	ExpiresAt    time.Time       `json:"expires_at"`
	CreatedAt    time.Time       `json:"created_at"`
	IsAnonymous  bool            `json:"is_anonymous"`
	ShowLocation bool            `json:"show_location"`
	IsPremium    *bool           `json:"is_premium"`
	Username     string          `json:"username"`
	AvatarURL    *string         `json:"avatar_url"`
	Lat          float64         `json:"lat"`
	Lng          float64         `json:"lng"`
	CropSettings json.RawMessage `json:"crop_settings,omitempty"`
}

// ArchivedStoryResponse is the DTO for archived story API responses
type ArchivedStoryResponse struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	StoryID           uuid.UUID  `json:"story_id"`
	MediaUrl          string     `json:"media_url"`
	MediaType         string     `json:"media_type"`
	Media         []MediaItem     `json:"media"`
	Caption           *string    `json:"caption"`
	Geohash           string     `json:"geohash"`
	IsAnonymous       bool       `json:"is_anonymous"`
	ShowLocation      bool       `json:"show_location"`
	OriginalCreatedAt time.Time  `json:"original_created_at"`
	ArchivedAt        *time.Time `json:"archived_at"`
	CreatedAt         *time.Time `json:"created_at"`
}

func toArchivedStoryResponse(row db.ArchivedStory) ArchivedStoryResponse {
	resp := ArchivedStoryResponse{
		ID:                row.ID,
		UserID:            row.UserID,
		StoryID:           row.StoryID,
		MediaUrl:          row.MediaUrl,
		MediaType:         row.MediaType,
		Geohash:           row.Geohash,
		IsAnonymous:       row.IsAnonymous.Bool,
		ShowLocation:      row.ShowLocation.Bool,
		OriginalCreatedAt: row.OriginalCreatedAt,
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

// Convert db.GetStoriesWithinRadiusRow to StoryResponse
func toStoryResponse(row db.GetStoriesWithinRadiusRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     row.Username,
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}

// Convert db.GetConnectionStoriesRow to StoryResponse
func toStoryResponseFromConnection(row db.GetConnectionStoriesRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:    row.MediaType,
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     row.Username,
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}

// Convert db.GetStoriesInBoundsRow to StoryResponse
func toStoryResponseFromBounds(row db.GetStoriesInBoundsRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     row.Username,
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}

	return resp
}

// Convert db.CreateStoryRow to StoryResponse
func toStoryResponseFromCreate(row db.CreateStoryRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     "",
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}

// Convert db.GetStoryByIDRow to StoryResponse
func toStoryResponseFromGet(row db.GetStoryByIDRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     "",
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}

// Convert db.UpdateStoryRow to StoryResponse
func toStoryResponseFromUpdate(row db.UpdateStoryRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     "",
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}

func toStoryResponseFromActive(row db.GetActiveStoriesByUserIDRow) StoryResponse {
	resp := StoryResponse{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaURL:     row.MediaUrl,
		MediaType:     row.MediaType,
		Media: []MediaItem{
			{
				URL:          row.MediaUrl,
				Type:         row.MediaType,
				Width:        func() *int32 { if row.Width.Valid { v := row.Width.Int32; return &v }; return nil }(),
				Height:       func() *int32 { if row.Height.Valid { v := row.Height.Int32; return &v }; return nil }(),
				AspectRatio:  func() *float64 { if row.AspectRatio.Valid { v := row.AspectRatio.Float64; return &v }; return nil }(),
				ThumbnailUrl: func() *string { if row.ThumbnailUrl.Valid { v := row.ThumbnailUrl.String; return &v }; return nil }(),
				BlurHash:     func() *string { if row.BlurHash.Valid { v := row.BlurHash.String; return &v }; return nil }(),
				Duration:     func() *int32 { if row.Duration.Valid { v := row.Duration.Int32; return &v }; return nil }(),
				FileSize:     func() *int32 { if row.FileSize.Valid { v := row.FileSize.Int32; return &v }; return nil }(),
				MimeType:     func() *string { if row.MimeType.Valid { v := row.MimeType.String; return &v }; return nil }(),
			},
		},
		Geohash:      row.Geohash,
		Visibility:   string(row.Visibility),
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		IsAnonymous:  row.IsAnonymous,
		ShowLocation: row.ShowLocation,
		Username:     row.Username,
		CropSettings: row.CropSettings.RawMessage,
	}

	if val, ok := row.Lat.(float64); ok {
		resp.Lat = val
	}
	if val, ok := row.Lng.(float64); ok {
		resp.Lng = val
	}

	if row.ThumbnailUrl.Valid {
		resp.ThumbnailURL = &row.ThumbnailUrl.String
	}

	if row.Caption.Valid {
		resp.Caption = &row.Caption.String
	}

	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}

	if row.IsPremium.Valid {
		resp.IsPremium = &row.IsPremium.Bool
	}

	return resp
}
