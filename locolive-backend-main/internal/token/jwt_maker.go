package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

// JWTMaker is a JSON Web Token maker
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey}, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *JWTMaker) CreateToken(username string, userID uuid.UUID, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, userID, role, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":         payload.ID.String(),
		"user_id":    payload.UserID.String(),
		"username":   payload.Username,
		"role":       payload.Role,
		"issued_at":  payload.IssuedAt.Format(time.RFC3339Nano),
		"expired_at": payload.ExpiredAt.Format(time.RFC3339Nano),
	})

	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// VerifyToken checks if the token is valid or not
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok || !jwtToken.Valid {
		return nil, ErrInvalidToken
	}

	// Parse claims safely
	id, _ := uuid.Parse(claims["id"].(string))
	userID, _ := uuid.Parse(claims["user_id"].(string))
	username := claims["username"].(string)
	role := claims["role"].(string)
	issuedAt, _ := time.Parse(time.RFC3339Nano, claims["issued_at"].(string))
	expiredAt, _ := time.Parse(time.RFC3339Nano, claims["expired_at"].(string))

	payload := &Payload{
		ID:        id,
		UserID:    userID,
		Username:  username,
		Role:      role,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}

	if err := payload.Valid(); err != nil {
		return nil, err
	}

	return payload, nil
}
