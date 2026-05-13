package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type LoginThrottleConfig struct {
	MaxFailures   int
	FailureWindow time.Duration
	LockoutWindow time.Duration
}

type LoginThrottle struct {
	redis  *redis.Client
	config LoginThrottleConfig
}

func NewLoginThrottle(redisClient *redis.Client, config LoginThrottleConfig) *LoginThrottle {
	return &LoginThrottle{
		redis:  redisClient,
		config: config,
	}
}

func (t *LoginThrottle) Check(ctx context.Context, email string, clientIP string) (time.Duration, error) {
	if t == nil || t.redis == nil {
		return 0, nil
	}

	blockKey := t.blockKey(email, clientIP)
	exists, err := t.redis.Exists(ctx, blockKey).Result()
	if err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, nil
	}

	ttl, err := t.redis.TTL(ctx, blockKey).Result()
	if err != nil {
		return 0, err
	}
	if ttl <= 0 {
		return time.Second, nil
	}

	return ttl, nil
}

func (t *LoginThrottle) RecordFailure(ctx context.Context, email string, clientIP string) (bool, time.Duration, error) {
	if t == nil || t.redis == nil {
		return false, 0, nil
	}

	failKey := t.failureKey(email, clientIP)
	count, err := t.redis.Incr(ctx, failKey).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := t.redis.Expire(ctx, failKey, t.config.FailureWindow).Err(); err != nil {
			return false, 0, err
		}
	}

	if count <= int64(t.config.MaxFailures) {
		return false, 0, nil
	}

	blockKey := t.blockKey(email, clientIP)
	if err := t.redis.Set(ctx, blockKey, "1", t.config.LockoutWindow).Err(); err != nil {
		return false, 0, err
	}
	if err := t.redis.Del(ctx, failKey).Err(); err != nil {
		return false, 0, err
	}

	return true, t.config.LockoutWindow, nil
}

func (t *LoginThrottle) Reset(ctx context.Context, email string, clientIP string) error {
	if t == nil || t.redis == nil {
		return nil
	}

	return t.redis.Del(ctx, t.failureKey(email, clientIP), t.blockKey(email, clientIP)).Err()
}

func (t *LoginThrottle) failureKey(email string, clientIP string) string {
	return "auth:login:fail:" + t.fingerprint(email, clientIP)
}

func (t *LoginThrottle) blockKey(email string, clientIP string) string {
	return "auth:login:block:" + t.fingerprint(email, clientIP)
}

func (t *LoginThrottle) fingerprint(email string, clientIP string) string {
	normalizedIP := strings.TrimSpace(clientIP)
	if normalizedIP == "" {
		normalizedIP = "unknown"
	}

	sum := sha256.Sum256([]byte(normalizeEmail(email) + "|" + normalizedIP))
	return hex.EncodeToString(sum[:])
}
