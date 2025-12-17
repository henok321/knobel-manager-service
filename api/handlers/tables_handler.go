package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/scores"
	"github.com/henok321/knobel-manager-service/gen/tables"
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

var (
	_ tables.ServerInterface = (*TablesHandler)(nil)
	_ scores.ServerInterface = (*TablesHandler)(nil)
)

func (t *TablesHandler) GetTables(writer http.ResponseWriter, request *http.Request, gameID, roundNumber int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := t.gamesService.FindByID(ctx, gameID, sub)
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
			apiTables := make([]tables.Table, len(round.Tables))
			for i, t := range round.Tables {
				apiTables[i] = entityTableToTablesTable(*t)
			}

			response := tables.TablesResponse{
				Tables: apiTables,
			}

			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(writer).Encode(response); err != nil {
				slog.InfoContext(ctx, "Could not write body", "error", err)
			}

			return
		}
	}

	JSONError(writer, "Round not found", http.StatusNotFound)
}

func (t *TablesHandler) GetTable(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := t.gamesService.FindByID(ctx, gameID, sub)
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
					response := entityTableToGamesTable(*currentTable)

					writer.Header().Set("Content-Type", "application/json")
					writer.WriteHeader(http.StatusOK)

					if err := json.NewEncoder(writer).Encode(response); err != nil {
						slog.InfoContext(ctx, "Could not write body", "error", err)
					}

					return
				}
			}
		}
	}

	JSONError(writer, "Round or table not found", http.StatusNotFound)
}

func (t *TablesHandler) UpdateScores(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	scoresRequest := scores.ScoresRequest{}

	if err := json.NewDecoder(request.Body).Decode(&scoresRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if len(scoresRequest.Scores) == 0 {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := t.tablesService.UpdateScore(ctx, gameID, roundNumber, tableNumber, sub, scoresRequest)
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

	updatedGame, err := t.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	response := games.GameResponse{
		Game: entityGameToGamesGame(updatedGame),
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}
