package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"firebase.google.com/go/v4/auth"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/api"
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

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	allGames, err := h.gamesService.FindAllByOwner(ctx, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiGames := make([]api.Game, len(allGames))
	ptrs := make([]*api.Game, len(allGames))

	for i, entry := range allGames {
		apiGames[i] = entityGameToAPIGame(entry)
		ptrs[i] = &apiGames[i]
	}

	h.enrichOwnerEmails(ctx, ptrs...)

	writeJSON(ctx, writer, http.StatusOK, api.GamesResponse{Games: apiGames})
}

func (h *GamesHandler) GetGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	gameByID, err := h.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiGame := entityGameToAPIGame(gameByID)
	h.enrichOwnerEmails(ctx, &apiGame)

	writeJSON(ctx, writer, http.StatusOK, api.GameResponse{Game: apiGame})
}

func (h *GamesHandler) CreateGame(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

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
		respondError(writer, err)
		return
	}

	apiGame := entityGameToAPIGame(createdGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writer.Header().Set("Location", fmt.Sprintf("/games/%d", createdGame.ID))
	writeJSON(ctx, writer, http.StatusCreated, api.GameResponse{Game: apiGame})
}

func (h *GamesHandler) UpdateGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

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

	if gameUpdateRequest.Status != nil && !gameUpdateRequest.Status.Valid() {
		JSONError(writer, "Invalid status", http.StatusBadRequest)
		return
	}

	updatedGame, err := h.gamesService.UpdateGame(ctx, gameID, sub, gameUpdateRequest)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writeJSON(ctx, writer, http.StatusOK, api.GameResponse{Game: apiGame})
}

func (h *GamesHandler) AddOwner(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
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

	updatedGame, err := h.gamesService.AddOwner(ctx, gameID, sub, body.Email)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writeJSON(ctx, writer, http.StatusOK, api.GameResponse{Game: apiGame})
}

func (h *GamesHandler) RemoveOwner(writer http.ResponseWriter, request *http.Request, gameID int, ownerSub string) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	updatedGame, err := h.gamesService.RemoveOwner(ctx, gameID, sub, ownerSub)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiGame := entityGameToAPIGame(updatedGame)
	h.enrichOwnerEmails(ctx, &apiGame)

	writeJSON(ctx, writer, http.StatusOK, api.GameResponse{Game: apiGame})
}

func (h *GamesHandler) DeleteGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := h.gamesService.DeleteGame(ctx, gameID, sub); err != nil {
		respondError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *GamesHandler) SetupGame(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := h.gamesService.AssignTables(ctx, gameID, sub); err != nil {
		respondError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *GamesHandler) ResetGameSetup(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if err := h.gamesService.ResetSetup(ctx, gameID, sub); err != nil {
		respondError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
