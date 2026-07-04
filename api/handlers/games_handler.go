package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"firebase.google.com/go/v4/auth"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type GamesHandler struct {
	gamesService *game.GamesService
	users        middleware.FirebaseAuth
}

func NewGamesHandler(gamesService *game.GamesService, users middleware.FirebaseAuth) *GamesHandler {
	return &GamesHandler{gamesService, users}
}

func (h *GamesHandler) enrichOwnerEmails(ctx context.Context, games ...*api.Game) {
	seen := map[string]struct{}{}

	var ids []auth.UserIdentifier

	for _, g := range games {
		for _, owner := range g.Owners {
			if _, ok := seen[owner.OwnerSub]; ok {
				continue
			}

			seen[owner.OwnerSub] = struct{}{}
			ids = append(ids, auth.UIDIdentifier{UID: owner.OwnerSub})
		}
	}

	if len(ids) == 0 {
		return
	}

	result, err := h.users.GetUsers(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "owner email enrichment failed", "error", err)
		return
	}

	emailByUID := make(map[string]string, len(result.Users))
	for _, user := range result.Users {
		emailByUID[user.UID] = user.Email
	}

	for _, g := range games {
		for i := range g.Owners {
			if email, ok := emailByUID[g.Owners[i].OwnerSub]; ok && email != "" {
				g.Owners[i].Email = &email
			}
		}
	}
}

func (h *GamesHandler) GetGames(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	allGames, err := h.gamesService.FindAllByOwner(ctx, sub)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	apiGames := make([]api.Game, len(allGames))
	ptrs := make([]*api.Game, len(allGames))

	for i, entry := range allGames {
		apiGames[i] = entityGameToAPIGame(entry)
		ptrs[i] = &apiGames[i]
	}

	h.enrichOwnerEmails(ctx, ptrs...)

	response := api.GamesResponse{
		Games: apiGames,
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) GetGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameByID, err := h.gamesService.FindByID(ctx, gameID, sub)
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

	apiGame := entityGameToAPIGame(gameByID)
	h.enrichOwnerEmails(ctx, &apiGame)
	response := api.GameResponse{Game: apiGame}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameCreateRequest := api.GameCreateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameCreateRequest); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if gameCreateRequest.Name == "" || gameCreateRequest.NumberOfRounds == 0 ||
		gameCreateRequest.TeamSize == 0 || gameCreateRequest.TableSize == 0 {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	createdGame, err := h.gamesService.CreateGame(ctx, sub, &gameCreateRequest)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Location", fmt.Sprintf("/games/%d", createdGame.ID))
	writer.WriteHeader(http.StatusCreated)

	apiGame := entityGameToAPIGame(createdGame)
	h.enrichOwnerEmails(ctx, &apiGame)
	response := api.GameResponse{
		Game: apiGame,
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameUpdateRequest := api.GameUpdateRequest{}

	if err := json.NewDecoder(request.Body).Decode(&gameUpdateRequest); err != nil {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	if gameUpdateRequest.Name == "" || gameUpdateRequest.NumberOfRounds == 0 ||
		gameUpdateRequest.TeamSize == 0 || gameUpdateRequest.TableSize == 0 {
		JSONError(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(ctx, gameID, sub, gameUpdateRequest)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "Not owner", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrInvalidGameSetup):
			JSONError(writer, "Invalid game setup", http.StatusConflict)
		case errors.Is(err, apperror.ErrGameIncomplete):
			JSONError(writer, "Game is complete", http.StatusConflict)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)
	response := api.GameResponse{
		Game: apiGame,
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) AddOwner(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	body := api.AddOwnerRequest{}

	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		JSONError(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Email == "" {
		JSONError(writer, "Missing required fields", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.AddOwner(ctx, gameID, userContext.Sub, body.Email)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrAlreadyOwner):
			JSONError(writer, "Already an owner", http.StatusConflict)
		case errors.Is(err, apperror.ErrUserNotFound):
			JSONError(writer, "No user found for the given email", http.StatusUnprocessableEntity)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(api.GameResponse{Game: apiGame}); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) RemoveOwner(writer http.ResponseWriter, request *http.Request, gameID int, ownerSub string) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	updatedGame, err := h.gamesService.RemoveOwner(ctx, gameID, userContext.Sub, ownerSub)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrNotOwner):
			JSONError(writer, "forbidden", http.StatusForbidden)
		case errors.Is(err, entity.ErrGameNotFound):
			JSONError(writer, "Game or owner not found", http.StatusNotFound)
		case errors.Is(err, apperror.ErrLastOwner):
			JSONError(writer, "Cannot remove the last owner", http.StatusConflict)
		default:
			JSONError(writer, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(api.GameResponse{Game: apiGame}); err != nil {
		slog.ErrorContext(ctx, "Could not write body", "error", err)
	}
}

func (h *GamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	if err := h.gamesService.DeleteGame(ctx, gameID, sub); err != nil {
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

func (h *GamesHandler) SetupGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	userContext, ok := middleware.UserFromContext(ctx)
	if !ok {
		JSONError(writer, "User context not found", http.StatusInternalServerError)
		return
	}

	sub := userContext.Sub

	gameToAssign, err := h.gamesService.FindByID(ctx, gameID, sub)
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

	if gameToAssign.Status != entity.StatusSetup {
		JSONError(writer, "Game is not in setup state", http.StatusBadRequest)
		return
	}

	if len(gameToAssign.Teams) < gameToAssign.TableSize {
		JSONError(writer, "Not enough teams to assign tables", http.StatusConflict)
		return
	}

	err = h.gamesService.AssignTables(ctx, gameToAssign)
	if err != nil {
		JSONError(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
