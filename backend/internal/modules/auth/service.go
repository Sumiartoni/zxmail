package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"zxmail/backend/internal/platform/auditlog"
	"zxmail/backend/internal/platform/logger"
)

const (
	RoleAdmin             = "admin"
	RoleCustomer          = "customer"
	AccessTokenCookieName = "zxmail_access_token"
	tokenIssuer           = "zxmail"
	dummyPasswordHash     = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLoginRateLimited   = errors.New("login rate limited")
)

type LoginRateLimitError struct {
	RetryAfter time.Duration
}

func (e *LoginRateLimitError) Error() string {
	return ErrLoginRateLimited.Error()
}

type AuthenticatedUser struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`
}

type LoginResult struct {
	Token string            `json:"token"`
	User  AuthenticatedUser `json:"user"`
}

type Claims struct {
	Email          string `json:"email"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id,omitempty"`
	jwt.RegisteredClaims
}

type Service struct {
	db                 *pgxpool.Pool
	log                *logger.Logger
	jwtSecret          []byte
	jwtTTL             time.Duration
	loginThrottle      *LoginThrottle
	firstAdminEmail    string
	firstAdminPassword string
}

func NewService(
	db *pgxpool.Pool,
	redisClient *redis.Client,
	log *logger.Logger,
	jwtSecret string,
	jwtTTL time.Duration,
	loginThrottleConfig LoginThrottleConfig,
	firstAdminEmail string,
	firstAdminPassword string,
) *Service {
	return &Service{
		db:                 db,
		log:                log,
		jwtSecret:          []byte(jwtSecret),
		jwtTTL:             jwtTTL,
		loginThrottle:      NewLoginThrottle(redisClient, loginThrottleConfig),
		firstAdminEmail:    normalizeEmail(firstAdminEmail),
		firstAdminPassword: firstAdminPassword,
	}
}

func (s *Service) EnsureFirstAdmin(ctx context.Context) error {
	if s.firstAdminEmail == "" || s.firstAdminPassword == "" {
		return nil
	}

	var exists bool
	if err := s.db.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`,
		s.firstAdminEmail,
	).Scan(&exists); err != nil {
		return err
	}

	if exists {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(s.firstAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)`,
		s.firstAdminEmail,
		string(passwordHash),
		RoleAdmin,
	)
	if err != nil {
		return err
	}

	s.log.Info("bootstrapped first admin %s", s.firstAdminEmail)
	return nil
}

func (s *Service) Login(ctx context.Context, email string, password string, clientIP string) (*LoginResult, error) {
	var (
		userID       uuid.UUID
		userEmail    string
		passwordHash string
		role         string
	)

	normalizedEmail := normalizeEmail(email)
	retryAfter, err := s.loginThrottle.Check(ctx, normalizedEmail, clientIP)
	if err != nil {
		s.log.Error("login throttle check failed for %s from %s: %v", normalizedEmail, clientIP, err)
	} else if retryAfter > 0 {
		s.recordAuditLog(ctx, nil, nil, "auth.login_blocked", "auth_attempt", nil, map[string]any{
			"email":               normalizedEmail,
			"client_ip":           clientIP,
			"retry_after_seconds": int(retryAfter.Seconds()),
		})
		return nil, &LoginRateLimitError{RetryAfter: retryAfter}
	}

	err = s.db.QueryRow(
		ctx,
		`SELECT id, email, password_hash, role FROM users WHERE email = $1`,
		normalizedEmail,
	).Scan(&userID, &userEmail, &passwordHash, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
			return nil, s.handleFailedLogin(ctx, normalizedEmail, clientIP, nil)
		}
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return nil, s.handleFailedLogin(ctx, normalizedEmail, clientIP, &userID)
	}

	var organizationID *uuid.UUID
	if role == RoleCustomer {
		var orgID uuid.UUID
		if err := s.db.QueryRow(ctx, `SELECT id FROM organizations WHERE owner_user_id = $1 LIMIT 1`, userID).Scan(&orgID); err == nil {
			organizationID = &orgID
		}
	}

	user := AuthenticatedUser{
		ID:             userID,
		Email:          userEmail,
		Role:           role,
		OrganizationID: organizationID,
	}

	token, err := SignToken(s.jwtSecret, s.jwtTTL, user)
	if err != nil {
		return nil, err
	}

	if _, err := s.db.Exec(ctx, `UPDATE users SET last_login = NOW() WHERE id = $1`, userID); err != nil {
		return nil, err
	}

	if err := s.loginThrottle.Reset(ctx, normalizedEmail, clientIP); err != nil {
		s.log.Error("login throttle reset failed for %s from %s: %v", normalizedEmail, clientIP, err)
	}

	if err := s.insertAuditLog(ctx, &userID, organizationID, "auth.login", "user", &userID, map[string]any{
		"email": userEmail,
		"role":  role,
		"ip":    clientIP,
	}); err != nil {
		return nil, err
	}

	return &LoginResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *Service) handleFailedLogin(ctx context.Context, email string, clientIP string, targetUserID *uuid.UUID) error {
	s.recordAuditLog(ctx, nil, nil, "auth.login_failed", "auth_attempt", targetUserID, map[string]any{
		"email":     email,
		"client_ip": clientIP,
	})

	blocked, retryAfter, err := s.loginThrottle.RecordFailure(ctx, email, clientIP)
	if err != nil {
		s.log.Error("login throttle update failed for %s from %s: %v", email, clientIP, err)
		return ErrInvalidCredentials
	}
	if blocked {
		s.recordAuditLog(ctx, nil, nil, "auth.login_blocked", "auth_attempt", targetUserID, map[string]any{
			"email":               email,
			"client_ip":           clientIP,
			"retry_after_seconds": int(retryAfter.Seconds()),
		})
		return &LoginRateLimitError{RetryAfter: retryAfter}
	}

	return ErrInvalidCredentials
}

func ParseToken(tokenString string, secret []byte) (AuthenticatedUser, error) {
	claims := &Claims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return AuthenticatedUser{}, errors.New("invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return AuthenticatedUser{}, err
	}

	var organizationID *uuid.UUID
	if claims.OrganizationID != "" {
		parsedOrgID, err := uuid.Parse(claims.OrganizationID)
		if err != nil {
			return AuthenticatedUser{}, err
		}
		organizationID = &parsedOrgID
	}

	return AuthenticatedUser{
		ID:             userID,
		Email:          claims.Email,
		Role:           claims.Role,
		OrganizationID: organizationID,
	}, nil
}

func SignToken(secret []byte, ttl time.Duration, user AuthenticatedUser) (string, error) {
	claims := Claims{
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	if user.OrganizationID != nil {
		claims.OrganizationID = user.OrganizationID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (s *Service) insertAuditLog(
	ctx context.Context,
	actorUserID *uuid.UUID,
	organizationID *uuid.UUID,
	action string,
	targetType string,
	targetID *uuid.UUID,
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

func (s *Service) recordAuditLog(
	ctx context.Context,
	actorUserID *uuid.UUID,
	organizationID *uuid.UUID,
	action string,
	targetType string,
	targetID *uuid.UUID,
	metadata map[string]any,
) {
	if err := s.insertAuditLog(ctx, actorUserID, organizationID, action, targetType, targetID, metadata); err != nil {
		s.log.Error("audit log write failed for %s: %v", action, err)
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
