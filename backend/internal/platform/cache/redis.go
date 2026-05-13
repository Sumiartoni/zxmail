package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	options, err := parseRedisOptions(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis addr=%s db=%d: %w", options.Addr, options.DB, err)
	}

	return client, nil
}

func parseRedisOptions(redisURL string) (*redis.Options, error) {
	trimmed := strings.TrimSpace(redisURL)
	if trimmed == "" {
		return nil, fmt.Errorf("parse redis config: REDIS_URL is empty")
	}

	if strings.Contains(trimmed, "://") {
		options, err := redis.ParseURL(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse redis config from URL %q: %w", trimmed, err)
		}
		return options, nil
	}

	return &redis.Options{
		Addr: trimmed,
		DB:   0,
	}, nil
}
