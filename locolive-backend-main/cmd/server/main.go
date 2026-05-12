package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"privacy-social-backend/internal/api"
	"privacy-social-backend/internal/config"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/service/storage"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// Default to JSON for production, ConsoleWriter for development
	if os.Getenv("ENVIRONMENT") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	// Load configuration
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	// Set gin to release mode in production
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}
	defer conn.Close()

	// Configure DB connection pool
	conn.SetMaxOpenConns(config.DBMaxOpenConns)
	conn.SetMaxIdleConns(config.DBMaxIdleConns)
	conn.SetConnMaxLifetime(config.DBConnMaxLifetime)
	conn.SetConnMaxIdleTime(config.DBConnMaxIdleTime)

	store := repository.NewStore(conn)

	// Initialize Storage Service (using R2/S3 if credentials provided, else Local)
	var storageService storage.Service
	if config.R2AccessKey != "" && config.R2AccessKey != "your_r2_access_key" && config.R2SecretKey != "" && config.R2AccountID != "" {
		storageService, err = storage.NewS3Service(context.Background(), config.R2AccountID, config.R2AccessKey, config.R2SecretKey, config.R2BucketName, config.R2PublicURL)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot initialize S3 storage service")
		}
	} else {
		// Use absolute path for local storage on Windows to avoid relative path issues
		uploadDir, _ := filepath.Abs("./uploads")
		log.Info().Str("path", uploadDir).Msg("Using Local storage service")
		storageService, err = storage.NewLocalStorageService(uploadDir, config.FrontendURL)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot initialize local storage service")
		}
	}

	// Create and start server
	server, err := api.NewServer(config, store, storageService)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	// Setup http.Server for graceful shutdown
	httpServer := &http.Server{
		Addr:    config.ServerAddress,
		Handler: server.GetRouter(), // We'll need to add this getter
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		log.Info().Str("address", config.ServerAddress).Msg("Starting Locolive API server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("cannot start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so no need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close other resources (Redis, etc.)
	if err := server.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing server resources")
	}

	log.Info().Msg("Server exiting")
}
