.PHONY: build run run-memory docker-build docker-run docker-up docker-down docker-logs db-shell test test-unit test-coverage test-verbose test-security test-security-verbose scan-security build-scanner clean help migrate-up migrate-down migrate-version migrate-steps migrate-force migrate-create build-migrate version

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -X 'todolist-api/internal/version.Version=$(VERSION)' \
          -X 'todolist-api/internal/version.Commit=$(COMMIT)' \
          -X 'todolist-api/internal/version.BuildDate=$(BUILD_DATE)'

# Build the application
build:
	@echo "Building application..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	go build -ldflags "$(LDFLAGS)" -o todolist-api ./cmd/server

# Run the application locally with PostgreSQL
run: build
	@echo "Starting server with PostgreSQL..."
	@echo "Make sure PostgreSQL is running and configured in .env"
	./todolist-api

# Run the application with in-memory storage (no database required)
run-memory: build
	@echo "Starting server with in-memory storage..."
	USE_MEMORY_STORAGE=true ./todolist-api

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t todolist-api .

# Run Docker container
docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 8080:8080 --name todolist-api todolist-api

# Start with Docker Compose
docker-up:
	@echo "Starting with Docker Compose..."
	docker-compose up --build

# Stop Docker Compose
docker-down:
	@echo "Stopping Docker Compose..."
	docker-compose down

# View Docker Compose logs
docker-logs:
	@echo "Viewing logs..."
	docker-compose logs -f

# Connect to PostgreSQL database shell
db-shell:
	@echo "Connecting to PostgreSQL..."
	docker-compose exec postgres psql -U todouser -d todolist

# Run integration API tests (requires server to be running)
test:
	@echo "Running integration API tests..."
	@./test-api.sh

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test ./... -short -count=1

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -short -coverprofile=coverage.out -covermode=atomic
	@echo "Coverage report saved to coverage.out"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report saved to coverage.html"

# Run tests in verbose mode
test-verbose:
	@echo "Running tests in verbose mode..."
	go test ./... -v -short -count=1

# Run security integration tests
test-security:
	@echo "Running security tests..."
	go test ./internal/security/... -v -count=1

# Run security tests in verbose mode
test-security-verbose:
	@echo "Running security tests (verbose)..."
	go test ./internal/security/... -v -count=1 -test.v

# Build security scanner
build-scanner:
	@echo "Building security scanner..."
	@go build -o bin/security-scanner ./cmd/security-scanner
	@echo "✅ Security scanner built: bin/security-scanner"

# Run security scanner against local instance
scan-security: build-scanner
	@echo "Running security scan..."
	@if [ -z "$(TARGET)" ]; then \
		echo "Using default target: https://localhost:8443"; \
		./bin/security-scanner --target https://localhost:8443 --skip-tls --verbose --output security-report.html; \
	else \
		./bin/security-scanner --target $(TARGET) --verbose --output security-report.html; \
	fi

# Run all tests including security
test-all: test-unit test-security
	@echo "✅ All tests completed"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f todolist-api
	@rm -f bin/migrate
	@rm -f bin/security-scanner
	@rm -f security-report.html
	@go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	go vet ./...

# Build the migration tool
build-migrate:
	@echo "Building migration tool..."
	@go build -o bin/migrate cmd/migrate/main.go
	@echo "✅ Migration tool built: bin/migrate"

# Database migration commands
migrate-up: build-migrate
	@echo "Applying migrations..."
	@./bin/migrate up

migrate-down: build-migrate
	@echo "Rolling back migration..."
	@./bin/migrate down

migrate-version: build-migrate
	@./bin/migrate version

migrate-steps: build-migrate
	@if [ -z "$(N)" ]; then echo "Error: N is required. Usage: make migrate-steps N=2"; exit 1; fi
	@./bin/migrate steps $(N)

migrate-force: build-migrate
	@if [ -z "$(V)" ]; then echo "Error: V is required. Usage: make migrate-force V=1"; exit 1; fi
	@./bin/migrate force $(V)

# Create new migration files
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=add_feature"; \
		exit 1; \
	fi
	@TIMESTAMP=$$(date +%s); \
	NUMBER=$$(printf "%06d" $$(($$TIMESTAMP % 1000000))); \
	UP_FILE="internal/migration/migrations/$${NUMBER}_$(NAME).up.sql"; \
	DOWN_FILE="internal/migration/migrations/$${NUMBER}_$(NAME).down.sql"; \
	echo "-- Add your UP migration here" > $$UP_FILE; \
	echo "-- Add your DOWN migration here" > $$DOWN_FILE; \
	echo "✅ Created migration files:"; \
	echo "   $$UP_FILE"; \
	echo "   $$DOWN_FILE"

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run with PostgreSQL (requires local PostgreSQL)"
	@echo "  run-memory    - Build and run with in-memory storage (no database)"
	@echo ""
	@echo "Database Migrations:"
	@echo "  migrate-up            - Apply all pending migrations"
	@echo "  migrate-down          - Rollback the last migration"
	@echo "  migrate-version       - Show current migration version"
	@echo "  migrate-steps N=2     - Run N migration steps (use N=-1 to go back)"
	@echo "  migrate-force V=1     - Force migration version (for dirty state)"
	@echo "  migrate-create NAME=x - Create new migration files"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Build and run Docker container"
	@echo "  docker-up     - Start with Docker Compose (PostgreSQL + API)"
	@echo "  docker-down   - Stop Docker Compose"
	@echo "  docker-logs   - View Docker Compose logs"
	@echo "  db-shell      - Connect to PostgreSQL database shell"
	@echo ""
	@echo "Testing:"
	@echo "  test                    - Run integration API tests (server must be running)"
	@echo "  test-unit               - Run unit tests"
	@echo "  test-security           - Run security integration tests"
	@echo "  test-security-verbose   - Run security tests in verbose mode"
	@echo "  test-all                - Run all tests (unit + security)"
	@echo "  test-coverage           - Run tests with coverage report"
	@echo "  test-verbose            - Run tests in verbose mode"
	@echo ""
	@echo "Security:"
	@echo "  build-scanner           - Build security scanner tool"
	@echo "  scan-security           - Scan local/remote instance (use TARGET=url)"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  version       - Show version information"
	@echo "  help          - Show this help message"
