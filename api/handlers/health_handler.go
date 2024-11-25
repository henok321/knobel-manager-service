package handlers

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func HealthCheck(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"status": "ok"}`))

	if err != nil {
		log.Error("Failed to write response", "err", err)
	}
}
