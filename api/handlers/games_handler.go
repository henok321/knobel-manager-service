package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/games"

	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"

	"github.com/henok321/knobel-manager-service/pkg/game"
)

type gameResponse struct {
	Game entity.Game `json:"game"`
}

type gamesResponse struct {
	ActiveGameID int           `json:"activeGameID,omitempty"`
	Games        []entity.Game `json:"games"`
}

type GamesHandler struct {
	gamesService game.GamesService
}

func NewGamesHandler(gamesService game.GamesService) *GamesHandler {
	return &GamesHandler{gamesService}
}

// Verify that GamesHandler implements the generated OpenAPI interface
var _ games.ServerInterface = (*GamesHandler)(nil)

func (h *GamesHandler) HandleValidationError(w http.ResponseWriter, _ *http.Request, err error) {
	if strings.Contains(err.Error(), "Invalid format for parameter gameID") {
		JSONError(w, "Invalid gameID", http.StatusBadRequest)
		return
	}
	JSONError(w, err.Error(), http.StatusBadRequest)
}

func (h *GamesHandler) GetGames(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	games, err := h.gamesService.FindAllByOwner(sub)
	if err != nil {
		http.Error(writer, "{'error': '"+err.Error()+"'}", http.StatusInternalServerError)
		return
	}

	var response gamesResponse

	activeGame, err := h.gamesService.GetActiveGame(sub)

	if err != nil {
		if errors.Is(err, entity.ErrGameNotFound) {
			slog.WarnContext(request.Context(), "Could not find active game", "error", err)

			response = gamesResponse{
				Games: games,
			}
		} else {
			slog.ErrorContext(request.Context(), "Unknown error error while querying active game", "error", err)
			JSONError(writer, "Internal server error", http.StatusInternalServerError)

			return
		}
	} else {
		response = gamesResponse{
			Games:        games,
			ActiveGameID: activeGame.ID,
		}
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) GetGamesGameID(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		http.Error(writer, `{'error': 'User logging not found'}`, http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameByID, err := h.gamesService.FindByID(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	gameResponse := gameResponse{
		Game: gameByID,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) PostGames(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := game.CreateOrUpdateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameCreateRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()

	err := validate.Struct(gameCreateRequest)
	if err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	createdGame, err := h.gamesService.CreateGame(sub, &gameCreateRequest)
	if err != nil {
		http.Error(writer, "{'error': '"+err.Error()+"'}", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", "/games/"+strconv.Itoa(createdGame.ID))
	writer.WriteHeader(http.StatusCreated)

	gameResponse := gameResponse{
		Game: createdGame,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) PutGamesGameID(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameUpdateRequest := game.CreateOrUpdateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	validate := validator.New()

	err := validate.Struct(gameUpdateRequest)
	if err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(gameID, sub, gameUpdateRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Not owner", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	responseBody := gameResponse{Game: updatedGame}

	if err := json.NewEncoder(writer).Encode(responseBody); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *GamesHandler) DeleteGamesGameID(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	if err := h.gamesService.DeleteGame(gameID, sub); err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *GamesHandler) PostGamesGameIDSetup(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameToAssign, err := h.gamesService.FindByID(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	err = h.gamesService.AssignTables(gameToAssign)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Location", "/games/"+strconv.Itoa(gameID)+"/tables")
	writer.WriteHeader(http.StatusCreated)
}

func (h *GamesHandler) PostGamesGameIDActivate(writer http.ResponseWriter, request *http.Request, gameID int) {
	userContext, ok := middleware.UserFromContext(request.Context())
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	err := h.gamesService.SetActiveGame(gameID, sub)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
