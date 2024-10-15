package middleware

import (
	"github.com/gin-gonic/gin"
)

func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		userInfo := map[string]interface{}{
			"sub":   "mock-sub",
			"email": "mock@example.com",
		}

		c.Set("user", userInfo)

		c.Next()
	}
}
