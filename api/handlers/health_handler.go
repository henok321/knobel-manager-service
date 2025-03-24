package handlers

import (
	"log/slog"
	"net/http"
)

func HealthCheck(writer http.ResponseWriter, request *http.Request) {
	slog.DebugContext(request.Context(), "Handle health request")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		slog.ErrorContext(request.Context(), "Failed to write response", "error", err)
	}
}
