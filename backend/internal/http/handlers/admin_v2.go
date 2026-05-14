package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	adminv2module "zxmail/backend/internal/modules/adminv2"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
)

type AdminV2Service interface {
	OverviewMetrics(ctx context.Context, actor authmodule.AuthenticatedUser) (*adminv2module.Overview, error)
	OrganizationDetailByID(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) (*adminv2module.OrganizationDetail, error)
	SuspendOrganization(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, reason string) error
	UnsuspendOrganization(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error
	DisableOrganizationCredentials(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string) error
	ForceRotateCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID string) (*credentialsmodule.CredentialSecretResponse, error)
	OrganizationRisk(ctx context.Context, actor authmodule.AuthenticatedUser) ([]adminv2module.RiskRecord, error)
	ListAuditLogs(ctx context.Context, actor authmodule.AuthenticatedUser, limit int) ([]adminv2module.AuditLogRecord, error)
}

type AdminV2Handler struct {
	service AdminV2Service
}

func NewAdminV2Handler(service AdminV2Service) *AdminV2Handler {
	return &AdminV2Handler{service: service}
}

func (h *AdminV2Handler) Overview(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	overview, err := h.service.OverviewMetrics(c.Request.Context(), actor)
	if err != nil {
		if errors.Is(err, adminv2module.ErrAdminV2Forbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load admin overview"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"overview": overview})
}

func (h *AdminV2Handler) OrganizationDetail(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	detail, err := h.service.OrganizationDetailByID(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, adminv2module.ErrAdminV2Invalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		case errors.Is(err, adminv2module.ErrAdminV2NotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organization detail"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"organization": detail})
}

func (h *AdminV2Handler) Suspend(c *gin.Context) {
	h.setSuspended(c, true)
}

func (h *AdminV2Handler) Unsuspend(c *gin.Context) {
	h.setSuspended(c, false)
}

func (h *AdminV2Handler) DisableCredentials(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	if err := h.service.DisableOrganizationCredentials(c.Request.Context(), actor, c.Param("id")); err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, adminv2module.ErrAdminV2Invalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable credentials"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AdminV2Handler) ForceRotate(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	response, err := h.service.ForceRotateCredential(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate credential"})
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AdminV2Handler) Risk(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	items, err := h.service.OrganizationRisk(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organization risk"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"organizations": items})
}

func (h *AdminV2Handler) AuditLogs(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := h.service.ListAuditLogs(c.Request.Context(), actor, limit)
	if err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit logs"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": items})
}

func (h *AdminV2Handler) setSuspended(c *gin.Context, suspended bool) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&request)
	var err error
	if suspended {
		err = h.service.SuspendOrganization(c.Request.Context(), actor, c.Param("id"), request.Reason)
	} else {
		err = h.service.UnsuspendOrganization(c.Request.Context(), actor, c.Param("id"))
	}
	if err != nil {
		switch {
		case errors.Is(err, adminv2module.ErrAdminV2Forbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, adminv2module.ErrAdminV2Invalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update organization suspension"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
