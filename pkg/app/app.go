package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/internal/db"
	firebaseauth "github.com/henok321/knobel-manager-service/pkg/firebase"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/gorm"
	"log"
)

type App struct {
	Router         *gin.Engine
	DB             *gorm.DB
	PlayersHandler handlers.PlayersHandler
	GamesHandler   handlers.GamesHandler
}

func (app *App) Initialize() {
	firebaseauth.InitFirebase()
	app.DB, _ = db.Connect()

	app.PlayersHandler = handlers.NewPlayersHandler(player.InitializePlayerModule(app.DB))
	app.GamesHandler = handlers.NewGamesHandler(game.InitializeGameModule(app.DB))

	app.Router = gin.Default()
	app.initializeRoutes()
}

func (app *App) initializeRoutes() {

	// Unauthenticated routes
	unauthenticated := app.Router
	unauthenticated.Use(middleware.RateLimiterMiddleware(5, 10))
	// health check
	unauthenticated.GET("/health", handlers.HealthCheck)

	// Authenticated routes
	authenticated := app.Router.Group("/")
	authenticated.Use(middleware.AuthMiddleware(), middleware.RateLimiterMiddleware(5, 10))

	// player routes
	authenticated.GET("/players", app.PlayersHandler.GetPlayers)

	// game routes
	authenticated.GET("/games", app.GamesHandler.GetGames)
	authenticated.GET("/games/:id", app.GamesHandler.GetGameByID)
	authenticated.POST("/games", app.GamesHandler.CreateGame)
	authenticated.PUT("/games/:id", app.GamesHandler.UpdateGame)
	authenticated.DELETE("/games/:id", app.GamesHandler.DeleteGame)

	log.Println("Routes setup completed")
}
