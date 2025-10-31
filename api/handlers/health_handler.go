package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/health"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

var _ health.ServerInterface = (*HealthHandler)(nil)

func (h *HealthHandler) HealthCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle health request")
	response := health.HealthCheckResponse{
		Status: "ok",
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}
