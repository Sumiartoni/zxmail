package app

import "testing"

func TestDescribePostgresTarget(t *testing.T) {
	host, database, user := describePostgresTarget("postgres://zxmail:secret@postgres:5432/zxmail?sslmode=disable")

	if host != "postgres" {
		t.Fatalf("expected host postgres, got %s", host)
	}
	if database != "zxmail" {
		t.Fatalf("expected database zxmail, got %s", database)
	}
	if user != "zxmail" {
		t.Fatalf("expected user zxmail, got %s", user)
	}
}

func TestDescribeRedisTarget(t *testing.T) {
	addr, db := describeRedisTarget("redis://redis:6379/2")

	if addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %s", addr)
	}
	if db != "2" {
		t.Fatalf("expected db 2, got %s", db)
	}
}

func TestDescribeRedisTargetRawAddress(t *testing.T) {
	addr, db := describeRedisTarget("redis:6379")

	if addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %s", addr)
	}
	if db != "0" {
		t.Fatalf("expected db 0, got %s", db)
	}
}
