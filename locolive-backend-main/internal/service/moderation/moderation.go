package moderation

import (
	"context"
	"strings"
	"sync"

	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"

	"github.com/google/uuid"
)

type Service struct {
	store    repository.Store
	keywords []string
	mu       sync.RWMutex
}

func NewService(store repository.Store) *Service {
	// Simple initial list of toxic keywords
	// In production, this would be a much larger list or an external AI call
	initialKeywords := []string{"spam", "scam", "offensive_word1", "offensive_word2", "toxic", "abuse"}

	return &Service{
		store:    store,
		keywords: initialKeywords,
	}
}

// FilterContent checks for toxic keywords and returns if it should be flagged
func (s *Service) FilterContent(text string) (isFlagged bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	textLower := strings.ToLower(text)
	for _, word := range s.keywords {
		if strings.Contains(textLower, word) {
			return true
		}
	}
	return false
}

// ProcessModerationAction handles trust score updates when an admin takes action
func (s *Service) ProcessModerationAction(ctx context.Context, userID uuid.UUID, actionType string) error {
	var delta int32
	switch actionType {
	case "content_deleted":
		delta = -10 // Serious violation
	case "spam_report_confirmed":
		delta = -5
	case "clean_record":
		delta = 2 // Reward for good behavior
	}

	if delta != 0 {
		return s.store.UpdateUserTrustScore(ctx, db.UpdateUserTrustScoreParams{
			ID:         userID,
			TrustScore: delta,
		})
	}
	return nil
}

// UpdateKeywords allows updating the keyword list dynamically (future enhancement)
func (s *Service) UpdateKeywords(newKeywords []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keywords = newKeywords
}
