package api

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"privacy-social-backend/internal/repository/db"
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.]+)`)

// extractMentions parses a string and returns a list of usernames mentioned (without the @)
func extractMentions(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	mentions := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			username := match[1]
			if !seen[username] {
				mentions = append(mentions, username)
				seen[username] = true
			}
		}
	}
	return mentions
}

// processMentions detects mentions in a text and sends notifications to the mentioned users
func (server *Server) processMentions(ctx context.Context, text string, senderID uuid.UUID, senderUsername string) {
	usernames := extractMentions(text)
	if len(usernames) == 0 {
		return
	}

	for _, username := range usernames {
		// Don't notify self
		if username == senderUsername {
			continue
		}

		targetUser, err := server.store.GetUserByUsername(ctx, username)
		if err != nil {
			continue // User not found, skip
		}

		// Send notification
		title := "Mentioned in a comment"
		message := fmt.Sprintf("%s mentioned you in a comment", senderUsername)
		
		// Use an existing notification type that the DB accepts
		server.createNotificationWithSound(ctx, targetUser.ID, db.NotificationTypeStoryReaction, "story_mention",
			title, message, map[string]uuid.UUID{"user": senderID})
	}
}
