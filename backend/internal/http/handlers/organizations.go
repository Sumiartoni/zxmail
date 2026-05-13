package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	organizationsmodule "zxmail/backend/internal/modules/organizations"
)

type OrganizationsService interface {
	ListOrganizations(ctx context.Context) ([]organizationsmodule.Organization, error)
	CreateCustomerOrganization(ctx context.Context, actorUserID string, input organizationsmodule.CreateCustomerOrganizationInput) (*organizationsmodule.Organization, error)
	GetOrganizationForOwner(ctx context.Context, ownerUserID string) (*organizationsmodule.Organization, error)
}

type OrganizationsHandler struct {
	service OrganizationsService
}

func NewOrganizationsHandler(service OrganizationsService) *OrganizationsHandler {
	return &OrganizationsHandler{service: service}
}

func (h *OrganizationsHandler) List(c *gin.Context) {
	organizations, err := h.service.ListOrganizations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": organizations})
}

func (h *OrganizationsHandler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	var request organizationsmodule.CreateCustomerOrganizationInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization payload"})
		return
	}

	organization, err := h.service.CreateCustomerOrganization(c.Request.Context(), actor.ID.String(), request)
	if err != nil {
		switch {
		case errors.Is(err, organizationsmodule.ErrAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "organization owner already exists"})
		case errors.Is(err, organizationsmodule.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": organization})
}

func (h *OrganizationsHandler) Current(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	if actor.Role != authmodule.RoleCustomer {
		c.JSON(http.StatusForbidden, gin.H{"error": "customer role required"})
		return
	}

	organization, err := h.service.GetOrganizationForOwner(c.Request.Context(), actor.ID.String())
	if err != nil {
		switch {
		case errors.Is(err, organizationsmodule.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organization"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": organization})
}
