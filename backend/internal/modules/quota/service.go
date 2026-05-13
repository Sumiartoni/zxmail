package quota

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

const EnforcementNote = "Production v1 customers send directly to Postal. Pre-send quota enforcement is limited until zxMail adds an SMTP gateway in front of Postal."

var (
	ErrQuotaInvalidInput      = errors.New("invalid quota input")
	ErrQuotaCredentialMissing = errors.New("credential not found")
	ErrQuotaForbidden         = errors.New("quota forbidden")
)

type Service struct {
	db    *pgxpool.Pool
	redis *redis.Client
	log   *logger.Logger
}

type SnapshotInput struct {
	CredentialID   uuid.UUID
	Enabled        bool
	PerMinuteLimit *int
	DailyLimit     *int
	DailyUsed      int
	MonthlyLimit   *int
	MonthlyUsed    int
	EvaluatedAt    time.Time
}

type Snapshot struct {
	PerMinuteLimit  *int     `json:"per_minute_limit,omitempty"`
	PerMinuteUsed   int      `json:"per_minute_used"`
	DailyLimit      *int     `json:"daily_limit,omitempty"`
	DailyUsed       int      `json:"daily_used"`
	MonthlyLimit    *int     `json:"monthly_limit,omitempty"`
	MonthlyUsed     int      `json:"monthly_used"`
	Status          string   `json:"status"`
	Limited         bool     `json:"limited"`
	Exceeded        []string `json:"exceeded,omitempty"`
	EnforcementNote string   `json:"enforcement_note"`
}

type UpdateCredentialQuotaInput struct {
	PerMinuteLimit    *int
	PerMinuteLimitSet bool
	DailyLimit        *int
	DailyLimitSet     bool
	MonthlyLimit      *int
	MonthlyLimitSet   bool
	ResetMinuteUsed   bool
	ResetDailyUsed    bool
	ResetMonthlyUsed  bool
}

type CredentialQuotaView struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	DomainID        uuid.UUID  `json:"domain_id"`
	DomainName      string     `json:"domain_name"`
	Username        string     `json:"username"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	PerMinuteLimit  *int       `json:"per_minute_limit,omitempty"`
	PerMinuteUsed   int        `json:"per_minute_used"`
	DailyLimit      *int       `json:"daily_limit,omitempty"`
	DailyUsed       int        `json:"daily_used"`
	MonthlyLimit    *int       `json:"monthly_limit,omitempty"`
	MonthlyUsed     int        `json:"monthly_used"`
	Limited         bool       `json:"limited"`
	Exceeded        []string   `json:"exceeded,omitempty"`
	EnforcementNote string     `json:"enforcement_note"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
}

type credentialRow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	DomainID       uuid.UUID
	DomainName     string
	Username       string
	Enabled        bool
	LastUsedAt     *time.Time
	PerMinuteLimit *int
	DailyLimit     *int
	DailyUsed      int
	MonthlyLimit   *int
	MonthlyUsed    int
}

func NewService(db *pgxpool.Pool, redisClient *redis.Client, log *logger.Logger) *Service {
	return &Service{db: db, redis: redisClient, log: log}
}

func (s *Service) BuildSnapshot(ctx context.Context, input SnapshotInput) (Snapshot, error) {
	evaluatedAt := input.EvaluatedAt.UTC()
	if input.EvaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}

	perMinuteUsed := 0
	if s.redis != nil {
		value, err := s.redis.Get(ctx, minuteUsageKey(input.CredentialID, evaluatedAt)).Int()
		if err != nil && !errors.Is(err, redis.Nil) {
			s.log.Error("quota minute usage lookup failed for credential %s: %v", input.CredentialID, err)
		} else {
			perMinuteUsed = value
		}
	}

	status, limited, exceeded := deriveStatus(
		input.Enabled,
		input.PerMinuteLimit,
		perMinuteUsed,
		input.DailyLimit,
		input.DailyUsed,
		input.MonthlyLimit,
		input.MonthlyUsed,
	)

	return Snapshot{
		PerMinuteLimit:  input.PerMinuteLimit,
		PerMinuteUsed:   perMinuteUsed,
		DailyLimit:      input.DailyLimit,
		DailyUsed:       input.DailyUsed,
		MonthlyLimit:    input.MonthlyLimit,
		MonthlyUsed:     input.MonthlyUsed,
		Status:          status,
		Limited:         limited,
		Exceeded:        exceeded,
		EnforcementNote: EnforcementNote,
	}, nil
}

func (s *Service) RecordAcceptedEvent(ctx context.Context, credentialID uuid.UUID, acceptedAt time.Time) error {
	if s.redis == nil {
		return nil
	}

	key := minuteUsageKey(credentialID, acceptedAt.UTC())
	pipe := s.redis.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateCredentialQuota(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, input UpdateCredentialQuotaInput) (*CredentialQuotaView, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrQuotaForbidden
	}
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(credentialID)
	if err != nil {
		return nil, ErrQuotaInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row, err := s.loadCredentialRow(ctx, tx, parsedID)
	if err != nil {
		return nil, err
	}

	err = tx.QueryRow(
		ctx,
		`UPDATE smtp_credentials
		 SET quota_per_minute_limit = CASE WHEN $2 THEN $3 ELSE quota_per_minute_limit END,
		     quota_daily_limit = CASE WHEN $4 THEN $5 ELSE quota_daily_limit END,
		     quota_daily_used = CASE WHEN $6 THEN 0 ELSE quota_daily_used END,
		     quota_monthly_limit = CASE WHEN $7 THEN $8 ELSE quota_monthly_limit END,
		     quota_monthly_used = CASE WHEN $9 THEN 0 ELSE quota_monthly_used END
		 WHERE id = $1
		 RETURNING quota_per_minute_limit, quota_daily_limit, quota_daily_used, quota_monthly_limit, quota_monthly_used`,
		parsedID,
		input.PerMinuteLimitSet,
		input.PerMinuteLimit,
		input.DailyLimitSet,
		input.DailyLimit,
		input.ResetDailyUsed,
		input.MonthlyLimitSet,
		input.MonthlyLimit,
		input.ResetMonthlyUsed,
	).Scan(
		&row.PerMinuteLimit,
		&row.DailyLimit,
		&row.DailyUsed,
		&row.MonthlyLimit,
		&row.MonthlyUsed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuotaCredentialMissing
		}
		return nil, err
	}

	if err := s.insertAuditLog(ctx, tx, actor.ID, &row.OrganizationID, "credential.quota.update", "smtp_credential", row.ID, map[string]any{
		"per_minute_limit_set": input.PerMinuteLimitSet,
		"daily_limit_set":      input.DailyLimitSet,
		"monthly_limit_set":    input.MonthlyLimitSet,
		"reset_minute_used":    input.ResetMinuteUsed,
		"reset_daily_used":     input.ResetDailyUsed,
		"reset_monthly_used":   input.ResetMonthlyUsed,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if input.ResetMinuteUsed {
		if err := s.resetMinuteUsage(ctx, row.ID, time.Now().UTC()); err != nil {
			s.log.Error("quota minute reset failed for credential %s: %v", row.ID, err)
		}
	}

	return s.buildCredentialQuotaView(ctx, row)
}

func (s *Service) DisableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*CredentialQuotaView, error) {
	return s.setCredentialEnabled(ctx, actor, credentialID, false)
}

func (s *Service) EnableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*CredentialQuotaView, error) {
	return s.setCredentialEnabled(ctx, actor, credentialID, true)
}

func validateUpdateInput(input UpdateCredentialQuotaInput) error {
	for _, value := range []*int{input.PerMinuteLimit, input.DailyLimit, input.MonthlyLimit} {
		if value != nil && *value < 0 {
			return ErrQuotaInvalidInput
		}
	}
	return nil
}

func deriveStatus(enabled bool, perMinuteLimit *int, perMinuteUsed int, dailyLimit *int, dailyUsed int, monthlyLimit *int, monthlyUsed int) (string, bool, []string) {
	if !enabled {
		return "disabled", false, nil
	}

	var exceeded []string
	if perMinuteLimit != nil && perMinuteUsed >= *perMinuteLimit {
		exceeded = append(exceeded, "per_minute")
	}
	if dailyLimit != nil && dailyUsed >= *dailyLimit {
		exceeded = append(exceeded, "daily")
	}
	if monthlyLimit != nil && monthlyUsed >= *monthlyLimit {
		exceeded = append(exceeded, "monthly")
	}

	if len(exceeded) > 0 {
		return "limited", true, exceeded
	}

	return "enabled", false, nil
}

func minuteUsageKey(credentialID uuid.UUID, at time.Time) string {
	return "quota:credential:" + credentialID.String() + ":minute:" + at.UTC().Format("200601021504")
}

func (s *Service) resetMinuteUsage(ctx context.Context, credentialID uuid.UUID, at time.Time) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, minuteUsageKey(credentialID, at)).Err()
}

func (s *Service) setCredentialEnabled(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, enabled bool) (*CredentialQuotaView, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrQuotaForbidden
	}

	parsedID, err := uuid.Parse(credentialID)
	if err != nil {
		return nil, ErrQuotaInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row, err := s.loadCredentialRow(ctx, tx, parsedID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE smtp_credentials SET enabled = $2 WHERE id = $1`, parsedID, enabled); err != nil {
		return nil, err
	}
	row.Enabled = enabled

	action := "credential.disable"
	if enabled {
		action = "credential.enable"
	}
	if err := s.insertAuditLog(ctx, tx, actor.ID, &row.OrganizationID, action, "smtp_credential", row.ID, map[string]any{
		"username": row.Username,
		"enabled":  enabled,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.buildCredentialQuotaView(ctx, row)
}

func (s *Service) buildCredentialQuotaView(ctx context.Context, row *credentialRow) (*CredentialQuotaView, error) {
	snapshot, err := s.BuildSnapshot(ctx, SnapshotInput{
		CredentialID:   row.ID,
		Enabled:        row.Enabled,
		PerMinuteLimit: row.PerMinuteLimit,
		DailyLimit:     row.DailyLimit,
		DailyUsed:      row.DailyUsed,
		MonthlyLimit:   row.MonthlyLimit,
		MonthlyUsed:    row.MonthlyUsed,
	})
	if err != nil {
		return nil, err
	}

	return &CredentialQuotaView{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		DomainID:        row.DomainID,
		DomainName:      row.DomainName,
		Username:        row.Username,
		Enabled:         row.Enabled,
		Status:          snapshot.Status,
		PerMinuteLimit:  snapshot.PerMinuteLimit,
		PerMinuteUsed:   snapshot.PerMinuteUsed,
		DailyLimit:      snapshot.DailyLimit,
		DailyUsed:       snapshot.DailyUsed,
		MonthlyLimit:    snapshot.MonthlyLimit,
		MonthlyUsed:     snapshot.MonthlyUsed,
		Limited:         snapshot.Limited,
		Exceeded:        snapshot.Exceeded,
		EnforcementNote: snapshot.EnforcementNote,
		LastUsedAt:      row.LastUsedAt,
	}, nil
}

func (s *Service) loadCredentialRow(ctx context.Context, querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, credentialID uuid.UUID) (*credentialRow, error) {
	row := &credentialRow{}
	err := querier.QueryRow(
		ctx,
		`SELECT c.id, c.organization_id, c.domain_id, d.name, c.username, c.enabled, c.last_used_at,
		        c.quota_per_minute_limit, c.quota_daily_limit, c.quota_daily_used, c.quota_monthly_limit, c.quota_monthly_used
		 FROM smtp_credentials c
		 JOIN domains d ON d.id = c.domain_id
		 WHERE c.id = $1`,
		credentialID,
	).Scan(
		&row.ID,
		&row.OrganizationID,
		&row.DomainID,
		&row.DomainName,
		&row.Username,
		&row.Enabled,
		&row.LastUsedAt,
		&row.PerMinuteLimit,
		&row.DailyLimit,
		&row.DailyUsed,
		&row.MonthlyLimit,
		&row.MonthlyUsed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuotaCredentialMissing
		}
		return nil, err
	}

	return row, nil
}

func (s *Service) insertAuditLog(
	ctx context.Context,
	querier interface {
		Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	},
	actorUserID uuid.UUID,
	organizationID *uuid.UUID,
	action string,
	targetType string,
	targetID uuid.UUID,
	metadata map[string]any,
) error {
	payload, err := auditlog.MarshalAuditDetails(metadata)
	if err != nil {
		return err
	}

	_, err = querier.Exec(
		ctx,
		`INSERT INTO audit_logs (actor_user_id, organization_id, action, target_type, target_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		actorUserID,
		organizationID,
		action,
		targetType,
		targetID,
		payload,
	)
	return err
}
