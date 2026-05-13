package auth

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestLoginThrottleBlocksAfterExceededFailures(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	throttle := NewLoginThrottle(client, LoginThrottleConfig{
		MaxFailures:   5,
		FailureWindow: 10 * time.Minute,
		LockoutWindow: 15 * time.Minute,
	})

	ctx := context.Background()
	email := "admin@example.com"
	clientIP := "198.51.100.10"

	for attempt := 1; attempt <= 5; attempt++ {
		blocked, retryAfter, err := throttle.RecordFailure(ctx, email, clientIP)
		if err != nil {
			t.Fatalf("record failure %d: %v", attempt, err)
		}
		if blocked {
			t.Fatalf("attempt %d should not be blocked yet", attempt)
		}
		if retryAfter != 0 {
			t.Fatalf("attempt %d should not set retry after", attempt)
		}
	}

	blocked, retryAfter, err := throttle.RecordFailure(ctx, email, clientIP)
	if err != nil {
		t.Fatalf("record failure 6: %v", err)
	}
	if !blocked {
		t.Fatalf("expected attempt 6 to trigger lockout")
	}
	if retryAfter != 15*time.Minute {
		t.Fatalf("expected 15 minute lockout, got %s", retryAfter)
	}

	remaining, err := throttle.Check(ctx, email, clientIP)
	if err != nil {
		t.Fatalf("check lockout: %v", err)
	}
	if remaining <= 0 {
		t.Fatalf("expected active lockout")
	}

	server.FastForward(15 * time.Minute)

	remaining, err = throttle.Check(ctx, email, clientIP)
	if err != nil {
		t.Fatalf("check after fast-forward: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected lockout to expire, got %s", remaining)
	}
}

func TestLoginThrottleResetClearsFailures(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	throttle := NewLoginThrottle(client, LoginThrottleConfig{
		MaxFailures:   5,
		FailureWindow: 10 * time.Minute,
		LockoutWindow: 15 * time.Minute,
	})

	ctx := context.Background()
	email := "customer@example.com"
	clientIP := "203.0.113.25"

	for attempt := 1; attempt <= 3; attempt++ {
		if _, _, err := throttle.RecordFailure(ctx, email, clientIP); err != nil {
			t.Fatalf("record failure %d: %v", attempt, err)
		}
	}

	if err := throttle.Reset(ctx, email, clientIP); err != nil {
		t.Fatalf("reset throttle: %v", err)
	}

	for attempt := 1; attempt <= 5; attempt++ {
		blocked, _, err := throttle.RecordFailure(ctx, email, clientIP)
		if err != nil {
			t.Fatalf("record failure after reset %d: %v", attempt, err)
		}
		if blocked {
			t.Fatalf("attempt %d after reset should not be blocked yet", attempt)
		}
	}
}
