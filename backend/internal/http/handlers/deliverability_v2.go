package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	deliverabilitymodule "zxmail/backend/internal/modules/deliverability"
)

type DeliverabilityV2Service interface {
	OverviewForActor(ctx context.Context, actor authmodule.AuthenticatedUser) (*deliverabilitymodule.Overview, error)
	DomainHealthForActor(ctx context.Context, actor authmodule.AuthenticatedUser, domainID string) (*deliverabilitymodule.DomainHealth, error)
	AdminDeliverabilityOverview(ctx context.Context, actor authmodule.AuthenticatedUser) (*deliverabilitymodule.Overview, error)
	ListAlerts(ctx context.Context, actor authmodule.AuthenticatedUser) ([]deliverabilitymodule.Alert, error)
	ResolveAlert(ctx context.Context, actor authmodule.AuthenticatedUser, alertID string) error
	RecheckDomain(ctx context.Context, actor authmodule.AuthenticatedUser, domainID string) (*deliverabilitymodule.DomainHealth, error)
	ListAllDomainHealth(ctx context.Context, actor authmodule.AuthenticatedUser) ([]deliverabilitymodule.DomainHealth, error)
	RecheckAllDomains(ctx context.Context, actor authmodule.AuthenticatedUser) error
}

type DeliverabilityV2Handler struct {
	service DeliverabilityV2Service
}

func NewDeliverabilityV2Handler(service DeliverabilityV2Service) *DeliverabilityV2Handler {
	return &DeliverabilityV2Handler{service: service}
}

func (h *DeliverabilityV2Handler) Overview(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	overview, err := h.service.OverviewForActor(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load deliverability overview"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"overview": overview})
}

func (h *DeliverabilityV2Handler) Domain(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	health, err := h.service.DomainHealthForActor(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "domain health not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load domain health"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"domain_health": health})
}

func (h *DeliverabilityV2Handler) AdminOverview(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	overview, err := h.service.AdminDeliverabilityOverview(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load admin deliverability overview"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"overview": overview})
}

func (h *DeliverabilityV2Handler) ListAlerts(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	alerts, err := h.service.ListAlerts(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alerts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *DeliverabilityV2Handler) ResolveAlert(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	if err := h.service.ResolveAlert(c.Request.Context(), actor, c.Param("id")); err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve alert"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *DeliverabilityV2Handler) RecheckDomain(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	health, err := h.service.RecheckDomain(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to recheck domain"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"domain_health": health})
}

func (h *DeliverabilityV2Handler) AdminDomainsHealth(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	items, err := h.service.ListAllDomainHealth(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domain health"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"domains": items})
}

func (h *DeliverabilityV2Handler) RecheckAll(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	if err := h.service.RecheckAllDomains(c.Request.Context(), actor); err != nil {
		switch {
		case errors.Is(err, deliverabilitymodule.ErrDeliverabilityForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to recheck domains"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
