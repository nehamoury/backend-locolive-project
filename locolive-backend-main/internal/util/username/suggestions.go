package username

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// SuggestionConfig controls suggestion generation behavior
type SuggestionConfig struct {
	MaxSuggestions int
	IncludeNumbers bool
	IncludePrefix  bool
	IncludeSuffix  bool
	RandomSuffixes []string
}

// DefaultSuggestionConfig returns the default configuration
func DefaultSuggestionConfig() SuggestionConfig {
	return SuggestionConfig{
		MaxSuggestions: 5,
		IncludeNumbers: true,
		IncludePrefix:  true,
		IncludeSuffix:  true,
		RandomSuffixes: []string{
			"live", "official", "real", "original", "verified",
			"the", "iam", "its", "mr", "mrs", "ms", "dr",
			"daily", "life", "world", "zone", "hub",
		},
	}
}

// GenerateSuggestions creates alternative username suggestions based on a base username
func GenerateSuggestions(baseUsername string, config SuggestionConfig) []string {
	if baseUsername == "" {
		return []string{}
	}

	// Normalize the base username first
	base := NormalizeUsername(baseUsername)
	if base == "" {
		return []string{}
	}

	// Ensure base doesn't exceed max length when we add suffixes
	maxBaseLen := MaxUsernameLength - 4 // Leave room for numbers/prefixes
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}

	suggestions := make([]string, 0, config.MaxSuggestions)
	seen := make(map[string]bool)
	seen[base] = true // Don't suggest the original if it's taken

	// Initialize random seed
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Strategy 1: Add numbers (e.g., john123, john_2024)
	if config.IncludeNumbers {
		numberSuggestions := generateNumberSuggestions(base, rng)
		for _, s := range numberSuggestions {
			if !seen[s] && len(suggestions) < config.MaxSuggestions {
				seen[s] = true
				suggestions = append(suggestions, s)
			}
		}
	}

	// Strategy 2: Add prefix (e.g., real_john, the_john)
	if config.IncludePrefix && len(suggestions) < config.MaxSuggestions {
		prefixSuggestions := generatePrefixSuggestions(base, config.RandomSuffixes, rng)
		for _, s := range prefixSuggestions {
			if !seen[s] && len(suggestions) < config.MaxSuggestions {
				seen[s] = true
				suggestions = append(suggestions, s)
			}
		}
	}

	// Strategy 3: Add suffix (e.g., john_live, john_official)
	if config.IncludeSuffix && len(suggestions) < config.MaxSuggestions {
		suffixSuggestions := generateSuffixSuggestions(base, config.RandomSuffixes, rng)
		for _, s := range suffixSuggestions {
			if !seen[s] && len(suggestions) < config.MaxSuggestions {
				seen[s] = true
				suggestions = append(suggestions, s)
			}
		}
	}

	// Strategy 4: Add underscores for readability (if base is long enough)
	if len(base) > 5 && len(suggestions) < config.MaxSuggestions {
		underscored := addUnderscores(base, rng)
		if !seen[underscored] && IsValidUsername(underscored) {
			seen[underscored] = true
			suggestions = append(suggestions, underscored)
		}
	}

	return suggestions
}

// generateNumberSuggestions creates number-based variations
func generateNumberSuggestions(base string, rng *rand.Rand) []string {
	suggestions := []string{}

	// Random 1-3 digit numbers
	for i := 0; i < 3; i++ {
		num := rng.Intn(999) + 1
		suffix := strconv.Itoa(num)
		candidate := base + suffix
		if len(candidate) <= MaxUsernameLength && IsValidUsername(candidate) {
			suggestions = append(suggestions, candidate)
		}
	}

	// Current year variations
	year := time.Now().Year()
	yearSuffixes := []string{
		strconv.Itoa(year),
		strconv.Itoa(year % 100), // Last 2 digits
	}

	for _, suffix := range yearSuffixes {
		// With underscore
		candidate := base + "_" + suffix
		if len(candidate) <= MaxUsernameLength && IsValidUsername(candidate) {
			suggestions = append(suggestions, candidate)
		}
	}

	// Birth year style (random 80s-00s)
	birthYear := 1980 + rng.Intn(25)
	candidate := base + strconv.Itoa(birthYear)
	if len(candidate) <= MaxUsernameLength && IsValidUsername(candidate) {
		suggestions = append(suggestions, candidate)
	}

	return suggestions
}

// generatePrefixSuggestions creates prefix-based variations
func generatePrefixSuggestions(base string, prefixes []string, rng *rand.Rand) []string {
	suggestions := []string{}

	// Shuffle prefixes
	rng.Shuffle(len(prefixes), func(i, j int) {
		prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
	})

	for _, prefix := range prefixes[:min(3, len(prefixes))] {
		candidate := prefix + "_" + base
		if len(candidate) <= MaxUsernameLength && IsValidUsername(candidate) {
			suggestions = append(suggestions, candidate)
		}
	}

	return suggestions
}

// generateSuffixSuggestions creates suffix-based variations
func generateSuffixSuggestions(base string, suffixes []string, rng *rand.Rand) []string {
	suggestions := []string{}

	// Shuffle suffixes
	rng.Shuffle(len(suffixes), func(i, j int) {
		suffixes[i], suffixes[j] = suffixes[j], suffixes[i]
	})

	for _, suffix := range suffixes[:min(3, len(suffixes))] {
		candidate := base + "_" + suffix
		if len(candidate) <= MaxUsernameLength && IsValidUsername(candidate) {
			suggestions = append(suggestions, candidate)
		}
	}

	return suggestions
}

// addUnderscores inserts underscores at word boundaries for readability
func addUnderscores(base string, rng *rand.Rand) string {
	if len(base) < 6 {
		return base
	}

	// Simple approach: insert underscore after 3-6 characters
	pos := 3 + rng.Intn(min(4, len(base)-3))
	result := base[:pos] + "_" + base[pos:]

	if len(result) > MaxUsernameLength {
		return base
	}

	return result
}

// GenerateRandomUsername creates a random username for users who don't provide one
func GenerateRandomUsername() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	adjectives := []string{
		"happy", "bright", "cool", "clever", "brave", "calm", "eager", "fancy",
		"gentle", "jolly", "kind", "lively", "proud", "silly", "witty", "zesty",
		"swift", "bold", "wild", "cosmic", "sunny", "lucky", "magic", "golden",
	}

	nouns := []string{
		"panda", "tiger", "eagle", "wolf", "fox", "bear", "lion", "hawk",
		"dolphin", "whale", "falcon", "raven", "cobra", "viper", "dragon",
		"phoenix", "unicorn", "wizard", "ninja", "pirate", "captain", "hero",
		"star", "comet", "meteor", "nova", "nebula", "quasar", "atom",
	}

	adj := adjectives[rng.Intn(len(adjectives))]
	noun := nouns[rng.Intn(len(nouns))]
	num := rng.Intn(9999) + 1

	// Try different formats
	formats := []string{
		fmt.Sprintf("%s_%s%d", adj, noun, num),
		fmt.Sprintf("%s%s%d", adj, noun, num),
		fmt.Sprintf("%s_%s_%d", adj, noun, num%100),
	}

	for _, format := range formats {
		if len(format) <= MaxUsernameLength && IsValidUsername(format) {
			return format
		}
	}

	// Fallback: simple format with truncated parts
	return fmt.Sprintf("user%d", rng.Intn(999999))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
