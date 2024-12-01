package handlers

import (
	"log/slog"
	"net/http"
)

func HealthCheck(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"status": "ok"}`))

	if err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}
