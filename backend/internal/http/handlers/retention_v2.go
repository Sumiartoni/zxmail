package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	retentionmodule "zxmail/backend/internal/modules/retention"
)

type RetentionV2Service interface {
	GetPolicies(ctx context.Context, actor authmodule.AuthenticatedUser) ([]retentionmodule.OrganizationPolicy, error)
	UpdatePolicy(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, retentionDays int) error
	RunCleanup(ctx context.Context, actor *authmodule.AuthenticatedUser, dryRun bool) (*retentionmodule.CleanupResult, error)
}

type RetentionV2Handler struct {
	service RetentionV2Service
}

func NewRetentionV2Handler(service RetentionV2Service) *RetentionV2Handler {
	return &RetentionV2Handler{service: service}
}

func (h *RetentionV2Handler) List(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	items, err := h.service.GetPolicies(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, retentionmodule.ErrRetentionForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list retention policies"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": items})
}

func (h *RetentionV2Handler) Update(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid retention payload"})
		return
	}
	if err := h.service.UpdatePolicy(c.Request.Context(), actor, c.Param("id"), request.RetentionDays); err != nil {
		switch {
		case errors.Is(err, retentionmodule.ErrRetentionForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, retentionmodule.ErrRetentionInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid retention payload"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update retention"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *RetentionV2Handler) Cleanup(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	dryRun, _ := strconv.ParseBool(c.DefaultQuery("dry_run", "true"))
	result, err := h.service.RunCleanup(c.Request.Context(), &actor, dryRun)
	if err != nil {
		switch {
		case errors.Is(err, retentionmodule.ErrRetentionForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run cleanup"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"cleanup": result})
}
