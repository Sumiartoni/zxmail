package organizations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	authmodule "zxmail/backend/internal/modules/auth"
	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
)

type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	OwnerUserID uuid.UUID `json:"owner_user_id"`
	OwnerEmail  string    `json:"owner_email"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateCustomerOrganizationInput struct {
	Name          string `json:"name"`
	OwnerEmail    string `json:"owner_email"`
	OwnerPassword string `json:"owner_password"`
}

type Service struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

func NewService(db *pgxpool.Pool, log *logger.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) ListOrganizations(ctx context.Context) ([]Organization, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT o.id, o.name, o.owner_user_id, COALESCE(u.email, ''), o.created_at
		 FROM organizations o
		 LEFT JOIN users u ON u.id = o.owner_user_id
		 ORDER BY o.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var organizations []Organization
	for rows.Next() {
		var organization Organization
		if err := rows.Scan(&organization.ID, &organization.Name, &organization.OwnerUserID, &organization.OwnerEmail, &organization.CreatedAt); err != nil {
			return nil, err
		}
		organizations = append(organizations, organization)
	}

	return organizations, rows.Err()
}

func (s *Service) CreateCustomerOrganization(
	ctx context.Context,
	actorUserID string,
	input CreateCustomerOrganizationInput,
) (*Organization, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.OwnerEmail = strings.ToLower(strings.TrimSpace(input.OwnerEmail))
	input.OwnerPassword = strings.TrimSpace(input.OwnerPassword)
	if input.Name == "" || input.OwnerEmail == "" || input.OwnerPassword == "" {
		return nil, ErrInvalidInput
	}

	actorUUID, err := uuid.Parse(actorUserID)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var ownerUserID uuid.UUID
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id`,
		input.OwnerEmail,
		string(passwordHash),
		authmodule.RoleCustomer,
	).Scan(&ownerUserID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	organization := &Organization{
		Name:        input.Name,
		OwnerUserID: ownerUserID,
		OwnerEmail:  input.OwnerEmail,
	}
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO organizations (name, owner_user_id) VALUES ($1, $2) RETURNING id, created_at`,
		input.Name,
		ownerUserID,
	).Scan(&organization.ID, &organization.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	if err := insertAuditLog(ctx, tx, actorUUID, &organization.ID, "organization.create", "organization", organization.ID, map[string]any{
		"name":        organization.Name,
		"owner_email": organization.OwnerEmail,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return organization, nil
}

func (s *Service) GetOrganizationForOwner(ctx context.Context, ownerUserID string) (*Organization, error) {
	ownerUUID, err := uuid.Parse(ownerUserID)
	if err != nil {
		return nil, err
	}

	organization := &Organization{}
	err = s.db.QueryRow(
		ctx,
		`SELECT o.id, o.name, o.owner_user_id, u.email, o.created_at
		 FROM organizations o
		 JOIN users u ON u.id = o.owner_user_id
		 WHERE o.owner_user_id = $1
		 LIMIT 1`,
		ownerUUID,
	).Scan(&organization.ID, &organization.Name, &organization.OwnerUserID, &organization.OwnerEmail, &organization.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return organization, nil
}

func insertAuditLog(
	ctx context.Context,
	tx pgx.Tx,
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

	_, err = tx.Exec(
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
