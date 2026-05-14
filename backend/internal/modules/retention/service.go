package retention

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrRetentionForbidden = errors.New("retention forbidden")
	ErrRetentionInvalid   = errors.New("invalid retention input")
)

type Service struct {
	db                   *pgxpool.Pool
	log                  *logger.Logger
	defaultRetentionDays int
}

type OrganizationPolicy struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	RetentionDays  int       `json:"retention_days"`
}

type CleanupResult struct {
	DryRun        bool `json:"dry_run"`
	MatchedLogs   int  `json:"matched_logs"`
	DeletedLogs   int  `json:"deleted_logs"`
	Organizations int  `json:"organizations"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger, defaultRetentionDays int) *Service {
	return &Service{db: db, log: log, defaultRetentionDays: defaultRetentionDays}
}

func (s *Service) GetPolicies(ctx context.Context, actor authmodule.AuthenticatedUser) ([]OrganizationPolicy, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrRetentionForbidden
	}
	rows, err := s.db.Query(ctx, `SELECT id, name, retention_days FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []OrganizationPolicy
	for rows.Next() {
		var policy OrganizationPolicy
		if err := rows.Scan(&policy.OrganizationID, &policy.Name, &policy.RetentionDays); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func (s *Service) UpdatePolicy(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, retentionDays int) error {
	if actor.Role != authmodule.RoleAdmin {
		return ErrRetentionForbidden
	}
	if retentionDays <= 0 {
		return ErrRetentionInvalid
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return ErrRetentionInvalid
	}
	if _, err := s.db.Exec(ctx, `UPDATE organizations SET retention_days = $2, updated_at = NOW() WHERE id = $1`, orgID, retentionDays); err != nil {
		return err
	}
	actorID := actor.ID
	return auditlog.Insert(ctx, s.db, &actorID, &orgID, "retention.organization.update", "organization", &orgID, map[string]any{
		"retention_days": retentionDays,
	})
}

func (s *Service) RunCleanup(ctx context.Context, actor *authmodule.AuthenticatedUser, dryRun bool) (*CleanupResult, error) {
	type orgPolicy struct {
		ID            uuid.UUID
		RetentionDays int
	}
	rows, err := s.db.Query(ctx, `SELECT id, retention_days FROM organizations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []orgPolicy
	for rows.Next() {
		var policy orgPolicy
		if err := rows.Scan(&policy.ID, &policy.RetentionDays); err != nil {
			return nil, err
		}
		if policy.RetentionDays <= 0 {
			policy.RetentionDays = s.defaultRetentionDays
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &CleanupResult{DryRun: dryRun, Organizations: len(policies)}
	for _, policy := range policies {
		cutoff := time.Now().UTC().Add(-time.Duration(policy.RetentionDays) * 24 * time.Hour)
		var matched int
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM send_logs WHERE organization_id = $1 AND created_at < $2`, policy.ID, cutoff).Scan(&matched); err != nil {
			return nil, err
		}
		result.MatchedLogs += matched
		if !dryRun && matched > 0 {
			tag, err := s.db.Exec(ctx, `DELETE FROM send_logs WHERE organization_id = $1 AND created_at < $2`, policy.ID, cutoff)
			if err != nil {
				return nil, err
			}
			result.DeletedLogs += int(tag.RowsAffected())
		}
	}

	if actor != nil {
		actorID := actor.ID
		if err := auditlog.Insert(ctx, s.db, &actorID, nil, "retention.cleanup.run", "send_logs", nil, map[string]any{
			"dry_run":      dryRun,
			"matched_logs": result.MatchedLogs,
			"deleted_logs": result.DeletedLogs,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *Service) RunAutomaticCleanup(ctx context.Context) error {
	_, err := s.RunCleanup(ctx, nil, false)
	return err
}
