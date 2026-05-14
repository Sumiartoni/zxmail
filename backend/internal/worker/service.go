package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	billingmodule "zxmail/backend/internal/modules/billing"
	deliverabilitymodule "zxmail/backend/internal/modules/deliverability"
	retentionmodule "zxmail/backend/internal/modules/retention"
	usagemodule "zxmail/backend/internal/modules/usage"
	"zxmail/backend/internal/platform/logger"
)

type Service struct {
	db             *pgxpool.Pool
	log            *logger.Logger
	billing        *billingmodule.Service
	usage          *usagemodule.Service
	deliverability *deliverabilitymodule.Service
	retention      *retentionmodule.Service
	port           string
	interval       time.Duration
}

func New(
	db *pgxpool.Pool,
	log *logger.Logger,
	billing *billingmodule.Service,
	usage *usagemodule.Service,
	deliverability *deliverabilitymodule.Service,
	retention *retentionmodule.Service,
	port string,
	interval time.Duration,
) *Service {
	return &Service{
		db:             db,
		log:            log,
		billing:        billing,
		usage:          usage,
		deliverability: deliverability,
		retention:      retention,
		port:           port,
		interval:       interval,
	}
}

func (s *Service) Run(ctx context.Context) error {
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		var latestRun *time.Time
		_ = s.db.QueryRow(c.Request.Context(), `SELECT MAX(started_at) FROM worker_job_runs`).Scan(&latestRun)
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"last_run_at": latestRun,
		})
	})

	server := &http.Server{
		Addr:              ":" + s.port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if err := s.runCycle(ctx, time.Now().UTC()); err != nil {
		s.log.Error("worker initial cycle failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		case err := <-errCh:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		case tick := <-ticker.C:
			if err := s.runCycle(ctx, tick.UTC()); err != nil {
				s.log.Error("worker cycle failed: %v", err)
			}
		}
	}
}

func (s *Service) runCycle(ctx context.Context, now time.Time) error {
	if err := s.runJob(ctx, "daily_usage_reset", jobKeyDay(now), s.usage.RunDailyReset); err != nil {
		return err
	}
	if now.Day() == 1 {
		if err := s.runJob(ctx, "monthly_usage_reset", jobKeyMonth(now), s.usage.RunMonthlyReset); err != nil {
			return err
		}
	}
	if err := s.runJob(ctx, "subscription_expiry_check", jobKeyDay(now), func(jobCtx context.Context) error {
		return s.billing.RunSubscriptionExpiryCheck(jobCtx, now)
	}); err != nil {
		return err
	}
	if err := s.runJob(ctx, "deliverability_snapshots", jobKeyHour(now), func(jobCtx context.Context) error {
		return s.deliverability.GenerateSnapshots(jobCtx, now)
	}); err != nil {
		return err
	}
	if err := s.runJob(ctx, "deliverability_alerts", jobKeyHour(now), func(jobCtx context.Context) error {
		return s.deliverability.GenerateAlerts(jobCtx, now)
	}); err != nil {
		return err
	}
	if err := s.runJob(ctx, "retention_cleanup", jobKeyDay(now), s.retention.RunAutomaticCleanup); err != nil {
		return err
	}
	return nil
}

func (s *Service) runJob(ctx context.Context, jobName string, runKey string, fn func(context.Context) error) error {
	var exists bool
	if err := s.db.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM worker_job_runs
			WHERE job_name = $1
			  AND status = 'succeeded'
			  AND metadata->>'run_key' = $2
		)`,
		jobName,
		runKey,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	var runID string
	if err := s.db.QueryRow(
		ctx,
		`INSERT INTO worker_job_runs (job_name, status, metadata) VALUES ($1, 'running', jsonb_build_object('run_key', $2)) RETURNING id::text`,
		jobName,
		runKey,
	).Scan(&runID); err != nil {
		return err
	}

	if err := fn(ctx); err != nil {
		_, _ = s.db.Exec(ctx, `UPDATE worker_job_runs SET status = 'failed', finished_at = NOW(), metadata = metadata || jsonb_build_object('error', $2) WHERE id = $1`, runID, err.Error())
		return err
	}

	_, err := s.db.Exec(ctx, `UPDATE worker_job_runs SET status = 'succeeded', finished_at = NOW() WHERE id = $1`, runID)
	return err
}

func jobKeyDay(now time.Time) string {
	return now.Format("2006-01-02")
}

func jobKeyMonth(now time.Time) string {
	return fmt.Sprintf("%04d-%02d", now.Year(), now.Month())
}

func jobKeyHour(now time.Time) string {
	return now.Format("2006-01-02T15")
}
