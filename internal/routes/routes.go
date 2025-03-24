package routes

import (
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/pkg/team"

	"github.com/henok321/knobel-manager-service/pkg/player"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type RouteSetup struct {
	database   *gorm.DB
	authClient middleware.FirebaseAuth
	router     *http.ServeMux
}

func SetupRouter(database *gorm.DB, authClient middleware.FirebaseAuth) *http.ServeMux {
	instance := RouteSetup{
		database:   database,
		authClient: authClient,
		router:     http.NewServeMux(),
	}
	instance.init()

	return instance.router
}

func (app *RouteSetup) publicEndpoint(handler http.Handler) http.Handler {
	return middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))
}

func (app *RouteSetup) authenticatedEndpoint(handler http.Handler) http.Handler {
	return middleware.Metrics(middleware.RequestLogging(slog.LevelInfo, middleware.Authentication(app.authClient, handler)))
}

func (app *RouteSetup) init() {

	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.database))
	playersHandler := handlers.NewPlayersHandler(player.InitializePlayerModule(app.database))
	tablesHandler := handlers.NewTablesHandler(game.InitializeGameModule(app.database))
	teamsHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.database))

	// health
	app.router.Handle("GET /health", app.publicEndpoint(http.HandlerFunc(handlers.HealthCheck)))

	// games
	app.router.Handle("GET /games", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GetGames)))
	app.router.Handle("GET /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GetGameByID)))
	app.router.Handle("POST /games", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.CreateGame)))
	app.router.Handle("PUT /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.UpdateGame)))
	app.router.Handle("DELETE /games/{gameID}", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.DeleteGame)))
	app.router.Handle("POST /games/{gameID}/activate", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.SetActiveGame)))

	// setup
	app.router.Handle("POST /games/{gameID}/setup", app.authenticatedEndpoint(http.HandlerFunc(gamesHandler.GameSetup)))

	// players
	app.router.Handle("POST /games/{gameID}/teams/{teamID}/players", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.CreatePlayer)))
	app.router.Handle("PUT /games/{gameID}/teams/{teamID}/players/{playerID}", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.UpdatePlayer)))
	app.router.Handle("DELETE /games/{gameID}/teams/{teamID}/players/{playerID}", app.authenticatedEndpoint(http.HandlerFunc(playersHandler.DeletePlayer)))

	// tables
	app.router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.GetTables)))
	app.router.Handle("GET /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.GetTable)))

	// scores
	app.router.Handle("PUT /games/{gameID}/rounds/{roundNumber}/tables/{tableNumber}/scores", app.authenticatedEndpoint(http.HandlerFunc(tablesHandler.UpdateTableScore)))

	// teams
	app.router.Handle("POST /games/{gameID}/teams", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.CreateTeam)))
	app.router.Handle("PUT /games/{gameID}/teams/{teamID}", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.UpdateTeam)))
	app.router.Handle("DELETE /games/{gameID}/teams/{teamID}", app.authenticatedEndpoint(http.HandlerFunc(teamsHandler.DeleteTeam)))
}
