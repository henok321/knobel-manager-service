package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/player"
)

type PlayersHandler struct {
	playersService player.PlayersService
}

func NewPlayersHandler(service player.PlayersService) *PlayersHandler {
	return &PlayersHandler{
		playersService: service,
	}
}

var _ players.ServerInterface = (*PlayersHandler)(nil)

func (h *PlayersHandler) CreatePlayer(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	playersRequest := players.PlayersRequest{}

	if err := json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if playersRequest.Name == "" {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	createPlayer, err := h.playersService.CreatePlayer(ctx, playersRequest, teamID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrTeamNotFound):
			JSONError(writer, "Team not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d/teams/%d/players/%d", gameID, teamID, createPlayer.ID))
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)

	response := players.PlayersResponse{
		Player: entityPlayerToAPIPlayer(createPlayer),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *PlayersHandler) UpdatePlayer(writer http.ResponseWriter, request *http.Request, _ /* gameID */, _ /* teamID */, playerID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	playersRequest := players.PlayersRequest{}

	if err := json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if playersRequest.Name == "" {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatePlayer, err := h.playersService.UpdatePlayer(ctx, playerID, playersRequest, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrTeamNotFound), errors.Is(err, apperror.ErrPlayerNotFound):
			JSONError(writer, err.Error(), http.StatusNotFound)
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := players.PlayersResponse{
		Player: entityPlayerToAPIPlayer(updatePlayer),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *PlayersHandler) DeletePlayer(writer http.ResponseWriter, request *http.Request, _ /* gameID */, _ /* teamID */, playerID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	err := h.playersService.DeletePlayer(ctx, playerID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrPlayerNotFound):
			JSONError(writer, err.Error(), http.StatusNotFound)
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
