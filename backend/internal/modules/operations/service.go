package operations

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/logger"
	"zxmail/backend/internal/postal"
)

var ErrOperationsForbidden = errors.New("operations forbidden")

type Service struct {
	db     *pgxpool.Pool
	redis  *redis.Client
	log    *logger.Logger
	postal *postal.Client
}

type SystemHealth struct {
	Postgres string            `json:"postgres"`
	Redis    string            `json:"redis"`
	Postal   string            `json:"postal"`
	Worker   string            `json:"worker"`
	Queue    string            `json:"queue"`
	Notes    map[string]string `json:"notes"`
}

type QueueHealth struct {
	Mode       string `json:"mode"`
	Pending    int64  `json:"pending"`
	InProgress int64  `json:"in_progress"`
	Note       string `json:"note"`
}

func NewService(db *pgxpool.Pool, redisClient *redis.Client, log *logger.Logger, postalClient *postal.Client) *Service {
	return &Service{db: db, redis: redisClient, log: log, postal: postalClient}
}

func (s *Service) AdminSystemHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*SystemHealth, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrOperationsForbidden
	}
	status := &SystemHealth{
		Postgres: "healthy",
		Redis:    "healthy",
		Postal:   "manual-check",
		Worker:   "degraded",
		Queue:    "manual",
		Notes:    map[string]string{},
	}

	if err := s.db.Ping(ctx); err != nil {
		status.Postgres = "degraded"
		status.Notes["postgres"] = err.Error()
	}
	if s.redis != nil {
		if err := s.redis.Ping(ctx).Err(); err != nil {
			status.Redis = "degraded"
			status.Notes["redis"] = err.Error()
		}
	}
	if s.postal != nil {
		result, err := s.postal.HealthCheck(ctx)
		if err != nil {
			status.Postal = "degraded"
			status.Notes["postal"] = err.Error()
		} else if !result.Reachable {
			status.Postal = "degraded"
			status.Notes["postal"] = result.Note
		} else {
			status.Postal = "ready"
			status.Notes["postal"] = result.Note
		}
	}

	var lastWorkerRun *time.Time
	if err := s.db.QueryRow(ctx, `SELECT MAX(started_at) FROM worker_job_runs`).Scan(&lastWorkerRun); err == nil && lastWorkerRun != nil {
		if time.Since(*lastWorkerRun) < 10*time.Minute {
			status.Worker = "healthy"
		}
	}
	status.Notes["queue"] = "Production v2 worker uses in-process scheduled jobs. Redis-backed queue depth is not yet exposed as a true SMTP job queue."

	return status, nil
}

func (s *Service) QueueHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*QueueHealth, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrOperationsForbidden
	}
	return &QueueHealth{
		Mode:       "scheduled_worker",
		Pending:    0,
		InProgress: 0,
		Note:       "No dedicated Redis queue has been provisioned yet. Scheduled worker jobs run in-process and report health via worker_job_runs.",
	}, nil
}

func (s *Service) PostalHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*postal.HealthCheckResult, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrOperationsForbidden
	}
	return s.postal.HealthCheck(ctx)
}
