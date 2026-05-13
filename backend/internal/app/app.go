package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"zxmail/backend/internal/config"
	httpapi "zxmail/backend/internal/http"
	authmodule "zxmail/backend/internal/modules/auth"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
	domainsmodule "zxmail/backend/internal/modules/domains"
	logsmodule "zxmail/backend/internal/modules/logs"
	organizationsmodule "zxmail/backend/internal/modules/organizations"
	quotamodule "zxmail/backend/internal/modules/quota"
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

	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	redisClient, err := cache.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("open redis: %w", err)
	}

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
	webhooksService := webhooksmodule.NewService(dbPool, log, quotaService)
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
