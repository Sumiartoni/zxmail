package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(c *gin.Context) {
	h.Ready(c)
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "zxmail-api",
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbReady := h.db.Ping(ctx) == nil
	redisReady := h.redis.Ping(ctx).Err() == nil

	statusCode := http.StatusOK
	status := "ready"
	if !dbReady || !redisReady {
		statusCode = http.StatusServiceUnavailable
		status = "degraded"
	}

	c.JSON(statusCode, gin.H{
		"status": status,
		"checks": gin.H{
			"postgres": dbReady,
			"redis":    redisReady,
		},
	})
}
