package api

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
)

func InitializeRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc, playersHandler handlers.PlayersHandler, gamesHandler handlers.GamesHandler) {
	// Unauthenticated routes
	unauthenticated := router
	unauthenticated.Use(middleware.RateLimiterMiddleware(5, 10), middleware.ErrorHandler())
	// health check
	unauthenticated.GET("/health", handlers.HealthCheck)

	// openapi
	router.StaticFile("/openapi.yaml", "./openapi.yaml")
	router.StaticFile("/docs", "./redoc.html")

	// Authenticated routes
	authenticated := router.Group("/")
	authenticated.Use(middleware.RateLimiterMiddleware(5, 10), middleware.ErrorHandler(), authMiddleware)

	// player routes
	authenticated.GET("/games/:gameID/players", playersHandler.GetPlayersByGame)
	authenticated.GET("/games/:gameID/teams/:teamID", playersHandler.GetPlayersByTeam)

	// game routes
	authenticated.GET("/games", gamesHandler.GetGames)
	authenticated.GET("/games/:gameID", gamesHandler.GetGameByID)
	authenticated.POST("/games", gamesHandler.CreateGame)
	authenticated.PUT("/games/:gameID", gamesHandler.UpdateGame)
	authenticated.DELETE("/games/:gameID", gamesHandler.DeleteGame)
}
