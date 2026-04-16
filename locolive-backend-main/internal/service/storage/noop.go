package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"
)

type NoOpService struct{}

func NewNoOpService() Service {
	return &NoOpService{}
}

// UploadFile returns a placeholder URL without uploading
func (n *NoOpService) UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	if file == nil || fileHeader == nil {
		return "", fmt.Errorf("file or fileHeader is nil")
	}

	// Generate a pseudo URL without actual upload
	ext := filepath.Ext(fileHeader.Filename)
	filename := uuid.New().String() + ext

	// Return a local path or placeholder URL
	return fmt.Sprintf("/uploads/%s", filename), nil
}
