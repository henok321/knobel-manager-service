package handlers

import (
	"encoding/json"
	"fmt"
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

	createPlayer, err := h.playersService.CreatePlayer(ctx, playersRequest, teamID, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d/teams/%d/players/%d", gameID, teamID, createPlayer.ID))
	writeJSON(ctx, writer, http.StatusCreated, api.PlayersResponse{Player: entityPlayerToAPIPlayer(createPlayer)})
}

func (h *PlayersHandler) UpdatePlayer(writer http.ResponseWriter, request *http.Request, _ /* gameID */, _ /* teamID */, playerID int) {
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

	updatePlayer, err := h.playersService.UpdatePlayer(ctx, playerID, playersRequest, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	writeJSON(ctx, writer, http.StatusOK, api.PlayersResponse{Player: entityPlayerToAPIPlayer(updatePlayer)})
}

func (h *PlayersHandler) DeletePlayer(writer http.ResponseWriter, request *http.Request, _ /* gameID */, _ /* teamID */, playerID int) {
	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := h.playersService.DeletePlayer(request.Context(), playerID, sub); err != nil {
		respondError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
