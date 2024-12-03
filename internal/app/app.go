package app

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

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

func (app *App) publicEndpoint(handler http.Handler) http.Handler {
	loggingMiddlewareDebug := middleware.NewRequestLoggingMiddleware(slog.LevelDebug)
	return middleware.Metrics(loggingMiddlewareDebug.RequestLogging(handler))
}

func (app *App) authenticatedEndpoint(handler http.Handler) http.Handler {
	loggingMiddlewareInfo := middleware.NewRequestLoggingMiddleware(slog.LevelInfo)
	return middleware.Metrics(loggingMiddlewareInfo.RequestLogging(app.AuthMiddleware.Authentication(handler)))
}

func (app *App) Initialize() http.Handler {

	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.Database))
	playersHandler := handlers.NewPlayersHandler(player.InitializePlayerModule(app.Database))
	tablesHandler := handlers.NewTablesHandler(game.InitializeGameModule(app.Database))
	teamsHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.Database))

	// health
	app.Router.Handle("GET /health", app.publicEndpoint(http.HandlerFunc(handlers.HealthCheck)))

	// metrics
	app.Router.Handle("GET /metrics", app.publicEndpoint(promhttp.Handler()))

	// games
	app.Router.Handle("GET /games", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GetGames)))
	app.Router.Handle("GET /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GetGameByID)))
	app.Router.Handle("POST /games", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.CreateGame)))
	app.Router.Handle("PUT /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.UpdateGame)))
	app.Router.Handle("DELETE /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.DeleteGame)))

	// setup
	app.Router.Handle("POST /games/{gameID}/setup", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GameSetup)))

	// players
	app.Router.Handle("POST /games/{gameID}/teams/{teamID}/players", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.CreatePlayer)))
	app.Router.Handle("PUT /games/{gameID}/teams/{teamID}/players/{playerID}", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.UpdatePlayer)))
	app.Router.Handle("DELETE /games/{gameID}/teams/{teamID}/players/{playerID}", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.DeletePlayer)))

	// tables
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.GetTables)))
	app.Router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.GetTable)))

	// scores
	app.Router.Handle("PUT /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}/scores", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.UpdateTableScore)))

	// teams
	app.Router.Handle("POST /games/{gameID}/teams", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.CreateTeam)))
	app.Router.Handle("PUT /games/{gameID}/teams/{teamID}", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.UpdateTeam)))
	app.Router.Handle("DELETE /games/{gameID}/teams/{teamID}", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.DeleteTeam)))

	return app.Router
}
