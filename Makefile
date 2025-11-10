.PHONY: build run docker-build docker-run docker-up docker-down test clean help

# Build the application
build:
	@echo "Building application..."
	go build -o todolist-api ./cmd/server

# Run the application locally
run: build
	@echo "Starting server..."
	./todolist-api

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

# Run API tests (requires server to be running)
test:
	@echo "Running API tests..."
	@./test-api.sh

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
	@echo "  build        - Build the application"
	@echo "  run          - Build and run the application locally"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  docker-up    - Start with Docker Compose"
	@echo "  docker-down  - Stop Docker Compose"
	@echo "  test         - Run API tests (server must be running)"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  help         - Show this help message"
