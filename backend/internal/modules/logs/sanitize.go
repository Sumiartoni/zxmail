package logs

import (
	"encoding/json"
	"strings"

	authmodule "zxmail/backend/internal/modules/auth"
)

func sanitizeEntriesForActor(actor authmodule.AuthenticatedUser, entries []Entry) []Entry {
	if actor.Role == authmodule.RoleAdmin {
		return entries
	}

	sanitized := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		entry.RawEvent = sanitizeRawEvent(entry.RawEvent, entry)
		sanitized = append(sanitized, entry)
	}

	return sanitized
}

func sanitizeRawEvent(raw json.RawMessage, entry Entry) json.RawMessage {
	if len(raw) == 0 {
		return buildRawEventSummary(entry)
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return buildRawEventSummary(entry)
	}

	sanitized := sanitizeRawValue(payload)
	if sanitizedMap, ok := sanitized.(map[string]any); ok {
		sanitizedMap["sanitized"] = true
		sanitized = sanitizedMap
	}
	output, err := json.Marshal(sanitized)
	if err != nil {
		return buildRawEventSummary(entry)
	}

	return output
}

func sanitizeRawValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, nestedValue := range typed {
			if isSensitiveField(key) {
				continue
			}
			sanitized[key] = sanitizeRawValue(nestedValue)
		}
		return sanitized
	case []any:
		sanitized := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitized = append(sanitized, sanitizeRawValue(item))
		}
		return sanitized
	default:
		return value
	}
}

func buildRawEventSummary(entry Entry) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
		"sanitized":         true,
		"summary_only":      true,
		"status":            entry.Status,
		"postal_message_id": entry.PostalMessageID,
		"message_id":        entry.MessageIDHeader,
		"recipient":         entry.Recipient,
		"timestamp":         entry.Timestamp.UTC().Format(timeLayoutRFC3339),
	})
	if err != nil {
		return json.RawMessage(`{"sanitized":true,"summary_only":true}`)
	}
	return payload
}

func isSensitiveField(key string) bool {
	normalized := normalizeSensitiveKey(key)
	if normalized == "" {
		return false
	}

	exact := map[string]struct{}{
		"authorization":     {},
		"token":             {},
		"secret":            {},
		"password":          {},
		"api_key":           {},
		"apikey":            {},
		"smtp_password":     {},
		"webhook_signature": {},
	}
	if _, ok := exact[normalized]; ok {
		return true
	}

	return strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "webhook_signature") ||
		strings.Contains(normalized, "smtp_password") ||
		strings.HasSuffix(normalized, "_token") ||
		strings.HasPrefix(normalized, "token_") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "apikey") ||
		strings.HasSuffix(normalized, "_secret") ||
		strings.HasPrefix(normalized, "secret_") ||
		strings.HasSuffix(normalized, "_password") ||
		strings.HasPrefix(normalized, "password_")
}

func normalizeSensitiveKey(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	replacer := strings.NewReplacer("-", "_", " ", "_")
	return replacer.Replace(normalized)
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"
