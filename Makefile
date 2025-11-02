.PHONY: help build build-api build-worker run run-api run-worker stop status clean test test-coverage test-race test-short test-integration test-integration-coverage test-all fmt lint vet install-tools deps migrate migrate-down db-setup db-create db-drop db-shell db-status logs logs-api logs-worker docker-build docker-up docker-down swagger-up swagger-down swagger-restart swagger-logs check dev e2e-verify e2e-test-setup test-transaction-extraction

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_DIR=bin
API_BINARY=$(BINARY_DIR)/api
WORKER_BINARY=$(BINARY_DIR)/worker
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

# Colors for output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

## help: Show this help message
help:
	@echo '$(GREEN)Blockchain Explorer - Available Commands$(RESET)'
	@echo ''
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo ''

## build: Build API and worker binaries
build:
	@echo '$(YELLOW)Building binaries...$(RESET)'
	@mkdir -p $(BINARY_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(API_BINARY) ./cmd/api
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(WORKER_BINARY) ./cmd/worker
	@echo '$(GREEN)✓ Build complete$(RESET)'
	@echo '  API: $(API_BINARY)'
	@echo '  Worker: $(WORKER_BINARY)'

## build-api: Build only the API server binary
build-api:
	@echo '$(YELLOW)Building API server...$(RESET)'
	@mkdir -p $(BINARY_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(API_BINARY) ./cmd/api
	@echo '$(GREEN)✓ API built: $(API_BINARY)$(RESET)'

## build-worker: Build only the worker binary
build-worker:
	@echo '$(YELLOW)Building worker...$(RESET)'
	@mkdir -p $(BINARY_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(WORKER_BINARY) ./cmd/worker
	@echo '$(GREEN)✓ Worker built: $(WORKER_BINARY)$(RESET)'

## run: Start both API server and worker using the run script
run:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		echo 'Copy .env.example to .env and configure it first:'; \
		echo '  cp .env.example .env'; \
		exit 1; \
	fi
	@./scripts/run.sh

## run-api: Run the API server in foreground
run-api:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		echo 'Loading environment from .env.example...'; \
		export $$(grep -v '^#' .env.example | xargs); \
	else \
		export $$(grep -v '^#' .env | xargs); \
	fi; \
	$(GO) run ./cmd/api/main.go

## run-worker: Run the worker in foreground
run-worker:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		echo 'Loading environment from .env.example...'; \
		export $$(grep -v '^#' .env.example | xargs); \
	else \
		export $$(grep -v '^#' .env | xargs); \
	fi; \
	$(GO) run ./cmd/worker/main.go

## stop: Stop all running services
stop:
	@./scripts/stop.sh

## status: Check status of running services
status:
	@./scripts/status.sh

## clean: Remove build artifacts and logs
clean:
	@echo '$(YELLOW)Cleaning build artifacts...$(RESET)'
	@rm -rf $(BINARY_DIR)
	@rm -f logs/*.log
	@rm -f logs/*.pid
	@$(GO) clean
	@echo '$(GREEN)✓ Clean complete$(RESET)'

## test: Run all tests
test:
	@echo '$(YELLOW)Running tests...$(RESET)'
	@$(GO) test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo '$(YELLOW)Running tests with coverage...$(RESET)'
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo '$(GREEN)✓ Coverage report generated: coverage.html$(RESET)'

## test-race: Run tests with race detector
test-race:
	@echo '$(YELLOW)Running tests with race detector...$(RESET)'
	@$(GO) test -v -race ./...

## test-short: Run tests excluding integration tests
test-short:
	@echo '$(YELLOW)Running unit tests...$(RESET)'
	@$(GO) test -v -short ./...

## test-integration: Run integration tests only (requires Docker)
test-integration:
	@echo '$(YELLOW)Running integration tests (timeout: 5 minutes)...$(RESET)'
	@if ! docker info >/dev/null 2>&1; then \
		echo '$(YELLOW)⚠️  Docker is not running. Integration tests require Docker.$(RESET)'; \
		echo 'Please start Docker and try again.'; \
		exit 1; \
	fi
	@$(GO) test -v -tags=integration -timeout=5m ./...
	@echo '$(GREEN)✓ Integration tests complete$(RESET)'

## test-integration-coverage: Run integration tests with coverage
test-integration-coverage:
	@echo '$(YELLOW)Running integration tests with coverage...$(RESET)'
	@if ! docker info >/dev/null 2>&1; then \
		echo '$(YELLOW)⚠️  Docker is not running. Integration tests require Docker.$(RESET)'; \
		exit 1; \
	fi
	@$(GO) test -v -tags=integration -timeout=5m -coverprofile=coverage-integration.out ./...
	@$(GO) tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo '$(GREEN)✓ Integration coverage report: coverage-integration.html$(RESET)'

## test-all: Run all tests (unit + integration)
test-all:
	@echo '$(YELLOW)Running all tests...$(RESET)'
	@$(MAKE) test
	@$(MAKE) test-integration
	@echo '$(GREEN)✓ All tests complete$(RESET)'

## fmt: Format Go code
fmt:
	@echo '$(YELLOW)Formatting code...$(RESET)'
	@$(GO) fmt ./...
	@echo '$(GREEN)✓ Code formatted$(RESET)'

## lint: Run golangci-lint
lint:
	@echo '$(YELLOW)Running linter...$(RESET)'
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo '$(GREEN)✓ Linting complete$(RESET)'; \
	else \
		echo '$(YELLOW)golangci-lint not installed. Run: make install-tools$(RESET)'; \
	fi

## vet: Run go vet
vet:
	@echo '$(YELLOW)Running go vet...$(RESET)'
	@$(GO) vet ./...
	@echo '$(GREEN)✓ Vet complete$(RESET)'

## install-tools: Install development tools
install-tools:
	@echo '$(YELLOW)Installing development tools...$(RESET)'
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo '$(GREEN)✓ Tools installed$(RESET)'

## deps: Download and tidy dependencies
deps:
	@echo '$(YELLOW)Downloading dependencies...$(RESET)'
	@$(GO) mod download
	@$(GO) mod tidy
	@echo '$(GREEN)✓ Dependencies updated$(RESET)'

## db-setup: Setup database (start Docker PostgreSQL and run migrations)
db-setup:
	@echo '$(YELLOW)Setting up database...$(RESET)'
	@if ! docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
		echo '$(YELLOW)Starting PostgreSQL with Docker...$(RESET)'; \
		docker-compose up -d postgres; \
		echo 'Waiting for PostgreSQL to be ready...'; \
		sleep 5; \
		until docker exec blockchain-explorer-db pg_isready -U postgres 2>/dev/null; do \
			echo 'Waiting for database...'; \
			sleep 2; \
		done; \
	else \
		echo '$(GREEN)PostgreSQL is already running$(RESET)'; \
	fi
	@$(MAKE) db-create
	@$(MAKE) migrate
	@echo '$(GREEN)✓ Database setup complete$(RESET)'

## migrate: Run database migrations (works with Docker or local PostgreSQL)
migrate:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	echo '$(YELLOW)Running database migrations...$(RESET)'; \
	if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
		echo 'Using Docker container...'; \
		for file in migrations/*_*.up.sql; do \
			echo "Applying migration: $$file"; \
			docker exec -i blockchain-explorer-db psql -U $$DB_USER -d $$DB_NAME < $$file || exit 1; \
		done; \
	else \
		echo 'Using local PostgreSQL...'; \
		for file in migrations/*_*.up.sql; do \
			echo "Applying migration: $$file"; \
			psql -h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} -U $$DB_USER -d $$DB_NAME -f $$file || exit 1; \
		done; \
	fi
	@echo '$(GREEN)✓ Migrations complete$(RESET)'

## migrate-down: Rollback database migrations (works with Docker or local PostgreSQL)
migrate-down:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	echo '$(YELLOW)⚠️  WARNING: Rolling back migrations will delete data!$(RESET)'; \
	read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo '$(YELLOW)Rolling back database migrations...$(RESET)'; \
		if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
			echo 'Using Docker container...'; \
			for file in $$(ls -r migrations/*_*.down.sql); do \
				echo "Rolling back: $$file"; \
				docker exec -i blockchain-explorer-db psql -U $$DB_USER -d $$DB_NAME < $$file || exit 1; \
			done; \
		else \
			echo 'Using local PostgreSQL...'; \
			for file in $$(ls -r migrations/*_*.down.sql); do \
				echo "Rolling back: $$file"; \
				psql -h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} -U $$DB_USER -d $$DB_NAME -f $$file || exit 1; \
			done; \
		fi; \
		echo '$(GREEN)✓ Migrations rolled back$(RESET)'; \
	else \
		echo 'Cancelled'; \
	fi

## db-create: Create the database (works with Docker or local PostgreSQL)
db-create:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	echo '$(YELLOW)Creating database...$(RESET)'; \
	if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
		echo 'Using Docker container...'; \
		docker exec blockchain-explorer-db psql -U $$DB_USER -c "CREATE DATABASE $$DB_NAME;" 2>/dev/null || true; \
	else \
		echo 'Using local PostgreSQL...'; \
		psql -h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} -U $$DB_USER -c "CREATE DATABASE $$DB_NAME;" || true; \
	fi
	@echo '$(GREEN)✓ Database created$(RESET)'

## db-drop: Drop the database (WARNING: destructive)
db-drop:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	echo '$(YELLOW)⚠️  WARNING: This will delete all data!$(RESET)'; \
	read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
			echo 'Using Docker container...'; \
			docker exec blockchain-explorer-db psql -U $$DB_USER -c "DROP DATABASE IF EXISTS $$DB_NAME;"; \
		else \
			echo 'Using local PostgreSQL...'; \
			psql -h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} -U $$DB_USER -c "DROP DATABASE IF EXISTS $$DB_NAME;"; \
		fi; \
		echo '$(GREEN)✓ Database dropped$(RESET)'; \
	else \
		echo 'Cancelled'; \
	fi

## db-shell: Connect to database shell (works with Docker or local PostgreSQL)
db-shell:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
		echo '$(GREEN)Connecting to Docker PostgreSQL...$(RESET)'; \
		docker exec -it blockchain-explorer-db psql -U $$DB_USER -d $$DB_NAME; \
	else \
		echo '$(GREEN)Connecting to local PostgreSQL...$(RESET)'; \
		psql -h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} -U $$DB_USER -d $$DB_NAME; \
	fi

## db-status: Check database status
db-status:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Warning: .env file not found$(RESET)'; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs); \
	if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then \
		echo '$(GREEN)✓ PostgreSQL Docker container is running$(RESET)'; \
		if docker exec blockchain-explorer-db pg_isready -U $$DB_USER -d $$DB_NAME 2>/dev/null; then \
			echo '$(GREEN)✓ Database is accepting connections$(RESET)'; \
			SIZE=$$(docker exec blockchain-explorer-db psql -U $$DB_USER -d $$DB_NAME -t -c "SELECT pg_size_pretty(pg_database_size('$$DB_NAME'));" 2>/dev/null | tr -d ' '); \
			echo "Database size: $$SIZE"; \
		else \
			echo '$(RED)✗ Database is not ready$(RESET)'; \
		fi; \
	else \
		echo '$(YELLOW)PostgreSQL Docker container is not running$(RESET)'; \
		echo 'Try: make docker-up or make db-setup'; \
	fi

## logs: Tail all log files
logs:
	@tail -f logs/*.log

## logs-api: Tail API server logs
logs-api:
	@tail -f logs/api.log

## logs-worker: Tail worker logs
logs-worker:
	@tail -f logs/worker.log

## docker-build: Build Docker images (if Dockerfile exists)
docker-build:
	@if [ -f Dockerfile ]; then \
		echo '$(YELLOW)Building Docker images...$(RESET)'; \
		docker build -t blockchain-explorer:latest .; \
		echo '$(GREEN)✓ Docker image built$(RESET)'; \
	else \
		echo '$(YELLOW)Dockerfile not found$(RESET)'; \
	fi

## docker-up: Start services using docker-compose
docker-up:
	@if [ -f docker-compose.yml ]; then \
		echo '$(YELLOW)Starting Docker services...$(RESET)'; \
		docker-compose up -d; \
		echo '$(GREEN)✓ Services started$(RESET)'; \
	else \
		echo '$(YELLOW)docker-compose.yml not found$(RESET)'; \
	fi

## docker-down: Stop services using docker-compose
docker-down:
	@if [ -f docker-compose.yml ]; then \
		echo '$(YELLOW)Stopping Docker services...$(RESET)'; \
		docker-compose down; \
		echo '$(GREEN)✓ Services stopped$(RESET)'; \
	else \
		echo '$(YELLOW)docker-compose.yml not found$(RESET)'; \
	fi

## swagger-up: Start Swagger UI for API documentation
swagger-up:
	@if [ -f docker-compose.swagger.yml ]; then \
		echo '$(YELLOW)Starting Swagger UI...$(RESET)'; \
		docker-compose -f docker-compose.swagger.yml up -d; \
		echo '$(GREEN)✓ Swagger UI started$(RESET)'; \
		echo ''; \
		echo 'Access Swagger UI at: $(BLUE)http://localhost:8081$(RESET)'; \
		echo 'API Documentation: See SWAGGER.md for details'; \
	else \
		echo '$(YELLOW)docker-compose.swagger.yml not found$(RESET)'; \
	fi

## swagger-down: Stop Swagger UI
swagger-down:
	@if [ -f docker-compose.swagger.yml ]; then \
		echo '$(YELLOW)Stopping Swagger UI...$(RESET)'; \
		docker-compose -f docker-compose.swagger.yml down; \
		echo '$(GREEN)✓ Swagger UI stopped$(RESET)'; \
	else \
		echo '$(YELLOW)docker-compose.swagger.yml not found$(RESET)'; \
	fi

## swagger-restart: Restart Swagger UI (useful after updating openapi.yaml)
swagger-restart:
	@$(MAKE) swagger-down
	@$(MAKE) swagger-up

## swagger-logs: View Swagger UI logs
swagger-logs:
	@docker-compose -f docker-compose.swagger.yml logs -f swagger-ui

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo '$(GREEN)✓ All checks passed$(RESET)'

## dev: Quick development cycle (format, build, run)
dev: fmt build run
	@echo '$(GREEN)✓ Development environment ready$(RESET)'

## e2e-verify: Run E2E verification tests for transaction extraction pipeline
e2e-verify:
	@if [ ! -f .env ]; then \
		echo '$(YELLOW)Setting up .env for E2E tests...$(RESET)'; \
		export DATABASE_URL="postgres://postgres:postgres@localhost:5432/blockchain_explorer"; \
	else \
		export $$(grep -v '^#' .env | xargs); \
	fi; \
	echo '$(YELLOW)Running E2E verification tests...$(RESET)'; \
	$(GO) run cmd/e2e-verify/main.go
	@echo '$(GREEN)✓ E2E verification complete$(RESET)'

## e2e-test-setup: Setup database and run worker for E2E testing
e2e-test-setup: db-setup
	@echo '$(YELLOW)Starting worker for E2E tests...$(RESET)'
	@export $$(grep -v '^#' .env | xargs); \
	export BACKFILL_START_HEIGHT=0; \
	export BACKFILL_END_HEIGHT=10; \
	timeout 60 $(GO) run ./cmd/worker/main.go || true
	@echo '$(GREEN)✓ Worker indexing complete. Run: make e2e-verify$(RESET)'

## test-transaction-extraction: Full E2E test for transaction extraction
test-transaction-extraction: build db-setup
	@echo '$(YELLOW)Testing transaction extraction pipeline...$(RESET)'
	@echo '  1. Building binaries...'; \
	$(MAKE) build > /dev/null; \
	echo '$(GREEN)  ✓ Binaries built$(RESET)'; \
	echo '  2. Starting worker with small backfill...'; \
	export $$(grep -v '^#' .env | xargs); \
	export BACKFILL_START_HEIGHT=0; \
	export BACKFILL_END_HEIGHT=5; \
	timeout 45 $(GO) run ./cmd/worker/main.go > logs/worker-e2e.log 2>&1 || true; \
	echo '$(GREEN)  ✓ Worker completed indexing$(RESET)'; \
	echo '  3. Running verification tests...'; \
	export DATABASE_URL="postgres://postgres:postgres@localhost:5432/blockchain_explorer"; \
	$(GO) run cmd/e2e-verify/main.go
	@echo '$(GREEN)✓ Transaction extraction E2E test complete$(RESET)'
