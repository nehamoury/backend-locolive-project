package privacy

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sqlc-dev/pqtype"

	"privacy-social-backend/internal/repository/db"
)

// AuditAction constants for consistent logging
const (
	AuditActionLogin          = "login"
	AuditActionLogout         = "logout"
	AuditActionPanicActivated = "panic_activated"
	AuditActionPanicDeactivated = "panic_deactivated"
	AuditActionGhostEnabled   = "ghost_mode_enabled"
	AuditActionGhostDisabled  = "ghost_mode_disabled"
	AuditActionAccountPrivate = "account_set_private"
	AuditActionAccountPublic  = "account_set_public"
	AuditActionUserBlocked    = "user_blocked"
	AuditActionUserUnblocked  = "user_unblocked"
	AuditActionPasswordChanged = "password_changed"
	AuditActionEmailChanged   = "email_changed"
	AuditActionAccountDeleted = "account_deleted"
	AuditActionProfileUpdated = "profile_updated"
)

// LogAction creates an audit log entry for a user action
func (s *Service) LogAction(ctx context.Context, userID uuid.UUID, action string, details map[string]interface{}, ipAddress, userAgent string) {
	var detailsJSON pqtype.NullRawMessage
	if details != nil {
		jsonBytes, err := json.Marshal(details)
		if err == nil {
			detailsJSON = pqtype.NullRawMessage{RawMessage: jsonBytes, Valid: true}
		}
	}

	_, err := s.store.CreateUserAuditLog(ctx, db.CreateUserAuditLogParams{
		UserID:    userID,
		Action:    action,
		Details:   detailsJSON,
		IpAddress: sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID.String()).
			Str("action", action).
			Msg("privacy: failed to create audit log")
	}
}
