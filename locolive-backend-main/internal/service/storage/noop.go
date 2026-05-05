package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
)

type NoOpService struct{}

func NewNoOpService() Service {
	return &NoOpService{}
}

// UploadFile returns a placeholder URL without uploading
func (n *NoOpService) UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Generate a pseudo URL without actual upload
	ext := filepath.Ext(filename)
	uniqueFilename := uuid.New().String() + ext

	// Return a local path or placeholder URL
	return fmt.Sprintf("/uploads/%s", uniqueFilename), nil
}
