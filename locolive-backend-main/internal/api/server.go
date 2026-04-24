package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"privacy-social-backend/internal/config"
	"privacy-social-backend/internal/realtime"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/service/admin"
	"privacy-social-backend/internal/service/location"
	"privacy-social-backend/internal/service/safety"
	"privacy-social-backend/internal/service/storage"
	"privacy-social-backend/internal/service/moderation"
	"privacy-social-backend/internal/service/story"
	"privacy-social-backend/internal/service/user"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
	"privacy-social-backend/internal/worker"

	"github.com/rs/zerolog/log"
)

// Server serves HTTP requests for our privacy social service
type Server struct {
	config     config.Config
	store      repository.Store
	tokenMaker token.Maker
	redis      *redis.Client
	router     *gin.Engine
	hub        *realtime.Hub
	safety     *safety.Monitor
	location   *location.RedisLocationService
	story      story.Service
	user       user.Service
	admin      admin.Service
	storage    storage.Service
	moderation *moderation.Service
	mailer     util.Mailer
}

// NewServer creates a new HTTP server and setup routing
func NewServer(
	config config.Config,
	store repository.Store,
	storageService storage.Service,
) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	var opt *redis.Options
	if config.RedisAddress != "" {
		var parseErr error
		opt, parseErr = redis.ParseURL(config.RedisAddress)
		if parseErr != nil {
			log.Warn().Err(parseErr).Str("address", config.RedisAddress).Msg("Failed to parse Redis URL, attempting simple address")
			opt = &redis.Options{
				Addr: config.RedisAddress,
			}
		}
	} else {
		log.Warn().Msg("REDIS_ADDRESS is empty, defaulting to localhost:6379 (may fail in Docker)")
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	rdb := redis.NewClient(opt)
	hub := realtime.NewHub(rdb)
	go hub.Run() // Start the hub in a goroutine

	safetyMonitor := safety.NewMonitor(rdb)
	locationService := location.NewRedisLocationService(rdb, store, hub)
	storyService := story.NewService(store, rdb, safetyMonitor, hub)
	userService := user.NewService(store, tokenMaker, user.TokenConfig{
		AccessTokenDuration:  config.AccessTokenDuration,
		RefreshTokenDuration: config.RefreshTokenDuration,
	})
	adminService := admin.NewService(store, rdb)
	modService := moderation.NewService(store)


	// Initialize SMTP Mailer (recommended for Gmail App Passwords)
	host := config.SMTPHost
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := config.SMTPPort
	if port == "" {
		port = "587"
	}
	mailer := util.NewGmailMailer(
		config.EmailSenderName,
		config.EmailSenderAddress,
		config.EmailSenderPassword,
		host,
		port,
		config.FrontendURL,
	)
	log.Info().Str("host", host).Msg("Email Service Initialized (SMTP)")

	// Initialize and Start Background Workers
	bgWorker := worker.NewCleanupWorker(store)
	bgWorker.Start()
	bgWorker.StartCrossingDetector()

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		redis:      rdb,
		safety:     safetyMonitor,
		hub:        hub,
		location:   locationService,
		story:      storyService,
		user:       userService,
		admin:      adminService,
		storage:    storageService,
		moderation: modService,
		mailer:     mailer,
	}

	server.setupRouter()
	return server, nil
}

// Start runs the HTTP/HTTPS server on a specific address
func (server *Server) Start(address string) error {
	// Check if TLS certificates are configured
	if server.config.TLSCertFile != "" && server.config.TLSKeyFile != "" {
		fmt.Printf("Starting HTTPS server on %s\n", address)
		fmt.Printf("TLS Cert: %s\n", server.config.TLSCertFile)
		fmt.Printf("TLS Key: %s\n", server.config.TLSKeyFile)
		return server.router.RunTLS(address, server.config.TLSCertFile, server.config.TLSKeyFile)
	}

	// HTTP mode (development only - production should use HTTPS)
	if gin.Mode() == gin.ReleaseMode {
		fmt.Println("WARNING: Running HTTP in production mode. Set TLS_CERT_FILE and TLS_KEY_FILE for HTTPS.")
	}
	fmt.Printf("Starting HTTP server on %s\n", address)
	return server.router.Run(address)
}
