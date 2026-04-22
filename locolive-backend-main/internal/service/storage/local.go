package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type LocalStorageService struct {
	uploadDir string
	baseURL   string
}

func NewLocalStorageService(uploadDir string, baseURL string) (Service, error) {
	// Ensure directory exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(uploadDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("could not create upload directory: %w", err)
		}
	}
	return &LocalStorageService{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}, nil
}

func (s *LocalStorageService) UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	if file == nil || fileHeader == nil {
		return "", fmt.Errorf("file or fileHeader is nil")
	}

	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := uuid.New().String() + ext
	dst := filepath.Join(s.uploadDir, filename)

	// Create destination file
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	// Copy content
	_, err = io.Copy(out, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Return public URL (Nginx or Go will serve this)
	return fmt.Sprintf("/uploads/%s", filename), nil
}
