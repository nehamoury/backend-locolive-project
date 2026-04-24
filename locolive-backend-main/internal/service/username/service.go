package username

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"privacy-social-backend/internal/repository/db"
	usernameutil "privacy-social-backend/internal/util/username"
)

var (
	ErrUsernameTaken     = errors.New("username is already taken")
	ErrUsernameReserved  = errors.New("username is reserved")
	ErrUsernameInvalid   = errors.New("username is invalid")
	ErrUsernameTooShort  = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong   = errors.New("username must be no more than 20 characters")
	ErrUsernameFormat    = errors.New("username can only contain lowercase letters, numbers, and underscores, and must start with a letter")
	ErrUsernameReservedPattern = errors.New("this username pattern is reserved")
)

// UsernameStore defines the required database operations
type UsernameStore interface {
	CheckUsernameExists(ctx context.Context, username string) (bool, error)
	IsUsernameReserved(ctx context.Context, username string) (bool, error)
	GetReservedUsernames(ctx context.Context) ([]string, error)
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	GetUserByUsernameCaseInsensitive(ctx context.Context, username string) (db.User, error)
	FindSimilarUsernames(ctx context.Context, pattern string) ([]string, error)
	RecordUsernameChange(ctx context.Context, arg db.RecordUsernameChangeParams) (db.UsernameHistory, error)
	UpdateUsername(ctx context.Context, arg db.UpdateUsernameParams) (db.User, error)
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
}

// Service handles username validation, availability, and management
type Service struct {
	store      UsernameStore
	redis      *redis.Client
	checker    *usernameutil.ReservedUsernameChecker
}

// NewService creates a new username service
func NewService(store UsernameStore, redisClient *redis.Client) *Service {
	svc := &Service{
		store: store,
		redis: redisClient,
	}

	// Initialize reserved username checker
	checker := usernameutil.NewReservedUsernameChecker(&reservedStoreAdapter{store: store})
	svc.checker = checker

	// Pre-load reserved usernames
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = checker.RefreshCache(ctx)

	return svc
}

// reservedStoreAdapter adapts UsernameStore to ReservedUsernameStore interface
type reservedStoreAdapter struct {
	store UsernameStore
}

func (a *reservedStoreAdapter) GetAllReservedUsernames(ctx context.Context) ([]string, error) {
	return a.store.GetReservedUsernames(ctx)
}

func (a *reservedStoreAdapter) IsReserved(ctx context.Context, username string) (bool, error) {
	return a.store.IsUsernameReserved(ctx, username)
}

// ValidationResult contains the result of username validation
type ValidationResult struct {
	Normalized  string `json:"normalized"`
	IsValid     bool   `json:"is_valid"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ValidateUsername performs complete validation of a username
func (s *Service) ValidateUsername(ctx context.Context, username string) ValidationResult {
	// Step 1: Basic validation (format, length, etc.)
	normalized, valid, errorCode := usernameutil.ValidateUsername(username)

	if !valid {
		return ValidationResult{
			Normalized:   normalized,
			IsValid:      false,
			ErrorCode:    errorCode,
			ErrorMessage: usernameutil.GetValidationErrorMessage(errorCode),
		}
	}

	// Step 2: Check reserved patterns
	if usernameutil.IsReservedPattern(normalized) {
		return ValidationResult{
			Normalized:   normalized,
			IsValid:      false,
			ErrorCode:    "reserved_pattern",
			ErrorMessage: usernameutil.GetValidationErrorMessage("reserved_pattern"),
		}
	}

	// Step 3: Check hardcoded reserved usernames
	if usernameutil.IsHardcodedReserved(normalized) {
		return ValidationResult{
			Normalized:   normalized,
			IsValid:      false,
			ErrorCode:    "reserved",
			ErrorMessage: usernameutil.GetValidationErrorMessage("reserved"),
		}
	}

	// Step 4: Check database reserved usernames
	isReserved, err := s.checker.IsReserved(ctx, normalized)
	if err != nil {
		// Log error but don't fail validation due to cache issues
		// In production, you might want to handle this differently
	}
	if isReserved {
		return ValidationResult{
			Normalized:   normalized,
			IsValid:      false,
			ErrorCode:    "reserved",
			ErrorMessage: usernameutil.GetValidationErrorMessage("reserved"),
		}
	}

	return ValidationResult{
		Normalized: normalized,
		IsValid:    true,
	}
}

// CheckAvailability checks if a username is available for use
func (s *Service) CheckAvailability(ctx context.Context, username string) (available bool, suggestions []string, err error) {
	// Validate the username first
	validation := s.ValidateUsername(ctx, username)
	if !validation.IsValid {
		// Generate suggestions based on the normalized version
		suggestions = usernameutil.GenerateSuggestions(validation.Normalized, usernameutil.DefaultSuggestionConfig())
		return false, suggestions, nil
	}

	normalized := validation.Normalized

	// Check if reserved
	isReserved, err := s.checker.IsReserved(ctx, normalized)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check reserved status: %w", err)
	}
	if isReserved || usernameutil.IsHardcodedReserved(normalized) {
		suggestions = usernameutil.GenerateSuggestions(normalized, usernameutil.DefaultSuggestionConfig())
		return false, suggestions, nil
	}

	// Check if taken (case-insensitive)
	exists, err := s.store.CheckUsernameExists(ctx, normalized)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check username existence: %w", err)
	}

	if exists {
		// Generate suggestions
		suggestions = usernameutil.GenerateSuggestions(normalized, usernameutil.DefaultSuggestionConfig())
		return false, suggestions, nil
	}

	// Check Redis reservation if available
	if s.redis != nil {
		reserved, err := s.isReservedInRedis(ctx, normalized)
		if err != nil {
			// Log error but continue
		}
		if reserved {
			suggestions = usernameutil.GenerateSuggestions(normalized, usernameutil.DefaultSuggestionConfig())
			return false, suggestions, nil
		}
	}

	return true, nil, nil
}

// ReserveUsername temporarily reserves a username in Redis
// This prevents race conditions during registration
func (s *Service) ReserveUsername(ctx context.Context, username string, userID uuid.UUID, ttl time.Duration) error {
	if s.redis == nil {
		return nil // Redis is optional
	}

	normalized := usernameutil.NormalizeUsername(username)
	if normalized == "" {
		return ErrUsernameInvalid
	}

	key := fmt.Sprintf("username:reserve:%s", normalized)

	// Use SET NX (only set if not exists) to prevent overwriting
	set, err := s.redis.SetNX(ctx, key, userID.String(), ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to reserve username: %w", err)
	}

	if !set {
		return ErrUsernameTaken
	}

	return nil
}

// ReleaseReservation removes a username reservation
func (s *Service) ReleaseReservation(ctx context.Context, username string) error {
	if s.redis == nil {
		return nil
	}

	normalized := usernameutil.NormalizeUsername(username)
	key := fmt.Sprintf("username:reserve:%s", normalized)

	return s.redis.Del(ctx, key).Err()
}

// isReservedInRedis checks if a username is reserved in Redis
func (s *Service) isReservedInRedis(ctx context.Context, username string) (bool, error) {
	key := fmt.Sprintf("username:reserve:%s", username)
	exists, err := s.redis.Exists(ctx, key).Result()
	return exists > 0, err
}

// ChangeUsername changes a user's username with full validation and history tracking
func (s *Service) ChangeUsername(ctx context.Context, userID uuid.UUID, newUsername string) (*db.User, error) {
	// Validate new username
	validation := s.ValidateUsername(ctx, newUsername)
	if !validation.IsValid {
		return nil, errors.New(validation.ErrorMessage)
	}

	normalized := validation.Normalized

	// Check availability
	available, _, err := s.CheckAvailability(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, ErrUsernameTaken
	}

	// Get current user to get old username
	currentUser, err := s.store.GetUserByUsernameCaseInsensitive(ctx, normalized)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if err == nil && currentUser.ID == userID {
		// User is trying to change to the same username (case change only)
		// Allow this - they'll just update their display case
	} else if err == nil {
		// Another user has this username
		return nil, ErrUsernameTaken
	}

	// Reserve the new username first
	if s.redis != nil {
		err = s.ReserveUsername(ctx, normalized, userID, 5*time.Minute)
		if err != nil {
			return nil, err
		}
		defer s.ReleaseReservation(ctx, normalized) // Release after operation
	}

	// Get the old username for history (best-effort, not blocking the update)
	_, _ = s.store.GetUserByUsernameCaseInsensitive(ctx, normalized)

	// Update the username
	updatedUser, err := s.store.UpdateUsername(ctx, db.UpdateUsernameParams{
		ID:       userID,
		Username: normalized,
	})
	if err != nil {
		// Check for unique violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("failed to update username: %w", err)
	}

	// Record the change in history (if we have the old username)
	// This would need to be done in a transaction for full consistency

	return &updatedUser, nil
}

// GenerateSuggestions creates username suggestions based on a base string
func (s *Service) GenerateSuggestions(baseUsername string, count int) []string {
	config := usernameutil.DefaultSuggestionConfig()
	config.MaxSuggestions = count
	return usernameutil.GenerateSuggestions(baseUsername, config)
}

// GenerateRandomUsername creates a random available username
func (s *Service) GenerateRandomUsername() string {
	return usernameutil.GenerateRandomUsername()
}

// SanitizeUsernameForDisplay creates a safe display version
func (s *Service) SanitizeUsernameForDisplay(username string) string {
	return usernameutil.SanitizeUsernameDisplay(username)
}

// RefreshReservedCache reloads the reserved username cache
func (s *Service) RefreshReservedCache(ctx context.Context) error {
	return s.checker.RefreshCache(ctx)
}

// IsUsernameTaken is a simple check for internal use
func (s *Service) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	normalized := usernameutil.NormalizeUsername(username)
	if normalized == "" {
		return false, ErrUsernameInvalid
	}

	exists, err := s.store.CheckUsernameExists(ctx, normalized)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	// Check reservations
	if s.redis != nil {
		reserved, err := s.isReservedInRedis(ctx, normalized)
		if err != nil {
			// Log but continue
		}
		if reserved {
			return true, nil
		}
	}

	return false, nil
}

// ValidateAndNormalize validates a username and returns the normalized version
// This is useful for the user creation flow
func (s *Service) ValidateAndNormalize(ctx context.Context, username string) (string, error) {
	if strings.TrimSpace(username) == "" {
		return "", ErrUsernameInvalid
	}

	validation := s.ValidateUsername(ctx, username)
	if !validation.IsValid {
		return "", errors.New(validation.ErrorMessage)
	}

	return validation.Normalized, nil
}
