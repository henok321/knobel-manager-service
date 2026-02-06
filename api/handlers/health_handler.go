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

	results := h.healthService.Liveness()

	response := health.HealthCheckResponse{
		Status: string(results.Status),
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}

func (h *HealthHandler) ReadinessCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle readiness check")

	results := h.healthService.Readiness(request.Context())

	checks := make(map[string]struct {
		Message *string                                        `json:"message,omitempty"`
		Status  health.HealthCheckDetailedResponseChecksStatus `json:"status"`
	})
	for name, result := range results.Checks {
		checkValue := struct {
			Message *string                                        `json:"message,omitempty"`
			Status  health.HealthCheckDetailedResponseChecksStatus `json:"status"`
		}{
			Status: health.HealthCheckDetailedResponseChecksStatus(result.Status),
		}
		if result.Message != "" {
			checkValue.Message = &result.Message
		}
		checks[name] = checkValue
	}

	response := health.HealthCheckDetailedResponse{
		Status: health.HealthCheckDetailedResponseStatus(results.Status),
	}
	if len(checks) > 0 {
		response.Checks = &checks
	}

	statusCode := http.StatusOK
	if results.Status != healthpkg.StatusPass {
		statusCode = http.StatusServiceUnavailable
		slog.WarnContext(request.Context(), "Readiness check failed",
			"status", results.Status,
			"checks", checks)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}
