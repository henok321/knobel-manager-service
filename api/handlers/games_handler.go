package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

	"github.com/henok321/knobel-manager-service/api/middleware"

	"github.com/henok321/knobel-manager-service/pkg/entity"

	"github.com/henok321/knobel-manager-service/pkg/game"
)

type gameResponse struct {
	Game entity.Game `json:"game"`
}

type gamesResponse struct {
	Games []entity.Game `json:"games"`
}

type GamesHandler interface {
	GetGames(writer http.ResponseWriter, request *http.Request)
	GetGameByID(writer http.ResponseWriter, request *http.Request)
	CreateGame(writer http.ResponseWriter, request *http.Request)
	UpdateGame(writer http.ResponseWriter, request *http.Request)
	DeleteGame(writer http.ResponseWriter, request *http.Request)
	GameSetup(writer http.ResponseWriter, request *http.Request)
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
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	games, err := h.gamesService.FindAllByOwner(sub)

	if err != nil {
		http.Error(writer, "{'error': '"+err.Error()+"'}", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	gamesResponse := gamesResponse{
		Games: games,
	}

	if err := json.NewEncoder(writer).Encode(gamesResponse); err != nil {
		slog.Error("Could not write body", "error", err)
	}
}

func (h *gamesHandler) GetGameByID(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		http.Error(writer, `{'error': 'User context not found'}`, http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	gameById, err := h.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
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
		slog.Error("Could not write body", "error", err)
	}
}

func (h *gamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {

	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User context not found", http.StatusUnauthorized)
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
		slog.Error("Could not write body", "error", err)
	}
}

func (h *gamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

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

	updatedGame, err := h.gamesService.UpdateGame(uint(gameID), sub, gameUpdateRequest)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
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
		slog.Error("Could not write body", "error", err)
	}
}

func (h *gamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request) {
	userContext, ok := request.Context().Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)
	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	if err := h.gamesService.DeleteGame(uint(gameID), sub); err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
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
		JSONError(writer, "User context not found", http.StatusUnauthorized)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		JSONError(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	gameToAssign, err := h.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
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
