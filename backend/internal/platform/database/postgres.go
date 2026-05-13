package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"create postgres pool host=%s db=%s user=%s: %w",
			cfg.ConnConfig.Host,
			cfg.ConnConfig.Database,
			cfg.ConnConfig.User,
			err,
		)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf(
			"ping postgres host=%s db=%s user=%s: %w",
			cfg.ConnConfig.Host,
			cfg.ConnConfig.Database,
			cfg.ConnConfig.User,
			err,
		)
	}

	return pool, nil
}
