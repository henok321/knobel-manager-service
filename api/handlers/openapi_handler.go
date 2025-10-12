package handlers

import (
	"log"
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
	openapiYAML, err := os.ReadFile(filepath.Join("spec", "openapi.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	writer.Header().Set("Content-Disposition", "inline")
	_, err = writer.Write(openapiYAML)
	if err != nil {
		log.Fatal(err)
	}
}

func (h *openAPIHandler) GetSwaggerDocs(writer http.ResponseWriter, _ *http.Request) {
	swaggerHTML, err := os.ReadFile(filepath.Join("spec", "swagger.html"))
	if err != nil {
		log.Fatal(err)
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header()
	_, err = writer.Write(swaggerHTML)
	if err != nil {
		log.Fatal(err)
	}
}
