package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"privacy-social-backend/internal/util"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
)

const (
	maxFileSize = 10 * 1024 * 1024 // 10MB
)

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".mp4":  true,
	".mov":  true,
	".webm": true,
}

var allowedMimeTypes = map[string]bool{
	"image/jpeg":       true,
	"image/png":        true,
	"image/gif":        true,
	"image/webp":       true,
	"video/mp4":        true,
	"video/quicktime":  true, // .mov
	"video/webm":       true,
}

func isValidExtension(ext string) bool {
	return allowedExtensions[strings.ToLower(ext)]
}

func isValidMimeType(data []byte) bool {
	mimetype := mimetype.Detect(data)
	return allowedMimeTypes[mimetype.String()]
}

func sanitizeFilename(filename string) string {
	// Remove any path components
	filename = filepath.Base(filename)
	// Remove any non-alphanumeric characters except dots and dashes
	var result strings.Builder
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

type uploadResponse struct {
	URL string `json:"url"`
}

func (server *Server) uploadFile(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("no file uploaded")))
		return
	}

	// Validate file size
	if fileHeader.Size > maxFileSize {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("file size exceeds maximum allowed size of 10MB")))
		return
	}

	// Validate file extension
	ext := filepath.Ext(fileHeader.Filename)
	if !isValidExtension(ext) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid file type. Allowed: jpg, jpeg, png, gif, webp, mp4, mov, webm")))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("failed to open file")))
		return
	}
	defer file.Close()

	// Validate actual file content (MIME type detection)
	// Read first 512 bytes for detection
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("failed to read file")))
		return
	}
	header = header[:n]

	if !isValidMimeType(header) {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("file content does not match allowed types")))
		return
	}

	// Reset file position for saving
	file.Seek(0, 0)

	// Generate safe filename: random string + sanitized original name
	safeFilename := sanitizeFilename(fileHeader.Filename)
	if safeFilename == "" || safeFilename == "." {
		safeFilename = "file" + ext
	}
	filename := util.RandomString(16) + "_" + safeFilename
	dst := filepath.Join("./uploads", filename)

	if err := ctx.SaveUploadedFile(fileHeader, dst); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("failed to save file")))
		return
	}

	publicURL := "/uploads/" + filename

	ctx.JSON(http.StatusOK, uploadResponse{
		URL: publicURL,
	})
}
