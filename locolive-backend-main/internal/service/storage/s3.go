package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type Service interface {
	UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
}

type S3Service struct {
	client     *s3.Client
	bucketName string
	endpoint   string
	baseURL    string // Optional: custom domain for public access
}

func NewS3Service(ctx context.Context, accountID, accessKey, secretKey, bucketName, baseURL string) (Service, error) {
	// R2 Endpoint: https://<accountid>.r2.cloudflarestorage.com
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
	})

	return &S3Service{
		client:     client,
		bucketName: bucketName,
		endpoint:   r2Endpoint,
		baseURL:    baseURL,
	}, nil
}

// UploadFile uploads a file stream to R2 and returns the public URL
func (s *S3Service) UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Generate unique filename
	extension := filepath.Ext(filename)
	key := fmt.Sprintf("%s%s", uuid.New().String(), extension)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucketName),
		Key:          aws.String(key),
		Body:         file,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=3600"),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Construction logic:
	// 1. If baseURL is provided, use it: https://<custom_domain>/<key>
	// 2. Otherwise fallback to sensible default: https://<bucket>.r2.dev/<key>
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.baseURL, "/"), key), nil
	}

	return fmt.Sprintf("https://%s.r2.dev/%s", s.bucketName, key), nil
}
