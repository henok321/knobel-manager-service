package routes

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/health"
	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/gen/scores"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"github.com/henok321/knobel-manager-service/pkg/table"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"golang.org/x/time/rate"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

func rateLimit() (rate.Limit, int) {
	maxRequestsPerSecond, err := strconv.Atoi(os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND"))
	if err != nil {
		slog.Info("Rate limit requests per seconds (rps) not set, defaulting to 100 requests per second")
		maxRequestsPerSecond = 100
	}
	burstSize, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST_SIZE"))
	if err != nil {
		slog.Info("Rate limit burst not set, defaulting to 20 requests")
		burstSize = 20
	}
	return rate.Limit(maxRequestsPerSecond), burstSize
}

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

func getIP(r *http.Request) string {
	return r.Header.Get("X-Forwarded-For")
}

func (app *RouteSetup) publicEndpoint(handler http.Handler) http.Handler {
	limit, burst := rateLimit()
	return middleware.RateLimit(getIP, limit, burst, middleware.SecurityHeaders(middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))))
}

func (app *RouteSetup) authenticatedEndpoint(handler http.Handler) http.Handler {
	limit, burst := rateLimit()

	return middleware.RateLimit(getIP, limit, burst, middleware.SecurityHeaders(middleware.Metrics(middleware.RequestLogging(slog.LevelInfo, middleware.Authentication(app.authClient, handler)))))
}

func (app *RouteSetup) setup() {
	gameService := game.InitializeGameModule(app.database)
	playerService := player.InitializePlayerModule(app.database)
	tableService := table.InitializeTablesModule(app.database)
	teamService := team.InitializeTeamsModule(app.database)

	healthHandler := handlers.NewHealthHandler()
	openAPIHandler := handlers.NewOpenAPIHandler()
	gamesHandler := handlers.NewGamesHandler(gameService)
	playersHandler := handlers.NewPlayersHandler(playerService)
	tablesHandler := handlers.NewTablesHandler(gameService, tableService)
	teamsHandler := handlers.NewTeamsHandler(teamService)

	app.router.Handle("/openapi.yaml", app.publicEndpoint(http.HandlerFunc(openAPIHandler.GetOpenAPIConfig)))
	app.router.Handle("/docs", app.publicEndpoint(http.HandlerFunc(openAPIHandler.GetSwaggerDocs)))

	app.router.Handle("/health", app.publicEndpoint(health.Handler(healthHandler)))

	gamesRoutes := games.HandlerWithOptions(gamesHandler, games.StdHTTPServerOptions{
		ErrorHandlerFunc: gamesHandler.HandleValidationError,
	})
	app.router.Handle("/games", app.authenticatedEndpoint(gamesRoutes))
	app.router.Handle("/games/", app.authenticatedEndpoint(gamesRoutes))

	teamsRoutes := teams.HandlerWithOptions(teamsHandler, teams.StdHTTPServerOptions{
		ErrorHandlerFunc: teamsHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/teams", app.authenticatedEndpoint(teamsRoutes))
	app.router.Handle("/games/{gameId}/teams/", app.authenticatedEndpoint(teamsRoutes))

	playersRoutes := players.HandlerWithOptions(playersHandler, players.StdHTTPServerOptions{
		ErrorHandlerFunc: playersHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/teams/{teamId}/players", app.authenticatedEndpoint(playersRoutes))
	app.router.Handle("/games/{gameId}/teams/{teamId}/players/", app.authenticatedEndpoint(playersRoutes))

	tablesRoutes := tables.HandlerWithOptions(tablesHandler, tables.StdHTTPServerOptions{
		ErrorHandlerFunc: tablesHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables", app.authenticatedEndpoint(tablesRoutes))
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables/", app.authenticatedEndpoint(tablesRoutes))

	scoresRoutes := scores.HandlerWithOptions(tablesHandler, scores.StdHTTPServerOptions{
		ErrorHandlerFunc: tablesHandler.HandleValidationError,
	})
	app.router.Handle("/games/{gameId}/rounds/{roundNumber}/tables/{tableNumber}/scores", app.authenticatedEndpoint(scoresRoutes))
}
