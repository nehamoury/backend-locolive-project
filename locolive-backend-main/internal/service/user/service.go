package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/token"
	"privacy-social-backend/internal/util"
	usernameutil "privacy-social-backend/internal/util/username"
)

type CreateUserParams struct {
	Phone    string
	Email    string
	Username string
	FullName string
	Password    string
	IsGhostMode bool
}

type LoginUserParams struct {
	Email     string
	Password  string
	UserAgent string
	ClientIP  string
}

type LoginUserResult struct {
	SessionID             uuid.UUID
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  db.User
}

type UpdateEmailParams struct {
	UserID uuid.UUID
	Email  string
}

type Service interface {
	CreateUser(ctx context.Context, params CreateUserParams) (db.User, error)
	LoginUser(ctx context.Context, params LoginUserParams) (*LoginUserResult, error)
	UpdateEmail(ctx context.Context, params UpdateEmailParams) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	SearchUsers(ctx context.Context, query string) ([]db.SearchUsersRow, error)
	UpdateTrustScore(ctx context.Context, userID uuid.UUID) (int32, error)
}

type ServiceImpl struct {
	store      repository.Store
	tokenMaker token.Maker
	config     TokenConfig
}

type TokenConfig struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

func NewService(store repository.Store, tokenMaker token.Maker, config TokenConfig) Service {
	return &ServiceImpl{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
	}
}

var (
	ErrInvalidUsername  = errors.New("invalid username format")
	ErrReservedUsername = errors.New("username is reserved")
)

func (s *ServiceImpl) CreateUser(ctx context.Context, req CreateUserParams) (db.User, error) {
	// Validate and normalize username
	normalized, valid, errorCode := usernameutil.ValidateUsername(req.Username)
	if !valid {
		return db.User{}, errors.New(usernameutil.GetValidationErrorMessage(errorCode))
	}

	// Check for reserved patterns
	if usernameutil.IsReservedPattern(normalized) {
		return db.User{}, ErrReservedUsername
	}

	// Check hardcoded reserved usernames
	if usernameutil.IsHardcodedReserved(normalized) {
		return db.User{}, ErrReservedUsername
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return db.User{}, err
	}

	arg := db.CreateUserParams{
		Phone:        req.Phone,
		Email:        sql.NullString{String: req.Email, Valid: req.Email != ""},
		Username:     normalized, // Store normalized username
		FullName:     req.FullName,
		PasswordHash: hashedPassword,
		IsGhostMode:  req.IsGhostMode,
	}

	user, err := s.store.CreateUser(ctx, arg)
	if err != nil {
		return db.User{}, err
	}

	return user, nil
}

func (s *ServiceImpl) LoginUser(ctx context.Context, req LoginUserParams) (*LoginUserResult, error) {
	user, err := s.store.GetUserByEmail(ctx, sql.NullString{String: req.Email, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		return nil, errors.New("incorrect password")
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(user.Username, user.ID, string(user.Role), s.config.AccessTokenDuration)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(user.Username, user.ID, string(user.Role), s.config.RefreshTokenDuration)
	if err != nil {
		return nil, err
	}

	session, err := s.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    req.UserAgent,
		ClientIp:     req.ClientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return nil, err
	}

	return &LoginUserResult{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  user,
	}, nil
}

func (s *ServiceImpl) UpdateEmail(ctx context.Context, req UpdateEmailParams) (db.User, error) {
	_, err := s.store.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		ID:    req.UserID,
		Email: sql.NullString{String: req.Email, Valid: true},
	})
	if err != nil {
		return db.User{}, err
	}
	// Convert UpdateUserEmailRow to User (manually or by just returning the compatible fields)
	// OR: Helper to fetch full user after update?
	// UpdateUserEmail returns specific fields. The ID should be enough to fetch full user if needed.
	// But UpdateUserEmailRow has most fields. Checking structure...
	// UpdateUserEmailRow has: ID, Phone, PasswordHash, Username, FullName, ...
	// It looks identical to User struct fields but different order/type maybe?
	// Actually generated code UpdateUserEmail returns UpdateUserEmailRow.
	// We should convert it or change interface return type.
	// Let's coerce it to db.User if fields match, or just fetch via ID.
	// Fetching via ID is safer and cleaner architecture-wise.
	return s.store.GetUserByID(ctx, req.UserID)
}

func (s *ServiceImpl) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return s.store.GetUserByID(ctx, id)
}

func (s *ServiceImpl) UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	err = util.CheckPassword(currentPassword, user.PasswordHash)
	if err != nil {
		return errors.New("incorrect current password")
	}

	hashedPassword, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	err = s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: hashedPassword,
	})
	return err
}

func (s *ServiceImpl) SearchUsers(ctx context.Context, query string) ([]db.SearchUsersRow, error) {
	return s.store.SearchUsers(ctx, query)
}

func (s *ServiceImpl) UpdateTrustScore(ctx context.Context, userID uuid.UUID) (int32, error) {
	// 1. Get user profile and stats
	profile, err := s.store.GetUserProfile(ctx, userID)
	if err != nil {
		return 0, err
	}

	// 2. Fetch full user object for extra fields
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Base score
	score := int32(50)

	// Account Age (+1 per day, max 20)
	daysOld := int32(time.Since(profile.CreatedAt).Hours() / 24)
	if daysOld > 20 {
		daysOld = 20
	}
	score += daysOld

	// Connections (+3 per connection, max 15)
	connBonus := int32(profile.ConnectionCount * 3)
	if connBonus > 15 {
		connBonus = 15
	}
	score += connBonus

	// Reports (-10 per report, max -30)
	reportCount, err := s.store.CountReportsForUser(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err == nil {
		reportPenalty := int32(reportCount) * 10
		if reportPenalty > 30 {
			reportPenalty = 30
		}
		score -= reportPenalty
	}

	// Verified status (+15)
	if user.IsVerified {
		score += 15
	}

	// Clamp 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// 3. Update DB
	_, err = s.store.UpdateUserTrust(ctx, db.UpdateUserTrustParams{
		ID:         userID,
		TrustLevel: score,
	})

	return score, err
}
