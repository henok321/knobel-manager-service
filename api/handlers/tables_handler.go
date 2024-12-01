package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type TablesHandler interface {
	GetTables(writer http.ResponseWriter, request *http.Request)
	GetTable(writer http.ResponseWriter, request *http.Request)
}

type tablesHandler struct {
	gamesService game.GamesService
}

func NewTablesHandler(gamesService game.GamesService) TablesHandler {
	return tablesHandler{gamesService: gamesService}
}

func (t tablesHandler) GetTables(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	roundNumber, err := strconv.ParseUint(request.PathValue("roundNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid roundNumber", http.StatusBadRequest)
		return
	}

	gameById, err := t.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
			return
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
			return
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == uint(roundNumber) {
			tables := round.Tables
			writer.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(writer).Encode(tables); err != nil {
				slog.Info("Could not write body", "error", err)
			}
			return
		}
	}

	JSONError(writer, "Round not found", http.StatusNotFound)
}

func (t tablesHandler) GetTable(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	roundNumber, err := strconv.ParseUint(request.PathValue("roundNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid roundNumber", http.StatusBadRequest)
		return
	}

	tablesNumber, err := strconv.ParseUint(request.PathValue("tableNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid tableNumber", http.StatusBadRequest)
		return
	}

	gameById, err := t.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
			return
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
			return
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == uint(roundNumber) {
			for _, table := range round.Tables {
				if table.TableNumber == uint(tablesNumber) {
					writer.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(writer).Encode(table); err != nil {
						slog.Info("Could not write body", "error", err)
					}
					return
				}
			}
		}
	}
	JSONError(writer, "Round or table not found", http.StatusNotFound)
}
