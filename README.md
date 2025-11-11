# Todo List REST API

A REST API service for managing multiple named todo lists with full CRUD operations, built with Go, Gin, and PostgreSQL.

## Features

- **JWT Authentication**: Secure user authentication with access and refresh tokens
- **User Management**: User registration, login, profile management, and password changes
- **Role-Based Access Control**: Support for user and admin roles
- **Multiple Named Lists**: Create and manage multiple todo lists per user
- **Full CRUD Operations**: Complete Create, Read, Update, Delete operations for both lists and todos
- **User Data Isolation**: Lists and todos are scoped to authenticated users
- **Rich Todo Items**: Each todo has description, priority (low/medium/high), and due date
- **Filtering & Sorting**: Filter todos by priority/completion status, sort by date/priority
- **Pagination**: Paginated list retrieval
- **PostgreSQL Database**: Persistent storage with GORM ORM
- **Containerized**: Docker and Docker Compose support for easy deployment
- **Flexible Storage**: Can use in-memory storage for development/testing (without auth)
- **Rate Limiting**: Configurable rate limiting to protect against abuse
- **Comprehensive Logging**: Structured logging with automatic log rotation and configurable retention
- **Security Hardened**: XSS protection, CORS, security headers, request size limits, UUID validation, bcrypt password hashing
- **HTTPS/TLS Support**: Secure communication with TLS 1.2/1.3, configurable cipher suites, and HTTP-to-HTTPS redirect

## API Specification

The API follows the OpenAPI 3.0 specification defined in [api/openapi.yaml](api/openapi.yaml).

### Base URL

**HTTP (Development):**
```
http://localhost:8080/api/v1
```

**HTTPS (Production):**
```
https://localhost:8443/api/v1
```

### Endpoints

#### Authentication (Public)
- `POST /auth/register` - Register a new user account
- `POST /auth/login` - Login and receive access + refresh tokens
- `POST /auth/refresh` - Refresh an access token using a refresh token
- `POST /auth/logout` - Logout and revoke refresh token

#### Authentication (Protected - Requires Authentication)
- `GET /auth/profile` - Get current user profile
- `PUT /auth/profile` - Update user profile (first name, last name)
- `PUT /auth/password` - Change password

#### Todo Lists (Protected - Requires Authentication)
- `GET /lists` - Get all todo lists (with pagination)
- `POST /lists` - Create a new todo list
- `GET /lists/{listId}` - Get a specific list
- `PUT /lists/{listId}` - Update a list
- `DELETE /lists/{listId}` - Delete a list and all its todos

#### Todos (Protected - Requires Authentication)
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

## Authentication

The API uses **JWT (JSON Web Token)** authentication for secure access control. All todo list and todo operations require authentication.

### Authentication Flow

1. **Register** a new user account
2. **Login** to receive an access token (15-minute expiration) and refresh token (7-day expiration)
3. Include the access token in the `Authorization` header for protected endpoints
4. **Refresh** the access token when it expires using the refresh token
5. **Logout** to revoke the refresh token when done

### Token Types

- **Access Token**: Short-lived JWT (15 minutes) used to authenticate API requests
- **Refresh Token**: Long-lived secure token (7 days) used to obtain new access tokens

### User Roles

- **user**: Default role, can manage their own todo lists and todos
- **admin**: Administrative role (reserved for future features)

## Usage Examples

### Authentication Examples

#### Register a New User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

**Response:**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "a1b2c3d4e5f6...",
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "firstName": "John",
    "lastName": "Doe",
    "role": "user",
    "isActive": true
  }
}
```

#### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'
```

**Response:** Same as registration response with access and refresh tokens.

#### Refresh Access Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "a1b2c3d4e5f6..."
  }'
```

**Response:** New access and refresh tokens.

#### Get User Profile

```bash
curl -X GET http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

#### Update Profile

```bash
curl -X PUT http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "firstName": "Jane",
    "lastName": "Smith"
  }'
```

#### Change Password

```bash
curl -X PUT http://localhost:8080/api/v1/auth/password \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "currentPassword": "SecurePassword123!",
    "newPassword": "NewSecurePassword456!"
  }'
```

#### Logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "a1b2c3d4e5f6..."
  }'
```

### Todo List Examples (with Authentication)

**Note:** All examples below require the `Authorization` header with a valid access token.

#### Create a Todo List

```bash
curl -X POST http://localhost:8080/api/v1/lists \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Work Tasks",
    "description": "Tasks for work projects"
  }'
```

#### Get All Lists

```bash
curl http://localhost:8080/api/v1/lists \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

#### Create a Todo

```bash
curl -X POST http://localhost:8080/api/v1/lists/{listId}/todos \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Complete project documentation",
    "priority": "high",
    "dueDate": "2025-11-15T23:59:59Z"
  }'
```

#### Get Todos with Filtering

```bash
# Get high priority todos
curl "http://localhost:8080/api/v1/lists/{listId}/todos?priority=high" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."

# Get incomplete todos sorted by due date
curl "http://localhost:8080/api/v1/lists/{listId}/todos?completed=false&sortBy=dueDate&sortOrder=asc" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

#### Update a Todo

```bash
curl -X PUT http://localhost:8080/api/v1/lists/{listId}/todos/{todoId} \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "completed": true
  }'
```

#### Delete a Todo

```bash
curl -X DELETE http://localhost:8080/api/v1/lists/{listId}/todos/{todoId} \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

## Project Structure

```
.
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 specification
│   └── examples.md           # API usage examples
├── cmd/
│   └── server/
│       └── main.go           # Application entry point with HTTP/HTTPS support
├── internal/
│   ├── auth/                 # Authentication package
│   │   ├── jwt.go            # JWT token generation and validation
│   │   ├── password.go       # Password hashing with bcrypt
│   │   └── service.go        # Authentication service (register, login, etc.)
│   ├── database/             # Database configuration
│   │   └── database.go       # PostgreSQL connection and migrations
│   ├── handlers/             # HTTP request handlers
│   │   ├── auth.go           # Authentication handlers
│   │   ├── lists.go          # List CRUD handlers
│   │   └── todos.go          # Todo CRUD handlers
│   ├── logging/              # Logging configuration
│   │   └── logging.go        # Logrus + lumberjack setup
│   ├── middleware/           # HTTP middleware
│   │   ├── auth.go           # JWT authentication middleware
│   │   ├── cors.go           # CORS middleware
│   │   ├── helpers.go        # Shared utility functions
│   │   ├── logging.go        # Request logging middleware
│   │   ├── ratelimit.go      # Rate limiting middleware
│   │   └── security.go       # Security middleware (XSS, size limits, etc.)
│   ├── models/               # Data models and DTOs
│   │   └── models.go         # GORM models with validation
│   ├── storage/              # Storage layer
│   │   ├── interface.go      # Storage interface
│   │   ├── storage.go        # In-memory implementation
│   │   └── postgres.go       # PostgreSQL/GORM implementation
│   └── tls/                  # TLS/HTTPS configuration
│       ├── tls.go            # TLS config and certificate handling
│       └── redirect.go       # HTTP to HTTPS redirect handler
├── scripts/
│   └── generate-certs.sh     # Self-signed certificate generator
├── Dockerfile                # Docker image definition
├── docker-compose.yml        # Docker Compose with PostgreSQL
├── .env.example              # Environment variables example
├── .gitignore                # Git ignore (includes certs/)
├── Makefile                  # Build and test targets
├── go.mod                    # Go module definition
├── SECURITY.md               # Security documentation
├── TESTING.md                # Testing documentation
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

**users table:**
- `id` (UUID, primary key)
- `email` (varchar(255), unique)
- `password_hash` (varchar(255))
- `first_name` (varchar(100))
- `last_name` (varchar(100))
- `role` (varchar(20): user/admin)
- `is_active` (boolean, default: true)
- `last_login_at` (timestamp, nullable)
- `created_at`, `updated_at`, `deleted_at` (timestamps)

**refresh_tokens table:**
- `id` (UUID, primary key)
- `user_id` (UUID, foreign key → users.id)
- `token_hash` (varchar(255), unique)
- `expires_at` (timestamp)
- `created_at` (timestamp)

**todo_lists table:**
- `id` (UUID, primary key)
- `user_id` (UUID, foreign key → users.id)
- `name` (varchar(100), unique per user)
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
- `USE_MEMORY_STORAGE`: Set to "true" to use in-memory storage instead of PostgreSQL (Note: In-memory mode does not support authentication)

### JWT Authentication Configuration
- `JWT_SECRET_KEY`: Secret key for signing JWT tokens (minimum 32 characters) - **CHANGE IN PRODUCTION**
- `JWT_ACCESS_TOKEN_MINUTES`: Access token expiration in minutes (default: 15)
- `JWT_REFRESH_TOKEN_DAYS`: Refresh token expiration in days (default: 7)
- `JWT_ISSUER`: JWT issuer identifier (default: todolist-api)

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

### TLS/HTTPS Configuration
- `TLS_ENABLED`: Enable HTTPS (default: false)
- `TLS_CERT_FILE`: Path to TLS certificate file (default: ./certs/server.crt)
- `TLS_KEY_FILE`: Path to TLS private key file (default: ./certs/server.key)
- `TLS_PORT`: HTTPS port (default: 8443, use 443 for production)
- `HTTP_PORT`: HTTP port when TLS enabled (default: 8080)
- `TLS_REDIRECT_HTTP`: Redirect HTTP to HTTPS (default: true)
- `TLS_MIN_VERSION`: Minimum TLS version - 1.0, 1.1, 1.2, 1.3 (default: 1.2)
- `TLS_MAX_VERSION`: Maximum TLS version (default: 1.3)
- `TLS_PREFER_SERVER_CIPHERS`: Prefer server cipher suites (default: true)

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
- Authentication: 95.8% (JWT, password hashing, auth service)
- Logging: 86.2%
- Middleware: 82.1% (includes security, CORS, rate limiting, logging, auth)
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

## HTTPS/TLS Support

The API includes built-in HTTPS/TLS support for secure communication in production environments.

### Features

- **TLS 1.2 and 1.3**: Modern, secure TLS versions (1.0 and 1.1 are deprecated)
- **Secure Cipher Suites**: Only strong, modern ciphers (AES-GCM, ChaCha20-Poly1305)
- **HTTP to HTTPS Redirect**: Automatically redirect HTTP requests to HTTPS
- **Flexible Configuration**: Environment-based configuration for different environments
- **Graceful Shutdown**: Proper handling of in-flight requests during shutdown
- **Certificate Validation**: Validates certificates on load

### Quick Start with HTTPS

#### 1. Generate Self-Signed Certificates (Development)

For development and testing, use the provided script to generate self-signed certificates:

```bash
# Generate certificates for localhost
./scripts/generate-certs.sh localhost

# Or for a specific domain
./scripts/generate-certs.sh example.com
```

This creates:
- `certs/server.key` - Private key (2048-bit RSA)
- `certs/server.crt` - Self-signed certificate (valid for 365 days)

**Note**: Self-signed certificates will show browser warnings. For production, use certificates from a trusted CA like Let's Encrypt.

#### 2. Enable HTTPS

Set the following in your `.env` file:

```bash
# Enable TLS
TLS_ENABLED=true

# Certificate paths
TLS_CERT_FILE=./certs/server.crt
TLS_KEY_FILE=./certs/server.key

# Ports
TLS_PORT=8443          # HTTPS port (use 443 for production)
HTTP_PORT=8080         # HTTP port (for redirect)

# Redirect HTTP to HTTPS
TLS_REDIRECT_HTTP=true
```

#### 3. Start the Server

```bash
# Build first
go build -o todolist-api ./cmd/server

# Run with HTTPS enabled
./todolist-api
```

You'll see:
```
INFO  Starting HTTPS server on port 8443
INFO  Starting HTTP redirect server on port 8080 -> HTTPS port 8443
```

#### 4. Test HTTPS Connection

```bash
# Using curl (accept self-signed cert)
curl -k https://localhost:8443/health

# Or specify the certificate
curl --cacert certs/server.crt https://localhost:8443/health

# HTTP will redirect to HTTPS
curl -L http://localhost:8080/health
```

### Production Setup

For production deployments, use proper certificates from a trusted Certificate Authority.

#### Option 1: Let's Encrypt (Recommended)

Use certbot or similar ACME client to obtain free certificates:

```bash
# Install certbot
sudo apt-get install certbot

# Obtain certificate
sudo certbot certonly --standalone -d yourdomain.com

# Certificates will be in /etc/letsencrypt/live/yourdomain.com/
```

Update `.env`:
```bash
TLS_ENABLED=true
TLS_CERT_FILE=/etc/letsencrypt/live/yourdomain.com/fullchain.pem
TLS_KEY_FILE=/etc/letsencrypt/live/yourdomain.com/privkey.pem
TLS_PORT=443
HTTP_PORT=80
TLS_REDIRECT_HTTP=true
```

#### Option 2: Commercial Certificate

If using a commercial CA (DigiCert, GlobalSign, etc.):

1. Generate a CSR (Certificate Signing Request):
```bash
openssl req -new -newkey rsa:2048 -nodes \
  -keyout server.key \
  -out server.csr \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=yourdomain.com"
```

2. Submit `server.csr` to your CA
3. Download the signed certificate
4. Update `.env` with certificate paths

#### Option 3: Reverse Proxy (Alternative)

For complex deployments, use a reverse proxy like nginx or Caddy to handle TLS:

**nginx example:**
```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

In this case, disable TLS in the app:
```bash
TLS_ENABLED=false
PORT=8080
```

### TLS Configuration Options

#### TLS Versions

**Recommended (Secure):**
```bash
TLS_MIN_VERSION=1.2  # TLS 1.2 minimum
TLS_MAX_VERSION=1.3  # TLS 1.3 maximum
```

**Legacy Support (Not Recommended):**
```bash
TLS_MIN_VERSION=1.0  # Allows TLS 1.0/1.1 (insecure)
```

#### Cipher Suites

The server uses only secure, modern cipher suites:

**TLS 1.3 Ciphers:**
- `TLS_AES_128_GCM_SHA256`
- `TLS_AES_256_GCM_SHA384`
- `TLS_CHACHA20_POLY1305_SHA256`

**TLS 1.2 Ciphers:**
- `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256`
- `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256`

Weak ciphers (RC4, DES, 3DES, MD5) are explicitly excluded.

### Testing HTTPS

#### Test TLS Configuration

```bash
# Check TLS version support
openssl s_client -connect localhost:8443 -tls1_2

# Check cipher suites
nmap --script ssl-enum-ciphers -p 8443 localhost

# Test certificate validity
openssl s_client -connect localhost:8443 -showcerts
```

#### Test HTTP Redirect

```bash
# HTTP should redirect to HTTPS (301 Moved Permanently)
curl -v http://localhost:8080/api/v1/lists

# Expected output:
# < HTTP/1.1 301 Moved Permanently
# < Location: https://localhost:8443/api/v1/lists
```

#### Test with API Requests

```bash
# Create a list via HTTPS
curl -k -X POST https://localhost:8443/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name": "Secure Tasks", "description": "Tasks over HTTPS"}'

# Get lists via HTTPS
curl -k https://localhost:8443/api/v1/lists
```

### Troubleshooting

**Certificate Not Found:**
```
Failed to create TLS config: open ./certs/server.crt: no such file or directory
```
→ Run `./scripts/generate-certs.sh` to create certificates

**Permission Denied on Port 443:**
```
Failed to start HTTPS server: listen tcp :443: bind: permission denied
```
→ Ports below 1024 require root/sudo, or use port 8443 for development

**Browser Shows Warning:**
```
Your connection is not private / NET::ERR_CERT_AUTHORITY_INVALID
```
→ This is expected with self-signed certificates. Click "Advanced" → "Proceed" for testing, or use proper CA certificates for production

**HTTP Requests Timing Out:**
```
curl: (7) Failed to connect to localhost port 8080
```
→ Check that `TLS_REDIRECT_HTTP=true` and the server started both listeners

### Security Best Practices

1. ✅ **Use TLS 1.2+ only** - Disable TLS 1.0/1.1 (known vulnerabilities)
2. ✅ **Use strong ciphers** - The default configuration only allows secure ciphers
3. ✅ **Use proper CA certificates in production** - Never use self-signed certs for public services
4. ✅ **Keep private keys secure** - Never commit `.key` files to version control (`.gitignore` already excludes them)
5. ✅ **Renew certificates before expiry** - Set up auto-renewal with Let's Encrypt
6. ✅ **Enable HTTP to HTTPS redirect** - Force all traffic through HTTPS
7. ✅ **Use HSTS headers** - Already included in security headers middleware
8. ✅ **Monitor certificate expiry** - Set up alerts for certificates expiring within 30 days

## Security

The API implements multiple layers of security protection. See [SECURITY.md](SECURITY.md) for complete security documentation.

### Implemented Security Features

✅ **JWT Authentication** - Secure token-based authentication with access and refresh tokens
✅ **Password Security** - bcrypt hashing with cost factor 12
✅ **User Data Isolation** - Database-level filtering ensures users only access their own data
✅ **Role-Based Access Control** - Support for user and admin roles
✅ **SQL Injection Protection** - GORM parameterized queries
✅ **XSS Prevention** - HTML escaping of all user input
✅ **DoS Protection** - Rate limiting (60 req/min per IP)
✅ **Request Size Limits** - Maximum 1MB request body
✅ **Security Headers** - X-Frame-Options, CSP, X-XSS-Protection, HSTS, etc.
✅ **CORS Protection** - Configurable origin whitelist
✅ **UUID Validation** - Format validation before database queries
✅ **Error Sanitization** - Generic errors to clients, detailed logs server-side
✅ **Memory Safety** - Go's built-in bounds checking and GC
✅ **HTTPS/TLS Support** - TLS 1.2/1.3 with secure cipher suites

### Security Configuration

**Production Settings:**
```bash
# JWT Authentication - CRITICAL: Change secret key!
JWT_SECRET_KEY=<your-secure-random-key-at-least-32-characters>
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_DAYS=7

# Enable HTTPS
TLS_ENABLED=true
TLS_PORT=443
TLS_REDIRECT_HTTP=true

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
JWT_SECRET_KEY=test-secret-key-32-characters!!
```

### Security Best Practices

1. **Change JWT secret key in production** - Generate a secure random key (at least 32 characters)
2. **Store secrets securely** - Use environment variables or a secrets manager, never commit secrets to version control
3. **Always use HTTPS in production** - Enable TLS or deploy behind nginx/load balancer with SSL
4. **Use proper certificates** - Get certificates from Let's Encrypt or commercial CA (never use self-signed in production)
5. **Configure CORS strictly** - Never use `*` wildcard in production
6. **Monitor rate limit logs** - Track suspicious IPs hitting limits
7. **Keep dependencies updated** - Regularly update Go modules
8. **Use strong database passwords** - Never use default credentials
9. **Protect private keys** - Never commit `.key` files to version control
10. **Rotate JWT secret keys periodically** - Implement key rotation for enhanced security

See [SECURITY.md](SECURITY.md) for detailed security information, testing procedures, and deployment checklist.

## Next Steps

- ✅ ~~Add database persistence (PostgreSQL/MongoDB)~~ - **COMPLETED**
- ✅ ~~Add unit and integration tests~~ - **COMPLETED**
- ✅ ~~Add rate limiting~~ - **COMPLETED**
- ✅ ~~Add request logging~~ - **COMPLETED**
- ✅ ~~Add security hardening (XSS, CORS, headers, size limits)~~ - **COMPLETED**
- ✅ ~~Add HTTPS/TLS support~~ - **COMPLETED**
- ✅ ~~Add JWT authentication and authorization~~ - **COMPLETED**
- Add metrics and monitoring (Prometheus)
- Add API documentation UI (Swagger/ReDoc)
- Add database connection pooling tuning
- Add health check with database connectivity status
- Add Let's Encrypt ACME support for automatic certificate management
- Add email verification for new user accounts
- Add password reset functionality
- Add multi-factor authentication (MFA/2FA)

## License

MIT
