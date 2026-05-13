package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	domainsmodule "zxmail/backend/internal/modules/domains"
)

type DomainsService interface {
	List(ctx context.Context, actor authmodule.AuthenticatedUser) ([]domainsmodule.DomainListItem, error)
	Create(ctx context.Context, actor authmodule.AuthenticatedUser, input domainsmodule.CreateDomainInput) (*domainsmodule.DomainResponse, error)
	Get(ctx context.Context, actor authmodule.AuthenticatedUser, id string) (*domainsmodule.DomainResponse, error)
	Verify(ctx context.Context, actor authmodule.AuthenticatedUser, id string) (*domainsmodule.VerificationResult, error)
}

type DomainsHandler struct {
	service DomainsService
}

func NewDomainsHandler(service DomainsService) *DomainsHandler {
	return &DomainsHandler{service: service}
}

func (h *DomainsHandler) List(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	domains, err := h.service.List(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"domains":  domains,
		"warnings": []string{"SMTP records must be DNS only in Cloudflare, not proxied."},
	})
}

func (h *DomainsHandler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	var request domainsmodule.CreateDomainInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain payload"})
		return
	}

	response, err := h.service.Create(c.Request.Context(), actor, request)
	if err != nil {
		switch {
		case errors.Is(err, domainsmodule.ErrDomainInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain input"})
		case errors.Is(err, domainsmodule.ErrDomainConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "domain already exists"})
		case errors.Is(err, domainsmodule.ErrOrganizationNeeded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "organization is required"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create domain"})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *DomainsHandler) Get(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, domainsmodule.ErrDomainInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		case errors.Is(err, domainsmodule.ErrDomainNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		case errors.Is(err, domainsmodule.ErrDomainForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load domain"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *DomainsHandler) Verify(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.Verify(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, domainsmodule.ErrDomainInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		case errors.Is(err, domainsmodule.ErrDomainNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		case errors.Is(err, domainsmodule.ErrDomainForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify domain"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}
