package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/types"
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

func (h *GamesHandler) HandleValidationError(w http.ResponseWriter, _ *http.Request, err error) {
	if strings.Contains(err.Error(), "Invalid format for parameter gameID") {
		JSONError(w, "Invalid gameID", http.StatusBadRequest)
		return
	}
	JSONError(w, err.Error(), http.StatusBadRequest)
}

func (h *GamesHandler) GetGames(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gamesList, err := h.gamesService.FindAllByOwner(sub)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	apiGames := make([]games.Game, len(gamesList))
	for i, g := range gamesList {
		apiGames[i] = entityGameToAPIGame(g)
	}

	response := games.GamesResponse{
		Games: apiGames,
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) GetGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := h.gamesService.FindByID(gameID, sub)
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

	response := entityGameToAPIGame(gameByID)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := types.GameCreateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameCreateRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if gameCreateRequest.Name == "" || gameCreateRequest.NumberOfRounds == 0 ||
		gameCreateRequest.TeamSize == 0 || gameCreateRequest.TableSize == 0 {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	createdGame, err := h.gamesService.CreateGame(sub, &gameCreateRequest)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", fmt.Sprintf("/games/%d", createdGame.ID))
	writer.WriteHeader(http.StatusCreated)

	response := games.GameResponse{
		Game: entityGameToAPIGame(createdGame),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameUpdateRequest := types.GameUpdateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	if gameUpdateRequest.Name == "" || gameUpdateRequest.NumberOfRounds == 0 ||
		gameUpdateRequest.TeamSize == 0 || gameUpdateRequest.TableSize == 0 {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(gameID, sub, gameUpdateRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Not owner", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrInvalidGameSetup):
			JSONError(writer, "Invalid game setup", http.StatusConflict)

		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	response := games.GameResponse{
		Game: entityGameToAPIGame(updatedGame),
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	if err := h.gamesService.DeleteGame(gameID, sub); err != nil {
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
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameToAssign, err := h.gamesService.FindByID(gameID, sub)
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

	err = h.gamesService.AssignTables(gameToAssign)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d", gameID))
	writer.WriteHeader(http.StatusCreated)
}
