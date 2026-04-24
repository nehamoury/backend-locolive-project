package moderation

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"

	"github.com/google/uuid"
)

type Service struct {
	store      repository.Store
	profanity  *regexp.Regexp
	spamWords  []string
	mu         sync.RWMutex
}

func NewService(store repository.Store) *Service {
	// Common profanity patterns (simplified for example)
	profanityRegex := regexp.MustCompile(`(?i)(badword1|badword2|offensive_word3)`)
	
	spamWords := []string{
		"win money", "free gift", "click here", "buy now", 
		"crypto scam", "investment opportunity", "telegram me",
	}

	return &Service{
		store:     store,
		profanity: profanityRegex,
		spamWords: spamWords,
	}
}

// ModerateText checks for profanity and spam
func (s *Service) ModerateText(text string) (isFlagged bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	textLower := strings.ToLower(text)

	// 1. Profanity Check
	if s.profanity.MatchString(text) {
		return true, "profanity"
	}

	// 2. Spam Check
	for _, word := range s.spamWords {
		if strings.Contains(textLower, word) {
			return true, "spam"
		}
	}

	return false, ""
}

// ModerateImage is a placeholder for external AI (e.g., Google Vision)
func (s *Service) ModerateImage(ctx context.Context, mediaURL string) (isFlagged bool, reason string) {
	// In production: call external AI API (Google Vision, AWS Rekognition, etc.)
	
	// Mock Implementation: Flag if URL contains specific suspicious patterns
	// This demonstrates the hook for the USER.
	urlLower := strings.ToLower(mediaURL)
	if strings.Contains(urlLower, "nsfw") || strings.Contains(urlLower, "adult") || strings.Contains(urlLower, "explicit") {
		return true, "potential_inappropriate_media"
	}
	
	return false, ""
}

// ProcessModerationAction handles trust score updates when an admin takes action
func (s *Service) ProcessModerationAction(ctx context.Context, userID uuid.UUID, actionType string) error {
	var delta int32
	switch actionType {
	case "content_deleted":
		delta = -10
	case "spam_report_confirmed":
		delta = -5
	case "clean_record":
		delta = 2
	}

	if delta != 0 {
		// Consolidate: update trust_level directly or through the score column
		// For now, we'll use the existing UpdateUserTrustScore to maintain compatibility
		// but eventually we should use the recalculation service.
		return s.store.UpdateUserTrustScore(ctx, db.UpdateUserTrustScoreParams{
			ID:         userID,
			TrustScore: delta,
		})
	}
	return nil
}
