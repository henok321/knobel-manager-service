package handlers

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func HealthCheck(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"status": "ok"}`))

	if err != nil {
		log.Error("Failed to write response", "err", err)
	}
}
