package api

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
)

func InitializeRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc, gamesHandler handlers.GamesHandler, teamsHandler handlers.TeamsHandler, playersHandler handlers.PlayersHandler) {
	// Unauthenticated routes
	unauthenticated := router
	unauthenticated.Use(middleware.RateLimiterMiddleware(20, 100), middleware.ErrorHandler())
	// health check
	unauthenticated.GET("/health", handlers.HealthCheck)

	// openapi
	router.StaticFile("/openapi.yaml", "./openapi.yaml")
	router.StaticFile("/docs", "./redoc.html")

	// Authenticated routes
	authenticated := router.Group("/")
	authenticated.Use(middleware.RateLimiterMiddleware(20, 100), middleware.ErrorHandler(), authMiddleware)

	// games routes
	authenticated.GET("/games/", gamesHandler.GetGames)
	authenticated.GET("/games/:gameID", gamesHandler.GetGameByID)
	authenticated.POST("/games/", gamesHandler.CreateGame)
	authenticated.PUT("/games/:gameID", gamesHandler.UpdateGame)
	authenticated.DELETE("/games/:gameID", gamesHandler.DeleteGame)

	// teams routes
	authenticated.POST("/games/:gameID/teams/", teamsHandler.CreateTeam)
	authenticated.PUT("/games/:gameID/teams/:teamID", teamsHandler.UpdateTeam)
	authenticated.DELETE("/games/:gameID/teams/:teamID", teamsHandler.DeleteTeam)

	// players routes
	authenticated.POST("/games/:gameID/teams/:teamID/players", playersHandler.CreatePlayer)
	authenticated.PUT("/games/:gameID/teams/:teamID/players/:playerID", playersHandler.UpdatePlayer)
	authenticated.DELETE("/games/:gameID/teams/:teamID/players/:playerID", playersHandler.DeletePlayer)

}
