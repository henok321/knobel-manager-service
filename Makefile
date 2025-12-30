
.DEFAULT_GOAL := all

GOARCH ?= $(shell uname -m)
GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
OUTPUT := knobel-manager-service
BUILD_FLAGS := -a -ldflags="-s -w -extldflags '-static'"
CMD_DIR := ./cmd

.PHONY: all help check-deps setup reset openapi lint lint-all update test build clean lint-go

all: help

help:
	@echo "Usage: make [target]"
	@echo "Targets: help, setup, reset, openapi, lint, update, test, build, clean"

check-deps:
	@echo "Checking dependencies..."
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed."; exit 1; }
	@command -v pre-commit >/dev/null 2>&1 || { echo >&2 "pre-commit is not installed."; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo >&2 "Docker is not installed."; exit 1; }
	@echo "Dependencies fulfilled!"

setup: check-deps
	@echo "Setting up commit hooks and local database..."
	./scripts/setup.sh

reset:
	@echo "Uninstall pre-commit hooks..."
	pre-commit uninstall
	@echo "Cleanup pre-commit cache..."
	pre-commit clean
	@echo "Cleanup local docker database..."
	docker compose down --volumes --remove-orphans

openapi:
	@echo "Cleanup generated files..."
	@command rm -rf ./gen
	@echo "Generate openapi code from spec..."
	@echo "Generating Health handler..."
	@cd openapi/config && go tool oapi-codegen --config=health.yaml ../openapi.yaml
	@echo "Generating Games handler..."
	@cd openapi/config && go tool oapi-codegen --config=games.yaml ../openapi.yaml
	@echo "Generating Teams handler..."
	@cd openapi/config && go tool oapi-codegen --config=teams.yaml ../openapi.yaml
	@echo "Generating Players handler..."
	@cd openapi/config && go tool oapi-codegen --config=players.yaml ../openapi.yaml
	@echo "Generating Tables handler..."
	@cd openapi/config && go tool oapi-codegen --config=tables.yaml ../openapi.yaml
	@echo "Generating Scores handler..."
	@cd openapi/config && go tool oapi-codegen --config=scores.yaml ../openapi.yaml
	@go mod tidy

lint:
	@echo "Running Go linter..."
	go fmt ./...
	go tool golangci-lint run --fix ./...

lint-all:
	@echo "Running linter..."
	go fmt ./...
	pre-commit run --all-files

update:
	@echo "Updating Go modules..."
	go get -u ./...
	go mod tidy

test:
	@echo "Running tests..."
	go test -v ./...

build: openapi
	@echo "Building the service..."
	CGO_ENABLED=0 GOARCH=$(GOARCH) GOOS=$(GOOS) go build $(BUILD_FLAGS) -o $(OUTPUT) $(CMD_DIR)/

clean:
	@echo "Cleaning build artifacts..."
	go clean
	@rm -f $(OUTPUT)
