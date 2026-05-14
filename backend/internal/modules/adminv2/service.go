package adminv2

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	billingmodule "zxmail/backend/internal/modules/billing"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
	usagemodule "zxmail/backend/internal/modules/usage"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrAdminV2Forbidden = errors.New("admin v2 forbidden")
	ErrAdminV2Invalid   = errors.New("invalid admin v2 input")
	ErrAdminV2NotFound  = errors.New("admin v2 not found")
)

type CredentialRotator interface {
	Rotate(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*credentialsmodule.CredentialSecretResponse, error)
}

type Service struct {
	db         *pgxpool.Pool
	log        *logger.Logger
	usage      *usagemodule.Service
	billing    *billingmodule.Service
	rotator    CredentialRotator
	smtpHost   string
	smtpTLS    string
	smtpSTART  string
}

type Overview struct {
	TotalEmailSent   int `json:"total_email_sent"`
	Delivered        int `json:"delivered"`
	Bounced          int `json:"bounced"`
	Rejected         int `json:"rejected"`
	ActiveCustomers  int `json:"active_customers"`
	ActiveDomains    int `json:"active_domains"`
	OpenAlerts       int `json:"open_alerts"`
	PastDuePayments  int `json:"past_due_payments"`
}

type OrganizationDetail struct {
	ID                     uuid.UUID  `json:"id"`
	Name                   string     `json:"name"`
	Suspended              bool       `json:"suspended"`
	SuspendedReason        string     `json:"suspended_reason,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	RetentionDays          int        `json:"retention_days"`
	CurrentSubscription    string     `json:"current_subscription_status"`
	PaymentStatus          string     `json:"payment_status"`
	VerifiedDomains        int        `json:"verified_domains"`
	EnabledCredentials     int        `json:"enabled_credentials"`
	LatestSendActivityAt   *time.Time `json:"latest_send_activity_at,omitempty"`
	RiskScore              int        `json:"risk_score"`
	BounceRate             float64    `json:"bounce_rate"`
}

type RiskRecord struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	RiskScore      int       `json:"risk_score"`
	BounceRate     float64   `json:"bounce_rate"`
	Suspended      bool      `json:"suspended"`
	PaymentStatus  string    `json:"payment_status"`
}

type AuditLogRecord struct {
	ID             uuid.UUID      `json:"id"`
	ActorUserID    *uuid.UUID     `json:"actor_user_id,omitempty"`
	ActorEmail     string         `json:"actor_email,omitempty"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	Action         string         `json:"action"`
	TargetType     string         `json:"target_type"`
	TargetID       *uuid.UUID     `json:"target_id,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

func NewService(
	db *pgxpool.Pool,
	log *logger.Logger,
	usageService *usagemodule.Service,
	billingService *billingmodule.Service,
	rotator CredentialRotator,
	smtpHost, smtpPortSTARTTLS, smtpPortTLS string,
) *Service {
	return &Service{
		db:        db,
		log:       log,
		usage:     usageService,
		billing:   billingService,
		rotator:   rotator,
		smtpHost:  smtpHost,
		smtpSTART: smtpPortSTARTTLS,
		smtpTLS:   smtpPortTLS,
	}
}

func (s *Service) OverviewMetrics(ctx context.Context, actor authmodule.AuthenticatedUser) (*Overview, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrAdminV2Forbidden
	}
	overview := &Overview{}
	err := s.db.QueryRow(
		ctx,
		`SELECT
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE metric = 'accepted'), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE metric = 'delivered'), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE metric = 'bounced'), 0),
		    COALESCE((SELECT SUM(quantity) FROM usage_records WHERE metric = 'rejected'), 0),
		    COALESCE((SELECT COUNT(*) FROM organizations WHERE suspended = FALSE), 0),
		    COALESCE((SELECT COUNT(*) FROM domains WHERE verified = TRUE), 0),
		    COALESCE((SELECT COUNT(*) FROM system_alerts WHERE status = 'open'), 0),
		    COALESCE((SELECT COUNT(*) FROM invoices WHERE status = 'failed'), 0)`,
	).Scan(
		&overview.TotalEmailSent,
		&overview.Delivered,
		&overview.Bounced,
		&overview.Rejected,
		&overview.ActiveCustomers,
		&overview.ActiveDomains,
		&overview.OpenAlerts,
		&overview.PastDuePayments,
	)
	if err != nil {
		return nil, err
	}
	return overview, nil
}

func (s *Service) OrganizationDetailByID(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) (*OrganizationDetail, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrAdminV2Forbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, ErrAdminV2Invalid
	}
	detail := &OrganizationDetail{}
	err = s.db.QueryRow(
		ctx,
		`SELECT
		    o.id,
		    o.name,
		    o.suspended,
		    COALESCE(o.suspended_reason, ''),
		    o.created_at,
		    o.retention_days,
		    COALESCE((SELECT status FROM subscriptions WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 'missing'),
		    COALESCE((SELECT status FROM payments WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 'not_required'),
		    COALESCE((SELECT COUNT(*) FROM domains WHERE organization_id = o.id AND verified = TRUE), 0),
		    COALESCE((SELECT COUNT(*) FROM smtp_credentials WHERE organization_id = o.id AND enabled = TRUE), 0),
		    (SELECT MAX(created_at) FROM send_logs WHERE organization_id = o.id),
		    COALESCE((SELECT score FROM deliverability_snapshots WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 0),
		    COALESCE((SELECT bounce_rate FROM deliverability_snapshots WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 0)
		 FROM organizations o
		 WHERE o.id = $1`,
		orgID,
	).Scan(
		&detail.ID,
		&detail.Name,
		&detail.Suspended,
		&detail.SuspendedReason,
		&detail.CreatedAt,
		&detail.RetentionDays,
		&detail.CurrentSubscription,
		&detail.PaymentStatus,
		&detail.VerifiedDomains,
		&detail.EnabledCredentials,
		&detail.LatestSendActivityAt,
		&detail.RiskScore,
		&detail.BounceRate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminV2NotFound
		}
		return nil, err
	}
	return detail, nil
}

func (s *Service) SuspendOrganization(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, reason string) error {
	return s.setOrganizationSuspended(ctx, actor, organizationID, true, reason)
}

func (s *Service) UnsuspendOrganization(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error {
	return s.setOrganizationSuspended(ctx, actor, organizationID, false, "")
}

func (s *Service) DisableOrganizationCredentials(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrAdminV2Forbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return ErrAdminV2Invalid
	}
	if _, err := s.db.Exec(ctx, `UPDATE smtp_credentials SET enabled = FALSE WHERE organization_id = $1`, orgID); err != nil {
		return err
	}
	actorID := actor.ID
	return auditlog.Insert(ctx, s.db, &actorID, &orgID, "admin.organization.disable_credentials", "organization", &orgID, map[string]any{
		"organization_id": orgID.String(),
	})
}

func (s *Service) ForceRotateCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*credentialsmodule.CredentialSecretResponse, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrAdminV2Forbidden
	}
	return s.rotator.Rotate(ctx, actor, credentialID, s.smtpHost, s.smtpSTART, s.smtpTLS)
}

func (s *Service) OrganizationRisk(ctx context.Context, actor authmodule.AuthenticatedUser) ([]RiskRecord, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrAdminV2Forbidden
	}
	rows, err := s.db.Query(
		ctx,
		`SELECT
		    o.id,
		    o.name,
		    COALESCE((SELECT score FROM deliverability_snapshots WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 0),
		    COALESCE((SELECT bounce_rate FROM deliverability_snapshots WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 0),
		    o.suspended,
		    COALESCE((SELECT status FROM payments WHERE organization_id = o.id ORDER BY created_at DESC LIMIT 1), 'not_required')
		 FROM organizations o
		 ORDER BY o.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var risks []RiskRecord
	for rows.Next() {
		var risk RiskRecord
		if err := rows.Scan(&risk.OrganizationID, &risk.Name, &risk.RiskScore, &risk.BounceRate, &risk.Suspended, &risk.PaymentStatus); err != nil {
			return nil, err
		}
		risks = append(risks, risk)
	}
	return risks, rows.Err()
}

func (s *Service) ListAuditLogs(ctx context.Context, actor authmodule.AuthenticatedUser, limit int) ([]AuditLogRecord, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrAdminV2Forbidden
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := s.db.Query(
		ctx,
		`SELECT a.id, a.actor_user_id, COALESCE(u.email, ''), a.organization_id, a.action, a.target_type, a.target_id, a.metadata, a.created_at
		 FROM audit_logs a
		 LEFT JOIN users u ON u.id = a.actor_user_id
		 ORDER BY a.created_at DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AuditLogRecord
	for rows.Next() {
		var record AuditLogRecord
		if err := rows.Scan(
			&record.ID,
			&record.ActorUserID,
			&record.ActorEmail,
			&record.OrganizationID,
			&record.Action,
			&record.TargetType,
			&record.TargetID,
			&record.Metadata,
			&record.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (s *Service) setOrganizationSuspended(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, suspended bool, reason string) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrAdminV2Forbidden
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return ErrAdminV2Invalid
	}
	var suspendedAt any
	if suspended {
		suspendedAt = time.Now().UTC()
	} else {
		suspendedAt = nil
		reason = ""
	}

	if _, err := s.db.Exec(
		ctx,
		`UPDATE organizations
		 SET suspended = $2,
		     suspended_reason = $3,
		     suspended_at = $4,
		     updated_at = NOW()
		 WHERE id = $1`,
		orgID,
		suspended,
		nullableString(reason),
		suspendedAt,
	); err != nil {
		return err
	}

	action := "admin.organization.unsuspend"
	if suspended {
		action = "admin.organization.suspend"
	}
	actorID := actor.ID
	return auditlog.Insert(ctx, s.db, &actorID, &orgID, action, "organization", &orgID, map[string]any{
		"suspended": suspended,
		"reason":    reason,
	})
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
