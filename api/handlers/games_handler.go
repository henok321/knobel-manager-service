package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type GamesHandler struct {
	gamesService game.GamesService
}

func NewGamesHandler(gamesService game.GamesService) *GamesHandler {
	return &GamesHandler{gamesService}
}

var _ games.ServerInterface = (*GamesHandler)(nil)

func (h *GamesHandler) GetGames(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	allGames, err := h.gamesService.FindAllByOwner(ctx, sub)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	apiGames := make([]games.Game, len(allGames))
	for i, entry := range allGames {
		apiGames[i] = entityGameToGamesGame(entry)
	}

	response := games.GamesResponse{
		Games: apiGames,
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) GetGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := h.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := entityGameToGamesGame(gameByID)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := games.GameCreateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameCreateRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if gameCreateRequest.Name == "" || gameCreateRequest.NumberOfRounds == 0 ||
		gameCreateRequest.TeamSize == 0 || gameCreateRequest.TableSize == 0 {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	createdGame, err := h.gamesService.CreateGame(ctx, sub, &gameCreateRequest)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", fmt.Sprintf("/games/%d", createdGame.ID))
	writer.WriteHeader(http.StatusCreated)

	response := games.GameResponse{
		Game: entityGameToGamesGame(createdGame),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameUpdateRequest := games.GameUpdateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	if gameUpdateRequest.Name == "" || gameUpdateRequest.NumberOfRounds == 0 ||
		gameUpdateRequest.TeamSize == 0 || gameUpdateRequest.TableSize == 0 {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(ctx, gameID, sub, gameUpdateRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Not owner", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrInvalidGameSetup):
			JSONError(writer, "Invalid game setup", http.StatusConflict)
		case errors.Is(err, apperror.ErrGameIncomplete):
			JSONError(writer, "Game is complete", http.StatusConflict)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
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

func (h *GamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	if err := h.gamesService.DeleteGame(ctx, gameID, sub); err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *GamesHandler) SetupGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameToAssign, err := h.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	if gameToAssign.Status != entity.StatusSetup {
		JSONError(writer, "Game is not in setup state", http.StatusBadRequest)
		return
	}

	err = h.gamesService.AssignTables(ctx, gameToAssign)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d", gameID))
	writer.WriteHeader(http.StatusCreated)
}
