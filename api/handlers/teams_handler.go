package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type TeamsHandler struct {
	service *team.TeamsService
}

func NewTeamsHandler(service *team.TeamsService) *TeamsHandler {
	return &TeamsHandler{service}
}

func (t *TeamsHandler) CreateTeam(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	teamsRequest := api.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(ctx, writer, http.StatusBadRequest, err.Error())
		return
	}

	if teamsRequest.Name == "" {
		JSONError(ctx, writer, http.StatusBadRequest, "Missing required fields")
		return
	}

	createdTeam, err := t.service.CreateTeam(ctx, gameID, sub, teamsRequest)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	writer.Header().Set("Location", request.URL.String()+"/"+strconv.FormatInt(int64(createdTeam.ID), 10))
	writeJSON(ctx, writer, http.StatusCreated, api.TeamResponse{Team: entityTeamToAPITeam(createdTeam)})
}

func (t *TeamsHandler) UpdateTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	teamsRequest := api.TeamsRequest{}

	if err := json.NewDecoder(request.Body).Decode(&teamsRequest); err != nil {
		JSONError(ctx, writer, http.StatusBadRequest, err.Error())
		return
	}

	if teamsRequest.Name == "" {
		JSONError(ctx, writer, http.StatusBadRequest, "Missing required fields")
		return
	}

	updatedGame, err := t.service.UpdateTeam(ctx, gameID, sub, teamID, teamsRequest)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	writeJSON(ctx, writer, http.StatusOK, api.TeamResponse{Team: entityTeamToAPITeam(updatedGame)})
}

func (t *TeamsHandler) DeleteTeam(writer http.ResponseWriter, request *http.Request, gameID, teamID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := t.service.DeleteTeam(ctx, gameID, sub, teamID); err != nil {
		respondError(ctx, writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
