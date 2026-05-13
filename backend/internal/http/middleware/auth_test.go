package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authmodule "zxmail/backend/internal/modules/auth"
)

func TestRequireRolesProtectsAdminRoute(t *testing.T) {
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

	router := gin.New()
	protected := router.Group("/")
	protected.Use(Auth(string(secret)))
	protected.GET("/api/v1/admin/organizations", RequireRoles(authmodule.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{name: "missing token", token: "", expectedStatus: http.StatusUnauthorized},
		{name: "customer denied", token: customerToken, expectedStatus: http.StatusForbidden},
		{name: "admin allowed", token: adminToken, expectedStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.expectedStatus {
				t.Fatalf("expected %d, got %d", test.expectedStatus, response.Code)
			}
		})
	}
}
