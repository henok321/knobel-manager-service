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

	"github.com/henok321/knobel-manager-service/pkg/game"
)

type gameResponse struct {
	Game entity.Game `json:"game"`
}

type gamesResponse struct {
	ActiveGameID int           `json:"activeGameID,omitempty"`
	Games        []entity.Game `json:"games"`
}

type GamesHandler interface {
	GetGames(writer http.ResponseWriter, request *http.Request)
	GetGameByID(writer http.ResponseWriter, request *http.Request)
	CreateGame(writer http.ResponseWriter, request *http.Request)
	UpdateGame(writer http.ResponseWriter, request *http.Request)
	DeleteGame(writer http.ResponseWriter, request *http.Request)
	GameSetup(writer http.ResponseWriter, request *http.Request)
	SetActiveGame(writer http.ResponseWriter, request *http.Request)
}

type gamesHandler struct {
	gamesService game.GamesService
}

func NewGamesHandler(gamesService game.GamesService) GamesHandler {
	return &gamesHandler{gamesService}
}

func (h *gamesHandler) GetGames(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
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
		if errors.Is(err, entity.ErrorGameNotFound) {
			slog.WarnContext(request.Context(), "Could not find active game", "error", err)

			response = gamesResponse{
				Games: games}

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

func (h *gamesHandler) GetGameByID(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		http.Error(writer, `{'error': 'User logging not found'}`, http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseInt(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	gameById, err := h.gamesService.FindByID(int(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	gameResponse := gameResponse{
		Game: gameById,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *gamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {

	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User logging not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := game.GameRequest{}

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
	writer.Header().Set("Location", "/games/"+strconv.Itoa(int(createdGame.ID)))
	writer.WriteHeader(http.StatusCreated)

	gameResponse := gameResponse{
		Game: createdGame,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		slog.ErrorContext(request.Context(), "Could not write body", "error", err)
	}
}

func (h *gamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request) {
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

	gameUpdateRequest := game.GameRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	validate := validator.New()

	err = validate.Struct(gameUpdateRequest)

	if err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(int(gameID), sub, gameUpdateRequest)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Not owner", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
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

func (h *gamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request) {
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

	if err := h.gamesService.DeleteGame(int(gameID), sub); err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		default:

			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *gamesHandler) GameSetup(writer http.ResponseWriter, request *http.Request) {
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

	gameToAssign, err := h.gamesService.FindByID(int(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
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

	writer.Header().Set("Location", "/games/"+strconv.Itoa(int(gameID))+"/tables")
	writer.WriteHeader(http.StatusCreated)
}

func (h *gamesHandler) SetActiveGame(writer http.ResponseWriter, request *http.Request) {
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

	err = h.gamesService.SetActiveGame(int(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, customError.NotOwner):
			JSONError(writer, "Forbidden", http.StatusForbidden)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
