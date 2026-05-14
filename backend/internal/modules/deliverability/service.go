package deliverability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	domainsmodule "zxmail/backend/internal/modules/domains"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrDeliverabilityForbidden = errors.New("deliverability forbidden")
	ErrDeliverabilityNotFound  = errors.New("deliverability not found")
	ErrDeliverabilityInvalid   = errors.New("invalid deliverability input")
)

type DomainVerifier interface {
	Verify(ctx context.Context, actor authmodule.AuthenticatedUser, id string) (*domainsmodule.VerificationResult, error)
}

type Service struct {
	db       *pgxpool.Pool
	log      *logger.Logger
	verifier DomainVerifier
}

type Overview struct {
	AcceptedCount      int     `json:"accepted_count"`
	DeliveredCount     int     `json:"delivered_count"`
	BouncedCount       int     `json:"bounced_count"`
	DeferredCount      int     `json:"deferred_count"`
	RejectedCount      int     `json:"rejected_count"`
	BounceRate         float64 `json:"bounce_rate"`
	DeferredRate       float64 `json:"deferred_rate"`
	RejectedRate       float64 `json:"rejected_rate"`
	DeliveredRate      float64 `json:"delivered_rate"`
	OpenAlerts         int     `json:"open_alerts"`
	AverageHealthScore int     `json:"average_health_score"`
}

type DomainHealth struct {
	DomainID        uuid.UUID `json:"domain_id"`
	DomainName      string    `json:"domain_name"`
	SPFFound        bool      `json:"spf_found"`
	DKIMFound       bool      `json:"dkim_found"`
	DMARCFound      bool      `json:"dmarc_found"`
	MXNoteFound     bool      `json:"mx_note_found"`
	RDNSStatus      string    `json:"rdns_status"`
	BounceRate      float64   `json:"bounce_rate"`
	DeferredRate    float64   `json:"deferred_rate"`
	RejectedRate    float64   `json:"rejected_rate"`
	QuotaLimited    bool      `json:"quota_limited"`
	HealthScore     int       `json:"health_score"`
	CheckedAt       time.Time `json:"checked_at"`
	LastVerifiedAt  *time.Time `json:"last_verified_at,omitempty"`
}

type Alert struct {
	ID         uuid.UUID  `json:"id"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	AlertType  string     `json:"alert_type"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger, verifier DomainVerifier) *Service {
	return &Service{db: db, log: log, verifier: verifier}
}

func (s *Service) OverviewForActor(ctx context.Context, actor authmodule.AuthenticatedUser) (*Overview, error) {
	orgID, err := s.resolveOrganizationID(ctx, actor)
	admin := actor.Role == authmodule.RoleAdmin
	if err != nil && !admin {
		return nil, err
	}
	return s.buildOverview(ctx, orgID, admin)
}

func (s *Service) DomainHealthForActor(ctx context.Context, actor authmodule.AuthenticatedUser, domainID string) (*DomainHealth, error) {
	parsedID, err := uuid.Parse(domainID)
	if err != nil {
		return nil, ErrDeliverabilityInvalid
	}
	return s.loadDomainHealth(ctx, actor, parsedID)
}

func (s *Service) AdminDeliverabilityOverview(ctx context.Context, actor authmodule.AuthenticatedUser) (*Overview, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrDeliverabilityForbidden
	}
	return s.buildOverview(ctx, uuid.Nil, true)
}

func (s *Service) ListAllDomainHealth(ctx context.Context, actor authmodule.AuthenticatedUser) ([]DomainHealth, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrDeliverabilityForbidden
	}
	rows, err := s.db.Query(
		ctx,
		`SELECT d.id, d.name, COALESCE(h.spf_found, FALSE), COALESCE(h.dkim_found, FALSE), COALESCE(h.dmarc_found, FALSE), COALESCE(h.mx_note_found, FALSE),
		        COALESCE(h.rdns_status, 'manual'), COALESCE(h.bounce_rate, 0), COALESCE(h.deferred_rate, 0), COALESCE(h.rejection_rate, 0), COALESCE(h.quota_limited, FALSE),
		        COALESCE(h.score, 0), COALESCE(h.checked_at, d.created_at), d.verified_at
		 FROM domains d
		 LEFT JOIN LATERAL (
		     SELECT spf_found, dkim_found, dmarc_found, mx_note_found, rdns_status, bounce_rate, deferred_rate, rejection_rate, quota_limited, score, checked_at
		     FROM domain_health_checks
		     WHERE domain_id = d.id
		     ORDER BY checked_at DESC
		     LIMIT 1
		 ) h ON TRUE
		 ORDER BY d.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DomainHealth
	for rows.Next() {
		var item DomainHealth
		if err := rows.Scan(
			&item.DomainID,
			&item.DomainName,
			&item.SPFFound,
			&item.DKIMFound,
			&item.DMARCFound,
			&item.MXNoteFound,
			&item.RDNSStatus,
			&item.BounceRate,
			&item.DeferredRate,
			&item.RejectedRate,
			&item.QuotaLimited,
			&item.HealthScore,
			&item.CheckedAt,
			&item.LastVerifiedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListAlerts(ctx context.Context, actor authmodule.AuthenticatedUser) ([]Alert, error) {
	orgID, err := s.resolveOrganizationID(ctx, actor)
	admin := actor.Role == authmodule.RoleAdmin
	if err != nil && !admin {
		return nil, err
	}

	query := `SELECT id, severity, status, alert_type, title, message, created_at, resolved_at FROM system_alerts`
	var args []any
	if !admin {
		query += ` WHERE organization_id = $1`
		args = append(args, orgID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(&alert.ID, &alert.Severity, &alert.Status, &alert.AlertType, &alert.Title, &alert.Message, &alert.CreatedAt, &alert.ResolvedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func (s *Service) ResolveAlert(ctx context.Context, actor authmodule.AuthenticatedUser, alertID string) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrDeliverabilityForbidden
	}
	parsedID, err := uuid.Parse(alertID)
	if err != nil {
		return ErrDeliverabilityInvalid
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(ctx, `UPDATE system_alerts SET status = 'resolved', resolved_at = $2 WHERE id = $1`, parsedID, now); err != nil {
		return err
	}
	actorID := actor.ID
	return auditlog.Insert(ctx, s.db, &actorID, nil, "deliverability.alert.resolve", "system_alert", &parsedID, map[string]any{
		"resolved_at": now,
	})
}

func (s *Service) RecheckDomain(ctx context.Context, actor authmodule.AuthenticatedUser, domainID string) (*DomainHealth, error) {
	if s.verifier == nil {
		return nil, ErrDeliverabilityInvalid
	}
	if _, err := s.verifier.Verify(ctx, actor, domainID); err != nil {
		return nil, err
	}
	return s.DomainHealthForActor(ctx, actor, domainID)
}

func (s *Service) RecheckAllDomains(ctx context.Context, actor authmodule.AuthenticatedUser) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrDeliverabilityForbidden
	}
	rows, err := s.db.Query(ctx, `SELECT id FROM domains ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domainID uuid.UUID
		if err := rows.Scan(&domainID); err != nil {
			return err
		}
		if _, err := s.RecheckDomain(ctx, actor, domainID.String()); err != nil {
			s.log.Error("domain recheck failed for %s: %v", domainID, err)
		}
	}

	actorID := actor.ID
	return auditlog.Insert(ctx, s.db, &actorID, nil, "deliverability.domain.recheck_all", "domain", nil, map[string]any{
		"scope": "all_domains",
	})
}

func (s *Service) GenerateSnapshots(ctx context.Context, now time.Time) error {
	windowStart := now.Add(-24 * time.Hour)
	rows, err := s.db.Query(ctx, `SELECT id, organization_id FROM domains ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domainID uuid.UUID
		var organizationID uuid.UUID
		if err := rows.Scan(&domainID, &organizationID); err != nil {
			return err
		}
		if err := s.generateDomainSnapshot(ctx, organizationID, domainID, windowStart, now); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) GenerateAlerts(ctx context.Context, now time.Time) error {
	rows, err := s.db.Query(
		ctx,
		`SELECT domain_id, organization_id, bounce_rate, rejection_rate, deferred_rate, score
		 FROM (
		     SELECT DISTINCT ON (domain_id) domain_id, organization_id, bounce_rate, rejection_rate, deferred_rate, score
		     FROM deliverability_snapshots
		     WHERE domain_id IS NOT NULL
		     ORDER BY domain_id, created_at DESC
		 ) latest`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domainID uuid.UUID
		var organizationID uuid.UUID
		var bounceRate float64
		var rejectionRate float64
		var deferredRate float64
		var score int
		if err := rows.Scan(&domainID, &organizationID, &bounceRate, &rejectionRate, &deferredRate, &score); err != nil {
			return err
		}
		if bounceRate >= 0.05 {
			if err := s.createAlertIfMissing(ctx, organizationID, &domainID, nil, "warning", "high_bounce_rate", "High bounce rate", fmt.Sprintf("Bounce rate reached %.2f%% in the latest snapshot.", bounceRate*100)); err != nil {
				return err
			}
		}
		if rejectionRate >= 0.05 || deferredRate >= 0.10 {
			if err := s.createAlertIfMissing(ctx, organizationID, &domainID, nil, "warning", "delivery_degradation", "Rejected or deferred rate is elevated", "Rejected/deferred activity is above the recommended threshold."); err != nil {
				return err
			}
		}
		if score < 60 {
			if err := s.createAlertIfMissing(ctx, organizationID, &domainID, nil, "critical", "domain_health_low", "Domain health score is low", "SPF, DKIM, DMARC, bounce rate, or quota state needs review."); err != nil {
				return err
			}
		}
	}

	return rows.Err()
}

func (s *Service) generateDomainSnapshot(ctx context.Context, organizationID uuid.UUID, domainID uuid.UUID, from time.Time, to time.Time) error {
	var domainName string
	var verifiedAt *time.Time
	if err := s.db.QueryRow(ctx, `SELECT name, verified_at FROM domains WHERE id = $1`, domainID).Scan(&domainName, &verifiedAt); err != nil {
		return err
	}

	counts, err := s.loadCounts(ctx, organizationID, &domainID, from, to)
	if err != nil {
		return err
	}
	score, quotaLimited, checks, err := s.computeHealthState(ctx, domainID, counts)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		ctx,
		`INSERT INTO deliverability_snapshots (organization_id, domain_id, period_start, period_end, accepted_count, delivered_count, bounced_count, deferred_count, rejected_count, bounce_rate, rejection_rate, deferred_rate, delivered_rate, score)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		organizationID,
		domainID,
		from,
		to,
		counts.Accepted,
		counts.Delivered,
		counts.Bounced,
		counts.Deferred,
		counts.Rejected,
		rate(counts.Bounced, counts.Accepted),
		rate(counts.Rejected, counts.Accepted),
		rate(counts.Deferred, counts.Accepted),
		rate(counts.Delivered, counts.Accepted),
		score,
	)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		ctx,
		`INSERT INTO domain_health_checks (domain_id, spf_found, dkim_found, dmarc_found, mx_note_found, rdns_status, bounce_rate, rejection_rate, deferred_rate, quota_limited, score, metadata, checked_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'{}'::jsonb,$12)`,
		domainID,
		checks.spf,
		checks.dkim,
		checks.dmarc,
		true,
		"manual",
		rate(counts.Bounced, counts.Accepted),
		rate(counts.Rejected, counts.Accepted),
		rate(counts.Deferred, counts.Accepted),
		quotaLimited,
		score,
		to,
	)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `UPDATE domains SET current_health_score = $2, last_rechecked_at = $3 WHERE id = $1`, domainID, score, to)
	return err
}

func (s *Service) buildOverview(ctx context.Context, organizationID uuid.UUID, admin bool) (*Overview, error) {
	counts, err := s.loadCounts(ctx, organizationID, nil, time.Now().UTC().Add(-30*24*time.Hour), time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var averageScore float64
	var openAlerts int
	if admin {
		if err := s.db.QueryRow(
			ctx,
			`SELECT COALESCE(AVG(score), 0)
			 FROM (
			     SELECT DISTINCT ON (domain_id) score
			     FROM deliverability_snapshots
			     WHERE domain_id IS NOT NULL
			     ORDER BY domain_id, created_at DESC
			 ) latest`,
		).Scan(&averageScore); err != nil {
			return nil, err
		}
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM system_alerts WHERE status = 'open'`).Scan(&openAlerts); err != nil {
			return nil, err
		}
	} else {
		if err := s.db.QueryRow(
			ctx,
			`SELECT COALESCE(AVG(score), 0)
			 FROM (
			     SELECT DISTINCT ON (domain_id) score
			     FROM deliverability_snapshots
			     WHERE organization_id = $1
			       AND domain_id IS NOT NULL
			     ORDER BY domain_id, created_at DESC
			 ) latest`,
			organizationID,
		).Scan(&averageScore); err != nil {
			return nil, err
		}
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM system_alerts WHERE status = 'open' AND organization_id = $1`, organizationID).Scan(&openAlerts); err != nil {
			return nil, err
		}
	}

	return &Overview{
		AcceptedCount:      counts.Accepted,
		DeliveredCount:     counts.Delivered,
		BouncedCount:       counts.Bounced,
		DeferredCount:      counts.Deferred,
		RejectedCount:      counts.Rejected,
		BounceRate:         rate(counts.Bounced, counts.Accepted),
		DeferredRate:       rate(counts.Deferred, counts.Accepted),
		RejectedRate:       rate(counts.Rejected, counts.Accepted),
		DeliveredRate:      rate(counts.Delivered, counts.Accepted),
		OpenAlerts:         openAlerts,
		AverageHealthScore: int(math.Round(averageScore)),
	}, nil
}

type aggregateCounts struct {
	Accepted  int
	Delivered int
	Bounced   int
	Deferred  int
	Rejected  int
}

type dnsState struct {
	spf   bool
	dkim  bool
	dmarc bool
}

func (s *Service) loadCounts(ctx context.Context, organizationID uuid.UUID, domainID *uuid.UUID, from time.Time, to time.Time) (*aggregateCounts, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN metric = 'accepted' THEN quantity ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN metric = 'delivered' THEN quantity ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN metric = 'bounced' THEN quantity ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN metric = 'deferred' THEN quantity ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN metric = 'rejected' THEN quantity ELSE 0 END), 0)
		FROM usage_records
		WHERE recorded_at >= $1 AND recorded_at <= $2`
	args := []any{from, to}
	if organizationID != uuid.Nil {
		query += fmt.Sprintf(" AND organization_id = $%d", len(args)+1)
		args = append(args, organizationID)
	}
	if domainID != nil {
		query += fmt.Sprintf(" AND domain_id = $%d", len(args)+1)
		args = append(args, *domainID)
	}

	counts := &aggregateCounts{}
	if err := s.db.QueryRow(ctx, query, args...).Scan(&counts.Accepted, &counts.Delivered, &counts.Bounced, &counts.Deferred, &counts.Rejected); err != nil {
		return nil, err
	}
	return counts, nil
}

func (s *Service) computeHealthState(ctx context.Context, domainID uuid.UUID, counts *aggregateCounts) (int, bool, dnsState, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT record_type, name, found
		 FROM dns_checks
		 WHERE domain_id = $1
		 ORDER BY checked_at DESC`,
		domainID,
	)
	if err != nil {
		return 0, false, dnsState{}, err
	}
	defer rows.Close()

	state := dnsState{}
	for rows.Next() {
		var recordType string
		var name string
		var found bool
		if err := rows.Scan(&recordType, &name, &found); err != nil {
			return 0, false, dnsState{}, err
		}
		if name == "" {
			continue
		}
		switch {
		case recordType == "TXT" && state.spf == false && found && len(name) > 0:
			if state.spf == false && found && !state.dkim && !state.dmarc {
				state.spf = true
			}
		}
		if found && len(name) > 0 {
			if len(name) >= 11 && name[:11] == "_dmarc." {
				state.dmarc = true
			}
			if len(name) >= 12 && nameContains(name, "._domainkey.") {
				state.dkim = true
			}
			if !nameContains(name, "_dmarc.") && !nameContains(name, "._domainkey.") {
				state.spf = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, false, dnsState{}, err
	}

	quotaLimited := false
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM smtp_credentials WHERE domain_id = $1 AND (manually_limited = TRUE OR enabled = FALSE))`, domainID).Scan(&quotaLimited); err != nil {
		return 0, false, dnsState{}, err
	}

	score := 100
	if !state.spf {
		score -= 20
	}
	if !state.dkim {
		score -= 20
	}
	if !state.dmarc {
		score -= 20
	}
	if rate(counts.Bounced, counts.Accepted) >= 0.05 {
		score -= 15
	}
	if rate(counts.Rejected, counts.Accepted) >= 0.05 {
		score -= 10
	}
	if rate(counts.Deferred, counts.Accepted) >= 0.10 {
		score -= 10
	}
	if quotaLimited {
		score -= 10
	}
	if score < 0 {
		score = 0
	}

	return score, quotaLimited, state, nil
}

func (s *Service) loadDomainHealth(ctx context.Context, actor authmodule.AuthenticatedUser, domainID uuid.UUID) (*DomainHealth, error) {
	query := `SELECT d.id, d.name, h.spf_found, h.dkim_found, h.dmarc_found, h.mx_note_found, h.rdns_status, h.bounce_rate, h.deferred_rate, h.rejection_rate, h.quota_limited, h.score, h.checked_at, d.verified_at
	          FROM domains d
	          LEFT JOIN LATERAL (
	              SELECT spf_found, dkim_found, dmarc_found, mx_note_found, rdns_status, bounce_rate, deferred_rate, rejection_rate, quota_limited, score, checked_at
	              FROM domain_health_checks
	              WHERE domain_id = d.id
	              ORDER BY checked_at DESC
	              LIMIT 1
	          ) h ON TRUE
	          WHERE d.id = $1`
	args := []any{domainID}
	if actor.Role != authmodule.RoleAdmin {
		orgID, err := s.resolveOrganizationID(ctx, actor)
		if err != nil {
			return nil, err
		}
		query += ` AND d.organization_id = $2`
		args = append(args, orgID)
	}

	var health DomainHealth
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&health.DomainID,
		&health.DomainName,
		&health.SPFFound,
		&health.DKIMFound,
		&health.DMARCFound,
		&health.MXNoteFound,
		&health.RDNSStatus,
		&health.BounceRate,
		&health.DeferredRate,
		&health.RejectedRate,
		&health.QuotaLimited,
		&health.HealthScore,
		&health.CheckedAt,
		&health.LastVerifiedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeliverabilityNotFound
		}
		return nil, err
	}
	return &health, nil
}

func (s *Service) resolveOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}
	var organizationID uuid.UUID
	if err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, actor.ID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrDeliverabilityNotFound
		}
		return uuid.Nil, err
	}
	return organizationID, nil
}

func (s *Service) createAlertIfMissing(ctx context.Context, organizationID uuid.UUID, domainID *uuid.UUID, credentialID *uuid.UUID, severity, alertType, title, message string) error {
	var exists bool
	if err := s.db.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM system_alerts
			WHERE organization_id = $1
			  AND alert_type = $2
			  AND status = 'open'
			  AND (domain_id IS NOT DISTINCT FROM $3)
			  AND (credential_id IS NOT DISTINCT FROM $4)
		)`,
		organizationID,
		alertType,
		domainID,
		credentialID,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := s.db.Exec(
		ctx,
		`INSERT INTO system_alerts (organization_id, domain_id, credential_id, severity, status, alert_type, title, message)
		 VALUES ($1,$2,$3,$4,'open',$5,$6,$7)`,
		organizationID,
		domainID,
		credentialID,
		severity,
		alertType,
		title,
		message,
	)
	return err
}

func rate(value int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total)
}

func nameContains(value string, needle string) bool {
	return strings.Contains(value, needle)
}
