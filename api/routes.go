package api

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
)

func InitializeRoutes(router *gin.Engine, playersHandler handlers.PlayersHandler, gamesHandler handlers.GamesHandler) {
	// Unauthenticated routes
	unauthenticated := router
	unauthenticated.Use(middleware.RateLimiterMiddleware(5, 10), middleware.ErrorHandler())
	// health check
	unauthenticated.GET("/health", handlers.HealthCheck)

	// Authenticated routes
	authenticated := router.Group("/")
	authenticated.Use(middleware.AuthMiddleware(), middleware.RateLimiterMiddleware(5, 10), middleware.ErrorHandler())

	// player routes
	authenticated.GET("/players", playersHandler.GetPlayers)

	// game routes
	authenticated.GET("/games", gamesHandler.GetGames)
	authenticated.GET("/games/:id", gamesHandler.GetGameByID)
	authenticated.POST("/games", gamesHandler.CreateGame)
	authenticated.PUT("/games/:id", gamesHandler.UpdateGame)
	authenticated.DELETE("/games/:id", gamesHandler.DeleteGame)
}
