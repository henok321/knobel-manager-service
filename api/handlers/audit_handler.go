package handlers

import (
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
		respondError(ctx, writer, err)
		return
	}

	apiEvents := make([]api.AuditEvent, len(events))
	for i, event := range events {
		apiEvents[i] = entityAuditEventToAPIAuditEvent(event)
	}

	writeJSON(ctx, writer, http.StatusOK, api.AuditResponse{Events: apiEvents})
}
