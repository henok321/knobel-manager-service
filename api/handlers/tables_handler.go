package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/scores"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/table"
)

type TablesHandler struct {
	gamesService  game.GamesService
	tablesService table.TablesService
}

func NewTablesHandler(gamesService game.GamesService, tablesService table.TablesService) *TablesHandler {
	return &TablesHandler{gamesService: gamesService, tablesService: tablesService}
}

// Verify that TablesHandler implements both generated OpenAPI interfaces
var (
	_ tables.ServerInterface = (*TablesHandler)(nil)
	_ scores.ServerInterface = (*TablesHandler)(nil)
)

func (t *TablesHandler) HandleValidationError(w http.ResponseWriter, _ *http.Request, err error) {
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "Invalid format for parameter gameID"):
		JSONError(w, "Invalid gameID", http.StatusBadRequest)
	case strings.Contains(errorMsg, "Invalid format for parameter roundNumber"):
		JSONError(w, "Invalid roundNumber", http.StatusBadRequest)
	case strings.Contains(errorMsg, "Invalid format for parameter tableNumber"):
		JSONError(w, "Invalid tableNumber", http.StatusBadRequest)
	default:
		JSONError(w, errorMsg, http.StatusBadRequest)
	}
}

func (t *TablesHandler) GetTables(writer http.ResponseWriter, request *http.Request, gameID, roundNumber int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := t.gamesService.FindByID(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
			return
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
			return
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, round := range gameByID.Rounds {
		if round.RoundNumber == roundNumber {
			// Convert entity tables to API tables
			apiTables := make([]tables.Table, len(round.Tables))
			for i, t := range round.Tables {
				apiTables[i] = entityTableToTablesAPITable(*t)
			}

			response := tables.TablesResponse{
				Tables: apiTables,
			}

			writer.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(writer).Encode(response); err != nil {
				slog.InfoContext(request.Context(), "Could not write body", "error", err)
			}

			return
		}
	}

	JSONError(writer, "Round not found", http.StatusNotFound)
}

func (t *TablesHandler) GetTable(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := t.gamesService.FindByID(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
			return
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
			return
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, round := range gameByID.Rounds {
		if round.RoundNumber == roundNumber {
			for _, currentTable := range round.Tables {
				if currentTable.TableNumber == tableNumber {
					response := entityTableToTablesAPITable(*currentTable)

					writer.WriteHeader(http.StatusOK)

					if err := json.NewEncoder(writer).Encode(response); err != nil {
						slog.InfoContext(request.Context(), "Could not write body", "error", err)
					}

					return
				}
			}
		}
	}

	JSONError(writer, "Round or table not found", http.StatusNotFound)
}

func (t *TablesHandler) UpdateScores(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	scoresRequest := types.ScoresRequest{}

	if err := json.NewDecoder(request.Body).Decode(&scoresRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if len(scoresRequest.Scores) == 0 {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := t.tablesService.UpdateScore(gameID, roundNumber, tableNumber, sub, scoresRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrInvalidScore):
			JSONError(writer, "Invalid score", http.StatusBadRequest)
		case errors.Is(err, apperror.ErrRoundOrTableNotFound):
			JSONError(writer, "Round or table not found", http.StatusNotFound)
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	updatedGame, err := t.gamesService.FindByID(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}
	}

	response := types.GameResponse{
		Game: entityGameToTypesAPIGame(updatedGame),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}
