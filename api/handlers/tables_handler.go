package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/customError"

	"github.com/go-playground/validator/v10"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type TablesHandler interface {
	GetTables(writer http.ResponseWriter, request *http.Request)
	GetTable(writer http.ResponseWriter, request *http.Request)
	UpdateTableScore(writer http.ResponseWriter, request *http.Request)
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
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseInt(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	roundNumber, err := strconv.ParseInt(request.PathValue("roundNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid roundNumber", http.StatusBadRequest)
		return
	}

	gameById, err := t.gamesService.FindByID(int(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
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
		if round.RoundNumber == int(roundNumber) {
			tables := round.Tables
			writer.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(writer).Encode(tables); err != nil {
				slog.InfoContext(request.Context(), "Could not write body", "error", err)
			}
			return
		}
	}

	JSONError(writer, "Round not found", http.StatusNotFound)
}

func (t tablesHandler) GetTable(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseInt(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	roundNumber, err := strconv.ParseInt(request.PathValue("roundNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid roundNumber", http.StatusBadRequest)
		return
	}

	tablesNumber, err := strconv.ParseInt(request.PathValue("tableNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid tableNumber", http.StatusBadRequest)
		return
	}

	gameById, err := t.gamesService.FindByID(int(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
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
		if round.RoundNumber == int(roundNumber) {
			for _, table := range round.Tables {
				if table.TableNumber == int(tablesNumber) {
					writer.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(writer).Encode(table); err != nil {
						slog.InfoContext(request.Context(), "Could not write body", "error", err)
					}
					return
				}
			}
		}
	}
	JSONError(writer, "Round or table not found", http.StatusNotFound)
}

func (t tablesHandler) UpdateTableScore(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseInt(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	roundNumber, err := strconv.ParseInt(request.PathValue("roundNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid roundNumber", http.StatusBadRequest)
		return
	}

	tableNumber, err := strconv.ParseInt(request.PathValue("tableNumber"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid tableNumber", http.StatusBadRequest)
		return
	}

	scoresRequest := game.ScoresRequest{}

	if err := json.NewDecoder(request.Body).Decode(&scoresRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()

	if err := validate.Struct(scoresRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := t.gamesService.UpdateScore(int(gameID), int(roundNumber), int(tableNumber), sub, scoresRequest)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, customError.InvalidScore):
			JSONError(writer, "Invalid score", http.StatusBadRequest)
		case errors.Is(err, customError.RoundOrTableNotFound):
			JSONError(writer, "Round or table not found", http.StatusNotFound)
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := json.NewEncoder(writer).Encode(gameResponse{Game: updatedGame}); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}
