package handlers

import (
	"log/slog"
	"net/http"
)

type OpenAPIHandler struct {
	openAPIConfig []byte
	swaggerDocs   []byte
}

func NewOpenAPIHandler(openAPIConfig, swaggerDocs []byte) *OpenAPIHandler {
	return &OpenAPIHandler{openAPIConfig: openAPIConfig, swaggerDocs: swaggerDocs}
}

func (h *OpenAPIHandler) GetOpenAPIConfig(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	writer.Header().Set("Content-Disposition", "inline")
	_, err := writer.Write(h.openAPIConfig)
	if err != nil {
		slog.Error("Could not write openapi.yaml", "error", err)
	}
}

func (h *OpenAPIHandler) GetSwaggerDocs(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := writer.Write(h.swaggerDocs)
	if err != nil {
		slog.Error("Could not write swagger.html", "error", err)
	}
}
