package util

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type VerificationKind string

const (
	VerificationKindEmail VerificationKind = "email"
	VerificationKindPhone VerificationKind = "phone"
)

type PreverifyClaims struct {
	SignupSessionID string           `json:"signup_session_id"`
	Email           string           `json:"email"`
	Phone           string           `json:"phone,omitempty"`
	Kind            VerificationKind `json:"kind"`
	jwt.RegisteredClaims
}

type PreverifyTokenMaker struct {
	secretKey string
}

func NewPreverifyTokenMaker(secretKey string) *PreverifyTokenMaker {
	return &PreverifyTokenMaker{secretKey: secretKey}
}

func (m *PreverifyTokenMaker) CreateToken(kind VerificationKind, sessionID, email, phone string, duration time.Duration) (string, error) {
	claims := PreverifyClaims{
		SignupSessionID: sessionID,
		Email:           email,
		Phone:           phone,
		Kind:            kind,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

func (m *PreverifyTokenMaker) VerifyToken(tokenString string) (*PreverifyClaims, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &PreverifyClaims{}, keyFunc)
	if err != nil {
		return nil, fmt.Errorf("invalid verification token")
	}

	claims, ok := parsed.Claims.(*PreverifyClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid verification token")
	}

	return claims, nil
}
