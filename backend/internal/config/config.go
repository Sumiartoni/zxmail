package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName               string
	AppEnv                string
	HTTPPort              string
	FrontendOrigin        string
	CORSAllowOrigins      []string
	CookieDomain          string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	JWTTokenTTL           time.Duration
	LoginMaxFailures      int
	LoginFailureWindow    time.Duration
	LoginLockoutWindow    time.Duration
	FirstAdminEmail       string
	FirstAdminPassword    string
	EncryptionKeyID       string
	EncryptionKey         string
	EncryptionKeys        string
	ActiveEncryptionKeyID string
	PostalBaseURL         string
	PostalAPIKey          string
	PostalWebhookSecret   string
	SMTPHost              string
	SMTPPortSTARTTLS      string
	SMTPPortTLS           string
	ShutdownTimeout       time.Duration
}

func Load() Config {
	frontendOrigin := getEnv("FRONTEND_ORIGIN", getEnv("FRONTEND_URL", "http://localhost:3000"))

	return Config{
		AppName:               getEnv("APP_NAME", "zxmail-api"),
		AppEnv:                getEnv("APP_ENV", "development"),
		HTTPPort:              getEnv("HTTP_PORT", "8080"),
		FrontendOrigin:        frontendOrigin,
		CORSAllowOrigins:      getCSVEnv("CORS_ALLOW_ORIGINS", []string{frontendOrigin}),
		CookieDomain:          getEnv("COOKIE_DOMAIN", ""),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://zxmail:zxmail@localhost:5432/zxmail?sslmode=disable"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:             getEnv("JWT_SECRET", "change-me"),
		JWTTokenTTL:           time.Duration(getIntEnv("JWT_TTL_HOURS", 24)) * time.Hour,
		LoginMaxFailures:      getIntEnv("LOGIN_MAX_FAILURES", 5),
		LoginFailureWindow:    time.Duration(getIntEnv("LOGIN_FAILURE_WINDOW_MINUTES", 10)) * time.Minute,
		LoginLockoutWindow:    time.Duration(getIntEnv("LOGIN_LOCKOUT_MINUTES", 15)) * time.Minute,
		FirstAdminEmail:       getEnv("FIRST_ADMIN_EMAIL", ""),
		FirstAdminPassword:    getEnv("FIRST_ADMIN_PASSWORD", ""),
		EncryptionKeyID:       getEnv("ENCRYPTION_KEY_ID", "default"),
		EncryptionKey:         getEnv("ENCRYPTION_KEY", ""),
		EncryptionKeys:        getEnv("ENCRYPTION_KEYS", ""),
		ActiveEncryptionKeyID: getEnv("ACTIVE_ENCRYPTION_KEY_ID", ""),
		PostalBaseURL:         getEnv("POSTAL_BASE_URL", "http://postal:5000"),
		PostalAPIKey:          getEnv("POSTAL_API_KEY", ""),
		PostalWebhookSecret:   getEnv("POSTAL_WEBHOOK_SECRET", "change-me"),
		SMTPHost:              getEnv("SMTP_PUBLIC_HOST", getEnv("SMTP_HOST", "smtp.zxmail.test")),
		SMTPPortSTARTTLS:      getEnv("SMTP_PORT_STARTTLS", "587"),
		SMTPPortTLS:           getEnv("SMTP_PORT_TLS", "465"),
		ShutdownTimeout:       time.Duration(getIntEnv("SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
	}
}

func FromEnv() Config {
	return Load()
}

func (c Config) Validate() error {
	var validationErrors []string

	if len(c.JWTSecret) < 32 || isUnsafeSecret(c.JWTSecret) {
		validationErrors = append(validationErrors, "JWT_SECRET must be at least 32 characters and not use a default placeholder")
	}
	if strings.TrimSpace(c.EncryptionKeyID) == "" {
		validationErrors = append(validationErrors, "ENCRYPTION_KEY_ID must be set")
	}
	if strings.TrimSpace(c.EncryptionKeys) == "" {
		if len(c.EncryptionKey) < 32 || isUnsafeSecret(c.EncryptionKey) {
			validationErrors = append(validationErrors, "ENCRYPTION_KEY must be at least 32 characters and not use a default placeholder when ENCRYPTION_KEYS is not configured")
		}
	} else if strings.TrimSpace(c.ActiveEncryptionKeyID) == "" {
		validationErrors = append(validationErrors, "ACTIVE_ENCRYPTION_KEY_ID must be set when ENCRYPTION_KEYS is configured")
	}
	if c.PostalWebhookSecret == "" || isUnsafeSecret(c.PostalWebhookSecret) {
		validationErrors = append(validationErrors, "POSTAL_WEBHOOK_SECRET must be set to a non-default value")
	}
	if len(c.CORSAllowOrigins) == 0 {
		validationErrors = append(validationErrors, "CORS_ALLOW_ORIGINS must include at least one explicit origin")
	}
	if c.LoginMaxFailures <= 0 {
		validationErrors = append(validationErrors, "LOGIN_MAX_FAILURES must be greater than zero")
	}
	if c.LoginFailureWindow <= 0 {
		validationErrors = append(validationErrors, "LOGIN_FAILURE_WINDOW_MINUTES must be greater than zero")
	}
	if c.LoginLockoutWindow <= 0 {
		validationErrors = append(validationErrors, "LOGIN_LOCKOUT_MINUTES must be greater than zero")
	}
	if c.AppEnv == "production" {
		if contains(c.CORSAllowOrigins, "*") {
			validationErrors = append(validationErrors, "CORS_ALLOW_ORIGINS cannot contain * in production")
		}
		if strings.HasPrefix(c.FrontendOrigin, "http://") {
			validationErrors = append(validationErrors, "FRONTEND_ORIGIN should use https in production")
		}
	}

	if len(validationErrors) == 0 {
		return nil
	}

	return errors.New(strings.Join(validationErrors, "; "))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getCSVEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	if len(out) == 0 {
		return fallback
	}

	return out
}

func isUnsafeSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "change-me", "replace-me", "replace-with-long-random-value", "replace-with-32-byte-random-value", "change-me-32-bytes-minimum":
		return true
	default:
		return false
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
