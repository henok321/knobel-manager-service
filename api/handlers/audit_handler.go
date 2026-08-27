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
		changes := make([]api.AuditChange, 0)

		// changes is written only by pkg/audit and never edited, so unreadable JSON
		// means a corrupted row. Degrade that one event to an empty diff rather than
		// failing the whole log.
		if err := json.Unmarshal([]byte(event.Changes), &changes); err != nil {
			slog.ErrorContext(ctx, "Could not decode audit changes", "auditEventID", event.ID, "error", err)

			changes = make([]api.AuditChange, 0)
		}

		apiEvents[i] = entityAuditEventToAPIAuditEvent(event, changes)
	}

	response := api.AuditResponse{Events: apiEvents}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.InfoContext(ctx, "Could not write body", "error", err)
	}
}
