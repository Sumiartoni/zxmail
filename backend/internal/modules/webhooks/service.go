package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	quotamodule "zxmail/backend/internal/modules/quota"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrInvalidWebhookPayload = errors.New("invalid webhook payload")
)

type Service struct {
	db    *pgxpool.Pool
	log   *logger.Logger
	quota *quotamodule.Service
}

type PostalEvent struct {
	Event      string `json:"event"`
	Reason     string `json:"reason"`
	BounceType string `json:"bounce_type"`
	Timestamp  string `json:"timestamp"`
	Message    struct {
		ID         string `json:"id"`
		MessageID  string `json:"message_id"`
		From       string `json:"from"`
		To         string `json:"to"`
		Subject    string `json:"subject"`
		Size       int    `json:"size"`
		Credential string `json:"credential"`
		Domain     string `json:"domain"`
	} `json:"message"`
}

type ProcessResult struct {
	Status        string `json:"status"`
	Duplicate     bool   `json:"duplicate"`
	PostalMessage string `json:"postal_message_id"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger, quotaService *quotamodule.Service) *Service {
	return &Service{db: db, log: log, quota: quotaService}
}

func (s *Service) ProcessPostalEvent(ctx context.Context, payload []byte) (*ProcessResult, error) {
	var event PostalEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, ErrInvalidWebhookPayload
	}

	if !isSupportedEvent(event.Event) {
		return nil, ErrInvalidWebhookPayload
	}

	eventTime := parseEventTime(event.Timestamp)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var (
		domainID       *uuid.UUID
		organizationID *uuid.UUID
		credentialID   *uuid.UUID
	)

	if event.Message.Credential != "" {
		var matchedCredentialID uuid.UUID
		var matchedOrgID uuid.UUID
		var matchedDomainID uuid.UUID
		err := tx.QueryRow(
			ctx,
			`SELECT id, organization_id, domain_id
			 FROM smtp_credentials
			 WHERE username = $1
			 LIMIT 1`,
			strings.TrimSpace(event.Message.Credential),
		).Scan(&matchedCredentialID, &matchedOrgID, &matchedDomainID)
		if err == nil {
			credentialID = &matchedCredentialID
			organizationID = &matchedOrgID
			domainID = &matchedDomainID
		}
	}

	if domainID == nil && event.Message.Domain != "" {
		var matchedDomainID uuid.UUID
		var matchedOrgID uuid.UUID
		err := tx.QueryRow(
			ctx,
			`SELECT id, organization_id FROM domains WHERE name = $1 LIMIT 1`,
			strings.ToLower(strings.TrimSpace(event.Message.Domain)),
		).Scan(&matchedDomainID, &matchedOrgID)
		if err == nil {
			domainID = &matchedDomainID
			organizationID = &matchedOrgID
		}
	}

	var insertedLogID uuid.UUID
	err = tx.QueryRow(
		ctx,
		`INSERT INTO send_logs (
			domain_id, credential_id, postal_message_id, message_id_header,
			from_addr, to_addr, subject, status, raw_event, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
		ON CONFLICT (postal_message_id, status) WHERE postal_message_id IS NOT NULL DO NOTHING
		RETURNING id`,
		domainID,
		credentialID,
		nullIfEmpty(event.Message.ID),
		nullIfEmpty(event.Message.MessageID),
		event.Message.From,
		event.Message.To,
		event.Message.Subject,
		event.Event,
		string(payload),
		eventTime,
	).Scan(&insertedLogID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &ProcessResult{
			Status:        event.Event,
			Duplicate:     true,
			PostalMessage: event.Message.ID,
		}, nil
	}

	if event.Event == "accepted" && credentialID != nil {
		if _, err := tx.Exec(
			ctx,
			`UPDATE smtp_credentials
			 SET quota_daily_used = quota_daily_used + 1,
			     quota_monthly_used = quota_monthly_used + 1
			 WHERE id = $1`,
			*credentialID,
		); err != nil {
			return nil, err
		}
	}

	if event.Event == "bounced" && organizationID != nil {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO bounces (domain_id, credential_id, recipient, reason, postal_message_id, created_at, disabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			domainID,
			credentialID,
			event.Message.To,
			nullIfEmpty(event.Reason),
			nullIfEmpty(event.Message.ID),
			eventTime,
			isHardBounce(event.BounceType, event.Reason),
		); err != nil {
			return nil, err
		}

		if isHardBounce(event.BounceType, event.Reason) {
			if _, err := tx.Exec(
				ctx,
				`INSERT INTO suppressions (organization_id, recipient, source, reason, active, created_at, released_at)
				 VALUES ($1, $2, 'bounce', $3, TRUE, $4, NULL)
				 ON CONFLICT (organization_id, recipient)
				 DO UPDATE SET source = 'bounce', reason = EXCLUDED.reason, active = TRUE, released_at = NULL`,
				*organizationID,
				event.Message.To,
				nullIfEmpty(event.Reason),
				eventTime,
			); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if event.Event == "accepted" && credentialID != nil && s.quota != nil {
		if err := s.quota.RecordAcceptedEvent(ctx, *credentialID, eventTime); err != nil {
			s.log.Error("quota minute counter update failed for credential %s: %v", credentialID.String(), err)
		}
	}

	return &ProcessResult{
		Status:        event.Event,
		Duplicate:     false,
		PostalMessage: event.Message.ID,
	}, nil
}

func isSupportedEvent(eventType string) bool {
	switch eventType {
	case "accepted", "delivered", "bounced", "deferred", "rejected":
		return true
	default:
		return false
	}
}

func isHardBounce(bounceType string, reason string) bool {
	normalizedType := strings.ToLower(strings.TrimSpace(bounceType))
	if normalizedType == "hard" || normalizedType == "permanent" {
		return true
	}

	normalizedReason := strings.ToLower(strings.TrimSpace(reason))
	keywords := []string{
		"user unknown",
		"recipient rejected",
		"mailbox unavailable",
		"no such user",
		"hard bounce",
		"permanent failure",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalizedReason, keyword) {
			return true
		}
	}

	return false
}

func parseEventTime(value string) time.Time {
	if value == "" {
		return time.Now().UTC()
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}

	return time.Now().UTC()
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
