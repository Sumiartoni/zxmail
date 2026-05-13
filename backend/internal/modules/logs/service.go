package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/logger"
)

type Service struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

type Filters struct {
	DomainID     string
	CredentialID string
	MessageID    string
	Recipient    string
	Status       string
	From         string
	To           string
	Limit        int
	Offset       int
}

type Entry struct {
	ID              uuid.UUID       `json:"id"`
	DomainID        *uuid.UUID      `json:"domain_id,omitempty"`
	CredentialID    *uuid.UUID      `json:"credential_id,omitempty"`
	PostalMessageID string          `json:"postal_message_id,omitempty"`
	MessageIDHeader string          `json:"message_id_header,omitempty"`
	From            string          `json:"from"`
	Recipient       string          `json:"recipient"`
	Subject         string          `json:"subject,omitempty"`
	Status          string          `json:"status"`
	Timestamp       time.Time       `json:"timestamp"`
	RawEvent        json.RawMessage `json:"raw_event"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger) *Service {
	return &Service{db: db, log: log}
}

func ParseFilters(values url.Values) Filters {
	limit := parseInt(values.Get("limit"), 50)
	offset := parseInt(values.Get("offset"), 0)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return Filters{
		DomainID:     strings.TrimSpace(values.Get("domain_id")),
		CredentialID: strings.TrimSpace(values.Get("credential_id")),
		MessageID:    strings.TrimSpace(values.Get("message_id")),
		Recipient:    strings.TrimSpace(values.Get("recipient")),
		Status:       strings.TrimSpace(values.Get("status")),
		From:         strings.TrimSpace(values.Get("from")),
		To:           strings.TrimSpace(values.Get("to")),
		Limit:        limit,
		Offset:       offset,
	}
}

func (s *Service) List(ctx context.Context, actor authmodule.AuthenticatedUser, filters Filters) ([]Entry, error) {
	query := `
		SELECT l.id, l.domain_id, l.credential_id, COALESCE(l.postal_message_id, ''), COALESCE(l.message_id_header, ''),
		       l.from_addr, l.to_addr, COALESCE(l.subject, ''), l.status, l.created_at, l.raw_event
		FROM send_logs l
		LEFT JOIN domains d ON d.id = l.domain_id
		WHERE 1=1`

	args := make([]any, 0, 8)
	index := 1

	if actor.Role != authmodule.RoleAdmin {
		orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
		if err != nil {
			return nil, err
		}
		query += fmt.Sprintf(" AND d.organization_id = $%d", index)
		args = append(args, orgID)
		index++
	}

	if filters.DomainID != "" {
		query += fmt.Sprintf(" AND l.domain_id = $%d", index)
		args = append(args, filters.DomainID)
		index++
	}
	if filters.CredentialID != "" {
		query += fmt.Sprintf(" AND l.credential_id = $%d", index)
		args = append(args, filters.CredentialID)
		index++
	}
	if filters.MessageID != "" {
		query += fmt.Sprintf(" AND (l.message_id_header = $%d OR l.postal_message_id = $%d)", index, index)
		args = append(args, filters.MessageID)
		index++
	}
	if filters.Recipient != "" {
		query += fmt.Sprintf(" AND l.to_addr ILIKE $%d", index)
		args = append(args, "%"+filters.Recipient+"%")
		index++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND l.status = $%d", index)
		args = append(args, filters.Status)
		index++
	}
	if filters.From != "" {
		query += fmt.Sprintf(" AND l.created_at >= $%d", index)
		args = append(args, filters.From)
		index++
	}
	if filters.To != "" {
		query += fmt.Sprintf(" AND l.created_at <= $%d", index)
		args = append(args, filters.To)
		index++
	}

	query += fmt.Sprintf(" ORDER BY l.created_at DESC LIMIT $%d OFFSET $%d", index, index+1)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(
			&entry.ID,
			&entry.DomainID,
			&entry.CredentialID,
			&entry.PostalMessageID,
			&entry.MessageIDHeader,
			&entry.From,
			&entry.Recipient,
			&entry.Subject,
			&entry.Status,
			&entry.Timestamp,
			&entry.RawEvent,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sanitizeEntriesForActor(actor, entries), nil
}

func (s *Service) resolveCustomerOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}

	var organizationID uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, actor.ID).Scan(&organizationID)
	if err != nil {
		return uuid.Nil, err
	}

	return organizationID, nil
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
