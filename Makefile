.PHONY: build run run-memory docker-build docker-run docker-up docker-down docker-logs db-shell test test-unit test-coverage test-verbose clean help

# Build the application
build:
	@echo "Building application..."
	go build -o todolist-api ./cmd/server

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

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f todolist-api
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

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run with PostgreSQL (requires local PostgreSQL)"
	@echo "  run-memory    - Build and run with in-memory storage (no database)"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Build and run Docker container"
	@echo "  docker-up     - Start with Docker Compose (PostgreSQL + API)"
	@echo "  docker-down   - Stop Docker Compose"
	@echo "  docker-logs   - View Docker Compose logs"
	@echo "  db-shell      - Connect to PostgreSQL database shell"
	@echo "  test          - Run integration API tests (server must be running)"
	@echo "  test-unit     - Run unit tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-verbose  - Run tests in verbose mode"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  help          - Show this help message"
