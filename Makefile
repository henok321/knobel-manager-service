.DEFAULT_GOAL := help

GO         		:= go
GOARCH 			:= $(shell uname -m)
GOOS 			:= $(shell uname -s | tr '[:upper:]' '[:lower:]')
CGO_ENABLED  	:= 0
OUTPUT       	:= knobel-manager-service
BUILD_FLAGS  	:= -a -ldflags="-s -w -extldflags '-static'"
CMD_DIR      	:= ./cmd
LINTER       	:= golangci-lint

.PHONY: help setup reset lint update test build clean check-deps

help:
	@echo "Usage: make [target]"
	@echo "Available targets:"
	@echo "  help      - Show this help message"
	@echo "  setup     - Install dependencies and setup project"
	@echo "  reset     - Reset local database"
	@echo "  lint      - Run linter"
	@echo "  update    - Update Go modules"
	@echo "  test      - Run tests"
	@echo "  build     - Build the service"
	@echo "  clean     - Remove build artifacts"

check-deps:
	@echo "Checking dependencies..."
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed."; exit 1; }
	@command -v pre-commit >/dev/null 2>&1 || { echo >&2 "pre-commit is not installed."; exit 1; }
	@command -v goose >/dev/null 2>&1 || { echo >&2 "goose is not installed."; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo >&2 "Docker is not installed."; exit 1; }

setup: check-deps
	@echo "Setting up commit hooks and local database..."
	./scripts/setup.sh

reset:
	@echo "Resetting local database..."
	docker compose down --volumes --remove-orphans

lint:
	@echo "Running linter..."
	$(LINTER) config verify --verbose --config .golangci.yml
	$(LINTER) run --fix --verbose

update:
	@echo "Updating Go modules..."
	$(GO) get -u ./...
	$(GO) mod tidy

test: lint
	@echo "Running tests..."
	$(GO) test -v ./...

build: test
	@echo "Building the service..."
	CGO_ENABLED=$(CGO_ENABLED) GOARCH=$(GOARCH) GOOS=$(GOOS) $(GO) build $(BUILD_FLAGS) -o $(OUTPUT) $(CMD_DIR)/

clean:
	@echo "Cleaning build artifacts..."
	@${GO} clean
	@rm -f $(OUTPUT)
