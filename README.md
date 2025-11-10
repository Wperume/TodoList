# Todo List REST API

A REST API service for managing multiple named todo lists with full CRUD operations, built with Go, Gin, and PostgreSQL.

## Features

- **Multiple Named Lists**: Create and manage multiple todo lists
- **Full CRUD Operations**: Complete Create, Read, Update, Delete operations for both lists and todos
- **Rich Todo Items**: Each todo has description, priority (low/medium/high), and due date
- **Filtering & Sorting**: Filter todos by priority/completion status, sort by date/priority
- **Pagination**: Paginated list retrieval
- **PostgreSQL Database**: Persistent storage with GORM ORM
- **Containerized**: Docker and Docker Compose support for easy deployment
- **Flexible Storage**: Can use in-memory storage for development/testing

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
- PostgreSQL 14+ (for local development without Docker)
- Docker and Docker Compose (for containerized deployment)

### Running with Docker Compose (Recommended)

This is the easiest way to get started. Docker Compose will start both the PostgreSQL database and the API server.

1. Build and start all services:
```bash
docker-compose up --build
```

2. The API will be available at `http://localhost:8080`.
   - PostgreSQL will be available on `localhost:5432`

3. To run in detached mode:
```bash
docker-compose up -d
```

4. View logs:
```bash
docker-compose logs -f todolist-api
```

5. To stop all services:
```bash
docker-compose down
```

6. To stop and remove all data:
```bash
docker-compose down -v
```

### Running Locally (Without Docker)

#### Option 1: With PostgreSQL

1. Start PostgreSQL and create the database:
```bash
# Using psql
createdb -U postgres todolist
createuser -U postgres todouser
psql -U postgres -c "ALTER USER todouser WITH PASSWORD 'todopass';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE todolist TO todouser;"
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. Install Go dependencies:
```bash
go mod download
```

4. Run the server:
```bash
go run cmd/server/main.go
```

The server will automatically run database migrations on startup.

#### Option 2: With In-Memory Storage (Development Only)

For quick testing without PostgreSQL:

```bash
USE_MEMORY_STORAGE=true go run cmd/server/main.go
```

Note: Data will be lost when the server restarts.

### Running with Docker Only

If you want to run just the API container and use an external PostgreSQL:

1. Build the image:
```bash
docker build -t todolist-api .
```

2. Run the container:
```bash
docker run -p 8080:8080 \
  -e DB_HOST=your-postgres-host \
  -e DB_PORT=5432 \
  -e DB_USER=todouser \
  -e DB_PASSWORD=todopass \
  -e DB_NAME=todolist \
  todolist-api
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
│   ├── openapi.yaml          # OpenAPI 3.0 specification
│   └── examples.md           # API usage examples
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── database/             # Database configuration
│   │   └── database.go       # PostgreSQL connection and migrations
│   ├── handlers/             # HTTP request handlers
│   │   ├── lists.go          # List CRUD handlers
│   │   └── todos.go          # Todo CRUD handlers
│   ├── models/               # Data models and DTOs
│   │   └── models.go         # GORM models with validation
│   └── storage/              # Storage layer
│       ├── interface.go      # Storage interface
│       ├── storage.go        # In-memory implementation
│       └── postgres.go       # PostgreSQL/GORM implementation
├── Dockerfile                # Docker image definition
├── docker-compose.yml        # Docker Compose with PostgreSQL
├── .env.example              # Environment variables example
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Database

The application uses **PostgreSQL** with **GORM** for persistent storage:

- **Auto-migrations**: Database schema is automatically created/updated on startup
- **Soft deletes**: Deleted records are marked as deleted (not physically removed)
- **Foreign keys**: Todos are linked to lists with cascade delete
- **Indexes**: Optimized queries with indexes on commonly searched fields
- **UUID primary keys**: Uses UUIDs for all entity IDs

### Database Schema

**todo_lists table:**
- `id` (UUID, primary key)
- `name` (varchar(100), unique)
- `description` (varchar(500))
- `created_at`, `updated_at`, `deleted_at` (timestamps)

**todos table:**
- `id` (UUID, primary key)
- `list_id` (UUID, foreign key → todo_lists.id)
- `description` (varchar(500))
- `priority` (varchar(10): low/medium/high)
- `due_date` (timestamp, nullable)
- `completed` (boolean, default: false)
- `completed_at` (timestamp, nullable)
- `created_at`, `updated_at`, `deleted_at` (timestamps)

## Configuration

The service can be configured using environment variables:

### Server Configuration
- `PORT`: Server port (default: 8080)

### Database Configuration
- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_USER`: Database user (default: todouser)
- `DB_PASSWORD`: Database password (default: todopass)
- `DB_NAME`: Database name (default: todolist)
- `DB_SSLMODE`: SSL mode (default: disable)
- `DB_LOG_LEVEL`: Set to "silent" to disable SQL logging

### Storage Configuration
- `USE_MEMORY_STORAGE`: Set to "true" to use in-memory storage instead of PostgreSQL

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

- ✅ ~~Add database persistence (PostgreSQL/MongoDB)~~ - **COMPLETED**
- Add JWT authentication and authorization
- Add unit and integration tests
- Add request logging and metrics
- Add CORS middleware
- Add rate limiting
- Add API documentation UI (Swagger/ReDoc)
- Add database connection pooling tuning
- Add health check with database connectivity status

## License

MIT
