package middleware

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		// Extract the token from the header
		parts := strings.Split(header, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		idToken := parts[1]

		app, err := firebase.NewApp(context.Background(), nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Firebase app"})
			return
		}
		client, err := app.Auth(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Firebase Auth client"})
			return
		}

		token, err := client.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Extract user information from token claims
		userInfo := map[string]interface{}{
			"sub":   token.UID,
			"name":  token.Claims["name"],
			"email": token.Claims["email"],
		}

		// Attach the user information to the context
		c.Set("user", userInfo)

		// Proceed to the next handler
		c.Next()
	}
}
