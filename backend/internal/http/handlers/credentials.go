package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
)

type CredentialsService interface {
	List(ctx context.Context, actor authmodule.AuthenticatedUser, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) ([]credentialsmodule.CredentialResponse, error)
	Create(ctx context.Context, actor authmodule.AuthenticatedUser, input credentialsmodule.CreateCredentialInput, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*credentialsmodule.CredentialSecretResponse, error)
	Get(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*credentialsmodule.CredentialResponse, error)
	Revoke(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*credentialsmodule.CredentialResponse, error)
	Rotate(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*credentialsmodule.CredentialSecretResponse, error)
}

type CredentialsHandler struct {
	service          CredentialsService
	smtpHost         string
	smtpPortSTARTTLS string
	smtpPortTLS      string
}

func NewCredentialsHandler(service CredentialsService, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) *CredentialsHandler {
	return &CredentialsHandler{
		service:          service,
		smtpHost:         smtpHost,
		smtpPortSTARTTLS: smtpPortSTARTTLS,
		smtpPortTLS:      smtpPortTLS,
	}
}

func (h *CredentialsHandler) List(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.List(c.Request.Context(), actor, h.smtpHost, h.smtpPortSTARTTLS, h.smtpPortTLS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credentials": response})
}

func (h *CredentialsHandler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	var request credentialsmodule.CreateCredentialInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential payload"})
		return
	}

	response, err := h.service.Create(c.Request.Context(), actor, request, h.smtpHost, h.smtpPortSTARTTLS, h.smtpPortTLS)
	if err != nil {
		switch {
		case errors.Is(err, credentialsmodule.ErrCredentialInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential input"})
		case errors.Is(err, credentialsmodule.ErrCredentialForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, credentialsmodule.ErrCredentialDomainState):
			c.JSON(http.StatusConflict, gin.H{"error": "domain must be verified"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create credential"})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *CredentialsHandler) Get(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.Get(c.Request.Context(), actor, c.Param("id"), h.smtpHost, h.smtpPortSTARTTLS, h.smtpPortTLS)
	if err != nil {
		switch {
		case errors.Is(err, credentialsmodule.ErrCredentialInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		case errors.Is(err, credentialsmodule.ErrCredentialNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		case errors.Is(err, credentialsmodule.ErrCredentialForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load credential"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *CredentialsHandler) Revoke(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.Revoke(c.Request.Context(), actor, c.Param("id"), h.smtpHost, h.smtpPortSTARTTLS, h.smtpPortTLS)
	if err != nil {
		switch {
		case errors.Is(err, credentialsmodule.ErrCredentialInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		case errors.Is(err, credentialsmodule.ErrCredentialNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		case errors.Is(err, credentialsmodule.ErrCredentialForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke credential"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *CredentialsHandler) Rotate(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	response, err := h.service.Rotate(c.Request.Context(), actor, c.Param("id"), h.smtpHost, h.smtpPortSTARTTLS, h.smtpPortTLS)
	if err != nil {
		switch {
		case errors.Is(err, credentialsmodule.ErrCredentialInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		case errors.Is(err, credentialsmodule.ErrCredentialNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		case errors.Is(err, credentialsmodule.ErrCredentialForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, credentialsmodule.ErrCredentialDisabled):
			c.JSON(http.StatusConflict, gin.H{"error": "credential is disabled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate credential"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}
