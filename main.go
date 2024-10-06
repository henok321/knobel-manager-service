package main

import (
	"knobel-manager-service/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	// Create a new Gin router
	r := gin.Default()

	r.Use(gin.WrapH(middlewares.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	r.Run("0.0.0.0:8080")

}
