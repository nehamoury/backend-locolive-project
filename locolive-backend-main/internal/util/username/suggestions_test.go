package username

import (
	"testing"
)

func TestGenerateSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		maxCount    int
		minExpected int
	}{
		{"normal name", "john", 5, 3},
		{"name with spaces", "  Jane  ", 5, 3},
		{"empty input", "", 5, 0},
		{"special chars", "user@name", 5, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSuggestionConfig()
			config.MaxSuggestions = tt.maxCount
			
			suggestions := GenerateSuggestions(tt.base, config)
			
			if len(suggestions) < tt.minExpected {
				t.Errorf("GenerateSuggestions(%q) returned %d suggestions, want at least %d",
					tt.base, len(suggestions), tt.minExpected)
			}
			
			// Check all suggestions are valid usernames
			for _, s := range suggestions {
				if !IsValidUsername(s) {
					t.Errorf("Generated suggestion %q is not a valid username", s)
				}
			}
			
			// Check no duplicates
			seen := make(map[string]bool)
			for _, s := range suggestions {
				if seen[s] {
					t.Errorf("Duplicate suggestion found: %q", s)
				}
				seen[s] = true
			}
		})
	}
}

func TestGenerateRandomUsername(t *testing.T) {
	// Generate multiple usernames and verify they are all valid
	for i := 0; i < 10; i++ {
		username := GenerateRandomUsername()
		
		if username == "" {
			t.Error("GenerateRandomUsername() returned empty string")
			continue
		}
		
		if !IsValidUsername(username) {
			t.Errorf("GenerateRandomUsername() returned invalid username: %q", username)
		}
		
		if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
			t.Errorf("GenerateRandomUsername() returned username with invalid length: %q (length: %d)",
				username, len(username))
		}
	}
}

func TestGenerateSuggestionsWithConfig(t *testing.T) {
	// Test with custom config that disables certain features
	config := SuggestionConfig{
		MaxSuggestions: 3,
		IncludeNumbers: false,
		IncludePrefix:  false,
		IncludeSuffix:  true,
		RandomSuffixes: []string{"live", "official"},
	}
	
	suggestions := GenerateSuggestions("testuser", config)
	
	if len(suggestions) > config.MaxSuggestions {
		t.Errorf("Got %d suggestions, max was %d", len(suggestions), config.MaxSuggestions)
	}
	
	// All suggestions should be valid
	for _, s := range suggestions {
		if !IsValidUsername(s) {
			t.Errorf("Invalid suggestion generated: %q", s)
		}
	}
}

func TestSuggestionUniqueness(t *testing.T) {
	// Generate suggestions multiple times and ensure no duplicates across generations
	allSuggestions := make(map[string]int)
	
	for i := 0; i < 5; i++ {
		suggestions := GenerateSuggestions("john", DefaultSuggestionConfig())
		for _, s := range suggestions {
			allSuggestions[s]++
		}
	}
	
	// Each suggestion should appear only once per generation
	// But may appear multiple times across generations (that's fine)
	// Just check that we got some variety
	if len(allSuggestions) < 3 {
		t.Errorf("Expected variety in suggestions, only got %d unique suggestions", len(allSuggestions))
	}
}
