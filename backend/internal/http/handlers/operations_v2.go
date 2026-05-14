package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	operationsmodule "zxmail/backend/internal/modules/operations"
	"zxmail/backend/internal/postal"
)

type OperationsV2Service interface {
	AdminSystemHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*operationsmodule.SystemHealth, error)
	QueueHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*operationsmodule.QueueHealth, error)
	PostalHealth(ctx context.Context, actor authmodule.AuthenticatedUser) (*postal.HealthCheckResult, error)
}

type OperationsV2Handler struct {
	service OperationsV2Service
}

func NewOperationsV2Handler(service OperationsV2Service) *OperationsV2Handler {
	return &OperationsV2Handler{service: service}
}

func (h *OperationsV2Handler) Health(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	response, err := h.service.AdminSystemHealth(c.Request.Context(), actor)
	if err != nil {
		if errors.Is(err, operationsmodule.ErrOperationsForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system health"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"health": response})
}

func (h *OperationsV2Handler) Queues(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	response, err := h.service.QueueHealth(c.Request.Context(), actor)
	if err != nil {
		if errors.Is(err, operationsmodule.ErrOperationsForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load queue health"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queues": response})
}

func (h *OperationsV2Handler) PostalHealth(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	response, err := h.service.PostalHealth(c.Request.Context(), actor)
	if err != nil {
		if errors.Is(err, operationsmodule.ErrOperationsForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load postal health"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"postal": response})
}
