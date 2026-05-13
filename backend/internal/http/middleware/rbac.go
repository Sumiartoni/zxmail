package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing actor",
			})
			return
		}

		if !slices.Contains(roles, actor.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient role",
			})
			return
		}

		c.Next()
	}
}
