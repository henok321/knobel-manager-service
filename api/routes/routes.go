package routes

import (
	"log/slog"
	"net/http"
	"slices"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/handlers"
	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/gen/health"
	"github.com/henok321/knobel-manager-service/pkg/audit"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"github.com/henok321/knobel-manager-service/pkg/table"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type apiServer struct {
	*handlers.GamesHandler
	*handlers.TeamsHandler
	*handlers.PlayersHandler
	*handlers.TablesHandler
	*handlers.AuditHandler
}

var _ api.ServerInterface = (*apiServer)(nil)

func chain(mw ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for _, v := range slices.Backward(mw) {
			h = v(h)
		}
		return h
	}
}

func serveBytes(contentType string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "inline")
		if _, err := w.Write(body); err != nil {
			slog.Error("Could not write response body", "error", err)
		}
	}
}

func SetupRouter(database *gorm.DB, authClient middleware.FirebaseAuth, healthService *healthpkg.Service, openAPIConfig, swaggerDocs []byte) *http.ServeMux {
	public := func(csp string) func(http.Handler) http.Handler {
		return chain(
			middleware.SecurityHeaders(csp),
			middleware.Metrics(),
			middleware.RequestLogging(slog.LevelDebug),
		)
	}

	authenticated := chain(
		middleware.SecurityHeaders("default-src 'self'"),
		middleware.Metrics(),
		middleware.RequestLogging(slog.LevelInfo),
		middleware.Authentication(authClient),
	)

	gameService := game.NewGamesService(game.NewGamesRepository(database), authClient)
	playerService := player.NewPlayersService(player.NewPlayersRepository(database), team.NewTeamsRepository(database), gameService)
	tableService := table.NewTablesService(table.NewTablesRepository(database))
	teamService := team.NewTeamsService(team.NewTeamsRepository(database), gameService)
	auditService := audit.NewEventsService(audit.NewEventsRepository(database), game.NewGamesRepository(database))

	healthHandler := handlers.NewHealthHandler(healthService)
	gamesHandler := handlers.NewGamesHandler(gameService, authClient)
	playersHandler := handlers.NewPlayersHandler(playerService)
	tablesHandler := handlers.NewTablesHandler(gameService, tableService)
	teamsHandler := handlers.NewTeamsHandler(teamService)
	auditHandler := handlers.NewAuditHandler(auditService)

	router := http.NewServeMux()

	router.Handle("/openapi.yaml", public("default-src 'self'")(serveBytes("text/yaml; charset=utf-8", openAPIConfig)))
	router.Handle("/docs", public("default-src 'self'; style-src 'self' https://unpkg.com; script-src 'self' https://unpkg.com 'unsafe-inline'; img-src 'self' data:")(serveBytes("text/html; charset=utf-8", swaggerDocs)))

	handleValidationErrors := func(w http.ResponseWriter, _ *http.Request, err error) {
		handlers.JSONError(w, err.Error(), http.StatusBadRequest)
	}

	health.HandlerWithOptions(healthHandler, health.StdHTTPServerOptions{
		BaseRouter:  router,
		Middlewares: []health.MiddlewareFunc{public("default-src 'self'")},
	})

	api.HandlerWithOptions(&apiServer{gamesHandler, teamsHandler, playersHandler, tablesHandler, auditHandler}, api.StdHTTPServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handleValidationErrors,
		Middlewares:      []api.MiddlewareFunc{authenticated},
	})

	return router
}
