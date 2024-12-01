package app

import (
	"net/http"

	"github.com/henok321/knobel-manager-service/pkg/player"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type App struct {
	Database       *gorm.DB
	AuthMiddleware func(next http.Handler) http.Handler
	Router         *http.ServeMux
}

func (app *App) Initialize() http.Handler {

	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.Database))
	playersHandler := handlers.NewPlayersHandler(player.InitializePlayerModule(app.Database))
	tablesHandler := handlers.NewTablesHandler(game.InitializeGameModule(app.Database))
	/*teamHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.DB))
	playerHandler :=   handlers.NewPlayersHandler(player.InitializePlayerModule(app.DB))
	app.TablesHandler = handlers.NewTablesHandler(game.InitializeGameModule(app.DB))
	app.Router = http.NewServeMux()*/

	// health
	app.Router.Handle("GET /health", http.HandlerFunc(handlers.HealthCheck))

	// games
	app.Router.Handle("GET /games", app.AuthMiddleware(http.HandlerFunc(gamesHandler.GetGames)))
	app.Router.Handle("GET /games/{gameID}", app.AuthMiddleware(http.HandlerFunc(gamesHandler.GetGameByID)))
	app.Router.Handle("POST /games", app.AuthMiddleware(http.HandlerFunc(gamesHandler.CreateGame)))
	app.Router.Handle("PUT /games/{gameID}", app.AuthMiddleware(http.HandlerFunc(gamesHandler.UpdateGame)))
	app.Router.Handle("DELETE /games/{gameID}", app.AuthMiddleware(http.HandlerFunc(gamesHandler.DeleteGame)))

	// setup
	app.Router.Handle("POST /games/{gameID}/setup", app.AuthMiddleware(http.HandlerFunc(gamesHandler.GameSetup)))

	// players
	app.Router.Handle("POST /games/{gameID}/teams/{teamID}/players", app.AuthMiddleware(http.HandlerFunc(playersHandler.CreatePlayer)))
	app.Router.Handle("PUT /games/{gameID}/teams/{teamID}/players/{playerID}", app.AuthMiddleware(http.HandlerFunc(playersHandler.UpdatePlayer)))
	app.Router.Handle("DELETE /games/{gameID}/teams/{teamID}/players/{playerID}", app.AuthMiddleware(http.HandlerFunc(playersHandler.DeletePlayer)))

	// tables
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables", app.AuthMiddleware(http.HandlerFunc(tablesHandler.GetTables)))
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}", app.AuthMiddleware(http.HandlerFunc(tablesHandler.GetTable)))

	return middleware.RequestLogging(app.Router)
}
