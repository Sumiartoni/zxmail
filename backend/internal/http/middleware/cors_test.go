package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSRejectsDisallowedPreflightOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"https://dashboard.example.com"}))
	router.OPTIONS("/api/v1/domains", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/domains", nil)
	request.Header.Set("Origin", "https://evil.example.com")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestCORSAllowsConfiguredOriginWithCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"https://dashboard.example.com"}))
	router.GET("/api/v1/domains", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/domains", nil)
	request.Header.Set("Origin", "https://dashboard.example.com")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "https://dashboard.example.com" {
		t.Fatalf("expected allow-origin header to match configured origin")
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected allow-credentials header")
	}
}
