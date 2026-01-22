.PHONY: all build run test test-coverage lint lint-fix clean docker-build docker-run migrate seed help

# Variables
BINARY_NAME=vendorplatform
GO=go
GOFLAGS=-ldflags="-s -w"
MAIN_PATH=./cmd/server

# Default target
all: lint test build

## Build Commands

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	$(GO) run $(MAIN_PATH)

run-dev: ## Run with hot reload (requires air)
	@air

## Test Commands

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	$(GO) test -race -v ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## Lint Commands

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run ./...

lint-fix: ## Run linters and fix issues
	@echo "Running linters with auto-fix..."
	@golangci-lint run --fix ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

## Database Commands

migrate: ## Run database migrations
	@echo "Running migrations..."
	psql $(DATABASE_URL) -f database/001_core_schema.sql

seed: ## Seed the database
	@echo "Seeding database..."
	psql $(DATABASE_URL) -f database/002_seed_data.sql

migrate-fresh: ## Drop and recreate database
	@echo "Recreating database..."
	dropdb vendorplatform --if-exists
	createdb vendorplatform
	$(MAKE) migrate
	$(MAKE) seed

## Docker Commands

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

docker-compose-up: ## Start all services with docker-compose
	@echo "Starting services..."
	docker-compose up -d

docker-compose-down: ## Stop all services
	@echo "Stopping services..."
	docker-compose down

## Code Generation

generate: ## Run go generate
	@echo "Running go generate..."
	$(GO) generate ./...

mocks: ## Generate mocks
	@echo "Generating mocks..."
	@mockgen -source=./internal/repository/repository.go -destination=./internal/repository/mocks/repository_mock.go
	@mockgen -source=./internal/service/service.go -destination=./internal/service/mocks/service_mock.go

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	@swag init -g cmd/server/main.go -o ./docs/swagger

## Dependency Management

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

deps-tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GO) mod tidy

## Python ML Service

ml-install: ## Install Python dependencies
	@echo "Installing Python dependencies..."
	pip install -r requirements.txt

ml-run: ## Run ML service
	@echo "Running ML service..."
	python recommendation-engine/ml_service.py

## Clean

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf tmp/
	$(GO) clean

## Utility

env-example: ## Create example env file
	@echo "Creating .env.example..."
	@cat > .env.example << 'EOF'
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/vendorplatform
REDIS_URL=redis://localhost:6379

# API
PORT=8080
ENV=development

# Services
NOTIFICATION_SERVICE_URL=http://localhost:8081
PAYMENT_SERVICE_URL=http://localhost:8082

# Security
JWT_SECRET=your-secret-key
API_KEY=your-api-key
EOF

## Help

help: ## Show this help
	@echo "VendorPlatform - Contextual Commerce Orchestration"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
