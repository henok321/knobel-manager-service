package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/customError"

	"github.com/go-playground/validator/v10"
	"github.com/henok321/knobel-manager-service/api/middleware"

	"github.com/henok321/knobel-manager-service/pkg/player"
)

type PlayersHandler struct {
	playersService *player.PlayersService
}

func NewPlayersHandler(service *player.PlayersService) PlayersHandler {
	return PlayersHandler{
		playersService: service,
	}
}

func (h PlayersHandler) CreatePlayer(writer http.ResponseWriter, request *http.Request) {
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

	teamID, err := strconv.ParseInt(request.PathValue("teamID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid teamID", http.StatusBadRequest)
		return
	}

	playersRequest := player.PlayersRequest{}

	if err = json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()

	err = validate.Struct(playersRequest)

	if err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	createPlayer, err := h.playersService.CreatePlayer(playersRequest, int(teamID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.TeamNotFound):
			JSONError(writer, "Team not found", http.StatusNotFound)
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return

	}

	writer.Header().Set("Location", fmt.Sprintf("/games/%d/teams/%d/players/%d", gameID, teamID, createPlayer.ID))
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)

	response := player.PlayersResponse{Player: player.Player{
		ID:     createPlayer.ID,
		Name:   createPlayer.Name,
		TeamID: createPlayer.TeamID,
	}}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}

}

func (h PlayersHandler) UpdatePlayer(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	playerID, err := strconv.ParseInt(request.PathValue("playerID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid playerID", http.StatusBadRequest)
		return
	}

	playersRequest := player.PlayersRequest{}

	if err = json.NewDecoder(request.Body).Decode(&playersRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()

	err = validate.Struct(playersRequest)

	if err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatePlayer, err := h.playersService.UpdatePlayer(int(playerID), playersRequest, sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.TeamNotFound), errors.Is(err, customError.PlayerNotFound):
			JSONError(writer, err.Error(), http.StatusNotFound)
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := player.PlayersResponse{Player: player.Player{
		ID:     updatePlayer.ID,
		Name:   updatePlayer.Name,
		TeamID: updatePlayer.TeamID,
	}}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h PlayersHandler) DeletePlayer(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	playerID, err := strconv.ParseInt(request.PathValue("playerID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid playerID", http.StatusBadRequest)
		return
	}

	err = h.playersService.DeletePlayer(int(playerID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.PlayerNotFound):
			JSONError(writer, err.Error(), http.StatusNotFound)
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusNoContent)

}
