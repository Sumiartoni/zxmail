package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
)

type stubAuthService struct {
	loginFn func(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error)
}

func (s *stubAuthService) Login(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error) {
	return s.loginFn(ctx, email, password, clientIP)
}

func TestAuthHandlerLoginSetsHttpOnlyCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := []byte("test-secret-with-32-characters-min")
	handler := NewAuthHandler(&stubAuthService{
		loginFn: func(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error) {
			token, err := authmodule.SignToken(secret, time.Hour, authmodule.AuthenticatedUser{
				ID:    uuid.New(),
				Email: email,
				Role:  authmodule.RoleAdmin,
			})
			if err != nil {
				return nil, err
			}

			return &authmodule.LoginResult{
				Token: token,
				User: authmodule.AuthenticatedUser{
					ID:    uuid.New(),
					Email: email,
					Role:  authmodule.RoleAdmin,
				},
			}, nil
		},
	}, "", false, time.Hour)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "secret123",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	cookieHeader := response.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, authmodule.AccessTokenCookieName+"=") {
		t.Fatalf("expected auth cookie to be set")
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Fatalf("expected HttpOnly cookie")
	}
	if !strings.Contains(cookieHeader, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax cookie")
	}
}

func TestAuthHandlerLoginInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(&stubAuthService{
		loginFn: func(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error) {
			return nil, authmodule.ErrInvalidCredentials
		},
	}, "", false, time.Hour)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "wrong",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestAuthHandlerLoginRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(&stubAuthService{
		loginFn: func(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error) {
			return nil, &authmodule.LoginRateLimitError{RetryAfter: 15 * time.Minute}
		},
	}, "", false, time.Hour)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "wrong",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "198.51.100.10:12345"
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", response.Code)
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header")
	}
}

func TestAuthHandlerLoginInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(&stubAuthService{
		loginFn: func(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error) {
			return nil, errors.New("db down")
		},
	}, "", false, time.Hour)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "secret123",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.Code)
	}
}

func TestAuthHandlerMeAcceptsCookieAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := []byte("test-secret-with-32-characters-min")
	user := authmodule.AuthenticatedUser{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Role:  authmodule.RoleAdmin,
	}
	token, err := authmodule.SignToken(secret, time.Hour, user)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := NewAuthHandler(&stubAuthService{}, "", false, time.Hour)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.Auth(string(secret)))
	protected.GET("/api/v1/me", handler.Me)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.AddCookie(&http.Cookie{
		Name:  authmodule.AccessTokenCookieName,
		Value: token,
	})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestAuthHandlerLogoutClearsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(&stubAuthService{}, ".example.com", true, time.Hour)

	router := gin.New()
	router.POST("/api/v1/auth/logout", handler.Logout)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	cookieHeader := response.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "Max-Age=0") && !strings.Contains(cookieHeader, "Expires=") {
		t.Fatalf("expected auth cookie to be cleared")
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Fatalf("expected cleared cookie to remain HttpOnly")
	}
}
