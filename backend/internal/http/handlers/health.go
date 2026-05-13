package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"zxmail/backend/internal/platform/logger"
)

type dependencyPinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	log   *logger.Logger
	db    dependencyPinger
	redis dependencyPinger
}

type redisPingerAdapter struct {
	client *redis.Client
}

func (r redisPingerAdapter) Ping(ctx context.Context) error {
	if r.client == nil {
		return errDependencyNotConfigured("redis")
	}
	return r.client.Ping(ctx).Err()
}

func NewHealthHandler(log *logger.Logger, db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	var dbPinger dependencyPinger
	if db != nil {
		dbPinger = db
	}

	return &HealthHandler{
		log:   log,
		db:    dbPinger,
		redis: redisPingerAdapter{client: redisClient},
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "zxmail-api",
	})
}

func (h *HealthHandler) Live(c *gin.Context) {
	h.Health(c)
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	postgresErr := h.pingDependency(ctx, "postgres", h.db)
	redisErr := h.pingDependency(ctx, "redis", h.redis)

	statusCode := http.StatusOK
	status := "ready"
	if postgresErr != nil || redisErr != nil {
		statusCode = http.StatusServiceUnavailable
		status = "degraded"
	}

	checks := gin.H{
		"postgres": dependencyStatus(postgresErr),
		"redis":    dependencyStatus(redisErr),
	}

	c.JSON(statusCode, gin.H{
		"status": status,
		"checks": checks,
	})
}

func (h *HealthHandler) pingDependency(ctx context.Context, name string, pinger dependencyPinger) error {
	if pinger == nil {
		err := errDependencyNotConfigured(name)
		if h.log != nil {
			h.log.Error("health %s ping failed: %v", name, err)
		}
		return err
	}

	err := pinger.Ping(ctx)
	if err != nil && h.log != nil {
		h.log.Error("health %s ping failed: %v", name, err)
	}
	return err
}

func dependencyStatus(err error) gin.H {
	if err == nil {
		return gin.H{"ready": true}
	}

	return gin.H{
		"ready": false,
		"error": err.Error(),
	}
}

type dependencyConfigError struct {
	name string
}

func (e dependencyConfigError) Error() string {
	return e.name + " client is not configured"
}

func errDependencyNotConfigured(name string) error {
	return dependencyConfigError{name: name}
}
