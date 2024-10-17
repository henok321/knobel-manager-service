package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != "permitted" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		userInfo := map[string]interface{}{
			"sub":   "sub-1",
			"email": "mock@example.com",
		}

		c.Set("user", userInfo)

		c.Next()
	}
}
