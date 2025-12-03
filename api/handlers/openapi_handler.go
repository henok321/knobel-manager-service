package handlers

import (
	"log/slog"
	"net/http"
)

type OpenAPIHandler interface {
	GetOpenAPIConfig(writer http.ResponseWriter, request *http.Request)
	GetSwaggerDocs(writer http.ResponseWriter, request *http.Request)
}

type openAPIHandler struct {
	openAPIConfig []byte
	swaggerDocs   []byte
}

func NewOpenAPIHandler(openAPIConfig, swaggerDocs []byte) OpenAPIHandler {
	return &openAPIHandler{openAPIConfig: openAPIConfig, swaggerDocs: swaggerDocs}
}

func (h *openAPIHandler) GetOpenAPIConfig(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	writer.Header().Set("Content-Disposition", "inline")
	_, err := writer.Write(h.openAPIConfig)
	if err != nil {
		slog.Error("Could not write openapi.yaml", "error", err)
	}
}

func (h *openAPIHandler) GetSwaggerDocs(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header()
	_, err := writer.Write(h.swaggerDocs)
	if err != nil {
		slog.Error("Could not write swagger.html", "error", err)
	}
}
