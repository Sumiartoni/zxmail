package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrPlanInvalidInput      = errors.New("invalid plan input")
	ErrPlanNotFound          = errors.New("plan not found")
	ErrBillingForbidden      = errors.New("billing forbidden")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrPaymentInvalidInput   = errors.New("invalid payment input")
	ErrSubscriptionConflict  = errors.New("subscription conflict")
	ErrInvoiceStateInvalid   = errors.New("invoice state invalid")
	ErrPaymentStateInvalid   = errors.New("payment state invalid")
	ErrPlanCodeAlreadyExists = errors.New("plan code already exists")
)

type Service struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

type Plan struct {
	ID                   uuid.UUID `json:"id"`
	Code                 string    `json:"code"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Active               bool      `json:"active"`
	Currency             string    `json:"currency"`
	PriceMonthly         int64     `json:"price_monthly"`
	DailyQuota           *int      `json:"daily_quota,omitempty"`
	MonthlyQuota         *int      `json:"monthly_quota,omitempty"`
	PerMinuteQuota       *int      `json:"per_minute_quota,omitempty"`
	CredentialQuota      *int      `json:"credential_quota,omitempty"`
	TrialDays            int       `json:"trial_days"`
	OveragePricePerEmail int64     `json:"overage_price_per_email"`
	PaymentMethods       []string  `json:"payment_methods"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type PlanInput struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Active               *bool    `json:"active,omitempty"`
	Currency             string   `json:"currency"`
	PriceMonthly         int64    `json:"price_monthly"`
	DailyQuota           *int     `json:"daily_quota,omitempty"`
	MonthlyQuota         *int     `json:"monthly_quota,omitempty"`
	PerMinuteQuota       *int     `json:"per_minute_quota,omitempty"`
	CredentialQuota      *int     `json:"credential_quota,omitempty"`
	TrialDays            int      `json:"trial_days"`
	OveragePricePerEmail int64    `json:"overage_price_per_email"`
	PaymentMethods       []string `json:"payment_methods"`
}

type AssignSubscriptionInput struct {
	PlanID          string `json:"plan_id"`
	PaymentProvider string `json:"payment_provider"`
	Notes           string `json:"notes"`
	StartTrial      bool   `json:"start_trial"`
}

type Subscription struct {
	ID                     uuid.UUID  `json:"id"`
	OrganizationID         uuid.UUID  `json:"organization_id"`
	PlanID                 uuid.UUID  `json:"plan_id"`
	Status                 string     `json:"status"`
	StartsAt               time.Time  `json:"starts_at"`
	CurrentPeriodStart     time.Time  `json:"current_period_start"`
	CurrentPeriodEnd       time.Time  `json:"current_period_end"`
	TrialEndsAt            *time.Time `json:"trial_ends_at,omitempty"`
	ExpiredAt              *time.Time `json:"expired_at,omitempty"`
	SuspendedAt            *time.Time `json:"suspended_at,omitempty"`
	QuotaDailyOverride     *int       `json:"quota_daily_override,omitempty"`
	QuotaMonthlyOverride   *int       `json:"quota_monthly_override,omitempty"`
	QuotaPerMinuteOverride *int       `json:"quota_per_minute_override,omitempty"`
	Notes                  string     `json:"notes,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type SubscriptionView struct {
	Subscription  Subscription `json:"subscription"`
	Plan          Plan         `json:"plan"`
	PaymentStatus string       `json:"payment_status"`
}

type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	InvoiceNumber  string     `json:"invoice_number"`
	Status         string     `json:"status"`
	Currency       string     `json:"currency"`
	Amount         int64      `json:"amount"`
	DueAt          *time.Time `json:"due_at,omitempty"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	IssuedAt       time.Time  `json:"issued_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
}

type Payment struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	InvoiceID      *uuid.UUID `json:"invoice_id,omitempty"`
	ProviderCode   string     `json:"provider_code"`
	Status         string     `json:"status"`
	Amount         int64      `json:"amount"`
	Reference      string     `json:"reference,omitempty"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	RejectedAt     *time.Time `json:"rejected_at,omitempty"`
	Notes          string     `json:"notes,omitempty"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) ListPlans(ctx context.Context, actor authmodule.AuthenticatedUser) ([]Plan, error) {
	query := `SELECT id, code, name, description, active, currency, price_monthly, daily_quota, monthly_quota, per_minute_quota, credential_quota, trial_days, overage_price_per_email, payment_methods, created_at, updated_at
	          FROM plans`
	if actor.Role != authmodule.RoleAdmin {
		query += ` WHERE active = TRUE`
	}
	query += ` ORDER BY price_monthly ASC, created_at ASC`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var plan Plan
		var paymentMethodsRaw []byte
		if err := rows.Scan(
			&plan.ID,
			&plan.Code,
			&plan.Name,
			&plan.Description,
			&plan.Active,
			&plan.Currency,
			&plan.PriceMonthly,
			&plan.DailyQuota,
			&plan.MonthlyQuota,
			&plan.PerMinuteQuota,
			&plan.CredentialQuota,
			&plan.TrialDays,
			&plan.OveragePricePerEmail,
			&paymentMethodsRaw,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		); err != nil {
			return nil, err
		}
		plan.PaymentMethods = unmarshalPaymentMethods(paymentMethodsRaw)
		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

func (s *Service) CreatePlan(ctx context.Context, actor authmodule.AuthenticatedUser, input PlanInput) (*Plan, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}
	normalized, err := normalizePlanInput(input, true)
	if err != nil {
		return nil, err
	}
	paymentMethodsJSON, err := marshalPaymentMethods(normalized.PaymentMethods)
	if err != nil {
		return nil, err
	}

	var plan Plan
	var paymentMethodsRaw []byte
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO plans (code, name, description, active, currency, price_monthly, daily_quota, monthly_quota, per_minute_quota, credential_quota, trial_days, overage_price_per_email, payment_methods)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id, code, name, description, active, currency, price_monthly, daily_quota, monthly_quota, per_minute_quota, credential_quota, trial_days, overage_price_per_email, payment_methods, created_at, updated_at`,
		normalized.Code,
		normalized.Name,
		normalized.Description,
		valueOrDefaultBool(normalized.Active, true),
		normalized.Currency,
		normalized.PriceMonthly,
		normalized.DailyQuota,
		normalized.MonthlyQuota,
		normalized.PerMinuteQuota,
		normalized.CredentialQuota,
		normalized.TrialDays,
		normalized.OveragePricePerEmail,
		paymentMethodsJSON,
	).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.Active,
		&plan.Currency,
		&plan.PriceMonthly,
		&plan.DailyQuota,
		&plan.MonthlyQuota,
		&plan.PerMinuteQuota,
		&plan.CredentialQuota,
		&plan.TrialDays,
		&plan.OveragePricePerEmail,
		&paymentMethodsRaw,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPlanCodeAlreadyExists
		}
		return nil, err
	}
	plan.PaymentMethods = unmarshalPaymentMethods(paymentMethodsRaw)

	actorID := actor.ID
	planID := plan.ID
	if err := auditlog.Insert(ctx, s.db, &actorID, nil, "billing.plan.create", "plan", &planID, map[string]any{
		"code": plan.Code,
		"name": plan.Name,
	}); err != nil {
		return nil, err
	}

	return &plan, nil
}

func (s *Service) UpdatePlan(ctx context.Context, actor authmodule.AuthenticatedUser, planID string, input PlanInput) (*Plan, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(planID))
	if err != nil {
		return nil, ErrPlanInvalidInput
	}
	normalized, err := normalizePlanInput(input, false)
	if err != nil {
		return nil, err
	}
	var paymentMethodsParam any
	if len(normalized.PaymentMethods) > 0 {
		paymentMethodsJSON, err := marshalPaymentMethods(normalized.PaymentMethods)
		if err != nil {
			return nil, err
		}
		paymentMethodsParam = paymentMethodsJSON
	}

	var plan Plan
	var paymentMethodsRaw []byte
	err = s.db.QueryRow(
		ctx,
		`UPDATE plans
		 SET code = COALESCE(NULLIF($2, ''), code),
		     name = COALESCE(NULLIF($3, ''), name),
		     description = COALESCE($4, description),
		     active = COALESCE($5, active),
		     currency = COALESCE(NULLIF($6, ''), currency),
		     price_monthly = COALESCE($7, price_monthly),
		     daily_quota = COALESCE($8, daily_quota),
		     monthly_quota = COALESCE($9, monthly_quota),
		     per_minute_quota = COALESCE($10, per_minute_quota),
		     credential_quota = COALESCE($11, credential_quota),
		     trial_days = COALESCE($12, trial_days),
		     overage_price_per_email = COALESCE($13, overage_price_per_email),
		     payment_methods = COALESCE($14, payment_methods),
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, code, name, description, active, currency, price_monthly, daily_quota, monthly_quota, per_minute_quota, credential_quota, trial_days, overage_price_per_email, payment_methods, created_at, updated_at`,
		parsedID,
		normalized.Code,
		normalized.Name,
		normalized.Description,
		normalized.Active,
		normalized.Currency,
		nullableInt64(normalized.PriceMonthly),
		normalized.DailyQuota,
		normalized.MonthlyQuota,
		normalized.PerMinuteQuota,
		normalized.CredentialQuota,
		nullableInt(normalized.TrialDays),
		nullableInt64(normalized.OveragePricePerEmail),
		paymentMethodsParam,
	).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.Active,
		&plan.Currency,
		&plan.PriceMonthly,
		&plan.DailyQuota,
		&plan.MonthlyQuota,
		&plan.PerMinuteQuota,
		&plan.CredentialQuota,
		&plan.TrialDays,
		&plan.OveragePricePerEmail,
		&paymentMethodsRaw,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrPlanCodeAlreadyExists
		}
		return nil, err
	}
	plan.PaymentMethods = unmarshalPaymentMethods(paymentMethodsRaw)

	actorID := actor.ID
	if err := auditlog.Insert(ctx, s.db, &actorID, nil, "billing.plan.update", "plan", &plan.ID, map[string]any{
		"code": plan.Code,
		"name": plan.Name,
	}); err != nil {
		return nil, err
	}

	return &plan, nil
}

func (s *Service) AssignSubscription(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID string, input AssignSubscriptionInput) (*SubscriptionView, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}

	orgID, err := uuid.Parse(strings.TrimSpace(organizationID))
	if err != nil {
		return nil, ErrPlanInvalidInput
	}
	planID, err := uuid.Parse(strings.TrimSpace(input.PlanID))
	if err != nil {
		return nil, ErrPlanInvalidInput
	}
	if !IsSupportedProvider(strings.TrimSpace(input.PaymentProvider)) {
		return nil, ErrUnsupportedPaymentProvider
	}

	plan, err := s.getPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	exists, err := s.organizationExists(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOrganizationNotFound
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`UPDATE subscriptions
		 SET status = 'canceled', updated_at = NOW()
		 WHERE organization_id = $1
		   AND status IN ('trialing', 'active', 'past_due', 'suspended')`,
		orgID,
	); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	periodStart := now
	periodEnd := now.Add(30 * 24 * time.Hour)
	status := "active"
	var trialEndsAt *time.Time
	if input.StartTrial && plan.TrialDays > 0 {
		status = "trialing"
		trialEnd := now.Add(time.Duration(plan.TrialDays) * 24 * time.Hour)
		trialEndsAt = &trialEnd
		periodEnd = trialEnd
	}

	var subscription Subscription
	err = tx.QueryRow(
		ctx,
		`INSERT INTO subscriptions (organization_id, plan_id, status, starts_at, current_period_start, current_period_end, trial_ends_at, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, organization_id, plan_id, status, starts_at, current_period_start, current_period_end, trial_ends_at, expired_at, suspended_at, quota_daily_override, quota_monthly_override, quota_per_minute_override, COALESCE(notes,''), created_at, updated_at`,
		orgID,
		plan.ID,
		status,
		now,
		periodStart,
		periodEnd,
		trialEndsAt,
		strings.TrimSpace(input.Notes),
	).Scan(
		&subscription.ID,
		&subscription.OrganizationID,
		&subscription.PlanID,
		&subscription.Status,
		&subscription.StartsAt,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.TrialEndsAt,
		&subscription.ExpiredAt,
		&subscription.SuspendedAt,
		&subscription.QuotaDailyOverride,
		&subscription.QuotaMonthlyOverride,
		&subscription.QuotaPerMinuteOverride,
		&subscription.Notes,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	paymentStatus := "not_required"
	if plan.PriceMonthly > 0 && status != "trialing" {
		invoice, payment, err := s.createInvoiceAndPayment(ctx, tx, orgID, subscription, *plan, strings.TrimSpace(input.PaymentProvider))
		if err != nil {
			return nil, err
		}
		if invoice.Status == "issued" {
			paymentStatus = payment.Status
			subscription.Status = "past_due"
			if _, err := tx.Exec(ctx, `UPDATE subscriptions SET status = 'past_due', updated_at = NOW() WHERE id = $1`, subscription.ID); err != nil {
				return nil, err
			}
		}
	}
	if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, orgID, subscription.Status); err != nil {
		return nil, err
	}

	actorID := actor.ID
	if err := auditlog.Insert(ctx, tx, &actorID, &orgID, "billing.subscription.assign", "subscription", &subscription.ID, map[string]any{
		"plan_id":   plan.ID.String(),
		"plan_code": plan.Code,
		"status":    subscription.Status,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &SubscriptionView{
		Subscription:  subscription,
		Plan:          *plan,
		PaymentStatus: paymentStatus,
	}, nil
}

func (s *Service) GetSubscription(ctx context.Context, actor authmodule.AuthenticatedUser) (*SubscriptionView, error) {
	orgID, err := s.resolveOrganizationID(ctx, actor)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(
		ctx,
		`SELECT s.id, s.organization_id, s.plan_id, s.status, s.starts_at, s.current_period_start, s.current_period_end,
		        s.trial_ends_at, s.expired_at, s.suspended_at, s.quota_daily_override, s.quota_monthly_override, s.quota_per_minute_override,
		        COALESCE(s.notes,''), s.created_at, s.updated_at,
		        p.id, p.code, p.name, p.description, p.active, p.currency, p.price_monthly, p.daily_quota, p.monthly_quota, p.per_minute_quota, p.credential_quota,
		        p.trial_days, p.overage_price_per_email, p.payment_methods, p.created_at, p.updated_at,
		        COALESCE((SELECT status FROM payments pay WHERE pay.organization_id = s.organization_id ORDER BY pay.created_at DESC LIMIT 1), 'not_required')
		 FROM subscriptions s
		 JOIN plans p ON p.id = s.plan_id
		 WHERE s.organization_id = $1
		 ORDER BY s.created_at DESC
		 LIMIT 1`,
		orgID,
	)

	var subscription Subscription
	var plan Plan
	var paymentMethodsRaw []byte
	var paymentStatus string
	if err := row.Scan(
		&subscription.ID,
		&subscription.OrganizationID,
		&subscription.PlanID,
		&subscription.Status,
		&subscription.StartsAt,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.TrialEndsAt,
		&subscription.ExpiredAt,
		&subscription.SuspendedAt,
		&subscription.QuotaDailyOverride,
		&subscription.QuotaMonthlyOverride,
		&subscription.QuotaPerMinuteOverride,
		&subscription.Notes,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.Active,
		&plan.Currency,
		&plan.PriceMonthly,
		&plan.DailyQuota,
		&plan.MonthlyQuota,
		&plan.PerMinuteQuota,
		&plan.CredentialQuota,
		&plan.TrialDays,
		&plan.OveragePricePerEmail,
		&paymentMethodsRaw,
		&plan.CreatedAt,
		&plan.UpdatedAt,
		&paymentStatus,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	plan.PaymentMethods = unmarshalPaymentMethods(paymentMethodsRaw)

	return &SubscriptionView{
		Subscription:  subscription,
		Plan:          plan,
		PaymentStatus: paymentStatus,
	}, nil
}

func (s *Service) ListInvoices(ctx context.Context, actor authmodule.AuthenticatedUser) ([]Invoice, error) {
	orgID, err := s.resolveOrganizationID(ctx, actor)
	if err != nil {
		return nil, err
	}
	return s.listInvoicesByOrganization(ctx, orgID)
}

func (s *Service) ListAdminInvoices(ctx context.Context, actor authmodule.AuthenticatedUser) ([]Invoice, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}
	rows, err := s.db.Query(
		ctx,
		`SELECT id, organization_id, subscription_id, invoice_number, status, currency, amount, due_at, period_start, period_end, issued_at, paid_at, failed_at
		 FROM invoices
		 ORDER BY issued_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var invoice Invoice
		if err := rows.Scan(
			&invoice.ID,
			&invoice.OrganizationID,
			&invoice.SubscriptionID,
			&invoice.InvoiceNumber,
			&invoice.Status,
			&invoice.Currency,
			&invoice.Amount,
			&invoice.DueAt,
			&invoice.PeriodStart,
			&invoice.PeriodEnd,
			&invoice.IssuedAt,
			&invoice.PaidAt,
			&invoice.FailedAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}

	return invoices, rows.Err()
}

func (s *Service) listInvoicesByOrganization(ctx context.Context, orgID uuid.UUID) ([]Invoice, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT id, organization_id, subscription_id, invoice_number, status, currency, amount, due_at, period_start, period_end, issued_at, paid_at, failed_at
		 FROM invoices
		 WHERE organization_id = $1
		 ORDER BY issued_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var invoice Invoice
		if err := rows.Scan(
			&invoice.ID,
			&invoice.OrganizationID,
			&invoice.SubscriptionID,
			&invoice.InvoiceNumber,
			&invoice.Status,
			&invoice.Currency,
			&invoice.Amount,
			&invoice.DueAt,
			&invoice.PeriodStart,
			&invoice.PeriodEnd,
			&invoice.IssuedAt,
			&invoice.PaidAt,
			&invoice.FailedAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}

	return invoices, rows.Err()
}

func (s *Service) MarkInvoicePaid(ctx context.Context, actor authmodule.AuthenticatedUser, invoiceID string) (*Invoice, error) {
	return s.transitionInvoice(ctx, actor, invoiceID, "paid")
}

func (s *Service) MarkInvoiceFailed(ctx context.Context, actor authmodule.AuthenticatedUser, invoiceID string) (*Invoice, error) {
	return s.transitionInvoice(ctx, actor, invoiceID, "failed")
}

func (s *Service) ListPayments(ctx context.Context, actor authmodule.AuthenticatedUser) ([]Payment, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}

	rows, err := s.db.Query(
		ctx,
		`SELECT id, organization_id, invoice_id, provider_code, status, amount, COALESCE(reference, ''), submitted_at, approved_at, rejected_at, COALESCE(notes, '')
		 FROM payments
		 ORDER BY submitted_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrganizationID,
			&payment.InvoiceID,
			&payment.ProviderCode,
			&payment.Status,
			&payment.Amount,
			&payment.Reference,
			&payment.SubmittedAt,
			&payment.ApprovedAt,
			&payment.RejectedAt,
			&payment.Notes,
		); err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	return payments, rows.Err()
}

func (s *Service) ApprovePayment(ctx context.Context, actor authmodule.AuthenticatedUser, paymentID string) (*Payment, error) {
	return s.transitionPayment(ctx, actor, paymentID, "approved")
}

func (s *Service) RejectPayment(ctx context.Context, actor authmodule.AuthenticatedUser, paymentID string) (*Payment, error) {
	return s.transitionPayment(ctx, actor, paymentID, "rejected")
}

func (s *Service) LatestSubscriptionState(ctx context.Context, organizationID uuid.UUID) (string, error) {
	var status string
	err := s.db.QueryRow(
		ctx,
		`SELECT status
		 FROM subscriptions
		 WHERE organization_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		organizationID,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "missing", nil
		}
		return "", err
	}
	return status, nil
}

func (s *Service) RunSubscriptionExpiryCheck(ctx context.Context, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(
		ctx,
		`UPDATE subscriptions
		 SET status = 'expired',
		     expired_at = COALESCE(expired_at, $2),
		     updated_at = NOW()
		 WHERE status IN ('trialing', 'active', 'past_due')
		   AND current_period_end < $1
		 RETURNING id, organization_id`,
		now.UTC(),
		now.UTC(),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredRecord struct {
		subscriptionID uuid.UUID
		organizationID uuid.UUID
	}

	var expired []expiredRecord
	for rows.Next() {
		var record expiredRecord
		if err := rows.Scan(&record.subscriptionID, &record.organizationID); err != nil {
			return err
		}
		expired = append(expired, record)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, record := range expired {
		if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, record.organizationID, "expired"); err != nil {
			return err
		}
		if err := auditlog.Insert(ctx, tx, nil, &record.organizationID, "billing.subscription.expire", "subscription", &record.subscriptionID, map[string]any{
			"reason":     "current_period_end_passed",
			"expired_at": now.UTC(),
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) createInvoiceAndPayment(ctx context.Context, tx pgx.Tx, organizationID uuid.UUID, subscription Subscription, plan Plan, providerCode string) (*Invoice, *Payment, error) {
	now := time.Now().UTC()
	invoiceNumber := fmt.Sprintf("ZXM-%s-%s", now.Format("20060102"), strings.ToUpper(subscription.ID.String()[:8]))

	var invoice Invoice
	err := tx.QueryRow(
		ctx,
		`INSERT INTO invoices (organization_id, subscription_id, invoice_number, status, currency, amount, due_at, period_start, period_end, issued_at)
		 VALUES ($1,$2,$3,'issued',$4,$5,$6,$7,$8,$9)
		 RETURNING id, organization_id, subscription_id, invoice_number, status, currency, amount, due_at, period_start, period_end, issued_at, paid_at, failed_at`,
		organizationID,
		subscription.ID,
		invoiceNumber,
		plan.Currency,
		plan.PriceMonthly,
		now.Add(7*24*time.Hour),
		subscription.CurrentPeriodStart,
		subscription.CurrentPeriodEnd,
		now,
	).Scan(
		&invoice.ID,
		&invoice.OrganizationID,
		&invoice.SubscriptionID,
		&invoice.InvoiceNumber,
		&invoice.Status,
		&invoice.Currency,
		&invoice.Amount,
		&invoice.DueAt,
		&invoice.PeriodStart,
		&invoice.PeriodEnd,
		&invoice.IssuedAt,
		&invoice.PaidAt,
		&invoice.FailedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	var payment Payment
	err = tx.QueryRow(
		ctx,
		`INSERT INTO payments (organization_id, invoice_id, provider_code, status, amount, submitted_at, notes)
		 VALUES ($1,$2,$3,'pending',$4,$5,$6)
		 RETURNING id, organization_id, invoice_id, provider_code, status, amount, COALESCE(reference,''), submitted_at, approved_at, rejected_at, COALESCE(notes,'')`,
		organizationID,
		invoice.ID,
		providerCode,
		plan.PriceMonthly,
		now,
		"Awaiting manual payment confirmation.",
	).Scan(
		&payment.ID,
		&payment.OrganizationID,
		&payment.InvoiceID,
		&payment.ProviderCode,
		&payment.Status,
		&payment.Amount,
		&payment.Reference,
		&payment.SubmittedAt,
		&payment.ApprovedAt,
		&payment.RejectedAt,
		&payment.Notes,
	)
	if err != nil {
		return nil, nil, err
	}

	return &invoice, &payment, nil
}

func (s *Service) transitionInvoice(ctx context.Context, actor authmodule.AuthenticatedUser, invoiceID string, target string) (*Invoice, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(invoiceID))
	if err != nil {
		return nil, ErrPlanInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var invoice Invoice
	err = tx.QueryRow(
		ctx,
		`SELECT id, organization_id, subscription_id, invoice_number, status, currency, amount, due_at, period_start, period_end, issued_at, paid_at, failed_at
		 FROM invoices WHERE id = $1`,
		parsedID,
	).Scan(
		&invoice.ID,
		&invoice.OrganizationID,
		&invoice.SubscriptionID,
		&invoice.InvoiceNumber,
		&invoice.Status,
		&invoice.Currency,
		&invoice.Amount,
		&invoice.DueAt,
		&invoice.PeriodStart,
		&invoice.PeriodEnd,
		&invoice.IssuedAt,
		&invoice.PaidAt,
		&invoice.FailedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	switch target {
	case "paid":
		if _, err := tx.Exec(ctx, `UPDATE invoices SET status = 'paid', paid_at = $2, updated_at = NOW() WHERE id = $1`, parsedID, now); err != nil {
			return nil, err
		}
		if invoice.SubscriptionID != nil {
			if _, err := tx.Exec(ctx, `UPDATE subscriptions SET status = 'active', expired_at = NULL, updated_at = NOW() WHERE id = $1`, *invoice.SubscriptionID); err != nil {
				return nil, err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'approved', approved_at = $2, updated_at = NOW() WHERE invoice_id = $1 AND status = 'pending'`, parsedID, now); err != nil {
			return nil, err
		}
		if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, invoice.OrganizationID, "active"); err != nil {
			return nil, err
		}
		invoice.Status = "paid"
		invoice.PaidAt = &now
	case "failed":
		if _, err := tx.Exec(ctx, `UPDATE invoices SET status = 'failed', failed_at = $2, updated_at = NOW() WHERE id = $1`, parsedID, now); err != nil {
			return nil, err
		}
		if invoice.SubscriptionID != nil {
			if _, err := tx.Exec(ctx, `UPDATE subscriptions SET status = 'past_due', expired_at = $2, updated_at = NOW() WHERE id = $1`, *invoice.SubscriptionID, now); err != nil {
				return nil, err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'failed', updated_at = NOW() WHERE invoice_id = $1 AND status = 'pending'`, parsedID); err != nil {
			return nil, err
		}
		if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, invoice.OrganizationID, "past_due"); err != nil {
			return nil, err
		}
		invoice.Status = "failed"
		invoice.FailedAt = &now
	default:
		return nil, ErrInvoiceStateInvalid
	}

	actorID := actor.ID
	if err := auditlog.Insert(ctx, tx, &actorID, &invoice.OrganizationID, "billing.invoice."+target, "invoice", &invoice.ID, map[string]any{
		"invoice_number": invoice.InvoiceNumber,
		"status":         target,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &invoice, nil
}

func (s *Service) transitionPayment(ctx context.Context, actor authmodule.AuthenticatedUser, paymentID string, target string) (*Payment, error) {
	if actor.Role != authmodule.RoleAdmin {
		return nil, ErrBillingForbidden
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(paymentID))
	if err != nil {
		return nil, ErrPaymentInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var payment Payment
	err = tx.QueryRow(
		ctx,
		`SELECT id, organization_id, invoice_id, provider_code, status, amount, COALESCE(reference,''), submitted_at, approved_at, rejected_at, COALESCE(notes,'')
		 FROM payments WHERE id = $1`,
		parsedID,
	).Scan(
		&payment.ID,
		&payment.OrganizationID,
		&payment.InvoiceID,
		&payment.ProviderCode,
		&payment.Status,
		&payment.Amount,
		&payment.Reference,
		&payment.SubmittedAt,
		&payment.ApprovedAt,
		&payment.RejectedAt,
		&payment.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	switch target {
	case "approved":
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'approved', approved_at = $2, updated_at = NOW() WHERE id = $1`, parsedID, now); err != nil {
			return nil, err
		}
		payment.Status = "approved"
		payment.ApprovedAt = &now
		if payment.InvoiceID != nil {
			if _, err := tx.Exec(ctx, `UPDATE invoices SET status = 'paid', paid_at = $2, updated_at = NOW() WHERE id = $1`, *payment.InvoiceID, now); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `UPDATE subscriptions SET status = 'active', expired_at = NULL, updated_at = NOW() WHERE id = (SELECT subscription_id FROM invoices WHERE id = $1)`, *payment.InvoiceID); err != nil {
				return nil, err
			}
		}
		if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, payment.OrganizationID, "active"); err != nil {
			return nil, err
		}
	case "rejected":
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'rejected', rejected_at = $2, updated_at = NOW() WHERE id = $1`, parsedID, now); err != nil {
			return nil, err
		}
		payment.Status = "rejected"
		payment.RejectedAt = &now
		if payment.InvoiceID != nil {
			if _, err := tx.Exec(ctx, `UPDATE invoices SET status = 'failed', failed_at = $2, updated_at = NOW() WHERE id = $1`, *payment.InvoiceID, now); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `UPDATE subscriptions SET status = 'past_due', expired_at = $2, updated_at = NOW() WHERE id = (SELECT subscription_id FROM invoices WHERE id = $1)`, *payment.InvoiceID, now); err != nil {
				return nil, err
			}
		}
		if err := s.syncOrganizationCredentialBillingLimit(ctx, tx, payment.OrganizationID, "past_due"); err != nil {
			return nil, err
		}
	default:
		return nil, ErrPaymentStateInvalid
	}

	actorID := actor.ID
	if err := auditlog.Insert(ctx, tx, &actorID, &payment.OrganizationID, "billing.payment."+target, "payment", &payment.ID, map[string]any{
		"provider_code": payment.ProviderCode,
		"status":        target,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &payment, nil
}

func (s *Service) getPlanByID(ctx context.Context, planID uuid.UUID) (*Plan, error) {
	var plan Plan
	var paymentMethodsRaw []byte
	err := s.db.QueryRow(
		ctx,
		`SELECT id, code, name, description, active, currency, price_monthly, daily_quota, monthly_quota, per_minute_quota, credential_quota, trial_days, overage_price_per_email, payment_methods, created_at, updated_at
		 FROM plans WHERE id = $1`,
		planID,
	).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.Active,
		&plan.Currency,
		&plan.PriceMonthly,
		&plan.DailyQuota,
		&plan.MonthlyQuota,
		&plan.PerMinuteQuota,
		&plan.CredentialQuota,
		&plan.TrialDays,
		&plan.OveragePricePerEmail,
		&paymentMethodsRaw,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}
	plan.PaymentMethods = unmarshalPaymentMethods(paymentMethodsRaw)
	return &plan, nil
}

func (s *Service) organizationExists(ctx context.Context, organizationID uuid.UUID) (bool, error) {
	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1)`, organizationID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Service) resolveOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.Role == authmodule.RoleAdmin && actor.OrganizationID == nil {
		return uuid.Nil, ErrBillingForbidden
	}
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}

	var organizationID uuid.UUID
	if err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, actor.ID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrOrganizationNotFound
		}
		return uuid.Nil, err
	}
	return organizationID, nil
}

func (s *Service) syncOrganizationCredentialBillingLimit(ctx context.Context, tx pgx.Tx, organizationID uuid.UUID, subscriptionStatus string) error {
	normalizedStatus := strings.TrimSpace(strings.ToLower(subscriptionStatus))
	if subscriptionStatusRequiresCredentialLimit(normalizedStatus) {
		_, err := tx.Exec(
			ctx,
			`UPDATE smtp_credentials
			 SET manually_limited = TRUE,
			     manual_limit_reason = $2,
			     manual_limit_updated_at = NOW()
			 WHERE organization_id = $1
			   AND (manually_limited = FALSE OR COALESCE(manual_limit_reason, '') LIKE 'subscription_%')`,
			organizationID,
			"subscription_"+normalizedStatus,
		)
		return err
	}

	_, err := tx.Exec(
		ctx,
		`UPDATE smtp_credentials
		 SET manually_limited = FALSE,
		     manual_limit_reason = NULL,
		     manual_limit_updated_at = NOW()
		 WHERE organization_id = $1
		   AND COALESCE(manual_limit_reason, '') LIKE 'subscription_%'`,
		organizationID,
	)
	return err
}

func subscriptionStatusRequiresCredentialLimit(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "expired", "suspended", "past_due":
		return true
	default:
		return false
	}
}

func normalizePlanInput(input PlanInput, requireCode bool) (PlanInput, error) {
	input.Code = strings.TrimSpace(strings.ToLower(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Currency == "" {
		input.Currency = "IDR"
	}
	if requireCode && (input.Code == "" || input.Name == "") {
		return PlanInput{}, ErrPlanInvalidInput
	}
	if input.PriceMonthly < 0 || input.OveragePricePerEmail < 0 || input.TrialDays < 0 {
		return PlanInput{}, ErrPlanInvalidInput
	}
	for _, value := range []*int{input.DailyQuota, input.MonthlyQuota, input.PerMinuteQuota, input.CredentialQuota} {
		if value != nil && *value < 0 {
			return PlanInput{}, ErrPlanInvalidInput
		}
	}
	var paymentMethods []string
	for _, method := range input.PaymentMethods {
		method = strings.TrimSpace(method)
		if method == "" {
			continue
		}
		if !IsSupportedProvider(method) {
			return PlanInput{}, ErrUnsupportedPaymentProvider
		}
		paymentMethods = append(paymentMethods, method)
	}
	if len(paymentMethods) == 0 && requireCode {
		paymentMethods = []string{"manual_bank_transfer", "manual_qris"}
	}
	input.PaymentMethods = paymentMethods
	return input, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func valueOrDefaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func nullableInt64(value int64) *int64 {
	return &value
}

func nullableInt(value int) *int {
	return &value
}

func marshalPaymentMethods(values []string) ([]byte, error) {
	if len(values) == 0 {
		values = []string{}
	}
	return json.Marshal(values)
}

func unmarshalPaymentMethods(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var methods []string
	if err := json.Unmarshal(raw, &methods); err != nil {
		return []string{}
	}
	return methods
}
