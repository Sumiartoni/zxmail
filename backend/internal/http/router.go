package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"zxmail/backend/internal/config"
	"zxmail/backend/internal/http/handlers"
	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	credentialsmodule "zxmail/backend/internal/modules/credentials"
	domainsmodule "zxmail/backend/internal/modules/domains"
	logsmodule "zxmail/backend/internal/modules/logs"
	organizationsmodule "zxmail/backend/internal/modules/organizations"
	quotamodule "zxmail/backend/internal/modules/quota"
	webhooksmodule "zxmail/backend/internal/modules/webhooks"
	"zxmail/backend/internal/platform/logger"
	"zxmail/backend/internal/postal"
)

type Dependencies struct {
	DB                *pgxpool.Pool
	Redis             *redis.Client
	Postal            *postal.Client
	Logger            *logger.Logger
	AuthService       *authmodule.Service
	OrgService        *organizationsmodule.Service
	DomainService     *domainsmodule.Service
	CredentialService *credentialsmodule.Service
	QuotaService      *quotamodule.Service
	LogsService       *logsmodule.Service
	WebhooksService   *webhooksmodule.Service
}

func NewRouter(cfg config.Config, deps Dependencies) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(deps.Logger))
	router.Use(middleware.CORS(cfg.CORSAllowOrigins))
	router.Use(middleware.Recovery(deps.Logger))

	healthHandler := handlers.NewHealthHandler(deps.DB, deps.Redis)
	authHandler := handlers.NewAuthHandler(
		deps.AuthService,
		cfg.CookieDomain,
		cfg.AppEnv == "production",
		cfg.JWTTokenTTL,
	)
	usersHandler := handlers.NewUsersHandler()
	organizationsHandler := handlers.NewOrganizationsHandler(deps.OrgService)
	domainsHandler := handlers.NewDomainsHandler(deps.DomainService)
	credentialsHandler := handlers.NewCredentialsHandler(deps.CredentialService, cfg.SMTPHost, cfg.SMTPPortSTARTTLS, cfg.SMTPPortTLS)
	logsHandler := handlers.NewLogsHandler(deps.LogsService)
	bouncesHandler := handlers.NewBouncesHandler()
	suppressionsHandler := handlers.NewSuppressionsHandler()
	quotaHandler := handlers.NewQuotaHandler(deps.QuotaService)
	adminHandler := handlers.NewAdminHandler()
	webhookHandler := handlers.NewPostalWebhookHandler(cfg.PostalWebhookSecret, deps.WebhooksService)

	router.GET("/health", healthHandler.Health)
	router.GET("/health/live", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)

		protected := api.Group("/")
		protected.Use(middleware.Auth(cfg.JWTSecret))
		{
			protected.GET("/me", authHandler.Me)
			protected.GET("/organization", organizationsHandler.Current)
			protected.GET("/users", usersHandler.List)
			protected.GET("/domains", domainsHandler.List)
			protected.POST("/domains", domainsHandler.Create)
			protected.GET("/domains/:id", domainsHandler.Get)
			protected.POST("/domains/:id/verify", domainsHandler.Verify)
			protected.GET("/credentials", credentialsHandler.List)
			protected.POST("/credentials", credentialsHandler.Create)
			protected.GET("/credentials/:id", credentialsHandler.Get)
			protected.POST("/credentials/:id/revoke", credentialsHandler.Revoke)
			protected.POST("/credentials/:id/rotate", credentialsHandler.Rotate)
			protected.GET("/logs", logsHandler.List)
			protected.GET("/bounces", bouncesHandler.List)
			protected.GET("/suppressions", suppressionsHandler.List)
			protected.POST("/suppressions", suppressionsHandler.Create)
			protected.DELETE("/suppressions/:suppressionID", suppressionsHandler.Release)
			protected.GET("/quota", quotaHandler.Get)

			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRoles(authmodule.RoleAdmin))
			{
				admin.GET("/overview", adminHandler.Overview)
				admin.GET("/organizations", organizationsHandler.List)
				admin.POST("/organizations", organizationsHandler.Create)
				admin.PATCH("/credentials/:id/quota", quotaHandler.UpdateCredentialQuota)
				admin.POST("/credentials/:id/disable", quotaHandler.DisableCredential)
				admin.POST("/credentials/:id/enable", quotaHandler.EnableCredential)
			}
		}
	}

	router.POST("/webhooks/postal/event", webhookHandler.Receive)

	return router
}
