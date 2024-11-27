package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type App struct {
	Router         *gin.Engine
	DB             *gorm.DB
	AuthMiddleware gin.HandlerFunc
}

func (app *App) Initialize() {
	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.DB))
	teamsHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.DB))
	playersHandler := handlers.NewPlayersHandler(player.InitializePlayerModule(app.DB))
	tablesHandler := handlers.NewTablesHandler(game.InitializeGameModule(app.DB))

	// Unauthenticated routes
	unauthenticated := app.Router
	unauthenticated.Use(middleware.RateLimiter(20, 100))

	// health check
	unauthenticated.GET("/health", handlers.HealthCheck)

	// openapi
	app.Router.StaticFile("/openapi.yaml", "./openapi.yaml")
	app.Router.StaticFile("/docs", "./redoc.html")

	// Authenticated routes
	authenticated := app.Router.Group("/")
	authenticated.Use(middleware.RateLimiter(20, 100), app.AuthMiddleware)
	authenticated.Use(middleware.RequestLogging())

	// games routes
	authenticated.GET("/games/", gamesHandler.GetGames)
	authenticated.GET("/games/:gameID", gamesHandler.GetGameByID)
	authenticated.POST("/games/", gamesHandler.CreateGame)
	authenticated.PUT("/games/:gameID", gamesHandler.UpdateGame)
	authenticated.DELETE("/games/:gameID", gamesHandler.DeleteGame)

	// game setup
	authenticated.POST("games/:gameID/setup", gamesHandler.GameSetup)

	// teams routes
	authenticated.POST("/games/:gameID/teams/", teamsHandler.CreateTeam)
	authenticated.PUT("/games/:gameID/teams/:teamID", teamsHandler.UpdateTeam)
	authenticated.DELETE("/games/:gameID/teams/:teamID", teamsHandler.DeleteTeam)

	// players routes
	authenticated.POST("/games/:gameID/teams/:teamID/players", playersHandler.CreatePlayer)
	authenticated.PUT("/games/:gameID/teams/:teamID/players/:playerID", playersHandler.UpdatePlayer)
	authenticated.DELETE("/games/:gameID/teams/:teamID/players/:playerID", playersHandler.DeletePlayer)

	// table routes
	authenticated.GET("/games/:gameID/rounds/:roundNumber/tables", tablesHandler.GetTables)
	authenticated.GET("/games/:gameID/rounds/:roundNumber/tables/:tableNumber", tablesHandler.GetTable)
}
