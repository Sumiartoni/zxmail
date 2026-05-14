package usage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrUsageForbidden    = errors.New("usage forbidden")
	ErrUsageInvalidInput = errors.New("invalid usage input")
	ErrUsageNotFound     = errors.New("usage target not found")
)

type Service struct {
	db    *pgxpool.Pool
	redis *redis.Client
	log   *logger.Logger
}

type Overview struct {
	OrganizationID    uuid.UUID `json:"organization_id"`
	AcceptedToday     int       `json:"accepted_today"`
	AcceptedMonth     int       `json:"accepted_month"`
	DeliveredMonth    int       `json:"delivered_month"`
	BouncedMonth      int       `json:"bounced_month"`
	DeferredMonth     int       `json:"deferred_month"`
	RejectedMonth     int       `json:"rejected_month"`
	EffectiveDaily    *int      `json:"effective_daily_quota,omitempty"`
	EffectiveMonthly  *int      `json:"effective_monthly_quota,omitempty"`
	EffectivePerMinute *int     `json:"effective_per_minute_quota,omitempty"`
	OverageCount      int       `json:"overage_count"`
	Status            string    `json:"status"`
	LastUpdatedAt     time.Time `json:"last_updated_at"`
}

type UpdateOrganizationQuotaInput struct {
	DailyQuota     *int   `json:"daily_quota,omitempty"`
	MonthlyQuota   *int   `json:"monthly_quota,omitempty"`
	PerMinuteQuota *int   `json:"per_minute_quota,omitempty"`
	Reason         string `json:"reason"`
}

func NewService(db *pgxpool.Pool, redisClient *redis.Client, log *logger.Logger) *Service {
	return &Service{db: db, redis: redisClient, log: log}
}

func (s *Service) RecordSendLogEvent(ctx context.Context, sendLogID uuid.UUID, organizationID uuid.UUID, credentialID *uuid.UUID, domainID *uuid.UUID, metric string, occurredAt time.Time) error {
	if !isSupportedMetric(metric) {
		return ErrUsageInvalidInput
	}

	periodDay := occurredAt.UTC().Format("2006-01-02")
	periodMonth := time.Date(occurredAt.UTC().Year(), occurredAt.UTC().Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	_, err := s.db.Exec(
		ctx,
		`INSERT INTO usage_records (organization_id, credential_id, domain_id, send_log_id, metric, quantity, recorded_at, period_day, period_month)
		 VALUES ($1,$2,$3,$4,$5,1,$6,$7,$8)
		 ON CONFLICT (send_log_id, metric) WHERE send_log_id IS NOT NULL DO NOTHING`,
		organizationID,
		credentialID,
		domainID,
		sendLogID,
		metric,
		occurredAt.UTC(),
		periodDay,
		periodMonth,
	)
	if err != nil {
		return err
	}

	if s.redis != nil {
		key := fmt.Sprintf("usage:org:%s:%s:%s", organizationID, metric, occurredAt.UTC().Format("200601021504"))
		pipe := s.redis.TxPipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 2*time.Minute)
		if _, err := pipe.Exec(ctx); err != nil {
			s.log.Error("usage redis advisory counter failed for org %s metric=%s: %v", organizationID, metric, err)
		}
	}

	return nil
}

func (s *Service) GetUsage(ctx context.Context, actor authmodule.AuthenticatedUser) (*Overview, error) {
	orgID, err := s.resolveOrganizationID(ctx, actor)
	if err != nil {
		return nil, err
	}
	return s.buildOverview(ctx, orgID)
}

func (s *Service) GetOrganizationUsage(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) (*Overview, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrUsageForbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, ErrUsageInvalidInput
	}
	return s.buildOverview(ctx, orgID)
}

func (s *Service) UpdateOrganizationQuota(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, input UpdateOrganizationQuotaInput) (*Overview, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrUsageForbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, ErrUsageInvalidInput
	}
	for _, value := range []*int{input.DailyQuota, input.MonthlyQuota, input.PerMinuteQuota} {
		if value != nil && *value < 0 {
			return nil, ErrUsageInvalidInput
		}
	}
	if _, err := s.db.Exec(
		ctx,
		`UPDATE organizations
		 SET quota_daily_override = $2,
		     quota_monthly_override = $3,
		     quota_per_minute_override = $4,
		     updated_at = NOW()
		 WHERE id = $1`,
		orgID,
		input.DailyQuota,
		input.MonthlyQuota,
		input.PerMinuteQuota,
	); err != nil {
		return nil, err
	}

	actorID := actor.ID
	if err := auditlog.Insert(ctx, s.db, &actorID, &orgID, "usage.organization.quota.update", "organization", &orgID, map[string]any{
		"reason":             input.Reason,
		"daily_quota":        input.DailyQuota,
		"monthly_quota":      input.MonthlyQuota,
		"per_minute_quota":   input.PerMinuteQuota,
	}); err != nil {
		return nil, err
	}

	return s.buildOverview(ctx, orgID)
}

func (s *Service) ResetOrganizationUsage(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrUsageForbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return ErrUsageInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE smtp_credentials SET quota_daily_used = 0, quota_monthly_used = 0 WHERE organization_id = $1`, orgID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO quota_events (organization_id, event_type, reason, metadata) VALUES ($1, 'reset_usage', 'manual admin reset', '{}'::jsonb)`, orgID); err != nil {
		return err
	}
	actorID := actor.ID
	if err := auditlog.Insert(ctx, tx, &actorID, &orgID, "usage.organization.reset", "organization", &orgID, map[string]any{
		"scope": "daily_and_monthly",
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) SetCredentialLimited(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, limited bool, reason string) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrUsageForbidden
	}
	parsedID, err := uuid.Parse(credentialID)
	if err != nil {
		return ErrUsageInvalidInput
	}
	var organizationID uuid.UUID
	if err := s.db.QueryRow(ctx, `SELECT organization_id FROM smtp_credentials WHERE id = $1`, parsedID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUsageNotFound
		}
		return err
	}
	if _, err := s.db.Exec(
		ctx,
		`UPDATE smtp_credentials
		 SET manually_limited = $2,
		     manual_limit_reason = $3,
		     manual_limit_updated_at = NOW()
		 WHERE id = $1`,
		parsedID,
		limited,
		nullableString(reason),
	); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO quota_events (organization_id, credential_id, event_type, reason, metadata) VALUES ($1, $2, $3, $4, '{}'::jsonb)`, organizationID, parsedID, eventTypeForLimit(limited), nullableString(reason)); err != nil {
		return err
	}
	actorID := actor.ID
	action := "usage.credential.limit"
	if !limited {
		action = "usage.credential.unlimit"
	}
	return auditlog.Insert(ctx, s.db, &actorID, &organizationID, action, "smtp_credential", &parsedID, map[string]any{
		"reason":  reason,
		"limited": limited,
	})
}

func (s *Service) RunDailyReset(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `UPDATE smtp_credentials SET quota_daily_used = 0`)
	return err
}

func (s *Service) RunMonthlyReset(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `UPDATE smtp_credentials SET quota_monthly_used = 0`)
	return err
}

func (s *Service) buildOverview(ctx context.Context, organizationID uuid.UUID) (*Overview, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	overview := &Overview{OrganizationID: organizationID, LastUpdatedAt: now}
	err := s.db.QueryRow(
		ctx,
		`SELECT
		    COALESCE((SELECT SUM(quota_daily_used) FROM smtp_credentials WHERE organization_id = $1), 0),
		    COALESCE((SELECT SUM(quota_monthly_used) FROM smtp_credentials WHERE organization_id = $1), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE organization_id = $1 AND metric = 'delivered' AND period_month = $2::date), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE organization_id = $1 AND metric = 'bounced' AND period_month = $2::date), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE organization_id = $1 AND metric = 'deferred' AND period_month = $2::date), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE organization_id = $1 AND metric = 'rejected' AND period_month = $2::date), 0),
		    o.quota_daily_override,
		    o.quota_monthly_override,
		    o.quota_per_minute_override,
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE organization_id = $1 AND metric = 'overage' AND period_month = $2::date), 0)
		 FROM organizations o
		 WHERE o.id = $1`,
		organizationID,
		monthStart,
	).Scan(
		&overview.AcceptedToday,
		&overview.AcceptedMonth,
		&overview.DeliveredMonth,
		&overview.BouncedMonth,
		&overview.DeferredMonth,
		&overview.RejectedMonth,
		&overview.EffectiveDaily,
		&overview.EffectiveMonthly,
		&overview.EffectivePerMinute,
		&overview.OverageCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUsageNotFound
		}
		return nil, err
	}

	if overview.EffectiveDaily == nil || overview.EffectiveMonthly == nil || overview.EffectivePerMinute == nil {
		planDaily, planMonthly, planPerMinute, err := s.loadSubscriptionQuotas(ctx, organizationID)
		if err == nil {
			if overview.EffectiveDaily == nil {
				overview.EffectiveDaily = planDaily
			}
			if overview.EffectiveMonthly == nil {
				overview.EffectiveMonthly = planMonthly
			}
			if overview.EffectivePerMinute == nil {
				overview.EffectivePerMinute = planPerMinute
			}
		}
	}

	overview.Status = "within_quota"
	if overview.EffectiveDaily != nil && overview.AcceptedToday >= *overview.EffectiveDaily {
		overview.Status = "daily_limited"
	} else if overview.EffectiveMonthly != nil && overview.AcceptedMonth >= *overview.EffectiveMonthly {
		overview.Status = "monthly_limited"
	}
	return overview, nil
}

func (s *Service) loadSubscriptionQuotas(ctx context.Context, organizationID uuid.UUID) (*int, *int, *int, error) {
	var daily *int
	var monthly *int
	var perMinute *int
	err := s.db.QueryRow(
		ctx,
		`SELECT COALESCE(s.quota_daily_override, p.daily_quota),
		        COALESCE(s.quota_monthly_override, p.monthly_quota),
		        COALESCE(s.quota_per_minute_override, p.per_minute_quota)
		 FROM subscriptions s
		 JOIN plans p ON p.id = s.plan_id
		 WHERE s.organization_id = $1
		 ORDER BY s.created_at DESC
		 LIMIT 1`,
		organizationID,
	).Scan(&daily, &monthly, &perMinute)
	if err != nil {
		return nil, nil, nil, err
	}
	return daily, monthly, perMinute, nil
}

func (s *Service) resolveOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}
	var organizationID uuid.UUID
	if err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, actor.ID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrUsageNotFound
		}
		return uuid.Nil, err
	}
	return organizationID, nil
}

func isSupportedMetric(metric string) bool {
	switch metric {
	case "accepted", "delivered", "bounced", "deferred", "rejected", "overage":
		return true
	default:
		return false
	}
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func eventTypeForLimit(limited bool) string {
	if limited {
		return "credential_limited"
	}
	return "credential_unlimited"
}
