package logs

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	authmodule "zxmail/backend/internal/modules/auth"
)

func TestSanitizeRawEventRemovesSensitiveFieldsForCustomer(t *testing.T) {
	entry := Entry{
		ID:              uuid.New(),
		PostalMessageID: "postal-1",
		MessageIDHeader: "<message-1@example.com>",
		Recipient:       "user@example.com",
		Status:          "accepted",
		Timestamp:       time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC),
		RawEvent: json.RawMessage(`{
			"event":"accepted",
			"authorization":"Bearer top-secret",
			"token":"abc123",
			"secret":"hidden",
			"password":"hidden-password",
			"api_key":"hidden-key",
			"smtp_password":"hidden-smtp",
			"headers":{
				"Authorization":"Bearer nested-secret",
				"X-Webhook-Signature":"signature-value",
				"X-Trace-Id":"trace-1"
			},
			"message":{
				"id":"postal-1",
				"credential":"apikey_visible",
				"metadata":{
					"webhook_signature":"nested-signature",
					"note":"keep-me"
				}
			}
		}`),
	}

	actor := authmodule.AuthenticatedUser{ID: uuid.New(), Role: authmodule.RoleCustomer}
	entries := sanitizeEntriesForActor(actor, []Entry{entry})
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}

	body := string(entries[0].RawEvent)
	for _, forbidden := range []string{
		"authorization",
		"token",
		"secret",
		"password",
		"api_key",
		"smtp_password",
		"webhook_signature",
		"signature-value",
		"hidden-key",
		"hidden-smtp",
		"nested-secret",
	} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Fatalf("expected sanitized raw_event to remove %q, got %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"X-Trace-Id":"trace-1"`) {
		t.Fatalf("expected non-sensitive headers to remain, got %s", body)
	}
	if !strings.Contains(body, `"credential":"apikey_visible"`) {
		t.Fatalf("expected non-sensitive message fields to remain, got %s", body)
	}
	if !strings.Contains(body, `"note":"keep-me"`) {
		t.Fatalf("expected nested safe fields to remain, got %s", body)
	}
	if !strings.Contains(body, `"sanitized":true`) {
		t.Fatalf("expected sanitized marker, got %s", body)
	}
}

func TestSanitizeEntriesForActorKeepsFullRawEventForAdmin(t *testing.T) {
	rawEvent := json.RawMessage(`{"authorization":"Bearer admin-can-see","event":"accepted"}`)
	entry := Entry{
		ID:       uuid.New(),
		Status:   "accepted",
		RawEvent: rawEvent,
	}

	actor := authmodule.AuthenticatedUser{ID: uuid.New(), Role: authmodule.RoleAdmin}
	entries := sanitizeEntriesForActor(actor, []Entry{entry})
	if string(entries[0].RawEvent) != string(rawEvent) {
		t.Fatalf("expected admin to receive full raw_event, got %s", entries[0].RawEvent)
	}
}

func TestSanitizeRawEventFallsBackToSummaryWhenPayloadInvalid(t *testing.T) {
	entry := Entry{
		ID:              uuid.New(),
		PostalMessageID: "postal-2",
		MessageIDHeader: "<message-2@example.com>",
		Recipient:       "user@example.com",
		Status:          "bounced",
		Timestamp:       time.Date(2026, 5, 13, 11, 30, 0, 0, time.UTC),
		RawEvent:        json.RawMessage(`{"broken":`),
	}

	sanitized := sanitizeRawEvent(entry.RawEvent, entry)
	body := string(sanitized)
	if !strings.Contains(body, `"summary_only":true`) {
		t.Fatalf("expected summary fallback, got %s", body)
	}
	if !strings.Contains(body, `"status":"bounced"`) {
		t.Fatalf("expected status in summary, got %s", body)
	}
	if !strings.Contains(body, `"recipient":"user@example.com"`) {
		t.Fatalf("expected recipient in summary, got %s", body)
	}
}
