package username

import (
	"regexp"
	"strings"
	"unicode"
)

// Constants for username validation
const (
	MinUsernameLength = 3
	MaxUsernameLength = 20
)

// Pre-compiled regex for performance
// Rules:
// - Must start with a letter (a-z)
// - Can contain letters, numbers, and underscores
// - Length: 3-20 characters
var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{2,19}$`)

// Common homoglyphs mapping for security (prevent impersonation)
// Maps lookalike characters to their normalized form
var homoglyphMap = map[rune]rune{
	'а': 'a', // Cyrillic а -> Latin a
	'е': 'e', // Cyrillic е -> Latin e
	'о': 'o', // Cyrillic о -> Latin o
	'р': 'p', // Cyrillic р -> Latin p
	'с': 'c', // Cyrillic с -> Latin c
	'х': 'x', // Cyrillic х -> Latin x
	'і': 'i', // Cyrillic і -> Latin i
	'ј': 'j', // Cyrillic ј -> Latin j
	'ԛ': 'q', // Cyrillic ԛ -> Latin q
	'ѕ': 's', // Cyrillic ѕ -> Latin s
	'ԝ': 'w', // Cyrillic ԝ -> Latin w
	'ζ': 'z', // Greek ζ -> Latin z
	'θ': '0', // Greek θ -> 0
	'ο': 'o', // Greek ο -> Latin o
	'ρ': 'p', // Greek ρ -> Latin p
	'𝟎': '0', // Mathematical 𝟎 -> 0
	'𝟏': '1', // Mathematical 𝟏 -> 1
	'𝟐': '2', // Mathematical 𝟐 -> 2
	'𝟑': '3', // Mathematical 𝟑 -> 3
	'𝟒': '4', // Mathematical 𝟒 -> 4
	'𝟓': '5', // Mathematical 𝟓 -> 5
	'𝟔': '6', // Mathematical 𝟔 -> 6
	'𝟕': '7', // Mathematical 𝟕 -> 7
	'𝟖': '8', // Mathematical 𝟖 -> 8
	'𝟗': '9', // Mathematical 𝟗 -> 9
}

// NormalizeUsername normalizes a username input
// Steps:
// 1. Trim whitespace
// 2. Convert to lowercase
// 3. Remove homoglyphs (security)
// 4. Remove invalid characters (keep only a-z, 0-9, _)
func NormalizeUsername(input string) string {
	if input == "" {
		return ""
	}

	// Trim whitespace
	username := strings.TrimSpace(input)

	// Convert to lowercase
	username = strings.ToLower(username)

	// Remove @ prefix if present (common user mistake)
	username = strings.TrimPrefix(username, "@")

	// Remove homoglyphs and normalize
	username = normalizeHomoglyphs(username)

	// Remove invalid characters (keep only a-z, 0-9, _)
	var validChars strings.Builder
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			validChars.WriteRune(r)
		}
	}

	return validChars.String()
}

// normalizeHomoglyphs converts lookalike Unicode characters to ASCII equivalents
func normalizeHomoglyphs(input string) string {
	var result strings.Builder
	for _, r := range input {
		if normalized, ok := homoglyphMap[r]; ok {
			result.WriteRune(normalized)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// IsValidUsername validates if a username meets all requirements
func IsValidUsername(username string) bool {
	if username == "" {
		return false
	}

	// Check length
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return false
	}

	// Check against regex pattern
	return usernameRegex.MatchString(username)
}

// ValidateUsername performs full validation and returns detailed error information
func ValidateUsername(input string) (normalized string, valid bool, errorCode string) {
	if input == "" {
		return "", false, "empty"
	}

	normalized = NormalizeUsername(input)

	if normalized == "" {
		return "", false, "invalid_chars"
	}

	if len(normalized) < MinUsernameLength {
		return normalized, false, "too_short"
	}

	if len(normalized) > MaxUsernameLength {
		return normalized, false, "too_long"
	}

	if !usernameRegex.MatchString(normalized) {
		// Check if starts with number
		if len(normalized) > 0 && normalized[0] >= '0' && normalized[0] <= '9' {
			return normalized, false, "starts_with_number"
		}
		return normalized, false, "invalid_format"
	}

	return normalized, true, ""
}

// IsReservedUsername checks if a username starts with reserved patterns
// These are typically system prefixes that shouldn't be used
func IsReservedPattern(username string) bool {
	if username == "" {
		return false
	}

	reservedPrefixes := []string{
		"system_",
		"admin_",
		"mod_",
		"support_",
		"test_",
		"temp_",
		"tmp_",
		"deleted_",
		"banned_",
		"user_",
		"guest_",
		"anonymous_",
	}

	lowerUsername := strings.ToLower(username)
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(lowerUsername, prefix) {
			return true
		}
	}

	return false
}

// SanitizeUsernameDisplay creates a safe display version of a username
// Used when showing usernames in UI to prevent homograph attacks
func SanitizeUsernameDisplay(username string) string {
	if username == "" {
		return ""
	}

	// Check for mixed scripts (potential spoofing)
	hasLatin := false
	hasNonLatin := false

	for _, r := range username {
		if unicode.Is(unicode.Latin, r) {
			hasLatin = true
		} else if unicode.IsLetter(r) {
			hasNonLatin = true
		}
	}

	// If mixed scripts detected, return normalized version
	if hasLatin && hasNonLatin {
		return NormalizeUsername(username)
	}

	return username
}

// GetValidationErrorMessage returns human-readable error message for validation codes
func GetValidationErrorMessage(errorCode string) string {
	messages := map[string]string{
		"empty":              "Username is required",
		"invalid_chars":      "Username contains only invalid characters",
		"too_short":          "Username must be at least 3 characters",
		"too_long":           "Username must be no more than 20 characters",
		"invalid_format":     "Username can only contain letters, numbers, and underscores",
		"starts_with_number": "Username must start with a letter",
		"reserved":           "This username is reserved",
		"reserved_pattern":   "This username pattern is reserved",
		"taken":              "This username is already taken",
	}

	if msg, ok := messages[errorCode]; ok {
		return msg
	}
	return "Invalid username"
}
