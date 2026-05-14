package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zxmail/backend/internal/config"
	httpapi "zxmail/backend/internal/http"
	adminv2module "zxmail/backend/internal/modules/adminv2"
	authmodule "zxmail/backend/internal/modules/auth"
	billingmodule "zxmail/backend/internal/modules/billing"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
	deliverabilitymodule "zxmail/backend/internal/modules/deliverability"
	domainsmodule "zxmail/backend/internal/modules/domains"
	logsmodule "zxmail/backend/internal/modules/logs"
	operationsmodule "zxmail/backend/internal/modules/operations"
	organizationsmodule "zxmail/backend/internal/modules/organizations"
	quotamodule "zxmail/backend/internal/modules/quota"
	retentionmodule "zxmail/backend/internal/modules/retention"
	usagemodule "zxmail/backend/internal/modules/usage"
	webhooksmodule "zxmail/backend/internal/modules/webhooks"
	"zxmail/backend/internal/platform/cache"
	"zxmail/backend/internal/platform/database"
	"zxmail/backend/internal/platform/logger"
	"zxmail/backend/internal/platform/security"
	"zxmail/backend/internal/postal"
)

type App struct {
	config config.Config
	server *http.Server
	log    *logger.Logger
	close  func()
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	log := logger.New(cfg.AppName)

	dbHost, dbName, dbUser := describePostgresTarget(cfg.DatabaseURL)
	log.Info("bootstrapping postgres connection host=%s database=%s user=%s", dbHost, dbName, dbUser)
	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres bootstrap failed host=%s database=%s user=%s error=%v", dbHost, dbName, dbUser, err)
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	log.Info("postgres connection ready host=%s database=%s user=%s", dbHost, dbName, dbUser)

	redisAddr, redisDB := describeRedisTarget(cfg.RedisURL)
	log.Info("bootstrapping redis connection addr=%s db=%s", redisAddr, redisDB)
	redisClient, err := cache.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Error("redis bootstrap failed addr=%s db=%s error=%v", redisAddr, redisDB, err)
		return nil, fmt.Errorf("open redis: %w", err)
	}
	log.Info("redis connection ready addr=%s db=%s", redisAddr, redisDB)

	postalClient := postal.NewClient(cfg.PostalBaseURL, cfg.PostalAPIKey, cfg.SMTPHost)
	authService := authmodule.NewService(
		dbPool,
		redisClient,
		log,
		cfg.JWTSecret,
		cfg.JWTTokenTTL,
		authmodule.LoginThrottleConfig{
			MaxFailures:   cfg.LoginMaxFailures,
			FailureWindow: cfg.LoginFailureWindow,
			LockoutWindow: cfg.LoginLockoutWindow,
		},
		cfg.FirstAdminEmail,
		cfg.FirstAdminPassword,
	)
	if err := authService.EnsureFirstAdmin(ctx); err != nil {
		return nil, fmt.Errorf("ensure first admin: %w", err)
	}

	orgService := organizationsmodule.NewService(dbPool, log)
	domainService := domainsmodule.NewService(dbPool, log)
	quotaService := quotamodule.NewService(dbPool, redisClient, log)
	usageService := usagemodule.NewService(dbPool, redisClient, log)
	keyring, err := security.NewKeyring(security.KeyringConfig{
		LegacyKey:   cfg.EncryptionKey,
		LegacyKeyID: cfg.EncryptionKeyID,
		EncodedKeys: cfg.EncryptionKeys,
		ActiveKeyID: cfg.ActiveEncryptionKeyID,
	})
	if err != nil {
		return nil, fmt.Errorf("build encryption keyring: %w", err)
	}
	credentialService := credentialsmodule.NewService(dbPool, log, keyring, quotaService)
	logsService := logsmodule.NewService(dbPool, log)
	billingService := billingmodule.NewService(dbPool, log)
	deliverabilityService := deliverabilitymodule.NewService(dbPool, log, domainService)
	retentionService := retentionmodule.NewService(dbPool, log, cfg.DefaultRetentionDays)
	operationsService := operationsmodule.NewService(dbPool, redisClient, log, postalClient)
	adminV2Service := adminv2module.NewService(
		dbPool,
		log,
		usageService,
		billingService,
		credentialService,
		cfg.SMTPHost,
		cfg.SMTPPortSTARTTLS,
		cfg.SMTPPortTLS,
	)
	webhooksService := webhooksmodule.NewService(dbPool, log, quotaService, usageService)
	router := httpapi.NewRouter(cfg, httpapi.Dependencies{
		DB:                dbPool,
		Redis:             redisClient,
		Postal:            postalClient,
		Logger:            log,
		AuthService:       authService,
		OrgService:        orgService,
		DomainService:     domainService,
		CredentialService: credentialService,
		QuotaService:      quotaService,
		LogsService:       logsService,
		WebhooksService:   webhooksService,
		BillingService:    billingService,
		UsageService:      usageService,
		Deliverability:    deliverabilityService,
		AdminV2Service:    adminV2Service,
		RetentionService:  retentionService,
		Operations:        operationsService,
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		config: cfg,
		server: server,
		log:    log,
		close: func() {
			redisClient.Close()
			dbPool.Close()
		},
	}, nil
}

func describePostgresTarget(databaseURL string) (host string, database string, user string) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "unknown", "unknown", "unknown"
	}

	if parsed.User != nil {
		user = parsed.User.Username()
	}
	if user == "" {
		user = "unknown"
	}

	host = parsed.Hostname()
	if host == "" {
		host = "unknown"
	}

	database = strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		database = "unknown"
	}

	return host, database, user
}

func describeRedisTarget(redisURL string) (addr string, db string) {
	trimmed := strings.TrimSpace(redisURL)
	if trimmed == "" {
		return "unknown", "unknown"
	}

	if !strings.Contains(trimmed, "://") {
		return trimmed, "0"
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "unknown", "unknown"
	}

	addr = parsed.Host
	if addr == "" {
		addr = "unknown"
	}

	db = strings.TrimPrefix(parsed.Path, "/")
	if db == "" {
		db = "0"
	}

	return addr, db
}

func (a *App) Run(ctx context.Context) error {
	defer a.close()

	errCh := make(chan error, 1)

	go func() {
		a.log.Info("http server listening on %s", a.server.Addr)
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()
		a.log.Info("shutdown signal received")
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
