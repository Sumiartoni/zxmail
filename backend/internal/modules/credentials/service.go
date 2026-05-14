package credentials

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	authmodule "zxmail/backend/internal/modules/auth"
	quotamodule "zxmail/backend/internal/modules/quota"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
	"zxmail/backend/internal/platform/security"
)

var (
	ErrCredentialInvalidInput = errors.New("invalid credential input")
	ErrCredentialForbidden    = errors.New("credential forbidden")
	ErrCredentialNotFound     = errors.New("credential not found")
	ErrCredentialDomainState  = errors.New("domain must be verified")
	ErrCredentialDisabled     = errors.New("credential disabled")
	ErrCredentialLimited      = errors.New("organization billing or suspension state blocks credential issuance")
)

type Service struct {
	db      *pgxpool.Pool
	log     *logger.Logger
	keyring *security.Keyring
	quota   *quotamodule.Service
}

type Credential struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	DomainID        uuid.UUID  `json:"domain_id"`
	DomainName      string     `json:"domain_name"`
	Username        string     `json:"username"`
	Label           string     `json:"label,omitempty"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	PerMinuteLimit  *int       `json:"per_minute_limit,omitempty"`
	PerMinuteUsed   int        `json:"per_minute_used"`
	DailyLimit      *int       `json:"daily_limit,omitempty"`
	DailyUsed       int        `json:"daily_used"`
	MonthlyLimit    *int       `json:"monthly_limit,omitempty"`
	MonthlyUsed     int        `json:"monthly_used"`
	Limited         bool       `json:"limited"`
	Exceeded        []string   `json:"exceeded,omitempty"`
	EnforcementNote string     `json:"enforcement_note"`
}

type SMTPConnectionInfo struct {
	Host         string `json:"host"`
	StartTLSPort string `json:"starttls_port"`
	TLSPort      string `json:"tls_port"`
	Username     string `json:"username"`
	PasswordNote string `json:"password_note"`
}

type CredentialResponse struct {
	Credential Credential         `json:"credential"`
	SMTP       SMTPConnectionInfo `json:"smtp"`
}

type CredentialSecretResponse struct {
	Credential Credential         `json:"credential"`
	SMTP       SMTPConnectionInfo `json:"smtp"`
	Secret     string             `json:"secret"`
}

type CreateCredentialInput struct {
	DomainID       string `json:"domain_id"`
	Label          string `json:"label"`
	PerMinuteLimit *int   `json:"per_minute_limit,omitempty"`
	DailyLimit     *int   `json:"daily_limit,omitempty"`
	MonthlyLimit   *int   `json:"monthly_limit,omitempty"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger, keyring *security.Keyring, quotaService *quotamodule.Service) *Service {
	return &Service{
		db:      db,
		log:     log,
		keyring: keyring,
		quota:   quotaService,
	}
}

func (s *Service) List(ctx context.Context, actor authmodule.AuthenticatedUser, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) ([]CredentialResponse, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if actor.Role == authmodule.RoleAdmin {
		rows, err = s.db.Query(
			ctx,
			`SELECT c.id, c.organization_id, c.domain_id, d.name, c.username, COALESCE(c.label, ''), c.enabled,
			        c.created_at, c.last_used_at, c.quota_per_minute_limit, c.quota_daily_limit, c.quota_daily_used, c.quota_monthly_limit, c.quota_monthly_used
			 FROM smtp_credentials c
			 JOIN domains d ON d.id = c.domain_id
			 ORDER BY c.created_at DESC`,
		)
	} else {
		orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
		if err != nil {
			return nil, err
		}
		rows, err = s.db.Query(
			ctx,
			`SELECT c.id, c.organization_id, c.domain_id, d.name, c.username, COALESCE(c.label, ''), c.enabled,
			        c.created_at, c.last_used_at, c.quota_per_minute_limit, c.quota_daily_limit, c.quota_daily_used, c.quota_monthly_limit, c.quota_monthly_used
			 FROM smtp_credentials c
			 JOIN domains d ON d.id = c.domain_id
			 WHERE c.organization_id = $1
			 ORDER BY c.created_at DESC`,
			orgID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []CredentialResponse
	for rows.Next() {
		credential, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		if err := s.applyRuntimeState(ctx, &credential); err != nil {
			return nil, err
		}

		credentials = append(credentials, CredentialResponse{
			Credential: credential,
			SMTP:       buildSMTPInfo(smtpHost, smtpPortSTARTTLS, smtpPortTLS, credential.Username),
		})
	}

	return credentials, rows.Err()
}

func (s *Service) Create(ctx context.Context, actor authmodule.AuthenticatedUser, input CreateCredentialInput, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*CredentialSecretResponse, error) {
	domainID, err := uuid.Parse(strings.TrimSpace(input.DomainID))
	if err != nil {
		return nil, ErrCredentialInvalidInput
	}

	domain, err := s.loadAuthorizedVerifiedDomain(ctx, actor, domainID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureOrganizationAllowsCredentialCreate(ctx, domain.OrganizationID); err != nil {
		return nil, err
	}

	username, err := s.generateUniqueUsername(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}

	encrypted, err := s.keyring.Encrypt(secret)
	if err != nil {
		return nil, err
	}

	var credential Credential
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO smtp_credentials (
			organization_id, domain_id, username, password_enc, password_key_id, label, enabled,
			quota_per_minute_limit, quota_daily_limit, quota_daily_used, quota_monthly_limit, quota_monthly_used
		) VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, $8, 0, $9, 0)
		RETURNING id, organization_id, domain_id, username, COALESCE(label, ''), enabled, created_at,
		          last_used_at, quota_per_minute_limit, quota_daily_limit, quota_daily_used, quota_monthly_limit, quota_monthly_used`,
		domain.OrganizationID,
		domain.ID,
		username,
		encrypted.Ciphertext,
		encrypted.KeyID,
		strings.TrimSpace(input.Label),
		input.PerMinuteLimit,
		input.DailyLimit,
		input.MonthlyLimit,
	).Scan(
		&credential.ID,
		&credential.OrganizationID,
		&credential.DomainID,
		&credential.Username,
		&credential.Label,
		&credential.Enabled,
		&credential.CreatedAt,
		&credential.LastUsedAt,
		&credential.PerMinuteLimit,
		&credential.DailyLimit,
		&credential.DailyUsed,
		&credential.MonthlyLimit,
		&credential.MonthlyUsed,
	)
	if err != nil {
		return nil, err
	}

	credential.DomainName = domain.Name
	if err := s.applyRuntimeState(ctx, &credential); err != nil {
		return nil, err
	}

	if err := s.insertAuditLog(ctx, actor.ID, &credential.OrganizationID, "credential.create", "smtp_credential", credential.ID, map[string]any{
		"domain_id": credential.DomainID.String(),
		"username":  credential.Username,
	}); err != nil {
		return nil, err
	}

	return &CredentialSecretResponse{
		Credential: credential,
		SMTP:       buildSMTPInfo(smtpHost, smtpPortSTARTTLS, smtpPortTLS, credential.Username),
		Secret:     secret,
	}, nil
}

func (s *Service) Get(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*CredentialResponse, error) {
	credentialID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrCredentialInvalidInput
	}

	credential, err := s.loadAuthorizedCredential(ctx, actor, credentialID)
	if err != nil {
		return nil, err
	}

	return &CredentialResponse{
		Credential: *credential,
		SMTP:       buildSMTPInfo(smtpHost, smtpPortSTARTTLS, smtpPortTLS, credential.Username),
	}, nil
}

func (s *Service) Revoke(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*CredentialResponse, error) {
	credentialID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrCredentialInvalidInput
	}

	credential, err := s.loadAuthorizedCredential(ctx, actor, credentialID)
	if err != nil {
		return nil, err
	}

	if _, err := s.db.Exec(ctx, `UPDATE smtp_credentials SET enabled = FALSE WHERE id = $1`, credentialID); err != nil {
		return nil, err
	}
	credential.Enabled = false
	if err := s.applyRuntimeState(ctx, credential); err != nil {
		return nil, err
	}

	if err := s.insertAuditLog(ctx, actor.ID, &credential.OrganizationID, "credential.revoke", "smtp_credential", credential.ID, map[string]any{
		"username": credential.Username,
	}); err != nil {
		return nil, err
	}

	return &CredentialResponse{
		Credential: *credential,
		SMTP:       buildSMTPInfo(smtpHost, smtpPortSTARTTLS, smtpPortTLS, credential.Username),
	}, nil
}

func (s *Service) Rotate(ctx context.Context, actor authmodule.AuthenticatedUser, id string, smtpHost, smtpPortSTARTTLS, smtpPortTLS string) (*CredentialSecretResponse, error) {
	credentialID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrCredentialInvalidInput
	}

	credential, err := s.loadAuthorizedCredential(ctx, actor, credentialID)
	if err != nil {
		return nil, err
	}
	if !credential.Enabled {
		return nil, ErrCredentialDisabled
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}

	encrypted, err := s.keyring.Encrypt(secret)
	if err != nil {
		return nil, err
	}

	if _, err := s.db.Exec(ctx, `UPDATE smtp_credentials SET password_enc = $2, password_key_id = $3 WHERE id = $1`, credentialID, encrypted.Ciphertext, encrypted.KeyID); err != nil {
		return nil, err
	}

	if err := s.insertAuditLog(ctx, actor.ID, &credential.OrganizationID, "credential.rotate", "smtp_credential", credential.ID, map[string]any{
		"username": credential.Username,
	}); err != nil {
		return nil, err
	}

	return &CredentialSecretResponse{
		Credential: *credential,
		SMTP:       buildSMTPInfo(smtpHost, smtpPortSTARTTLS, smtpPortTLS, credential.Username),
		Secret:     secret,
	}, nil
}

type authorizedDomain struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
}

func (s *Service) loadAuthorizedVerifiedDomain(ctx context.Context, actor authmodule.AuthenticatedUser, domainID uuid.UUID) (*authorizedDomain, error) {
	var domain authorizedDomain
	var verified bool
	err := s.db.QueryRow(
		ctx,
		`SELECT id, organization_id, name, verified FROM domains WHERE id = $1`,
		domainID,
	).Scan(&domain.ID, &domain.OrganizationID, &domain.Name, &verified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}

	if actor.Role != authmodule.RoleAdmin {
		orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if domain.OrganizationID != orgID {
			return nil, ErrCredentialForbidden
		}
	}

	if !verified {
		return nil, ErrCredentialDomainState
	}

	return &domain, nil
}

func (s *Service) loadAuthorizedCredential(ctx context.Context, actor authmodule.AuthenticatedUser, credentialID uuid.UUID) (*Credential, error) {
	row, err := s.db.Query(
		ctx,
			`SELECT c.id, c.organization_id, c.domain_id, d.name, c.username, COALESCE(c.label, ''), c.enabled,
			        c.created_at, c.last_used_at, c.quota_per_minute_limit, c.quota_daily_limit, c.quota_daily_used, c.quota_monthly_limit, c.quota_monthly_used
			 FROM smtp_credentials c
			 JOIN domains d ON d.id = c.domain_id
			 WHERE c.id = $1`,
		credentialID,
	)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrCredentialNotFound
	}

	credential, err := scanCredential(row)
	if err != nil {
		return nil, err
	}

	if actor.Role != authmodule.RoleAdmin {
		orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if credential.OrganizationID != orgID {
			return nil, ErrCredentialForbidden
		}
	}

	if err := s.applyRuntimeState(ctx, &credential); err != nil {
		return nil, err
	}

	return &credential, nil
}

func scanCredential(scanner interface {
	Scan(dest ...any) error
}) (Credential, error) {
	var credential Credential
	err := scanner.Scan(
		&credential.ID,
		&credential.OrganizationID,
		&credential.DomainID,
		&credential.DomainName,
		&credential.Username,
		&credential.Label,
		&credential.Enabled,
		&credential.CreatedAt,
		&credential.LastUsedAt,
		&credential.PerMinuteLimit,
		&credential.DailyLimit,
		&credential.DailyUsed,
		&credential.MonthlyLimit,
		&credential.MonthlyUsed,
	)
	if err != nil {
		return Credential{}, err
	}

	credential.Status = statusFromEnabled(credential.Enabled)
	credential.EnforcementNote = quotamodule.EnforcementNote
	return credential, nil
}

func (s *Service) resolveCustomerOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}

	var organizationID uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, actor.ID).Scan(&organizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrCredentialForbidden
		}
		return uuid.Nil, err
	}

	return organizationID, nil
}

func buildSMTPInfo(host, starttlsPort, tlsPort, username string) SMTPConnectionInfo {
	return SMTPConnectionInfo{
		Host:         host,
		StartTLSPort: starttlsPort,
		TLSPort:      tlsPort,
		Username:     username,
		PasswordNote: "Password is shown only once when the credential is created or rotated.",
	}
}

func statusFromEnabled(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func (s *Service) applyRuntimeState(ctx context.Context, credential *Credential) error {
	if s.quota == nil {
		credential.Status = statusFromEnabled(credential.Enabled)
		credential.EnforcementNote = quotamodule.EnforcementNote
		return nil
	}

	forcedReasons, forcedNote, err := s.organizationRuntimeRestrictions(ctx, credential.OrganizationID, credential.ID)
	if err != nil {
		return err
	}

	snapshot, err := s.quota.BuildSnapshot(ctx, quotamodule.SnapshotInput{
		CredentialID:   credential.ID,
		Enabled:        credential.Enabled,
		PerMinuteLimit: credential.PerMinuteLimit,
		DailyLimit:     credential.DailyLimit,
		DailyUsed:      credential.DailyUsed,
		MonthlyLimit:   credential.MonthlyLimit,
		MonthlyUsed:    credential.MonthlyUsed,
	})
	if err != nil {
		return err
	}

	credential.PerMinuteUsed = snapshot.PerMinuteUsed
	credential.Status = snapshot.Status
	credential.Limited = snapshot.Limited
	credential.Exceeded = snapshot.Exceeded
	credential.EnforcementNote = snapshot.EnforcementNote
	if len(forcedReasons) > 0 {
		credential.Status = "limited"
		credential.Limited = true
		credential.Exceeded = append(credential.Exceeded, forcedReasons...)
		if forcedNote != "" {
			credential.EnforcementNote = forcedNote
		}
	}
	return nil
}

func (s *Service) ensureOrganizationAllowsCredentialCreate(ctx context.Context, organizationID uuid.UUID) error {
	reasons, _, err := s.organizationRuntimeRestrictions(ctx, organizationID, uuid.Nil)
	if err != nil {
		return err
	}
	if len(reasons) > 0 {
		return ErrCredentialLimited
	}
	return nil
}

func (s *Service) organizationRuntimeRestrictions(ctx context.Context, organizationID uuid.UUID, credentialID uuid.UUID) ([]string, string, error) {
	var (
		suspended       bool
		subscriptionStatus string
		manuallyLimited bool
	)
	if err := s.db.QueryRow(ctx, `SELECT suspended FROM organizations WHERE id = $1`, organizationID).Scan(&suspended); err != nil {
		return nil, "", err
	}
	if err := s.db.QueryRow(ctx, `SELECT COALESCE((SELECT status FROM subscriptions WHERE organization_id = $1 ORDER BY created_at DESC LIMIT 1), 'missing')`, organizationID).Scan(&subscriptionStatus); err != nil {
		return nil, "", err
	}
	if credentialID != uuid.Nil {
		_ = s.db.QueryRow(ctx, `SELECT manually_limited FROM smtp_credentials WHERE id = $1`, credentialID).Scan(&manuallyLimited)
	}

	var reasons []string
	note := quotamodule.EnforcementNote
	if suspended {
		reasons = append(reasons, "organization_suspended")
		note = "Organization is suspended. Customer can still sign in, but new SMTP operations are restricted until the suspension is cleared."
	}
	switch subscriptionStatus {
	case "expired", "suspended", "past_due":
		reasons = append(reasons, "subscription_"+subscriptionStatus)
		note = "Organization billing or subscription state is limiting SMTP operations. Review the current subscription and payment status."
	}
	if manuallyLimited {
		reasons = append(reasons, "manual_limit")
		note = "This credential was manually limited by an administrator."
	}
	return reasons, note, nil
}

func generateSecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Service) generateUniqueUsername(ctx context.Context) (string, error) {
	for range 10 {
		suffix, err := randomToken(10)
		if err != nil {
			return "", err
		}

		username := "apikey_" + suffix
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM smtp_credentials WHERE username = $1)`, username).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}
	}

	return "", errors.New("failed to generate unique username")
}

func randomToken(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(random[i])%len(alphabet)]
	}
	return string(buf), nil
}

func (s *Service) insertAuditLog(
	ctx context.Context,
	actorUserID uuid.UUID,
	organizationID *uuid.UUID,
	action string,
	targetType string,
	targetID uuid.UUID,
	metadata map[string]any,
) error {
	payload, err := auditlog.MarshalAuditDetails(metadata)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		ctx,
		`INSERT INTO audit_logs (actor_user_id, organization_id, action, target_type, target_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		actorUserID,
		organizationID,
		action,
		targetType,
		targetID,
		payload,
	)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
