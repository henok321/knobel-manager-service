package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

type OpenAPIHandler interface {
	GetOpenAPIConfig(writer http.ResponseWriter, request *http.Request)
	GetSwaggerDocs(writer http.ResponseWriter, request *http.Request)
}

type openAPIHandler struct{}

func NewOpenAPIHandler() OpenAPIHandler {
	return &openAPIHandler{}
}

func (h *openAPIHandler) GetOpenAPIConfig(writer http.ResponseWriter, _ *http.Request) {
	openapiYAML, err := os.ReadFile(filepath.Join("openapi", "openapi.yaml"))
	if err != nil {
		slog.Error("Could not read openapi.yaml", "error", err)
	}

	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	writer.Header().Set("Content-Disposition", "inline")
	_, err = writer.Write(openapiYAML)
	if err != nil {
		slog.Error("Could not write openapi.yaml", "error", err)
	}
}

func (h *openAPIHandler) GetSwaggerDocs(writer http.ResponseWriter, _ *http.Request) {
	swaggerHTML, err := os.ReadFile(filepath.Join("openapi", "swagger.html"))
	if err != nil {
		slog.Error("Could not read swagger.html", "error", err)
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header()
	_, err = writer.Write(swaggerHTML)
	if err != nil {
		slog.Error("Could not write swagger.html", "error", err)
	}
}
