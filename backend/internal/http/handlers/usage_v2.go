package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	usagemodule "zxmail/backend/internal/modules/usage"
)

type UsageV2Service interface {
	GetUsage(ctx context.Context, actor authmodule.AuthenticatedUser) (*usagemodule.Overview, error)
	GetOrganizationUsage(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) (*usagemodule.Overview, error)
	UpdateOrganizationQuota(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, input usagemodule.UpdateOrganizationQuotaInput) (*usagemodule.Overview, error)
	ResetOrganizationUsage(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error
	SetCredentialLimited(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string, limited bool, reason string) error
}

type UsageV2Handler struct {
	service UsageV2Service
}

func NewUsageV2Handler(service UsageV2Service) *UsageV2Handler {
	return &UsageV2Handler{service: service}
}

func (h *UsageV2Handler) GetUsage(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	overview, err := h.service.GetUsage(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": overview})
}

func (h *UsageV2Handler) GetOrganizationUsage(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	overview, err := h.service.GetOrganizationUsage(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, usagemodule.ErrUsageForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, usagemodule.ErrUsageInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organization usage"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": overview})
}

func (h *UsageV2Handler) UpdateOrganizationQuota(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request usagemodule.UpdateOrganizationQuotaInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota payload"})
		return
	}
	overview, err := h.service.UpdateOrganizationQuota(c.Request.Context(), actor, c.Param("id"), request)
	if err != nil {
		switch {
		case errors.Is(err, usagemodule.ErrUsageForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, usagemodule.ErrUsageInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota payload"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update quota"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": overview})
}

func (h *UsageV2Handler) ResetOrganizationUsage(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	if err := h.service.ResetOrganizationUsage(c.Request.Context(), actor, c.Param("id")); err != nil {
		switch {
		case errors.Is(err, usagemodule.ErrUsageForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, usagemodule.ErrUsageInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset usage"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *UsageV2Handler) LimitCredential(c *gin.Context) {
	h.setCredentialLimit(c, true)
}

func (h *UsageV2Handler) UnlimitCredential(c *gin.Context) {
	h.setCredentialLimit(c, false)
}

func (h *UsageV2Handler) setCredentialLimit(c *gin.Context, limited bool) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&request)
	if err := h.service.SetCredentialLimited(c.Request.Context(), actor, c.Param("id"), limited, request.Reason); err != nil {
		switch {
		case errors.Is(err, usagemodule.ErrUsageForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, usagemodule.ErrUsageInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		case errors.Is(err, usagemodule.ErrUsageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change credential limit"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
