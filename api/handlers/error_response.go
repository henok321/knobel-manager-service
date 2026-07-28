package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
)

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

func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrNotOwner):
		JSONError(w, "Forbidden", http.StatusForbidden)
	case errors.Is(err, apperror.ErrGameNotFound):
		JSONError(w, "Game not found", http.StatusNotFound)
	case errors.Is(err, apperror.ErrTeamNotFound):
		JSONError(w, "Team not found", http.StatusNotFound)
	case errors.Is(err, apperror.ErrPlayerNotFound):
		JSONError(w, "Player not found", http.StatusNotFound)
	case errors.Is(err, apperror.ErrRoundOrTableNotFound):
		JSONError(w, "Round or table not found", http.StatusNotFound)
	case errors.Is(err, apperror.ErrInvalidScore):
		JSONError(w, "Invalid score", http.StatusBadRequest)
	case errors.Is(err, apperror.ErrTeamSizeNotAllowed):
		JSONError(w, "Invalid team size", http.StatusBadRequest)
	case errors.Is(err, apperror.ErrInvalidGameSetup):
		JSONError(w, "Invalid game setup", http.StatusConflict)
	case errors.Is(err, apperror.ErrGameIncomplete):
		JSONError(w, "Game is complete", http.StatusConflict)
	case errors.Is(err, apperror.ErrAlreadyOwner):
		JSONError(w, "Already an owner", http.StatusConflict)
	case errors.Is(err, apperror.ErrLastOwner):
		JSONError(w, "Cannot remove the last owner", http.StatusConflict)
	case errors.Is(err, apperror.ErrUserNotFound):
		JSONError(w, "No user found for the given email", http.StatusUnprocessableEntity)
	default:
		JSONError(w, "Internal server error", http.StatusInternalServerError)
	}
}
