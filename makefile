.DEFAULT_GOAL := help

GO         		:= go
GOARCH 			:= $(shell uname -m)
GOOS 			:= $(shell uname -s | tr '[:upper:]' '[:lower:]')
CGO_ENABLED  	:= 0
OUTPUT       	:= knobel-manager-service
BUILD_FLAGS  	:= -a -ldflags="-s -w -extldflags '-static'"
CMD_DIR      	:= ./cmd
LINTER       	:= golangci-lint

.PHONY: help setup reset lint update test build run clean

help:
	@echo "Usage: make [target]"
	@echo "Available targets:"
	@echo "  help   - Show this help message"
	@echo "  setup  - Setup commit hooks and local database"
	@echo "  reset  - Reset local database"
	@echo "  lint   - Run linter"
	@echo "  update - Update Go modules"
	@echo "  test   - Run tests"
	@echo "  build  - Build the service"
	@echo "  run    - Run the service"
	@echo "  clean  - Remove build artifacts"

setup:
	@echo "Setting up commit hooks and local database..."
	pre-commit install --hook-type pre-commit --hook-type pre-push
	./scripts/db.sh

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

run: build setup
	@echo "Running service..."
	./scripts/run.sh

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(OUTPUT)
