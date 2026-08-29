package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/audit"
)

type AuditHandler struct {
	service *audit.EventsService
}

func NewAuditHandler(service *audit.EventsService) *AuditHandler {
	return &AuditHandler{service}
}

func (a *AuditHandler) GetAuditLog(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	events, err := a.service.FindByGameID(ctx, gameID, sub)
	if err != nil {
		respondError(writer, err)
		return
	}

	apiEvents := make([]api.AuditEvent, len(events))
	for i, event := range events {
		apiEvents[i] = entityAuditEventToAPIAuditEvent(
			event,
			decodeAuditRow(ctx, event.ID, event.OldRow),
			decodeAuditRow(ctx, event.ID, event.NewRow),
		)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(api.AuditResponse{Events: apiEvents}); err != nil {
		slog.InfoContext(ctx, "Could not write body", "error", err)
	}
}

// The jsonb columns are written only by the audit trigger, so unreadable JSON means a
// corrupted row rather than a normal case. Drop that one side of the event instead of
// failing the whole log.
func decodeAuditRow(ctx context.Context, eventID int64, raw *string) *map[string]any {
	if raw == nil {
		return nil
	}

	row := map[string]any{}
	if err := json.Unmarshal([]byte(*raw), &row); err != nil {
		slog.ErrorContext(ctx, "Could not decode audit row", "auditEventID", eventID, "error", err)
		return nil
	}

	return &row
}
