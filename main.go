package main

import (
	"knobel-manager-service/db"
	"knobel-manager-service/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	// Create a new Gin router
	r := gin.Default()

	// Define a simple GET route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	database, _ := db.NewDB()
	sampleService := services.NewExampleService(database)

	r.GET("/sample", func(c *gin.Context) {
		data, err := sampleService.SampleData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)

		// Run the service on port 8080
	})
	r.Run(":8080") // By default, Gin listens on 0.0.0.0:8080

}
