package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/audit"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type AuditHandler struct {
	gamesService *game.GamesService
	auditService *audit.EventsService
}

func NewAuditHandler(gamesService *game.GamesService, auditService *audit.EventsService) *AuditHandler {
	return &AuditHandler{gamesService: gamesService, auditService: auditService}
}

func (a *AuditHandler) GetAuditLog(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	if _, err := a.gamesService.FindByID(ctx, gameID, sub); err != nil {
		respondError(writer, err)
		return
	}

	events, err := a.auditService.FindByGameID(ctx, gameID)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiEvents := make([]api.AuditEvent, len(events))
	for i, event := range events {
		apiEvents[i] = entityAuditEventToAPIAuditEvent(event)
	}

	response := api.AuditResponse{Events: apiEvents}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(ctx, "Could not write body", "error", err)
	}
}
