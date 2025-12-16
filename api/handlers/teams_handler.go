package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type TeamsHandler struct {
	service team.TeamsService
}

func NewTeamsHandler(service team.TeamsService) *TeamsHandler {
	return &TeamsHandler{service}
}

var _ teams.ServerInterface = (*TeamsHandler)(nil)

func (t *TeamsHandler) CreateTeam(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	teamsRequest := types.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if teamsRequest.Name == "" {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	createdTeam, err := t.service.CreateTeam(ctx, gameID, sub, teamsRequest)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, apperror.ErrTeamSizeNotAllowed):
			JSONError(writer, "Invalid team size", http.StatusBadRequest)

		default:
			JSONError(writer, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", request.URL.String()+"/"+strconv.FormatInt(int64(createdTeam.ID), 10))
	writer.WriteHeader(http.StatusCreated)

	response := types.TeamResponse{
		Team: entityTeamToAPITeam(createdTeam),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(ctx, "Could not write body", "error", err)
	}
}

func (t *TeamsHandler) UpdateTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	teamsRequest := types.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if teamsRequest.Name == "" {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	updatedGame, err := t.service.UpdateTeam(ctx, gameID, sub, teamID, teamsRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrTeamNotFound):
			JSONError(writer, "Team not found", http.StatusNotFound)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := types.TeamResponse{
		Team: entityTeamToAPITeam(updatedGame),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(ctx, "Could not write body", "error", err)
	}
}

func (t *TeamsHandler) DeleteTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	err := t.service.DeleteTeam(ctx, gameID, sub, teamID)
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

	writer.WriteHeader(http.StatusNoContent)
}
