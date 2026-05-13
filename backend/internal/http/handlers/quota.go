package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	quotamodule "zxmail/backend/internal/modules/quota"
)

type QuotaService interface {
	UpdateCredentialQuota(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, input quotamodule.UpdateCredentialQuotaInput) (*quotamodule.CredentialQuotaView, error)
	DisableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error)
	EnableCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*quotamodule.CredentialQuotaView, error)
}

type QuotaHandler struct {
	service QuotaService
}

func NewQuotaHandler(service QuotaService) *QuotaHandler {
	return &QuotaHandler{service: service}
}

func (h *QuotaHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enforcement_note": quotamodule.EnforcementNote,
	})
}

func (h *QuotaHandler) UpdateCredentialQuota(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	input, err := parseUpdateCredentialQuotaInput(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota payload"})
		return
	}

	response, err := h.service.UpdateCredentialQuota(c.Request.Context(), actor, c.Param("id"), input)
	if err != nil {
		switch {
		case errors.Is(err, quotamodule.ErrQuotaInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota input"})
		case errors.Is(err, quotamodule.ErrQuotaForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, quotamodule.ErrQuotaCredentialMissing):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update credential quota"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": response})
}

func (h *QuotaHandler) DisableCredential(c *gin.Context) {
	h.setEnabled(c, false)
}

func (h *QuotaHandler) EnableCredential(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *QuotaHandler) setEnabled(c *gin.Context, enabled bool) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	var (
		response *quotamodule.CredentialQuotaView
		err      error
	)
	if enabled {
		response, err = h.service.EnableCredential(c.Request.Context(), actor, c.Param("id"))
	} else {
		response, err = h.service.DisableCredential(c.Request.Context(), actor, c.Param("id"))
	}
	if err != nil {
		switch {
		case errors.Is(err, quotamodule.ErrQuotaInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		case errors.Is(err, quotamodule.ErrQuotaForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, quotamodule.ErrQuotaCredentialMissing):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update credential state"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": response})
}

func parseUpdateCredentialQuotaInput(c *gin.Context) (quotamodule.UpdateCredentialQuotaInput, error) {
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}

	var input quotamodule.UpdateCredentialQuotaInput
	if err := decodeOptionalInt(raw, "per_minute_limit", &input.PerMinuteLimit, &input.PerMinuteLimitSet); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}
	if err := decodeOptionalInt(raw, "daily_limit", &input.DailyLimit, &input.DailyLimitSet); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}
	if err := decodeOptionalInt(raw, "monthly_limit", &input.MonthlyLimit, &input.MonthlyLimitSet); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}
	if err := decodeOptionalBool(raw, "reset_minute_used", &input.ResetMinuteUsed); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}
	if err := decodeOptionalBool(raw, "reset_daily_used", &input.ResetDailyUsed); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}
	if err := decodeOptionalBool(raw, "reset_monthly_used", &input.ResetMonthlyUsed); err != nil {
		return quotamodule.UpdateCredentialQuotaInput{}, err
	}

	return input, nil
}

func decodeOptionalInt(raw map[string]json.RawMessage, key string, target **int, set *bool) error {
	value, ok := raw[key]
	if !ok {
		return nil
	}

	*set = true
	if string(value) == "null" {
		*target = nil
		return nil
	}

	var decoded int
	if err := json.Unmarshal(value, &decoded); err != nil {
		return err
	}
	*target = &decoded
	return nil
}

func decodeOptionalBool(raw map[string]json.RawMessage, key string, target *bool) error {
	value, ok := raw[key]
	if !ok {
		return nil
	}

	return json.Unmarshal(value, target)
}
