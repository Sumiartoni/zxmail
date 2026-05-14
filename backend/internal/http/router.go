package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"zxmail/backend/internal/config"
	"zxmail/backend/internal/http/handlers"
	"zxmail/backend/internal/http/middleware"
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
	BillingService    *billingmodule.Service
	UsageService      *usagemodule.Service
	Deliverability    *deliverabilitymodule.Service
	AdminV2Service    *adminv2module.Service
	RetentionService  *retentionmodule.Service
	Operations        *operationsmodule.Service
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

	healthHandler := handlers.NewHealthHandler(deps.Logger, deps.DB, deps.Redis)
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
	billingV2Handler := handlers.NewBillingV2Handler(deps.BillingService)
	usageV2Handler := handlers.NewUsageV2Handler(deps.UsageService)
	deliverabilityV2Handler := handlers.NewDeliverabilityV2Handler(deps.Deliverability)
	adminV2Handler := handlers.NewAdminV2Handler(deps.AdminV2Service)
	retentionV2Handler := handlers.NewRetentionV2Handler(deps.RetentionService)
	operationsV2Handler := handlers.NewOperationsV2Handler(deps.Operations)

	router.GET("/health", healthHandler.Health)
	router.GET("/health/live", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/ready", healthHandler.Ready)

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

	apiV2 := router.Group("/api/v2")
	{
		protected := apiV2.Group("/")
		protected.Use(middleware.Auth(cfg.JWTSecret))
		{
			protected.GET("/plans", billingV2Handler.ListPlans)
			protected.GET("/subscription", billingV2Handler.GetSubscription)
			protected.GET("/invoices", billingV2Handler.ListInvoices)
			protected.GET("/usage", usageV2Handler.GetUsage)
			protected.GET("/deliverability/overview", deliverabilityV2Handler.Overview)
			protected.GET("/deliverability/domains/:id", deliverabilityV2Handler.Domain)
			protected.GET("/alerts", deliverabilityV2Handler.ListAlerts)
			protected.POST("/domains/:id/recheck", deliverabilityV2Handler.RecheckDomain)
			protected.GET("/domains/:id/health", deliverabilityV2Handler.Domain)
			protected.GET("/organization", organizationsHandler.Current)
			protected.GET("/logs", logsHandler.List)

			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRoles(authmodule.RoleAdmin))
			{
				admin.POST("/plans", billingV2Handler.CreatePlan)
				admin.PATCH("/plans/:id", billingV2Handler.UpdatePlan)
				admin.POST("/organizations/:id/subscription", billingV2Handler.AssignSubscription)
				admin.POST("/invoices/:id/mark-paid", billingV2Handler.MarkInvoicePaid)
				admin.POST("/invoices/:id/mark-failed", billingV2Handler.MarkInvoiceFailed)
				admin.GET("/invoices", billingV2Handler.ListAdminInvoices)
				admin.GET("/payments", billingV2Handler.ListPayments)
				admin.POST("/payments/:id/approve", billingV2Handler.ApprovePayment)
				admin.POST("/payments/:id/reject", billingV2Handler.RejectPayment)
				admin.GET("/overview", adminV2Handler.Overview)
				admin.GET("/organizations", organizationsHandler.List)
				admin.GET("/organizations/:id/detail", adminV2Handler.OrganizationDetail)
				admin.POST("/organizations/:id/suspend", adminV2Handler.Suspend)
				admin.POST("/organizations/:id/unsuspend", adminV2Handler.Unsuspend)
				admin.POST("/organizations/:id/disable-credentials", adminV2Handler.DisableCredentials)
				admin.GET("/risk/organizations", adminV2Handler.Risk)
				admin.GET("/organizations/:id/usage", usageV2Handler.GetOrganizationUsage)
				admin.PATCH("/organizations/:id/quota", usageV2Handler.UpdateOrganizationQuota)
				admin.POST("/organizations/:id/reset-usage", usageV2Handler.ResetOrganizationUsage)
				admin.POST("/credentials/:id/limit", usageV2Handler.LimitCredential)
				admin.POST("/credentials/:id/unlimit", usageV2Handler.UnlimitCredential)
				admin.POST("/credentials/:id/force-rotate", adminV2Handler.ForceRotate)
				admin.GET("/deliverability", deliverabilityV2Handler.AdminOverview)
				admin.GET("/domains/health", deliverabilityV2Handler.AdminDomainsHealth)
				admin.POST("/domains/recheck-all", deliverabilityV2Handler.RecheckAll)
				admin.POST("/alerts/:id/resolve", deliverabilityV2Handler.ResolveAlert)
				admin.GET("/retention", retentionV2Handler.List)
				admin.PATCH("/organizations/:id/retention", retentionV2Handler.Update)
				admin.POST("/retention/run-cleanup", retentionV2Handler.Cleanup)
				admin.GET("/audit-logs", adminV2Handler.AuditLogs)
				admin.GET("/system/health", operationsV2Handler.Health)
				admin.GET("/system/queues", operationsV2Handler.Queues)
				admin.GET("/system/postal-health", operationsV2Handler.PostalHealth)
			}
		}
	}

	router.POST("/webhooks/postal/event", webhookHandler.Receive)

	return router
}
