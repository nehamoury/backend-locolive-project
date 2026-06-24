package util

import (
	"regexp"
	"strings"
)

// hashtagRegex matches valid hashtags containing letters, numbers, and underscores.
var hashtagRegex = regexp.MustCompile(`#([a-zA-Z0-9_]+)`)

// ExtractHashtags extracts unique hashtags from a caption.
// Returns a slice of strings without the '#' prefix.
func ExtractHashtags(caption string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(caption, -1)
	
	// Use a map to keep hashtags unique and prevent duplicate insertions
	uniqueTags := make(map[string]bool)
	var tags []string
	
	for _, match := range matches {
		if len(match) > 1 {
			tagName := match[1]
			// ToLower is usually good for deduplication and slugs
			tagNameLower := strings.ToLower(tagName)
			if !uniqueTags[tagNameLower] {
				uniqueTags[tagNameLower] = true
				tags = append(tags, tagNameLower)
			}
		}
	}
	
	return tags
}
