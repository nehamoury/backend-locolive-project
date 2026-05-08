package api

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"privacy-social-backend/internal/repository/db"
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.]+)`)

const MaxMentionsPerComment = 5

type MentionedUser struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
}

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

func (server *Server) processCommentMentions(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	text string,
	senderID uuid.UUID,
	senderUsername string,
) []MentionedUser {
	usernames := extractMentions(text)
	if len(usernames) == 0 {
		return nil
	}

	if len(usernames) > MaxMentionsPerComment {
		usernames = usernames[:MaxMentionsPerComment]
	}

	var mentionedUsers []MentionedUser

	for _, username := range usernames {
		if username == senderUsername {
			continue
		}

		targetUser, err := server.store.GetUserByUsername(ctx, username)
		if err != nil {
			continue
		}

		blocked, err := server.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
			BlockerID: senderID,
			BlockedID: targetUser.ID,
		})
		if err == nil && blocked {
			log.Debug().Str("sender", senderUsername).Str("target", username).Msg("skipping mention - blocked user")
			continue
		}

		blockedByTarget, err := server.store.IsUserBlocked(ctx, db.IsUserBlockedParams{
			BlockerID: targetUser.ID,
			BlockedID: senderID,
		})
		if err == nil && blockedByTarget {
			log.Debug().Str("sender", senderUsername).Str("target", username).Msg("skipping mention - blocked by target")
			continue
		}

		_, err = server.store.CreateMention(ctx, db.CreateMentionParams{
			EntityType:        entityType,
			EntityID:          entityID,
			MentionedUserID:   targetUser.ID,
			MentionedByUserID: senderID,
		})
		if err != nil {
			log.Error().Err(err).Str("username", username).Msg("failed to create mention record")
			continue
		}

		mentionedUsers = append(mentionedUsers, MentionedUser{
			UserID:   targetUser.ID,
			Username: username,
		})

		title := "Mentioned in a comment"
		message := fmt.Sprintf("%s mentioned you in a comment", senderUsername)
		server.createNotificationWithSound(ctx, targetUser.ID, db.NotificationTypeCommentMention, "comment_mention",
			title, message, map[string]uuid.UUID{"user": senderID})
	}

	return mentionedUsers
}
