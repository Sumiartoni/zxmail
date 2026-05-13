package auditlog

import (
	"encoding/json"
	"strings"
)

func SanitizeAuditDetails(metadata map[string]any) map[string]any {
	return sanitizeAuditDetails(metadata)
}

func sanitizeAuditDetails(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	sanitized := sanitizeAuditDetailsValue(metadata)
	sanitizedMap, ok := sanitized.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return sanitizedMap
}

func MarshalAuditDetails(metadata map[string]any) (string, error) {
	payload, err := json.Marshal(SanitizeAuditDetails(metadata))
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func sanitizeAuditDetailsValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, nestedValue := range typed {
			if isSensitiveAuditField(key) {
				continue
			}
			sanitized[key] = sanitizeAuditDetailsValue(nestedValue)
		}
		return sanitized
	case []any:
		sanitized := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitized = append(sanitized, sanitizeAuditDetailsValue(item))
		}
		return sanitized
	default:
		return value
	}
}

func isSensitiveAuditField(key string) bool {
	normalized := normalizeAuditKey(key)
	if normalized == "" {
		return false
	}

	exact := map[string]struct{}{
		"password":             {},
		"password_hash":        {},
		"secret":               {},
		"smtp_secret":          {},
		"smtp_password":        {},
		"token":                {},
		"api_key":              {},
		"apikey":               {},
		"authorization":        {},
		"authorization_header": {},
		"webhook_secret":       {},
		"webhook_signature":    {},
	}
	if _, ok := exact[normalized]; ok {
		return true
	}

	return strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "webhook_secret") ||
		strings.Contains(normalized, "webhook_signature") ||
		strings.Contains(normalized, "smtp_password") ||
		strings.Contains(normalized, "smtp_secret") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "apikey") ||
		strings.HasSuffix(normalized, "_token") ||
		strings.HasPrefix(normalized, "token_") ||
		strings.HasSuffix(normalized, "_password") ||
		strings.HasPrefix(normalized, "password_") ||
		strings.HasSuffix(normalized, "_secret") ||
		strings.HasPrefix(normalized, "secret_")
}

func normalizeAuditKey(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	replacer := strings.NewReplacer("-", "_", " ", "_")
	return replacer.Replace(normalized)
}
