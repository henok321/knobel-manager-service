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

type routeSetup struct {
	database      *gorm.DB
	authClient    middleware.FirebaseAuth
	healthService *healthpkg.Service
	openAPIConfig []byte
	swaggerDocs   []byte
}

func SetupRouter(database *gorm.DB, authClient middleware.FirebaseAuth, healthClient *healthpkg.Service, openAPIConfig, swaggerDocs []byte) *http.ServeMux {
	instance := routeSetup{
		database:      database,
		authClient:    authClient,
		healthService: healthClient,
		openAPIConfig: openAPIConfig,
		swaggerDocs:   swaggerDocs,
	}
	return instance.setup()
}

func (app *routeSetup) publicEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'", middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))))
}

func (app *routeSetup) publicOpenAPIEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'; style-src 'self' https://unpkg.com; script-src 'self' https://unpkg.com 'unsafe-inline'; img-src 'self' data:", middleware.Metrics(middleware.RequestLogging(slog.LevelDebug, handler))))
}

func (app *routeSetup) authenticatedEndpoint(handler http.Handler) http.Handler {
	return middleware.RateLimit(rateLimitConfig(), middleware.SecurityHeaders("default-src 'self'", middleware.Metrics(middleware.RequestLogging(slog.LevelInfo, middleware.Authentication(app.authClient, handler)))))
}

func (app *routeSetup) setup() *http.ServeMux {
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

	router := http.NewServeMux()

	router.Handle("/openapi.yaml", app.publicEndpoint(http.HandlerFunc(openAPIHandler.GetOpenAPIConfig)))
	router.Handle("/docs", app.publicOpenAPIEndpoint(http.HandlerFunc(openAPIHandler.GetSwaggerDocs)))

	handleValidationErrors := func(w http.ResponseWriter, _ *http.Request, err error) {
		handlers.JSONError(w, err.Error(), http.StatusBadRequest)
	}

	health.HandlerWithOptions(healthHandler, health.StdHTTPServerOptions{
		BaseRouter:  router,
		Middlewares: []health.MiddlewareFunc{app.publicEndpoint},
	})

	games.HandlerWithOptions(gamesHandler, games.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []games.MiddlewareFunc{app.authenticatedEndpoint},
	})

	teams.HandlerWithOptions(teamsHandler, teams.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []teams.MiddlewareFunc{app.authenticatedEndpoint},
	})

	players.HandlerWithOptions(playersHandler, players.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []players.MiddlewareFunc{app.authenticatedEndpoint},
	})

	tables.HandlerWithOptions(tablesHandler, tables.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []tables.MiddlewareFunc{app.authenticatedEndpoint},
	})

	scores.HandlerWithOptions(tablesHandler, scores.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []scores.MiddlewareFunc{app.authenticatedEndpoint},
	})

	return router
}
