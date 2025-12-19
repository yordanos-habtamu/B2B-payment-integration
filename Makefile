# B2B Payments API Makefile

.PHONY: help build clean test lint fmt docs dev docker-build docker-run docker-stop migrate

# Default target
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
dev: ## Run all services in development mode
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.dev.yml up --build

dev-api: ## Run only API server
	@echo "Starting API server..."
	@go run ./cmd/api/main.go

dev-worker: ## Run only background worker
	@echo "Starting background worker..."
	@go run ./cmd/worker/main.go

dev-proxy: ## Run only load balancer
	@echo "Starting load balancer..."
	@go run ./cmd/proxy/main.go

# Build targets
build: ## Build all binaries
	@echo "Building all services..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api
	@go build -o bin/worker ./cmd/worker
	@go build -o bin/proxy ./cmd/proxy
	@echo "Build completed: bin/api, bin/worker, bin/proxy"

build-api: ## Build API server binary
	@echo "Building API server..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api

build-worker: ## Build worker binary
	@echo "Building background worker..."
	@mkdir -p bin
	@go build -o bin/worker ./cmd/worker

build-proxy: ## Build proxy binary
	@echo "Building load balancer..."
	@mkdir -p bin
	@go build -o bin/proxy ./cmd/proxy

# Test targets
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

# Quality targets
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

security: ## Run security scan
	@echo "Running security scan..."
	@gosec ./...

# Documentation targets
docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@swag init -g cmd/api/main.go
	@swag gen
	@echo "Documentation generated: docs/swagger.yaml, docs/swagger.html"

docs-serve: ## Serve documentation locally
	@echo "Starting documentation server..."
	@cd docs && python3 -m http.server 8080

# Database targets
migrate-up: ## Run database migrations
	@echo "Running database migrations..."
	@go run ./migrations/main.go up

migrate-down: ## Rollback database migrations
	@echo "Rolling back database migrations..."
	@go run ./migrations/main.go down

migrate-create: ## Create new migration (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME parameter is required"; \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	@go run ./migrations/main.go create $(NAME)

migrate-status: ## Show migration status
	@echo "Checking migration status..."
	@go run ./migrations/main.go status

# Certificate targets
certs: ## Generate development certificates
	@echo "Generating development certificates..."
	@mkdir -p certs
	@./scripts/certs/generate-dev-certs.sh

certs-clean: ## Clean generated certificates
	@echo "Cleaning certificates..."
	@rm -rf certs/*

# Docker targets
docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker build -t b2b-payments-api .
	@docker build -t b2b-payments-worker -f Dockerfile.worker .
	@docker build -t b2b-payments-proxy -f Dockerfile.proxy .

docker-run: ## Run with Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up --build

docker-stop: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose down

docker-clean: ## Clean Docker resources
	@echo "Cleaning Docker resources..."
	@docker-compose down -v
	@docker system prune -f

# Performance targets
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

load-test: ## Run load tests
	@echo "Running load tests..."
	@k6 run tests/load/payments.js

# Deployment targets
deploy-dev: ## Deploy to development environment
	@echo "Deploying to development..."
	@./scripts/deploy.sh dev

deploy-staging: ## Deploy to staging environment
	@echo "Deploying to staging..."
	@./scripts/deploy.sh staging

deploy-prod: ## Deploy to production environment
	@echo "Deploying to production..."
	@./scripts/deploy.sh prod

# Utility targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f docs/swagger.yaml docs/swagger.html

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Monitoring targets
metrics: ## Show service metrics
	@echo "Fetching service metrics..."
	@curl -s http://localhost:9090/metrics | head -20

health: ## Check service health
	@echo "Checking service health..."
	@curl -k --cert certs/tenant-123.crt --key certs/tenant-123.key https://localhost:8443/health
	@curl http://localhost:8080/health

# Setup targets
setup: ## Initial project setup
	@echo "Setting up project..."
	@mkdir -p bin certs logs
	@go mod download
	@make certs
	@make migrate-up
	@echo "Setup completed!"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@brew install k6 || echo "k6 already installed or not available"

# Release targets
version: ## Show version information
	@echo "B2B Payments API"
	@echo "Git commit: $$(git rev-parse HEAD)"
	@echo "Git branch: $$(git rev-parse --abbrev-ref HEAD)"
	@echo "Go version: $$(go version)"
	@echo "Build time: $$(date)"

release: ## Create release (usage: make release VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@git tag $(VERSION)
	@git push origin $(VERSION)
	@echo "Release $(VERSION) created and pushed!"