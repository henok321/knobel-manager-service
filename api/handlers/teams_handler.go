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
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type teamResponse struct {
	Team entity.Team `json:"team"`
}

type TeamsHandler struct {
	service team.TeamsService
}

func NewTeamsHandler(service team.TeamsService) *TeamsHandler {
	return &TeamsHandler{service}
}

func (t TeamsHandler) CreateTeam(writer http.ResponseWriter, request *http.Request) {
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

	teamsRequest := team.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()
	if err := validate.Struct(teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	createdTeam, err := t.service.CreateTeam(int(gameID), sub, teamsRequest)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, customError.TeamSizeNotAllowed):
			JSONError(writer, "Invalid team size", http.StatusBadRequest)

		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusCreated)
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", request.URL.String()+"/"+strconv.FormatInt(int64(createdTeam.ID), 10))

	teamResponse := teamResponse{Team: createdTeam}

	if err := json.NewEncoder(writer).Encode(teamResponse); err != nil {
		slog.InfoContext(request.Context(), "Could not write body", "error", err)
	}

}

func (t TeamsHandler) UpdateTeam(writer http.ResponseWriter, request *http.Request) {
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

	teamsRequest := team.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()

	if err := validate.Struct(teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	updatedGame, err := t.service.UpdateTeam(int(gameID), sub, int(teamID), teamsRequest)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, customError.TeamNotFound):
			JSONError(writer, "Team not found", http.StatusNotFound)
		}
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "application/json")

	response := teamResponse{Team: updatedGame}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(request.Context(), "Could not write body", "error", err)
	}
}

func (t TeamsHandler) DeleteTeam(writer http.ResponseWriter, request *http.Request) {
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

	err = t.service.DeleteTeam(int(gameID), sub, int(teamID))

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusNoContent)

}
