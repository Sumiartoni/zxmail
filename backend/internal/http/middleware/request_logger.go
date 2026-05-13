package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/platform/logger"
)

func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.Info(
			"%s %s status=%d duration=%s ip=%s request_id=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start).String(),
			c.ClientIP(),
			c.GetString("requestID"),
		)
	}
}
