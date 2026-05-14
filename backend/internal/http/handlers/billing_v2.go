package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	billingmodule "zxmail/backend/internal/modules/billing"
)

type BillingV2Service interface {
	ListPlans(ctx context.Context, actor authmodule.AuthenticatedUser) ([]billingmodule.Plan, error)
	CreatePlan(ctx context.Context, actor authmodule.AuthenticatedUser, input billingmodule.PlanInput) (*billingmodule.Plan, error)
	UpdatePlan(ctx context.Context, actor authmodule.AuthenticatedUser, planID string, input billingmodule.PlanInput) (*billingmodule.Plan, error)
	AssignSubscription(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, input billingmodule.AssignSubscriptionInput) (*billingmodule.SubscriptionView, error)
	GetSubscription(ctx context.Context, actor authmodule.AuthenticatedUser) (*billingmodule.SubscriptionView, error)
	ListInvoices(ctx context.Context, actor authmodule.AuthenticatedUser) ([]billingmodule.Invoice, error)
	ListAdminInvoices(ctx context.Context, actor authmodule.AuthenticatedUser) ([]billingmodule.Invoice, error)
	MarkInvoicePaid(ctx context.Context, actor authmodule.AuthenticatedUser, invoiceID string) (*billingmodule.Invoice, error)
	MarkInvoiceFailed(ctx context.Context, actor authmodule.AuthenticatedUser, invoiceID string) (*billingmodule.Invoice, error)
	ListPayments(ctx context.Context, actor authmodule.AuthenticatedUser) ([]billingmodule.Payment, error)
	ApprovePayment(ctx context.Context, actor authmodule.AuthenticatedUser, paymentID string) (*billingmodule.Payment, error)
	RejectPayment(ctx context.Context, actor authmodule.AuthenticatedUser, paymentID string) (*billingmodule.Payment, error)
}

type BillingV2Handler struct {
	service BillingV2Service
}

func NewBillingV2Handler(service BillingV2Service) *BillingV2Handler {
	return &BillingV2Handler{service: service}
}

func (h *BillingV2Handler) ListPlans(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	plans, err := h.service.ListPlans(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list plans"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *BillingV2Handler) CreatePlan(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request billingmodule.PlanInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan payload"})
		return
	}
	plan, err := h.service.CreatePlan(c.Request.Context(), actor, request)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, billingmodule.ErrPlanInvalidInput), errors.Is(err, billingmodule.ErrUnsupportedPaymentProvider):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, billingmodule.ErrPlanCodeAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "plan code already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create plan"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan": plan})
}

func (h *BillingV2Handler) UpdatePlan(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request billingmodule.PlanInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan payload"})
		return
	}
	plan, err := h.service.UpdatePlan(c.Request.Context(), actor, c.Param("id"), request)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, billingmodule.ErrPlanInvalidInput), errors.Is(err, billingmodule.ErrUnsupportedPaymentProvider):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, billingmodule.ErrPlanNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		case errors.Is(err, billingmodule.ErrPlanCodeAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "plan code already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plan"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"plan": plan})
}

func (h *BillingV2Handler) AssignSubscription(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var request billingmodule.AssignSubscriptionInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription payload"})
		return
	}
	response, err := h.service.AssignSubscription(c.Request.Context(), actor, c.Param("id"), request)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, billingmodule.ErrPlanInvalidInput), errors.Is(err, billingmodule.ErrUnsupportedPaymentProvider):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, billingmodule.ErrPlanNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		case errors.Is(err, billingmodule.ErrOrganizationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign subscription"})
		}
		return
	}
	c.JSON(http.StatusCreated, response)
}

func (h *BillingV2Handler) GetSubscription(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	response, err := h.service.GetSubscription(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrSubscriptionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load subscription"})
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BillingV2Handler) ListInvoices(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	invoices, err := h.service.ListInvoices(c.Request.Context(), actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

func (h *BillingV2Handler) ListAdminInvoices(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	invoices, err := h.service.ListAdminInvoices(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admin invoices"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

func (h *BillingV2Handler) MarkInvoicePaid(c *gin.Context) {
	h.transitionInvoice(c, true)
}

func (h *BillingV2Handler) MarkInvoiceFailed(c *gin.Context) {
	h.transitionInvoice(c, false)
}

func (h *BillingV2Handler) ListPayments(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	payments, err := h.service.ListPayments(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list payments"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

func (h *BillingV2Handler) ApprovePayment(c *gin.Context) {
	h.transitionPayment(c, true)
}

func (h *BillingV2Handler) RejectPayment(c *gin.Context) {
	h.transitionPayment(c, false)
}

func (h *BillingV2Handler) transitionInvoice(c *gin.Context, paid bool) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var (
		invoice *billingmodule.Invoice
		err     error
	)
	if paid {
		invoice, err = h.service.MarkInvoicePaid(c.Request.Context(), actor, c.Param("id"))
	} else {
		invoice, err = h.service.MarkInvoiceFailed(c.Request.Context(), actor, c.Param("id"))
	}
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, billingmodule.ErrPlanInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		case errors.Is(err, billingmodule.ErrInvoiceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update invoice"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoice": invoice})
}

func (h *BillingV2Handler) transitionPayment(c *gin.Context, approve bool) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}
	var (
		payment *billingmodule.Payment
		err     error
	)
	if approve {
		payment, err = h.service.ApprovePayment(c.Request.Context(), actor, c.Param("id"))
	} else {
		payment, err = h.service.RejectPayment(c.Request.Context(), actor, c.Param("id"))
	}
	if err != nil {
		switch {
		case errors.Is(err, billingmodule.ErrBillingForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, billingmodule.ErrPaymentInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment id"})
		case errors.Is(err, billingmodule.ErrPaymentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update payment"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"payment": payment})
}
