package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/internal/db"
	firebaseauth "github.com/henok321/knobel-manager-service/pkg/firebase"
	"github.com/henok321/knobel-manager-service/pkg/health"
	"github.com/henok321/knobel-manager-service/pkg/middleware"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/gorm"
	"log"
)

type App struct {
	Router *gin.Engine
	DB     *gorm.DB

	PlayerHandler *player.PlayersHandler
}

func (app *App) Initialize() {
	firebaseauth.InitFirebase()
	app.DB, _ = db.Connect()
	app.PlayerHandler = player.InitializePlayerModule(app.DB)
	app.Router = gin.Default()
	app.initializeRoutes()
}

func (app *App) initializeRoutes() {
	protected := app.Router.Group("/")
	protected.Use(middleware.AuthMiddleware())

	app.Router.GET("/health", health.HealthCheck)
	protected.GET("/players", app.PlayerHandler.GetPlayers)
	log.Println("Routes setup completed")
}
