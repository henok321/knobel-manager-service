package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/player"
)

type PlayersHandler struct {
	playersService *player.PlayersService
}

func NewPlayersHandler(service *player.PlayersService) *PlayersHandler {
	return &PlayersHandler{
		playersService: service,
	}
}

func (h *PlayersHandler) CreatePlayer(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	playersRequest := api.PlayersRequest{}

	if err := json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if playersRequest.Name == "" {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	createPlayer, err := h.playersService.CreatePlayer(ctx, playersRequest, gameID, teamID, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d/teams/%d/players/%d", gameID, teamID, createPlayer.ID))
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)

	response := api.PlayersResponse{
		Player: entityPlayerToAPIPlayer(createPlayer),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *PlayersHandler) UpdatePlayer(writer http.ResponseWriter, request *http.Request, gameID, _ /* teamID */, playerID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	playersRequest := api.PlayersRequest{}

	if err := json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if playersRequest.Name == "" {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatePlayer, err := h.playersService.UpdatePlayer(ctx, gameID, playerID, playersRequest, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := api.PlayersResponse{
		Player: entityPlayerToAPIPlayer(updatePlayer),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *PlayersHandler) DeletePlayer(writer http.ResponseWriter, request *http.Request, gameID, _ /* teamID */, playerID int) {
	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := h.playersService.DeletePlayer(request.Context(), gameID, playerID, sub); err != nil {
		respondError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
