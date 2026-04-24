package username

import (
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Rahul_123", "rahul_123"},
		{"  Rahul123  ", "rahul123"},
		{"@username", "username"},
		{"USER_NAME_123", "user_name_123"},
		{"user@name", "username"},
		{"user name", "username"},
		{"user-name", "username"},
		{"user.name", "username"},
		{"", ""},
		{"!!!", ""},
		{"admin", "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeUsername(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		username string
		valid    bool
	}{
		{"rahul", true},
		{"rahul_123", true},
		{"user_name", true},
		{"abc", true}, // minimum length
		{"abcdefghijklmnopqrst", true}, // maximum length (20)
		{"ab", false},  // too short
		{"abcdefghijklmnopqrstu", false}, // too long (21)
		{"123user", false}, // starts with number
		{"_user", false}, // starts with underscore
		{"user@name", false}, // invalid character
		{"user-name", false}, // invalid character
		{"user.name", false}, // invalid character
		{"user name", false}, // invalid character
		{"", false}, // empty
		{"ADMIN", false}, // uppercase (normalized would be valid, but raw input is invalid)
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			result := IsValidUsername(tt.username)
			if result != tt.valid {
				t.Errorf("IsValidUsername(%q) = %v, want %v", tt.username, result, tt.valid)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		input        string
		wantValid    bool
		wantErrorCode string
	}{
		{"rahul", true, ""},
		{"", false, "empty"},
		{"ab", false, "too_short"},
		{"abcdefghijklmnopqrstu", false, "too_long"},
		{"123user", false, "starts_with_number"},
		{"!!!", false, "invalid_chars"},
		{"user@name", false, "invalid_format"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, valid, errorCode := ValidateUsername(tt.input)
			if valid != tt.wantValid {
				t.Errorf("ValidateUsername(%q) valid = %v, want %v", tt.input, valid, tt.wantValid)
			}
			if errorCode != tt.wantErrorCode {
				t.Errorf("ValidateUsername(%q) errorCode = %q, want %q", tt.input, errorCode, tt.wantErrorCode)
			}
		})
	}
}

func TestIsReservedPattern(t *testing.T) {
	tests := []struct {
		username string
		reserved bool
	}{
		{"admin_user", true},
		{"mod_something", true},
		{"support_team", true},
		{"test_account", true},
		{"temp_user", true},
		{"normaluser", false},
		{"rahul_123", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			result := IsReservedPattern(tt.username)
			if result != tt.reserved {
				t.Errorf("IsReservedPattern(%q) = %v, want %v", tt.username, result, tt.reserved)
			}
		})
	}
}

func TestIsHardcodedReserved(t *testing.T) {
	tests := []struct {
		username string
		reserved bool
	}{
		{"admin", true},
		{"ADMIN", true}, // case insensitive
		{"Admin", true},
		{"root", true},
		{"system", true},
		{"support", true},
		{"locolive", true},
		{"rahul", false},
		{"john_doe", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			result := IsHardcodedReserved(tt.username)
			if result != tt.reserved {
				t.Errorf("IsHardcodedReserved(%q) = %v, want %v", tt.username, result, tt.reserved)
			}
		})
	}
}

func TestGetValidationErrorMessage(t *testing.T) {
	tests := []struct {
		code     string
		contains string
	}{
		{"empty", "required"},
		{"too_short", "at least 3"},
		{"too_long", "no more than 20"},
		{"invalid_format", "only contain"},
		{"reserved", "reserved"},
		{"unknown_code", "Invalid username"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			msg := GetValidationErrorMessage(tt.code)
			if msg == "" {
				t.Errorf("GetValidationErrorMessage(%q) returned empty string", tt.code)
			}
		})
	}
}

func TestNormalizeHomoglyphs(t *testing.T) {
	// Test that Cyrillic lookalikes are normalized
	tests := []struct {
		input    string
		expected string
	}{
		{"аdmin", "admin"}, // Cyrillic а + dmin
		{"раypal", "paypal"}, // Cyrillic р and а
		{"normal", "normal"}, // All Latin
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHomoglyphs(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeHomoglyphs(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
