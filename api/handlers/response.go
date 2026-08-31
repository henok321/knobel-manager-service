package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
)

func writeJSON(ctx context.Context, w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSONError(w http.ResponseWriter, errorMessage string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(&ErrorResponse{Error: errorMessage}); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}

func userSub(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		JSONError(w, "User context not found", http.StatusInternalServerError)
		return "", false
	}

	return user.Sub, true
}

var errorResponses = []struct {
	err     error
	message string
	status  int
}{
	{apperror.ErrNotOwner, "Forbidden", http.StatusForbidden},
	{apperror.ErrGameNotFound, "Game not found", http.StatusNotFound},
	{apperror.ErrTeamNotFound, "Team not found", http.StatusNotFound},
	{apperror.ErrPlayerNotFound, "Player not found", http.StatusNotFound},
	{apperror.ErrRoundOrTableNotFound, "Round or table not found", http.StatusNotFound},
	{apperror.ErrInvalidScore, "Invalid score", http.StatusBadRequest},
	{apperror.ErrTeamSizeNotAllowed, "Invalid team size", http.StatusBadRequest},
	{apperror.ErrInvalidGameSetup, "Invalid game setup", http.StatusConflict},
	{apperror.ErrGameIncomplete, "Game is incomplete", http.StatusConflict},
	{apperror.ErrAlreadyOwner, "Already an owner", http.StatusConflict},
	{apperror.ErrGameNotEditable, "Game is not editable", http.StatusConflict},
	{apperror.ErrInvalidStatusTransition, "Invalid status transition", http.StatusConflict},
	{apperror.ErrGameNotInProgress, "Game is not in progress", http.StatusConflict},
	{apperror.ErrNotEnoughTeams, "Not enough teams to assign tables", http.StatusConflict},
	{apperror.ErrTableAssignment, "Cannot assign players to tables", http.StatusConflict},
	{apperror.ErrGameAlreadySetUp, "Game setup already assigned, reset the setup first", http.StatusConflict},
	{apperror.ErrLastOwner, "Cannot remove the last owner", http.StatusConflict},
	{apperror.ErrUserNotFound, "No user found for the given email", http.StatusUnprocessableEntity},
}

func respondError(w http.ResponseWriter, err error) {
	for _, response := range errorResponses {
		if errors.Is(err, response.err) {
			JSONError(w, response.message, response.status)
			return
		}
	}

	JSONError(w, "Internal server error", http.StatusInternalServerError)
}
