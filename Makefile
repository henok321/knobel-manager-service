.PHONY: all build check-deps clean help lint openapi reset setup test update

.DEFAULT_GOAL := all

GOARCH := $(shell uname -m)
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
OUTPUT := knobel-manager-service
BUILD_FLAGS := -a -ldflags="-s -w -extldflags '-static'"
CMD_DIR := ./cmd

all: help

help:
	@echo "Usage: make [target]"
	@echo "Targets: help, setup, reset, openapi, lint, update, test, build, clean"

check-deps:
	@echo "Checking dependencies..."
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed."; exit 1; }
	@command -v pre-commit >/dev/null 2>&1 || { echo >&2 "pre-commit is not installed."; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo >&2 "Docker is not installed."; exit 1; }

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
	@echo "Generate openapi code from spec..."
	@echo "Generating shared types..."
	@cd spec/config && go tool oapi-codegen --config=types.yaml ../api.yaml
	@echo "Generating Health handler..."
	@cd spec/config && go tool oapi-codegen --config=health.yaml ../api.yaml
	@echo "Generating Games handler..."
	@cd spec/config && go tool oapi-codegen --config=games.yaml ../api.yaml
	@echo "Generating Teams handler..."
	@cd spec/config && go tool oapi-codegen --config=teams.yaml ../api.yaml
	@echo "Generating Players handler..."
	@cd spec/config && go tool oapi-codegen --config=players.yaml ../api.yaml
	@echo "Generating Tables handler..."
	@cd spec/config && go tool oapi-codegen --config=tables.yaml ../api.yaml
	@echo "Generating Scores handler..."
	@cd spec/config && go tool oapi-codegen --config=scores.yaml ../api.yaml
	@go mod tidy

lint: openapi
	@echo "Running linter..."
	pre-commit run --all-files

update:
	@echo "Updating Linter and Go modules..."
	pre-commit autoupdate
	pre-commit migrate-config
	go get -u ./...
	go mod tidy

test: lint
	@echo "Running tests..."
	go test -v ./...

build: test
	@echo "Building the service..."
	CGO_ENABLED=0 GOARCH=$(GOARCH) GOOS=$(GOOS) go build $(BUILD_FLAGS) -o $(OUTPUT) $(CMD_DIR)/

clean:
	@echo "Cleaning build artifacts..."
	go clean
	@rm -f $(OUTPUT)
