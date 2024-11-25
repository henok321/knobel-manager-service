package app

import (
	"net/http"

	"github.com/henok321/knobel-manager-service/api/handlers"
	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"gorm.io/gorm"
)

type App struct {
	DB *gorm.DB
}

func (app *App) Initialize(authMiddleware func(next http.Handler) http.Handler) http.Handler {

	gamesHandler := handlers.NewGamesHandler(game.InitializeGameModule(app.DB))

	/*teamHandler := handlers.NewTeamsHandler(team.InitializeTeamsModule(app.DB))
	playerHandler :=   handlers.NewPlayersHandler(player.InitializePlayerModule(app.DB))
	app.TablesHandler = handlers.NewTablesHandler(game.InitializeGameModule(app.DB))
	app.Router = http.NewServeMux()*/

	router := http.NewServeMux()

	router.Handle("GET /health", http.HandlerFunc(handlers.HealthCheck))

	router.Handle("GET /games", authMiddleware(http.HandlerFunc(gamesHandler.GetGames)))
	router.Handle("GET /games/{gameID}", authMiddleware(http.HandlerFunc(gamesHandler.GetGameByID)))
	router.Handle("POST /games", authMiddleware(http.HandlerFunc(gamesHandler.CreateGame)))
	router.Handle("PUT /games/{gameID}", authMiddleware(http.HandlerFunc(gamesHandler.UpdateGame)))
	router.Handle("DELETE /games/{gameID}", authMiddleware(http.HandlerFunc(gamesHandler.DeleteGame)))

	return middleware.RequestLogging(router)
}
