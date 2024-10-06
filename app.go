package main

import (
	"gorm.io/gorm"
	"knobel-manager-service/db"
	"knobel-manager-service/player"

	"github.com/gin-gonic/gin"
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
	app.Router.GET("/players", app.PlayerHandler.GetPlayers)
}
