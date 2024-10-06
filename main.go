package main

import (
	"knobel-manager-service/middlewares"
	"knobel-manager-service/services"
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

	sampleService := services.NewExampleService()

	r.GET("/sample", func(c *gin.Context) {
		data, err := sampleService.SampleData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)

		// Run the service on port 8080
	})

	r.Run("0.0.0.0:8080")

}
