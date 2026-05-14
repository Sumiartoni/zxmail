package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	adminv2module "zxmail/backend/internal/modules/adminv2"
	billingmodule "zxmail/backend/internal/modules/billing"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
	deliverabilitymodule "zxmail/backend/internal/modules/deliverability"
	domainsmodule "zxmail/backend/internal/modules/domains"
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
	"zxmail/backend/internal/worker"

	"zxmail/backend/internal/config"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate config: %v", err)
	}

	appLogger := logger.New(cfg.AppName + "-worker")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer dbPool.Close()

	redisClient, err := cache.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()

	postalClient := postal.NewClient(cfg.PostalBaseURL, cfg.PostalAPIKey, cfg.SMTPHost)
	domainService := domainsmodule.NewService(dbPool, appLogger)
	quotaService := quotamodule.NewService(dbPool, redisClient, appLogger)
	usageService := usagemodule.NewService(dbPool, redisClient, appLogger)
	keyring, err := security.NewKeyring(security.KeyringConfig{
		LegacyKey:   cfg.EncryptionKey,
		LegacyKeyID: cfg.EncryptionKeyID,
		EncodedKeys: cfg.EncryptionKeys,
		ActiveKeyID: cfg.ActiveEncryptionKeyID,
	})
	if err != nil {
		log.Fatalf("build encryption keyring: %v", err)
	}
	credentialService := credentialsmodule.NewService(dbPool, appLogger, keyring, quotaService)
	billingService := billingmodule.NewService(dbPool, appLogger)
	deliverabilityService := deliverabilitymodule.NewService(dbPool, appLogger, domainService)
	retentionService := retentionmodule.NewService(dbPool, appLogger, cfg.DefaultRetentionDays)

	_ = organizationsmodule.NewService(dbPool, appLogger)
	_ = operationsmodule.NewService(dbPool, redisClient, appLogger, postalClient)
	_ = adminv2module.NewService(dbPool, appLogger, usageService, billingService, credentialService, cfg.SMTPHost, cfg.SMTPPortSTARTTLS, cfg.SMTPPortTLS)
	_ = webhooksmodule.NewService(dbPool, appLogger, quotaService, usageService)

	service := worker.New(
		dbPool,
		appLogger,
		billingService,
		usageService,
		deliverabilityService,
		retentionService,
		cfg.WorkerPort,
		time.Duration(cfg.WorkerScheduleSeconds)*time.Second,
	)

	if err := service.Run(ctx); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
