# Todo List REST API

A REST API service for managing multiple named todo lists with full CRUD operations, built with Go and Gin.

## Features

- **Multiple Named Lists**: Create and manage multiple todo lists
- **Full CRUD Operations**: Complete Create, Read, Update, Delete operations for both lists and todos
- **Rich Todo Items**: Each todo has description, priority (low/medium/high), and due date
- **Filtering & Sorting**: Filter todos by priority/completion status, sort by date/priority
- **Pagination**: Paginated list retrieval
- **Containerized**: Docker and Docker Compose support for easy deployment

## API Specification

The API follows the OpenAPI 3.0 specification defined in [api/openapi.yaml](api/openapi.yaml).

### Base URL

```
http://localhost:8080/api/v1
```

### Endpoints

#### Todo Lists
- `GET /lists` - Get all todo lists (with pagination)
- `POST /lists` - Create a new todo list
- `GET /lists/{listId}` - Get a specific list
- `PUT /lists/{listId}` - Update a list
- `DELETE /lists/{listId}` - Delete a list and all its todos

#### Todos
- `GET /lists/{listId}/todos` - Get all todos in a list (with filtering/sorting)
- `POST /lists/{listId}/todos` - Create a new todo
- `GET /lists/{listId}/todos/{todoId}` - Get a specific todo
- `PUT /lists/{listId}/todos/{todoId}` - Update a todo
- `DELETE /lists/{listId}/todos/{todoId}` - Delete a todo

#### Health Check
- `GET /health` - Health check endpoint

## Quick Start

### Prerequisites

- Go 1.23 or later (for local development)
- Docker and Docker Compose (for containerized deployment)

### Running Locally (Without Docker)

1. Install dependencies:
```bash
go mod download
```

2. Run the server:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`.

### Running with Docker Compose (Recommended)

1. Build and start the container:
```bash
docker-compose up --build
```

2. The API will be available at `http://localhost:8080`.

3. To run in detached mode:
```bash
docker-compose up -d
```

4. To stop the container:
```bash
docker-compose down
```

### Running with Docker Only

1. Build the image:
```bash
docker build -t todolist-api .
```

2. Run the container:
```bash
docker run -p 8080:8080 todolist-api
```

## Usage Examples

### Create a Todo List

```bash
curl -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Work Tasks",
    "description": "Tasks for work projects"
  }'
```

### Get All Lists

```bash
curl http://localhost:8080/api/v1/lists
```

### Create a Todo

```bash
curl -X POST http://localhost:8080/api/v1/lists/{listId}/todos \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Complete project documentation",
    "priority": "high",
    "dueDate": "2025-11-15T23:59:59Z"
  }'
```

### Get Todos with Filtering

```bash
# Get high priority todos
curl "http://localhost:8080/api/v1/lists/{listId}/todos?priority=high"

# Get incomplete todos sorted by due date
curl "http://localhost:8080/api/v1/lists/{listId}/todos?completed=false&sortBy=dueDate&sortOrder=asc"
```

### Update a Todo

```bash
curl -X PUT http://localhost:8080/api/v1/lists/{listId}/todos/{todoId} \
  -H "Content-Type: application/json" \
  -d '{
    "completed": true
  }'
```

### Delete a Todo

```bash
curl -X DELETE http://localhost:8080/api/v1/lists/{listId}/todos/{todoId}
```

## Project Structure

```
.
├── api/
│   └── openapi.yaml          # OpenAPI 3.0 specification
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── handlers/             # HTTP request handlers
│   │   ├── lists.go          # List CRUD handlers
│   │   └── todos.go          # Todo CRUD handlers
│   ├── models/               # Data models and DTOs
│   │   └── models.go
│   └── storage/              # Storage layer
│       └── storage.go        # In-memory storage implementation
├── Dockerfile                # Docker image definition
├── docker-compose.yml        # Docker Compose configuration
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Storage

Currently uses in-memory storage. Data is lost when the service restarts. This is suitable for development and testing. For production use, you would integrate a database (PostgreSQL, MongoDB, etc.).

## Configuration

The service can be configured using environment variables:

- `PORT`: Server port (default: 8080)

## Development

### Building

```bash
go build -o todolist-api ./cmd/server
```

### Running Tests

```bash
go test ./...
```

## Next Steps

- Add authentication and authorization
- Add database persistence (PostgreSQL/MongoDB)
- Add unit and integration tests
- Add request logging and metrics
- Add CORS middleware
- Add rate limiting

## License

MIT
