package routes

import (
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/health"
	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/gen/scores"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/pkg/table"
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
	instance.setup()

	return instance.router
}

func (app *RouteSetup) publicEndpoint(handler http.Handler) http.Handler {
	return middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))
}

func (app *RouteSetup) authenticatedEndpoint(handler http.Handler) http.Handler {
	return middleware.Metrics(middleware.RequestLogging(slog.LevelInfo, middleware.Authentication(app.authClient, handler)))
}

func (app *RouteSetup) setup() {
	// Initialize services
	gameService := game.InitializeGameModule(app.database)
	playerService := player.InitializePlayerModule(app.database)
	tableService := table.InitializeTablesModule(app.database)
	teamService := team.InitializeTeamsModule(app.database)

	// Create handlers that implement OpenAPI interfaces
	healthHandler := handlers.NewHealthHandler()
	gamesHandler := handlers.NewGamesHandler(gameService)
	playersHandler := handlers.NewPlayersHandler(playerService)
	tablesHandler := handlers.NewTablesHandler(gameService, tableService)
	teamsHandler := handlers.NewTeamsHandler(teamService)

	// Register routes using generated OpenAPI handlers
	// Health endpoint (public)
	app.router.Handle("/health", app.publicEndpoint(health.Handler(healthHandler)))

	// Games endpoints (authenticated)
	gamesRoutes := games.HandlerWithOptions(gamesHandler, games.StdHTTPServerOptions{
		ErrorHandlerFunc: gamesHandler.HandleValidationError,
	})
	app.router.Handle("/games", app.authenticatedEndpoint(gamesRoutes))
	app.router.Handle("/games/", app.authenticatedEndpoint(gamesRoutes))

	// Teams endpoints (authenticated)
	teamsRoutes := teams.HandlerWithOptions(teamsHandler, teams.StdHTTPServerOptions{
		ErrorHandlerFunc: teamsHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/teams", app.authenticatedEndpoint(teamsRoutes))
	app.router.Handle("/games/{gameId}/teams/", app.authenticatedEndpoint(teamsRoutes))

	// Players endpoints (authenticated)
	playersRoutes := players.HandlerWithOptions(playersHandler, players.StdHTTPServerOptions{
		ErrorHandlerFunc: playersHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/teams/{teamId}/players", app.authenticatedEndpoint(playersRoutes))
	app.router.Handle("/games/{gameId}/teams/{teamId}/players/", app.authenticatedEndpoint(playersRoutes))

	// Tables endpoints (authenticated)
	tablesRoutes := tables.HandlerWithOptions(tablesHandler, tables.StdHTTPServerOptions{
		ErrorHandlerFunc: tablesHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables", app.authenticatedEndpoint(tablesRoutes))
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables/", app.authenticatedEndpoint(tablesRoutes))

	// Scores endpoints (authenticated) - uses same tablesHandler which implements scores.ServerInterface
	scoresRoutes := scores.HandlerWithOptions(tablesHandler, scores.StdHTTPServerOptions{
		ErrorHandlerFunc: tablesHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables/{tableNumber}/scores", app.authenticatedEndpoint(scoresRoutes))
}
