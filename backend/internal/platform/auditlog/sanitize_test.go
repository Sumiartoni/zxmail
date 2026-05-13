package auditlog

import (
	"strings"
	"testing"
)

func TestSanitizeAuditDetailsRemovesSecretsRecursively(t *testing.T) {
	input := map[string]any{
		"email":    "audit@example.com",
		"username": "apikey_visible",
		"password": "should-not-be-kept",
		"api_key":  "should-not-be-kept",
		"token":    "should-not-be-kept",
		"headers": map[string]any{
			"Authorization":       "Bearer hidden",
			"X-Webhook-Signature": "hidden-signature",
			"X-Request-ID":        "keep-me",
		},
		"nested": map[string]any{
			"smtp_password":  "hidden-smtp-password",
			"webhook_secret": "hidden-webhook-secret",
			"note":           "keep-this",
		},
	}

	payload, err := MarshalAuditDetails(input)
	if err != nil {
		t.Fatalf("marshal audit details: %v", err)
	}

	lower := strings.ToLower(payload)
	for _, forbidden := range []string{
		"password",
		"api_key",
		"token",
		"authorization",
		"webhook_signature",
		"webhook_secret",
		"hidden-smtp-password",
		"hidden-webhook-secret",
		"bearer hidden",
	} {
		if strings.Contains(lower, strings.ToLower(forbidden)) {
			t.Fatalf("expected payload to remove %q, got %s", forbidden, payload)
		}
	}

	for _, expected := range []string{
		`"email":"audit@example.com"`,
		`"username":"apikey_visible"`,
		`"X-Request-ID":"keep-me"`,
		`"note":"keep-this"`,
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("expected payload to keep %s, got %s", expected, payload)
		}
	}
}
