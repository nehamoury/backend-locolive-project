package api

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
)

const (
	maxFileSize = 100 * 1024 * 1024 // 100MB
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
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("file size exceeds maximum allowed size of 100MB")))
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

	// Reset file position
	file.Seek(0, 0)

	// Check for crop parameters
	cropXStr := ctx.PostForm("cropX")
	cropYStr := ctx.PostForm("cropY")
	cropWStr := ctx.PostForm("cropWidth")
	cropHStr := ctx.PostForm("cropHeight")
	aspectRatio := ctx.PostForm("aspectRatio")

	isImage := strings.HasPrefix(mimetype.Detect(header).String(), "image/")
	var uploadReader io.Reader = file
	contentType := fileHeader.Header.Get("Content-Type")
	filename := fileHeader.Filename

	if isImage && cropWStr != "" && cropHStr != "" {
		cX, _ := strconv.Atoi(cropXStr)
		cY, _ := strconv.Atoi(cropYStr)
		cW, _ := strconv.Atoi(cropWStr)
		cH, _ := strconv.Atoi(cropHStr)

		processedImg, pContentType, pErr := server.processImage(file, cX, cY, cW, cH, aspectRatio)
		if pErr == nil {
			uploadReader = bytes.NewReader(processedImg)
			contentType = pContentType
			// If we processed it, it's now a JPEG
			if !strings.HasSuffix(strings.ToLower(filename), ".jpg") && !strings.HasSuffix(strings.ToLower(filename), ".jpeg") {
				filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".jpg"
			}
		} else {
			// Fallback to original but log error
			fmt.Printf("Image processing failed: %v\n", pErr)
			file.Seek(0, 0)
		}
	}

	// Use the storage service
	fileURL, err := server.storage.UploadFile(ctx, uploadReader, filename, contentType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("upload failed: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, uploadResponse{
		URL: fileURL,
	})
}

func (server *Server) processImage(file io.Reader, cX, cY, cW, cH int, aspectRatio string) ([]byte, string, error) {
	img, err := imaging.Decode(file)
	if err != nil {
		return nil, "", err
	}

	// 1. Crop
	if cW > 0 && cH > 0 {
		img = imaging.Crop(img, image.Rect(cX, cY, cX+cW, cY+cH))
	}

	// 2. Resize to target dimensions
	var targetW, targetH int
	switch aspectRatio {
	case "1/1":
		targetW, targetH = 1080, 1080
	case "4/5":
		targetW, targetH = 1080, 1350
	case "16/9":
		targetW, targetH = 1920, 1080
	case "9/16":
		targetW, targetH = 1080, 1920
	default:
		bounds := img.Bounds()
		targetW = bounds.Dx()
		targetH = bounds.Dy()
		if targetW > 1920 {
			targetW = 1920
			targetH = (targetH * 1920) / bounds.Dx()
		}
	}

	img = imaging.Resize(img, targetW, targetH, imaging.Lanczos)

	// 3. Encode (JPEG 85 for good quality/size ratio)
	var buf bytes.Buffer
	err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(85))
	if err != nil {
		return nil, "", err
	}

	return buf.Bytes(), "image/jpeg", nil
}
