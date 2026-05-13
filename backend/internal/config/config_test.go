package config

import "testing"

func TestValidateRejectsLocalDependencyFallbacksInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:                "production",
		FrontendOrigin:        "https://dashboard.zxmail.site",
		CORSAllowOrigins:      []string{"https://dashboard.zxmail.site"},
		DatabaseURL:           "postgres://zxmail:zxmail@localhost:5432/zxmail?sslmode=disable",
		RedisURL:              "redis://localhost:6379/0",
		JWTSecret:             "12345678901234567890123456789012",
		LoginMaxFailures:      5,
		LoginFailureWindow:    10,
		LoginLockoutWindow:    15,
		EncryptionKeyID:       "legacy-v1",
		EncryptionKey:         "12345678901234567890123456789012",
		PostalWebhookSecret:   "postal-webhook-secret-that-is-not-default",
		PostalBaseURL:         "https://postal.zxmail.site",
		ActiveEncryptionKeyID: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected production validation to reject localhost dependency fallbacks")
	}
}
