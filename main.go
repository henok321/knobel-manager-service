package main

import (
	"knobel-manager-service/handlers"
	"knobel-manager-service/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create a new Gin router
	r := gin.Default()

	r.Use(gin.WrapH(middlewares.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))

	// Use the handler for the health endpoint
	r.GET("/health", handlers.HealthCheck)
	r.GET("/players", handlers.GetPlayers)

	r.Run("0.0.0.0:8080")
}
