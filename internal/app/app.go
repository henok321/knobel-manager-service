package app

import (
	"net/http"

	"github.com/henok321/knobel-manager-service/pkg/team"

	"github.com/henok321/knobel-manager-service/pkg/player"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type App struct {
	Database       *gorm.DB
	AuthMiddleware middleware.AuthenticationMiddleware
	Router         *http.ServeMux
}

func (app *App) Initialize() http.Handler {

	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.Database))
	playersHandler := handlers.NewPlayersHandler(player.InitializePlayerModule(app.Database))
	tablesHandler := handlers.NewTablesHandler(game.InitializeGameModule(app.Database))
	teamsHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.Database))

	// health
	app.Router.Handle("GET /health", http.HandlerFunc(handlers.HealthCheck))

	// games
	app.Router.Handle("GET /games", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.GetGames)))
	app.Router.Handle("GET /games/{gameID}", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.GetGameByID)))
	app.Router.Handle("POST /games", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.CreateGame)))
	app.Router.Handle("PUT /games/{gameID}", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.UpdateGame)))
	app.Router.Handle("DELETE /games/{gameID}", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.DeleteGame)))

	// setup
	app.Router.Handle("POST /games/{gameID}/setup", app.AuthMiddleware.Authentication(http.HandlerFunc(gamesHandler.GameSetup)))

	// players
	app.Router.Handle("POST /games/{gameID}/teams/{teamID}/players", app.AuthMiddleware.Authentication(http.HandlerFunc(playersHandler.CreatePlayer)))
	app.Router.Handle("PUT /games/{gameID}/teams/{teamID}/players/{playerID}", app.AuthMiddleware.Authentication(http.HandlerFunc(playersHandler.UpdatePlayer)))
	app.Router.Handle("DELETE /games/{gameID}/teams/{teamID}/players/{playerID}", app.AuthMiddleware.Authentication(http.HandlerFunc(playersHandler.DeletePlayer)))

	// tables
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables", app.AuthMiddleware.Authentication(http.HandlerFunc(tablesHandler.GetTables)))
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}", app.AuthMiddleware.Authentication(http.HandlerFunc(tablesHandler.GetTable)))

	// scores
	app.Router.Handle("PUT /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}/scores", app.AuthMiddleware.Authentication(http.HandlerFunc(tablesHandler.UpdateTableScore)))

	// teams
	app.Router.Handle("POST /games/{gameID}/teams", app.AuthMiddleware.Authentication(http.HandlerFunc(teamsHandler.CreateTeam)))
	app.Router.Handle("PUT /games/{gameID}/teams/{teamID}", app.AuthMiddleware.Authentication(http.HandlerFunc(teamsHandler.UpdateTeam)))
	app.Router.Handle("DELETE /games/{gameID}/teams/{teamID}", app.AuthMiddleware.Authentication(http.HandlerFunc(teamsHandler.DeleteTeam)))

	return middleware.RequestLogging(app.Router)
}
