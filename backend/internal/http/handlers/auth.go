package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
)

type AuthService interface {
	Login(ctx context.Context, email string, password string, clientIP string) (*authmodule.LoginResult, error)
}

type AuthHandler struct {
	service        AuthService
	cookieDomain   string
	cookieSecure   bool
	cookieTTL      time.Duration
	cookieSameSite http.SameSite
}

func NewAuthHandler(service AuthService, cookieDomain string, cookieSecure bool, cookieTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		service:        service,
		cookieDomain:   cookieDomain,
		cookieSecure:   cookieSecure,
		cookieTTL:      cookieTTL,
		cookieSameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login payload"})
		return
	}

	result, err := h.service.Login(c.Request.Context(), request.Email, request.Password, c.ClientIP())
	if err != nil {
		var rateLimitErr *authmodule.LoginRateLimitError
		if errors.As(err, &rateLimitErr) {
			c.Header("Retry-After", formatRetryAfter(rateLimitErr.RetryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts"})
			return
		}
		if errors.Is(err, authmodule.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	h.writeAccessTokenCookie(c, result.Token, h.cookieTTL)
	c.JSON(http.StatusOK, gin.H{"user": result.User})
}

func formatRetryAfter(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAccessTokenCookie(c)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": actor})
}

func (h *AuthHandler) writeAccessTokenCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(h.cookieSameSite)
	c.SetCookie(
		authmodule.AccessTokenCookieName,
		token,
		int(ttl.Seconds()),
		"/",
		h.cookieDomain,
		h.cookieSecure,
		true,
	)
}

func (h *AuthHandler) clearAccessTokenCookie(c *gin.Context) {
	c.SetSameSite(h.cookieSameSite)
	c.SetCookie(
		authmodule.AccessTokenCookieName,
		"",
		-1,
		"/",
		h.cookieDomain,
		h.cookieSecure,
		true,
	)
}
