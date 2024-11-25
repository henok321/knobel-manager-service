package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/api/middleware"
	log "github.com/sirupsen/logrus"

	"github.com/henok321/knobel-manager-service/pkg/entity"

	"github.com/henok321/knobel-manager-service/pkg/game"
)

type gameResponse struct {
	Game entity.Game
}

type gamesResponse struct {
	Games []entity.Game
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
	userContext, err := middleware.GetUserFromCtx(request.Context())

	if err != nil {
		if errors.Is(err, middleware.ErrUserContextNotFound) {
			http.Error(writer, "User context not found", http.StatusUnauthorized)
			return
		}
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	games, err := h.gamesService.FindAllByOwner(sub)

	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	gamesResponse := gamesResponse{
		Games: games,
	}

	if err := json.NewEncoder(writer).Encode(gamesResponse); err != nil {
		log.Error("Could not write body", "err", err)
	}
}

func (h *gamesHandler) GetGameByID(writer http.ResponseWriter, request *http.Request) {
	userContext, err := middleware.GetUserFromCtx(request.Context())

	if err != nil {
		if errors.Is(err, middleware.ErrUserContextNotFound) {
			http.Error(writer, "User context not found", http.StatusUnauthorized)
			return
		}
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		http.Error(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	gameById, err := h.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			http.Error(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			http.Error(writer, "Game not found", http.StatusNotFound)
		default:

			http.Error(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// c.JSON(http.StatusOK, gin.H{"game": gameById})
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	gameResponse := gameResponse{
		Game: gameById,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		log.Error("Could not write body", "err", err)
	}
}

func (h *gamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {

	userContext, err := middleware.GetUserFromCtx(request.Context())

	if err != nil {
		if errors.Is(err, middleware.ErrUserContextNotFound) {
			http.Error(writer, "User context not found", http.StatusUnauthorized)
			return
		}
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := game.GameRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameCreateRequest); err != nil {
		http.Error(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdGame, err := h.gamesService.CreateGame(sub, &gameCreateRequest)

	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", "/games/"+strconv.Itoa(int(createdGame.ID)))
	writer.WriteHeader(http.StatusCreated)

	gameResponse := gameResponse{
		Game: createdGame,
	}

	if err := json.NewEncoder(writer).Encode(gameResponse); err != nil {
		log.Error("Could not write body", "err", err)
	}
}

func (h *gamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request) {
	userContext, err := middleware.GetUserFromCtx(request.Context())

	if err != nil {
		if errors.Is(err, middleware.ErrUserContextNotFound) {
			http.Error(writer, "User context not found", http.StatusUnauthorized)
			return
		}
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		// c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		http.Error(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	gameUpdateRequest := game.GameRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		http.Error(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(uint(gameID), sub, gameUpdateRequest)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			http.Error(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			http.Error(writer, "Game not found", http.StatusNotFound)
		default:
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	responseBody := gameResponse{Game: updatedGame}

	if err := json.NewEncoder(writer).Encode(responseBody); err != nil {
		log.Error("Could not write body", "err", err)
	}
}

func (h *gamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request) {
	userContext, err := middleware.GetUserFromCtx(request.Context())

	if err != nil {
		if errors.Is(err, middleware.ErrUserContextNotFound) {
			http.Error(writer, "User context not found", http.StatusUnauthorized)
			return
		}
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)
	if err != nil {
		http.Error(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	if err := h.gamesService.DeleteGame(uint(gameID), sub); err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			http.Error(writer, "Forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrorGameNotFound):
			//	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			http.Error(writer, "Game not found", http.StatusNotFound)
		default:

			http.Error(writer, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *gamesHandler) GameSetup(writer http.ResponseWriter, request *http.Request) {
	gameID, err := strconv.ParseUint(request.PathValue("gameID"), 10, 64)

	if err != nil {
		http.Error(writer, "Invalid gameID", http.StatusBadRequest)
		return
	}

	err = h.gamesService.AssignTables(uint(gameID))

	if err != nil {
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Location", "/games/"+strconv.Itoa(int(gameID))+"/tables")
	writer.WriteHeader(http.StatusCreated)
}
