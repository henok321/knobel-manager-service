package app

import (
	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/api"
	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/internal/db"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type App struct {
	Router       *gin.Engine
	DB           *gorm.DB
	GamesHandler handlers.GamesHandler
	TeamsHandler handlers.TeamsHandler
}

func (app *App) Initialize(authMiddleware gin.HandlerFunc) {

	app.DB, _ = db.Connect()

	app.GamesHandler = handlers.NewGamesHandler(game.InitializeGameModule(app.DB))
	app.TeamsHandler = handlers.NewTeamsHandler(team.InitalizeTeamsModule(app.DB))

	app.Router = gin.Default()
	api.InitializeRoutes(app.Router, authMiddleware, app.GamesHandler, app.TeamsHandler)
}
