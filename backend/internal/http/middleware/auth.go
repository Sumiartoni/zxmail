package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authmodule "zxmail/backend/internal/modules/auth"
)

const actorContextKey = "actor"

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := readBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			if cookieToken, err := c.Cookie(authmodule.AccessTokenCookieName); err == nil {
				token = strings.TrimSpace(cookieToken)
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authentication token",
			})
			return
		}

		actor, err := authmodule.ParseToken(token, []byte(jwtSecret))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authentication token",
			})
			return
		}

		c.Set(actorContextKey, actor)
		c.Next()
	}
}

func readBearerToken(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func ActorFromContext(c *gin.Context) (authmodule.AuthenticatedUser, bool) {
	value, exists := c.Get(actorContextKey)
	if !exists {
		return authmodule.AuthenticatedUser{}, false
	}

	actor, ok := value.(authmodule.AuthenticatedUser)
	return actor, ok
}
