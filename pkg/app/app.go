package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/internal/db"
	firebaseauth "github.com/henok321/knobel-manager-service/pkg/firebase"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/health"
	"github.com/henok321/knobel-manager-service/pkg/middleware"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/gorm"
	"log"
)

type App struct {
	Router         *gin.Engine
	DB             *gorm.DB
	PlayersHandler player.PlayersHandler
	GamesHandler   game.GamesHandler
}

func (app *App) Initialize() {
	firebaseauth.InitFirebase()
	app.DB, _ = db.Connect()

	app.PlayersHandler = player.InitializePlayerModule(app.DB)
	app.GamesHandler = game.InitializeGameModule(app.DB)

	app.Router = gin.Default()
	app.initializeRoutes()
}

func (app *App) initializeRoutes() {

	// Unauthenticated routes
	unauthenticated := app.Router
	unauthenticated.Use(middleware.RateLimiterMiddleware(5, 10))
	// health check
	unauthenticated.GET("/health", health.HealthCheck)

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

	log.Println("Routes setup completed")
}
