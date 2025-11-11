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
- **Rate Limiting**: Configurable rate limiting to protect against abuse
- **Comprehensive Logging**: Structured logging with automatic log rotation and configurable retention
- **Security Hardened**: XSS protection, CORS, security headers, request size limits, UUID validation

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

### Rate Limiting Configuration
- `RATE_LIMIT_ENABLED`: Enable/disable rate limiting (default: true)
- `RATE_LIMIT_REQUESTS_PER_MIN`: Maximum requests per minute per IP (default: 60)
- `RATE_LIMIT_REQUESTS_PER_HOUR`: Maximum requests per hour per IP (default: 1000, reserved for future use)
- `RATE_LIMIT_BURST`: Burst size for rate limiting (default: 10, reserved for future use)

### Logging Configuration
- `LOG_FILE_ENABLED`: Enable/disable file logging (default: true)
- `LOG_FILE_PATH`: Path to log file (default: ./logs/todolist-api.log)
- `LOG_MAX_SIZE_MB`: Maximum log file size in MB before rotation (default: 100)
- `LOG_MAX_BACKUPS`: Number of old log files to retain (default: 3)
- `LOG_MAX_AGE_DAYS`: Maximum days to retain old log files (default: 28)
- `LOG_COMPRESS`: Compress rotated log files (default: true)
- `LOG_LEVEL`: Log level - trace, debug, info, warn, error, fatal, panic (default: info)
- `LOG_JSON_FORMAT`: Use JSON format instead of text (default: false)

### Security Configuration
- `MAX_REQUEST_BODY_SIZE`: Maximum request body size in bytes (default: 1048576 = 1MB)
- `ENABLE_XSS_PROTECTION`: Enable XSS input sanitization (default: true)
- `TRUSTED_PROXIES`: Comma-separated list of trusted proxy IPs (optional)

### CORS Configuration
- `CORS_ENABLED`: Enable/disable CORS (default: true)
- `CORS_ALLOWED_ORIGINS`: Allowed origins, `*` for all or comma-separated list (default: *)
- `CORS_ALLOWED_METHODS`: Allowed HTTP methods (default: GET,POST,PUT,DELETE,OPTIONS,PATCH)
- `CORS_ALLOWED_HEADERS`: Allowed request headers
- `CORS_EXPOSE_HEADERS`: Headers exposed to client
- `CORS_ALLOW_CREDENTIALS`: Allow credentials like cookies (default: false)
- `CORS_MAX_AGE`: Preflight cache duration in seconds (default: 3600)

## Development

### Building

```bash
go build -o todolist-api ./cmd/server
```

### Running Tests

The project includes comprehensive unit tests with high coverage:

```bash
# Run all unit tests
make test-unit

# Run tests with coverage report
make test-coverage

# Run tests in verbose mode
make test-verbose

# Or use go test directly
go test ./... -v
```

**Test Coverage:**
- Models: 100%
- Logging: 86.2%
- Middleware: 82.1% (includes security, CORS, rate limiting, logging)
- Storage Layer: 80.2%

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Rate Limiting

The API includes configurable rate limiting to protect against abuse and ensure fair usage.

### Configuration

Rate limiting is controlled via environment variables (see [.env.example](.env.example)):

```bash
RATE_LIMIT_ENABLED=true                # Enable/disable rate limiting
RATE_LIMIT_REQUESTS_PER_MIN=60         # Maximum requests per minute per IP
RATE_LIMIT_REQUESTS_PER_HOUR=1000      # Reserved for future use
RATE_LIMIT_BURST=10                    # Reserved for future use
```

### Behavior

- **Global limit**: Applied to all endpoints by default (60 requests/minute per IP)
- **Per-IP tracking**: Rate limits are tracked separately for each IP address
- **Response on limit exceeded**: Returns HTTP 429 (Too Many Requests) with retry information:

```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Too many requests. Please try again later.",
  "retryAfter": 60
}
```

### Disabling Rate Limiting

For development or testing, you can disable rate limiting:

```bash
RATE_LIMIT_ENABLED=false go run cmd/server/main.go
```

### Custom Rate Limits

The middleware also provides separate rate limiters for read and write operations (currently not applied but available):

- **ReadRateLimiter**: Double the global limit (120 req/min) for GET requests
- **WriteRateLimiter**: Half the global limit (30 req/min) for POST/PUT/DELETE requests

These can be applied to specific route groups in [cmd/server/main.go](cmd/server/main.go:57).

## Logging

The API includes comprehensive request logging with automatic log rotation and configurable retention policies.

### Features

- **Request Logging**: Every HTTP request is logged with detailed information
- **Automatic Log Rotation**: Log files are automatically rotated when they reach the size limit
- **Configurable Retention**: Control how many old logs to keep and for how long
- **Compression**: Old log files are automatically compressed to save disk space
- **Multiple Formats**: Support for both human-readable text and machine-parseable JSON
- **Structured Logging**: Uses logrus for structured, leveled logging
- **Rate Limit Tracking**: Automatically logs when rate limits are exceeded

### Logged Information

Each request log entry includes:

- **Timestamp**: ISO 8601 formatted timestamp
- **Client IP**: IP address of the requesting client
- **Method & Path**: HTTP method and request path
- **Query Parameters**: URL query string
- **Status Code**: HTTP response status
- **Latency**: Request processing time in milliseconds
- **Response Size**: Size of the response body
- **User Agent**: Client user agent string
- **API Key Prefix**: First 8 characters of API key (when authentication is added)
- **Rate Limited**: Flag indicating if the request was rate limited
- **Errors**: Any errors that occurred during request processing

### Log Format Examples

**Text Format (default):**
```
time="2025-11-10 15:04:05" level=info msg="Request completed" client_ip=192.168.1.1 method=GET path=/api/v1/lists status=200 latency_ms=25 response_size=1024
```

**JSON Format:**
```json
{
  "time": "2025-11-10T15:04:05-05:00",
  "level": "info",
  "msg": "Request completed",
  "client_ip": "192.168.1.1",
  "method": "GET",
  "path": "/api/v1/lists",
  "query": "",
  "status": 200,
  "latency_ms": 25,
  "response_size": 1024,
  "user_agent": "Mozilla/5.0..."
}
```

### Log Rotation

Logs are automatically rotated using lumberjack:

- **Size-based**: When a log file reaches `LOG_MAX_SIZE_MB` (default: 100MB)
- **Retention by count**: Keep `LOG_MAX_BACKUPS` old files (default: 3)
- **Retention by age**: Delete files older than `LOG_MAX_AGE_DAYS` (default: 28 days)
- **Compression**: Old logs are gzipped to save disk space

Example log file structure:
```
logs/
├── todolist-api.log           # Current log file
├── todolist-api-2025-11-09.log.gz
├── todolist-api-2025-11-08.log.gz
└── todolist-api-2025-11-07.log.gz
```

### Log Levels

Configure logging verbosity with `LOG_LEVEL`:

- **trace**: Very detailed debugging information
- **debug**: Detailed debugging information
- **info** (default): General operational information
- **warn**: Warning messages (4xx errors, rate limits)
- **error**: Error messages (5xx errors)
- **fatal**: Fatal errors that cause the application to exit
- **panic**: Panic-level errors

### Rate Limit Logging

When a client exceeds the rate limit, a warning is logged:

```
time="2025-11-10 15:04:05" level=warning msg="Rate limit exceeded" client_ip=192.168.1.100 path=/api/v1/lists method=POST rate_limited=true limit_per_min=60
```

### Configuration Examples

**Production (JSON format, info level):**
```bash
LOG_FILE_ENABLED=true
LOG_FILE_PATH=/var/log/todolist-api/app.log
LOG_MAX_SIZE_MB=100
LOG_MAX_BACKUPS=10
LOG_MAX_AGE_DAYS=90
LOG_COMPRESS=true
LOG_LEVEL=info
LOG_JSON_FORMAT=true
```

**Development (text format, debug level):**
```bash
LOG_FILE_ENABLED=true
LOG_FILE_PATH=./logs/dev.log
LOG_MAX_SIZE_MB=10
LOG_MAX_BACKUPS=2
LOG_MAX_AGE_DAYS=7
LOG_COMPRESS=false
LOG_LEVEL=debug
LOG_JSON_FORMAT=false
```

**Testing (stdout only):**
```bash
LOG_FILE_ENABLED=false
LOG_LEVEL=warn
```

## Security

The API implements multiple layers of security protection. See [SECURITY.md](SECURITY.md) for complete security documentation.

### Implemented Security Features

✅ **SQL Injection Protection** - GORM parameterized queries
✅ **XSS Prevention** - HTML escaping of all user input
✅ **DoS Protection** - Rate limiting (60 req/min per IP)
✅ **Request Size Limits** - Maximum 1MB request body
✅ **Security Headers** - X-Frame-Options, CSP, X-XSS-Protection, etc.
✅ **CORS Protection** - Configurable origin whitelist
✅ **UUID Validation** - Format validation before database queries
✅ **Error Sanitization** - Generic errors to clients, detailed logs server-side
✅ **Memory Safety** - Go's built-in bounds checking and GC

### Security Configuration

**Production Settings:**
```bash
# Strict CORS - DO NOT use wildcard!
CORS_ALLOWED_ORIGINS=https://yourdomain.com

# Reasonable rate limits
RATE_LIMIT_REQUESTS_PER_MIN=30

# XSS protection enabled
ENABLE_XSS_PROTECTION=true

# Request size limit
MAX_REQUEST_BODY_SIZE=524288  # 512KB
```

**Development Settings:**
```bash
# Relaxed for development
CORS_ALLOWED_ORIGINS=*
RATE_LIMIT_ENABLED=false
```

### What's NOT Implemented (Yet)

⚠️ **Authentication** - No JWT/API key authentication (planned)
⚠️ **Authorization** - No user-level access control (planned)
⚠️ **HTTPS Enforcement** - Should be deployed behind HTTPS proxy

### Security Best Practices

1. **Always use HTTPS in production** - Deploy behind nginx/load balancer with SSL
2. **Configure CORS strictly** - Never use `*` wildcard in production
3. **Monitor rate limit logs** - Track suspicious IPs hitting limits
4. **Keep dependencies updated** - Regularly update Go modules
5. **Use strong database passwords** - Never use default credentials

See [SECURITY.md](SECURITY.md) for detailed security information, testing procedures, and deployment checklist.

## Next Steps

- ✅ ~~Add database persistence (PostgreSQL/MongoDB)~~ - **COMPLETED**
- ✅ ~~Add unit and integration tests~~ - **COMPLETED**
- ✅ ~~Add rate limiting~~ - **COMPLETED**
- ✅ ~~Add request logging~~ - **COMPLETED**
- ✅ ~~Add security hardening (XSS, CORS, headers, size limits)~~ - **COMPLETED**
- Add JWT authentication and authorization
- Add metrics and monitoring (Prometheus)
- Add API documentation UI (Swagger/ReDoc)
- Add database connection pooling tuning
- Add health check with database connectivity status
- Add HTTPS/TLS support

## License

MIT
