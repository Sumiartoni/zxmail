package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubDependencyPinger struct {
	err error
}

func (s stubDependencyPinger) Ping(context.Context) error {
	return s.err
}

func TestHealthHandlerHealthAlwaysReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &HealthHandler{
		db:    stubDependencyPinger{err: errors.New("postgres down")},
		redis: stubDependencyPinger{err: errors.New("redis down")},
	}

	router := gin.New()
	router.GET("/health", handler.Health)

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", payload["status"])
	}
}

func TestHealthHandlerReadyReturnsDependencyState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &HealthHandler{
		db:    stubDependencyPinger{err: errors.New("auth failed")},
		redis: stubDependencyPinger{},
	}

	router := gin.New()
	router.GET("/health/ready", handler.Ready)

	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.Code)
	}

	var payload struct {
		Status string `json:"status"`
		Checks struct {
			Postgres struct {
				Ready bool   `json:"ready"`
				Error string `json:"error"`
			} `json:"postgres"`
			Redis struct {
				Ready bool `json:"ready"`
			} `json:"redis"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload.Status != "degraded" {
		t.Fatalf("expected degraded status, got %s", payload.Status)
	}
	if payload.Checks.Postgres.Ready {
		t.Fatalf("expected postgres readiness false")
	}
	if payload.Checks.Postgres.Error == "" {
		t.Fatalf("expected postgres error details")
	}
	if !payload.Checks.Redis.Ready {
		t.Fatalf("expected redis readiness true")
	}
}
