package util

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomString generates a cryptographically secure random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := int64(len(alphabet))

	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(k))
		if err != nil {
			// Fallback (should never happen with crypto/rand)
			idx = big.NewInt(0)
		}
		sb.WriteByte(alphabet[idx.Int64()])
	}

	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// RandomEmail generates a random email address
func RandomEmail() string {
	return RandomString(8) + "@example.com"
}

const digits = "0123456789"

// RandomDigitString generates a cryptographically secure random string of digits of length n
func RandomDigitString(n int) string {
	var sb strings.Builder
	k := int64(len(digits))

	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(k))
		if err != nil {
			idx = big.NewInt(0)
		}
		sb.WriteByte(digits[idx.Int64()])
	}

	return sb.String()
}
