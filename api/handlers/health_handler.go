package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/gen/health"
)

type HealthHandler struct {
	healthService *healthpkg.Service
}

func NewHealthHandler(healthService *healthpkg.Service) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

var _ health.ServerInterface = (*HealthHandler)(nil)

func (h *HealthHandler) LivenessCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle liveness check")

	response := health.HealthCheckResponse{Status: "pass"}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}

func (h *HealthHandler) ReadinessCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle readiness check")

	response := h.healthService.Readiness(request.Context())

	statusCode := http.StatusOK
	if response.Status != health.HealthCheckDetailedResponseStatusPass {
		statusCode = http.StatusServiceUnavailable
		slog.WarnContext(request.Context(), "Readiness check failed",
			"status", response.Status,
			"checks", response.Checks)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}
