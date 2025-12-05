package routes

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

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
	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

func rateLimitConfig() middleware.RateConfig {
	maxRequestsPerSecond, err := strconv.Atoi(os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND"))
	if err != nil {
		maxRequestsPerSecond = 20
		slog.Info("Rate limit requests per seconds (rps) not set, fallback to default", "defaultMaxRequestsPerSecond", maxRequestsPerSecond)
	}
	burstSize, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST_SIZE"))
	if err != nil {
		burstSize = 40
		slog.Info("Rate limit burst not set, fallback to default", "defaultBurstSize", burstSize)
	}

	cacheDefaultDuration, err := time.ParseDuration(os.Getenv("RATE_LIMIT_CACHE_DEFAULT_DURATION"))
	if err != nil {
		cacheDefaultDuration = 5 * time.Minute
		slog.Info("Rate limit cache default duration not set, fallback to default", "defaultCacheDefaultDuration", cacheDefaultDuration)
	}

	cacheCleanupPeriod, err := time.ParseDuration(os.Getenv("RATE_LIMIT_CACHE_CLEANUP_PERIOD"))
	if err != nil {
		slog.Info("Rate limit cache cleanup period not set, defaulting to 1 minute")
		cacheCleanupPeriod = 1 * time.Minute
	}

	return middleware.RateConfig{
		Limit:                rate.Limit(maxRequestsPerSecond),
		Burst:                burstSize,
		CacheDefaultDuration: cacheDefaultDuration,
		CacheCleanupPeriod:   cacheCleanupPeriod,
	}
}

type RouteSetup struct {
	database      *gorm.DB
	authClient    middleware.FirebaseAuth
	router        *http.ServeMux
	healthService *healthpkg.Service
	openAPIConfig []byte
	swaggerDocs   []byte
}

func SetupRouter(database *gorm.DB, authClient middleware.FirebaseAuth, healthClient *healthpkg.Service, openAPIConfig, swaggerDocs []byte) *http.ServeMux {
	instance := RouteSetup{
		database:      database,
		authClient:    authClient,
		router:        http.NewServeMux(),
		healthService: healthClient,
		openAPIConfig: openAPIConfig,
		swaggerDocs:   swaggerDocs,
	}
	instance.setup()

	return instance.router
}

func (app *RouteSetup) publicEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'", middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))))
}

func (app *RouteSetup) publicOpenAPIEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'; style-src 'self' https://unpkg.com; script-src 'self' https://unpkg.com 'unsafe-inline'; img-src 'self' data:", middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))))
}

func (app *RouteSetup) authenticatedEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'", middleware.Metrics(middleware.RequestLogging(slog.LevelInfo, middleware.Authentication(app.authClient, handler)))))
}

func (app *RouteSetup) setup() {
	gameService := game.InitializeGameModule(app.database)
	playerService := player.InitializePlayerModule(app.database)
	tableService := table.InitializeTablesModule(app.database)
	teamService := team.InitializeTeamsModule(app.database)

	healthHandler := handlers.NewHealthHandler(app.healthService)
	openAPIHandler := handlers.NewOpenAPIHandler(app.openAPIConfig, app.swaggerDocs)
	gamesHandler := handlers.NewGamesHandler(gameService)
	playersHandler := handlers.NewPlayersHandler(playerService)
	tablesHandler := handlers.NewTablesHandler(gameService, tableService)
	teamsHandler := handlers.NewTeamsHandler(teamService)

	app.router.Handle("/openapi.yaml", app.publicEndpoint(http.HandlerFunc(openAPIHandler.GetOpenAPIConfig)))
	app.router.Handle("/docs", app.publicOpenAPIEndpoint(http.HandlerFunc(openAPIHandler.GetSwaggerDocs)))

	healthRoutes := health.Handler(healthHandler)
	app.router.Handle("/health", app.publicEndpoint(healthRoutes))
	app.router.Handle("/health/", app.publicEndpoint(healthRoutes))

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
