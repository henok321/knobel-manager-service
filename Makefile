
.DEFAULT_GOAL := all

GOARCH ?= $(shell uname -m)
GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
OUTPUT := knobel-manager-service
BUILD_FLAGS := -a -ldflags="-s -w -extldflags '-static'"
CMD_DIR := ./cmd

.PHONY: all help setup reset openapi-generate openapi-validate lint lint-all update test build clean

all: help

help:
	@echo "Usage: make [target]"
	@grep -hE '^[a-z][a-z-]*:' $(MAKEFILE_LIST) | cut -d: -f1 | sort | paste -sd' ' -

setup:
	@echo "Setting up commit hooks and local database..."
	./scripts/setup.sh

reset:
	@echo "Uninstall pre-commit hooks..."
	pre-commit uninstall
	@echo "Cleanup pre-commit cache..."
	pre-commit clean
	@echo "Cleanup local docker database..."
	docker compose down --volumes --remove-orphans

openapi-generate:
	@echo "Cleanup generated files..."
	@command rm -rf ./gen
	@echo "Generate openapi code from spec..."
	@echo "Generating Health handler..."
	@cd openapi/config && go tool oapi-codegen --config=health.yaml ../openapi.yaml
	@echo "Generating API handlers..."
	@cd openapi/config && go tool oapi-codegen --config=api.yaml ../openapi.yaml
	@go mod tidy
	@echo "✓ Generated code updated. Review changes with 'git diff gen/' and commit if needed."

openapi-validate:
	@echo "Validating OpenAPI generated code..."
	@./scripts/validate-openapi.sh

# Run twice: the hook's entry is `golangci-lint run --fix`, and pre-commit fails a hook that
# modified files even when the tool exited 0. The second run reports only real issues.
lint:
	@echo "Running Go linter..."
	pre-commit run golangci-lint-full --all-files || pre-commit run golangci-lint-full --all-files

lint-all:
	@echo "Running linter..."
	pre-commit run --all-files

update:
	@echo "Updating Go modules..."
	go get -u ./...
	go mod tidy

test:
	@echo "Running tests..."
	go test -v ./...

build:
	@echo "Building the service..."
	CGO_ENABLED=0 GOARCH=$(GOARCH) GOOS=$(GOOS) go build $(BUILD_FLAGS) -o $(OUTPUT) $(CMD_DIR)/

clean:
	@echo "Cleaning build artifacts..."
	go clean
	@rm -f $(OUTPUT)
