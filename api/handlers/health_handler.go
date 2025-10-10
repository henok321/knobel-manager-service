package handlers

import (
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/health"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Verify that HealthHandler implements the generated OpenAPI interface
var _ health.ServerInterface = (*HealthHandler)(nil)

func (h *HealthHandler) HealthCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle health request")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}
