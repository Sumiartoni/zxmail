package cache

import "testing"

func TestParseRedisOptionsSupportsURL(t *testing.T) {
	options, err := parseRedisOptions("redis://redis:6379/2")
	if err != nil {
		t.Fatalf("parse redis URL: %v", err)
	}

	if options.Addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %s", options.Addr)
	}
	if options.DB != 2 {
		t.Fatalf("expected db 2, got %d", options.DB)
	}
}

func TestParseRedisOptionsSupportsRawAddress(t *testing.T) {
	options, err := parseRedisOptions("redis:6379")
	if err != nil {
		t.Fatalf("parse raw redis address: %v", err)
	}

	if options.Addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %s", options.Addr)
	}
	if options.DB != 0 {
		t.Fatalf("expected db 0, got %d", options.DB)
	}
}

func TestParseRedisOptionsRejectsEmptyValue(t *testing.T) {
	_, err := parseRedisOptions("   ")
	if err == nil {
		t.Fatalf("expected error for empty redis config")
	}
}
