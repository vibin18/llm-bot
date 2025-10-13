.PHONY: build test lint docker-build docker-run clean dev help

# Variables
APP_NAME := whatsapp-llm-bot
DOCKER_IMAGE := $(APP_NAME):latest
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w"

# Default target
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/bot
	@echo "Build complete: bin/$(APP_NAME)"

test: ## Run tests
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-e TZ=Europe/Brussels \
		-v $(PWD)/data:/data \
		-v $(PWD)/config.yaml:/config/config.yaml \
		$(DOCKER_IMAGE)
	@echo "Container started. Access admin UI at http://localhost:8080"

docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) && docker rm $(APP_NAME)

docker-logs: ## Show Docker container logs
	docker logs -f $(APP_NAME)

run: build ## Build and run the application locally
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME)

dev: ## Run in development mode with auto-reload (requires air)
	@echo "Starting development mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without auto-reload..."; \
		$(MAKE) run; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html
	rm -rf whatsapp_session/
	@echo "Clean complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

upgrade-deps: ## Upgrade dependencies
	@echo "Upgrading dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

install: ## Install the application
	@echo "Installing $(APP_NAME)..."
	$(GO) install $(LDFLAGS) ./cmd/bot

all: clean deps fmt vet lint test build ## Run all checks and build
