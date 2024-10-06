package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/internal/db"
	"github.com/henok321/knobel-manager-service/pkg/health"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/gorm"
)

type App struct {
	Router *gin.Engine
	DB     *gorm.DB

	PlayerHandler *player.PlayersHandler
}

func (app *App) Initialize() {
	// Initialize database
	app.DB, _ = db.Connect()

	// Initialize modules
	app.PlayerHandler = player.InitializePlayerModule(app.DB)

	// Initialize router
	app.Router = gin.Default()
	app.initializeRoutes()
}

func (app *App) initializeRoutes() {
	app.Router.GET("/health", health.HealthCheck)
	app.Router.GET("/players", app.PlayerHandler.GetPlayers)
}
