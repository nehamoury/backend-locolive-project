package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"privacy-social-backend/internal/api"
	"privacy-social-backend/internal/config"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/service/storage"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	// Connect to database
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	store := repository.NewStore(conn)

	// Initialize Storage Service (using R2/S3 if credentials provided, else Local)
	var storageService storage.Service
	if config.R2AccessKey != "" && config.R2AccessKey != "your_r2_access_key" && config.R2SecretKey != "" && config.R2AccountID != "" {
		storageService, err = storage.NewS3Service(context.Background(), config.R2AccountID, config.R2AccessKey, config.R2SecretKey, config.R2BucketName)
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

	log.Info().Str("address", config.ServerAddress).Msg("Starting Locolive API server")
	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")

	}
}
