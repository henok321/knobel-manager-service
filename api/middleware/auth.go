package middleware

import (
	"context"
	"net/http"
	"strings"

	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"

	"github.com/gin-gonic/gin"
)

func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		tokenParts := strings.Split(authorizationHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		idToken := tokenParts[1]

		firebaseApp := firebaseauth.App

		client, err := firebaseApp.Auth(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Firebase Auth client"})
			return
		}

		token, err := client.VerifyIDToken(context.Background(), idToken)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		userInfo := map[string]interface{}{
			"sub":   token.UID,
			"email": token.Claims["email"],
		}

		c.Set("user", userInfo)

		c.Next()
	}
}
