package domains

import (
	"context"
	"errors"
	"net"
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

const cloudflareSMTPWarning = "SMTP records must be DNS only in Cloudflare, not proxied."

var (
	ErrDomainNotFound     = errors.New("domain not found")
	ErrDomainForbidden    = errors.New("domain forbidden")
	ErrDomainInvalidInput = errors.New("invalid domain input")
	ErrDomainConflict     = errors.New("domain already exists")
	ErrOrganizationNeeded = errors.New("organization required")
)

type Service struct {
	db        *pgxpool.Pool
	log       *logger.Logger
	resolvers []string
}

type Domain struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Name           string     `json:"name"`
	Verified       bool       `json:"verified"`
	DKIMSelector   string     `json:"dkim_selector"`
	CreatedAt      time.Time  `json:"created_at"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
}

type DNSRequirement struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
	Note     string `json:"note"`
}

type DNSCheck struct {
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	ExpectedValue string    `json:"expected_value"`
	FoundValue    string    `json:"found_value,omitempty"`
	Found         bool      `json:"found"`
	Required      bool      `json:"required"`
	CheckedAt     time.Time `json:"checked_at"`
}

type DomainResponse struct {
	Domain          Domain           `json:"domain"`
	DNSRequirements []DNSRequirement `json:"dns_requirements"`
	DNSChecks       []DNSCheck       `json:"dns_checks"`
	Warnings        []string         `json:"warnings"`
}

type DomainListItem struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Verified             bool       `json:"verified"`
	DKIMSelector         string     `json:"dkim_selector"`
	CreatedAt            time.Time  `json:"created_at"`
	VerifiedAt           *time.Time `json:"verified_at,omitempty"`
	RequiredRecordsTotal int        `json:"required_records_total"`
	RequiredRecordsFound int        `json:"required_records_found"`
}

type VerificationResult struct {
	Status               string     `json:"status"`
	Verified             bool       `json:"verified"`
	VerifiedAt           *time.Time `json:"verified_at,omitempty"`
	RequiredRecordsTotal int        `json:"required_records_total"`
	RequiredRecordsFound int        `json:"required_records_found"`
	DNSChecks            []DNSCheck `json:"dns_checks"`
	Warnings             []string   `json:"warnings"`
}

type CreateDomainInput struct {
	Name           string     `json:"name"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`
}

func NewService(db *pgxpool.Pool, log *logger.Logger) *Service {
	return &Service{
		db:        db,
		log:       log,
		resolvers: []string{"1.1.1.1:53", "8.8.8.8:53"},
	}
}

func (s *Service) List(ctx context.Context, actor authmodule.AuthenticatedUser) ([]DomainListItem, error) {
	if actor.Role == authmodule.RoleAdmin {
		rows, err := s.db.Query(
			ctx,
			`SELECT d.id, d.name, d.verified, d.dkim_selector, d.created_at, d.verified_at
			 FROM domains d
			 ORDER BY d.created_at DESC`,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var domains []DomainListItem
		for rows.Next() {
			var item DomainListItem
			if err := rows.Scan(&item.ID, &item.Name, &item.Verified, &item.DKIMSelector, &item.CreatedAt, &item.VerifiedAt); err != nil {
				return nil, err
			}
			item.RequiredRecordsTotal = 3
			item.RequiredRecordsFound = 3
			if !item.Verified {
				item.RequiredRecordsFound = 0
			}
			domains = append(domains, item)
		}

		return domains, rows.Err()
	}

	orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		ctx,
		`SELECT d.id, d.name, d.verified, d.dkim_selector, d.created_at, d.verified_at
		 FROM domains d
		 WHERE d.organization_id = $1
		 ORDER BY d.created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []DomainListItem
	for rows.Next() {
		var item DomainListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Verified, &item.DKIMSelector, &item.CreatedAt, &item.VerifiedAt); err != nil {
			return nil, err
		}

		found, err := s.countLatestRequiredChecks(ctx, item.ID)
		if err != nil {
			return nil, err
		}

		item.RequiredRecordsTotal = 3
		item.RequiredRecordsFound = found
		domains = append(domains, item)
	}

	return domains, rows.Err()
}

func (s *Service) Create(ctx context.Context, actor authmodule.AuthenticatedUser, input CreateDomainInput) (*DomainResponse, error) {
	domainName := normalizeDomain(input.Name)
	if domainName == "" {
		return nil, ErrDomainInvalidInput
	}

	orgID, err := s.resolveOrganizationForCreate(ctx, actor, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	recordSpec := buildRecordSpec(domainName)
	var domain Domain
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO domains (organization_id, name, verified, dkim_selector, dkim_public, spf_record, dmarc_record)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, organization_id, name, verified, dkim_selector, created_at, verified_at`,
		orgID,
		domainName,
		false,
		recordSpec.dkimSelector,
		recordSpec.dkimPublic,
		recordSpec.spfValue,
		recordSpec.dmarcValue,
	).Scan(
		&domain.ID,
		&domain.OrganizationID,
		&domain.Name,
		&domain.Verified,
		&domain.DKIMSelector,
		&domain.CreatedAt,
		&domain.VerifiedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDomainConflict
		}
		return nil, err
	}

	if err := s.insertAuditLog(ctx, actor.ID, &domain.OrganizationID, "domain.create", "domain", domain.ID, map[string]any{
		"name": domain.Name,
	}); err != nil {
		return nil, err
	}

	return &DomainResponse{
		Domain:          domain,
		DNSRequirements: recordSpec.requirements(),
		DNSChecks:       []DNSCheck{},
		Warnings:        buildWarnings(),
	}, nil
}

func (s *Service) Get(ctx context.Context, actor authmodule.AuthenticatedUser, id string) (*DomainResponse, error) {
	domainID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrDomainInvalidInput
	}

	domainRow, spec, err := s.loadAuthorizedDomain(ctx, actor, domainID)
	if err != nil {
		return nil, err
	}

	checks, err := s.loadLatestChecks(ctx, domainID)
	if err != nil {
		return nil, err
	}

	return &DomainResponse{
		Domain:          *domainRow,
		DNSRequirements: spec.requirements(),
		DNSChecks:       checks,
		Warnings:        buildWarnings(),
	}, nil
}

func (s *Service) Verify(ctx context.Context, actor authmodule.AuthenticatedUser, id string) (*VerificationResult, error) {
	domainID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrDomainInvalidInput
	}

	domainRow, spec, err := s.loadAuthorizedDomain(ctx, actor, domainID)
	if err != nil {
		return nil, err
	}

	checks, err := s.checkAndPersistDNS(ctx, *domainRow, spec)
	if err != nil {
		return nil, err
	}

	requiredTotal := 0
	requiredFound := 0
	for _, check := range checks {
		if !check.Required {
			continue
		}
		requiredTotal++
		if check.Found {
			requiredFound++
		}
	}

	verified := requiredTotal > 0 && requiredTotal == requiredFound
	var verifiedAt *time.Time
	if verified {
		now := time.Now().UTC()
		verifiedAt = &now
		_, err = s.db.Exec(ctx, `UPDATE domains SET verified = TRUE, verified_at = $2 WHERE id = $1`, domainID, now)
	} else {
		_, err = s.db.Exec(ctx, `UPDATE domains SET verified = FALSE, verified_at = NULL WHERE id = $1`, domainID)
	}
	if err != nil {
		return nil, err
	}

	if err := s.insertAuditLog(ctx, actor.ID, &domainRow.OrganizationID, "domain.verify", "domain", domainRow.ID, map[string]any{
		"name":                   domainRow.Name,
		"verified":               verified,
		"required_records_found": requiredFound,
		"required_records_total": requiredTotal,
	}); err != nil {
		return nil, err
	}

	status := "pending"
	if verified {
		status = "verified"
	}

	return &VerificationResult{
		Status:               status,
		Verified:             verified,
		VerifiedAt:           verifiedAt,
		RequiredRecordsTotal: requiredTotal,
		RequiredRecordsFound: requiredFound,
		DNSChecks:            checks,
		Warnings:             buildWarnings(),
	}, nil
}

type recordSpec struct {
	domain       string
	dkimSelector string
	dkimPublic   string
	spfValue     string
	dmarcValue   string
}

func buildRecordSpec(domain string) recordSpec {
	return recordSpec{
		domain:       domain,
		dkimSelector: "postal",
		dkimPublic:   "v=DKIM1; k=rsa; p=REPLACE_WITH_POSTAL_DKIM_PUBLIC_KEY",
		spfValue:     "v=spf1 include:" + domain + " ~all",
		dmarcValue:   "v=DMARC1; p=none; rua=mailto:postmaster@" + domain,
	}
}

func (r recordSpec) requirements() []DNSRequirement {
	return []DNSRequirement{
		{
			Type:     "TXT",
			Name:     r.domain,
			Value:    r.spfValue,
			Required: true,
			Note:     "SPF record for transactional sending.",
		},
		{
			Type:     "TXT",
			Name:     r.dkimSelector + "._domainkey." + r.domain,
			Value:    r.dkimPublic,
			Required: true,
			Note:     "DKIM placeholder until Postal key wiring is added.",
		},
		{
			Type:     "TXT",
			Name:     "_dmarc." + r.domain,
			Value:    r.dmarcValue,
			Required: true,
			Note:     "DMARC baseline policy.",
		},
		{
			Type:     "MX",
			Name:     "return-path." + r.domain,
			Value:    "Optional bounce or return-path note only. Do not proxy SMTP records through Cloudflare.",
			Required: false,
			Note:     "Optional MX or bounce note for operator guidance.",
		},
	}
}

func (s *Service) loadAuthorizedDomain(ctx context.Context, actor authmodule.AuthenticatedUser, domainID uuid.UUID) (*Domain, recordSpec, error) {
	var domain Domain
	var dkimPublic string
	var spfRecord string
	var dmarcRecord string
	err := s.db.QueryRow(
		ctx,
		`SELECT id, organization_id, name, verified, dkim_selector, dkim_public, spf_record, dmarc_record, created_at, verified_at
		 FROM domains
		 WHERE id = $1`,
		domainID,
	).Scan(
		&domain.ID,
		&domain.OrganizationID,
		&domain.Name,
		&domain.Verified,
		&domain.DKIMSelector,
		&dkimPublic,
		&spfRecord,
		&dmarcRecord,
		&domain.CreatedAt,
		&domain.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recordSpec{}, ErrDomainNotFound
		}
		return nil, recordSpec{}, err
	}

	if actor.Role == authmodule.RoleCustomer {
		orgID, err := s.resolveCustomerOrganizationID(ctx, actor)
		if err != nil {
			return nil, recordSpec{}, err
		}
		if domain.OrganizationID != orgID {
			return nil, recordSpec{}, ErrDomainForbidden
		}
	}

	return &domain, recordSpec{
		domain:       domain.Name,
		dkimSelector: domain.DKIMSelector,
		dkimPublic:   dkimPublic,
		spfValue:     spfRecord,
		dmarcValue:   dmarcRecord,
	}, nil
}

func (s *Service) loadLatestChecks(ctx context.Context, domainID uuid.UUID) ([]DNSCheck, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT record_type, name, expected_value, COALESCE(found_value, ''), found, checked_at
		 FROM dns_checks
		 WHERE domain_id = $1
		 ORDER BY checked_at DESC, name ASC`,
		domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []DNSCheck
	for rows.Next() {
		var check DNSCheck
		if err := rows.Scan(&check.Type, &check.Name, &check.ExpectedValue, &check.FoundValue, &check.Found, &check.CheckedAt); err != nil {
			return nil, err
		}
		check.Required = check.Type == "TXT"
		checks = append(checks, check)
	}

	return checks, rows.Err()
}

func (s *Service) countLatestRequiredChecks(ctx context.Context, domainID uuid.UUID) (int, error) {
	checks, err := s.loadLatestChecks(ctx, domainID)
	if err != nil {
		return 0, err
	}

	found := 0
	for _, check := range checks {
		if check.Required && check.Found {
			found++
		}
	}
	return found, nil
}

func (s *Service) checkAndPersistDNS(ctx context.Context, domain Domain, spec recordSpec) ([]DNSCheck, error) {
	requirements := spec.requirements()
	now := time.Now().UTC()
	checks := make([]DNSCheck, 0, 3)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM dns_checks WHERE domain_id = $1`, domain.ID); err != nil {
		return nil, err
	}

	for _, requirement := range requirements {
		if !requirement.Required {
			continue
		}

		foundValue, found := s.lookupTXTMatch(ctx, requirement.Name, requirement.Value)
		check := DNSCheck{
			Type:          requirement.Type,
			Name:          requirement.Name,
			ExpectedValue: requirement.Value,
			FoundValue:    foundValue,
			Found:         found,
			Required:      true,
			CheckedAt:     now,
		}
		checks = append(checks, check)

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO dns_checks (domain_id, record_type, name, expected_value, found_value, found, checked_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			domain.ID,
			check.Type,
			check.Name,
			check.ExpectedValue,
			nullIfEmpty(check.FoundValue),
			check.Found,
			check.CheckedAt,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return checks, nil
}

func (s *Service) lookupTXTMatch(ctx context.Context, recordName string, expected string) (string, bool) {
	expectedNormalized := normalizeTXT(expected)

	for _, server := range s.resolvers {
		values, err := lookupTXTWithResolver(ctx, server, recordName)
		if err != nil {
			continue
		}

		for _, value := range values {
			normalized := normalizeTXT(value)
			if normalized == expectedNormalized {
				return value, true
			}
		}

		if len(values) > 0 {
			return strings.Join(values, " | "), false
		}
	}

	return "", false
}

func lookupTXTWithResolver(ctx context.Context, resolverAddr string, host string) ([]string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: 4 * time.Second}
			return dialer.DialContext(ctx, "udp", resolverAddr)
		},
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return resolver.LookupTXT(lookupCtx, host)
}

func (s *Service) resolveOrganizationForCreate(ctx context.Context, actor authmodule.AuthenticatedUser, organizationID *uuid.UUID) (uuid.UUID, error) {
	if actor.Role == authmodule.RoleAdmin {
		if organizationID == nil {
			return uuid.Nil, ErrOrganizationNeeded
		}
		return *organizationID, nil
	}

	return s.resolveCustomerOrganizationID(ctx, actor)
}

func (s *Service) resolveCustomerOrganizationID(ctx context.Context, actor authmodule.AuthenticatedUser) (uuid.UUID, error) {
	if actor.OrganizationID != nil {
		return *actor.OrganizationID, nil
	}

	var organizationID uuid.UUID
	err := s.db.QueryRow(
		ctx,
		`SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`,
		actor.ID,
	).Scan(&organizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrOrganizationNeeded
		}
		return uuid.Nil, err
	}

	return organizationID, nil
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

func buildWarnings() []string {
	return []string{
		cloudflareSMTPWarning,
		"Add TXT records at your registrar or DNS provider. Cloudflare automation is not enabled in Production v1.",
	}
}

func normalizeDomain(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.TrimSuffix(value, ".")))
}

func normalizeTXT(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
