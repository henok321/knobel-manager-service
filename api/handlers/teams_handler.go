package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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

// Verify that TeamsHandler implements the generated OpenAPI interface
var _ teams.ServerInterface = (*TeamsHandler)(nil)

func (t *TeamsHandler) HandleValidationError(w http.ResponseWriter, _ *http.Request, err error) {
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "Invalid format for parameter gameID"):
		JSONError(w, "Invalid gameID", http.StatusBadRequest)
	case strings.Contains(errorMsg, "Invalid format for parameter teamID"):
		JSONError(w, "Invalid teamID", http.StatusBadRequest)
	default:
		JSONError(w, errorMsg, http.StatusBadRequest)
	}
}

func (t *TeamsHandler) CreateTeam(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
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

	createdTeam, err := t.service.CreateTeam(gameID, sub, teamsRequest)
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

	writer.WriteHeader(http.StatusCreated)
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", request.URL.String()+"/"+strconv.FormatInt(int64(createdTeam.ID), 10))

	response := teams.TeamResponse{
		Team: entityTeamToTeamsAPITeam(createdTeam),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(request.Context(), "Could not write body", "error", err)
	}
}

func (t *TeamsHandler) UpdateTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
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

	updatedGame, err := t.service.UpdateTeam(gameID, sub, teamID, teamsRequest)
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

	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "application/json")

	response := teams.TeamResponse{
		Team: entityTeamToTeamsAPITeam(updatedGame),
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(request.Context(), "Could not write body", "error", err)
	}
}

func (t *TeamsHandler) DeleteTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	err := t.service.DeleteTeam(gameID, sub, teamID)
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
