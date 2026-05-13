package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	quotamodule "zxmail/backend/internal/modules/quota"
)

type stubQuotaService struct {
	updateFn  func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, input quotamodule.UpdateCredentialQuotaInput) (*quotamodule.CredentialQuotaView, error)
	disableFn func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error)
	enableFn  func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error)
}

func (s *stubQuotaService) UpdateCredentialQuota(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, input quotamodule.UpdateCredentialQuotaInput) (*quotamodule.CredentialQuotaView, error) {
	return s.updateFn(ctx, actor, credentialID, input)
}

func (s *stubQuotaService) DisableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error) {
	return s.disableFn(ctx, actor, credentialID)
}

func (s *stubQuotaService) EnableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error) {
	return s.enableFn(ctx, actor, credentialID)
}

func TestQuotaHandlerUpdateCredentialQuotaProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := []byte("test-secret")
	adminToken, err := authmodule.SignToken(secret, time.Hour, authmodule.AuthenticatedUser{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Role:  authmodule.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("sign admin token: %v", err)
	}

	customerToken, err := authmodule.SignToken(secret, time.Hour, authmodule.AuthenticatedUser{
		ID:    uuid.New(),
		Email: "customer@example.com",
		Role:  authmodule.RoleCustomer,
	})
	if err != nil {
		t.Fatalf("sign customer token: %v", err)
	}

	handler := NewQuotaHandler(&stubQuotaService{
		updateFn: func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, input quotamodule.UpdateCredentialQuotaInput) (*quotamodule.CredentialQuotaView, error) {
			if credentialID != "cred-1" {
				t.Fatalf("expected credential id cred-1, got %s", credentialID)
			}
			if !input.PerMinuteLimitSet || input.PerMinuteLimit == nil || *input.PerMinuteLimit != 60 {
				t.Fatalf("expected per_minute_limit to be set to 60")
			}
			if !input.ResetDailyUsed {
				t.Fatalf("expected reset_daily_used to be true")
			}
			return &quotamodule.CredentialQuotaView{
				ID:              uuid.New(),
				OrganizationID:  uuid.New(),
				DomainID:        uuid.New(),
				DomainName:      "mail.example.com",
				Username:        "apikey_test",
				Enabled:         true,
				Status:          "enabled",
				PerMinuteLimit:  input.PerMinuteLimit,
				PerMinuteUsed:   0,
				DailyUsed:       0,
				MonthlyUsed:     0,
				EnforcementNote: quotamodule.EnforcementNote,
			}, nil
		},
		disableFn: func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error) {
			return nil, nil
		},
		enableFn: func(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error) {
			return nil, nil
		},
	})

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.Auth(string(secret)))
	protected.PATCH("/api/v1/admin/credentials/:id/quota", middleware.RequireRoles(authmodule.RoleAdmin), handler.UpdateCredentialQuota)

	body, _ := json.Marshal(map[string]any{
		"per_minute_limit": 60,
		"reset_daily_used": true,
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{name: "customer denied", token: customerToken, expectedStatus: http.StatusForbidden},
		{name: "admin allowed", token: adminToken, expectedStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/credentials/cred-1/quota", bytes.NewReader(body))
			request.Header.Set("Authorization", "Bearer "+test.token)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.expectedStatus {
				t.Fatalf("expected %d, got %d", test.expectedStatus, response.Code)
			}
		})
	}
}
