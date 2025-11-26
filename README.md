# Todo List REST API

[![CI/CD Pipeline](https://github.com/pulsifer/TodoList/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/pulsifer/TodoList/actions/workflows/ci-cd.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

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
- **Health Checks**: Comprehensive health endpoints with database connectivity, migration status, and system metrics
- **API Documentation**: Interactive Swagger/OpenAPI documentation for all endpoints
- **Graceful Shutdown**: Clean shutdown with proper resource cleanup and in-flight request completion
- **CI/CD Pipeline**: Automated testing, building, security scanning, and deployment to Oracle Cloud Infrastructure

## Table of Contents

- [Features](#features)
- [API Specification](#api-specification)
- [Getting Started](#getting-started)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [Usage Examples](#usage-examples)
- [Project Structure](#project-structure)
- [Database](#database)
- [Configuration](#configuration)
- [Development](#development)
- [Rate Limiting](#rate-limiting)
- [Logging](#logging)
- [Health Checks](#health-checks)
- [HTTPS/TLS Support](#httpstls-support)
- [Security](#security)
- [Oracle Cloud Infrastructure (OCI) Deployment](#oracle-cloud-infrastructure-oci-deployment)
- [Next Steps](#next-steps)
- [License](#license)

## API Specification

The API follows the OpenAPI 3.0 specification defined in [api/openapi.yaml](api/openapi.yaml).

### Base URL

**Development:**
```
http://localhost:8080/api/v1   # HTTP
https://localhost:8443/api/v1  # HTTPS (with self-signed cert)
```

**Production:**
```
http://localhost:80/api/v1     # HTTP (redirects to HTTPS)
https://localhost:443/api/v1   # HTTPS (with proper certificate)
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

#### Health Checks (Public)
- `GET /health` - Basic health check (simple status response)
- `GET /health/detailed` - Detailed health check with database connectivity, migration status, and system metrics
- `GET /health/ready` - Kubernetes-style readiness probe (checks if app can handle requests)
- `GET /health/live` - Kubernetes-style liveness probe (checks if app is running)

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed on your system:

#### Required

| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | 1.25.3+ | Build and run the application |
| **Git** | 2.0+ | Clone the repository |

#### Optional (Choose based on your setup)

| Tool | Version | Purpose | When Required |
|------|---------|---------|---------------|
| **Docker** | 20.0+ | Run with containers | Docker deployment |
| **Docker Compose** | 2.0+ | Multi-container orchestration | Docker deployment |
| **PostgreSQL** | 14+ | Database server | Local development without Docker |
| **Make** | Any | Build automation | Optional convenience |
| **golangci-lint** | Latest | Code linting | Development/CI |

### Installation Instructions

Choose your operating system:

<details>
<summary><b>🪟 Windows</b></summary>

#### Install Go
1. Download installer from [go.dev/dl](https://go.dev/dl/)
2. Run the MSI installer
3. Verify installation:
   ```powershell
   go version
   ```

#### Install Git
1. Download from [git-scm.com](https://git-scm.com/download/win)
2. Run installer with default options
3. Verify installation:
   ```powershell
   git --version
   ```

#### Install Docker Desktop (Optional)
1. Download from [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/)
2. Run installer
3. Start Docker Desktop
4. Verify installation:
   ```powershell
   docker --version
   docker compose version
   ```

#### Install PostgreSQL (Optional - for local development)
1. Download from [postgresql.org/download/windows](https://www.postgresql.org/download/windows/)
2. Run installer
3. Remember the password you set for the `postgres` user
4. Verify installation:
   ```powershell
   psql --version
   ```

#### Install Make (Optional)
Using Chocolatey:
```powershell
choco install make
```
Or download from [gnuwin32.sourceforge.net](http://gnuwin32.sourceforge.net/packages/make.htm)

#### Install golangci-lint (Optional)
```powershell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

</details>

<details>
<summary><b>🍎 macOS</b></summary>

#### Install Homebrew (if not already installed)
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

#### Install Go
```bash
brew install go
go version  # Verify installation
```

#### Install Git
```bash
brew install git
git --version  # Verify installation
```

#### Install Docker Desktop (Optional)
**Option 1 - GUI Installer:**
1. Download from [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/)
2. Drag to Applications folder
3. Start Docker Desktop

**Option 2 - Homebrew:**
```bash
brew install --cask docker
```

Verify installation:
```bash
docker --version
docker compose version
```

#### Install PostgreSQL (Optional - for local development)
```bash
brew install postgresql@15
brew services start postgresql@15
psql --version  # Verify installation
```

#### Install Make (Optional)
Already included in macOS Xcode Command Line Tools:
```bash
xcode-select --install
make --version  # Verify installation
```

#### Install golangci-lint (Optional)
```bash
brew install golangci-lint
# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

</details>

<details>
<summary><b>🐧 Linux (Ubuntu/Debian)</b></summary>

#### Install Go
```bash
# Download and install
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin

# Reload shell configuration
source ~/.bashrc  # or source ~/.zshrc

# Verify installation
go version
```

#### Install Git
```bash
sudo apt-get update
sudo apt-get install git -y
git --version  # Verify installation
```

#### Install Docker (Optional)
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add your user to docker group (avoid using sudo)
sudo usermod -aG docker $USER
newgrp docker

# Verify installation
docker --version
docker compose version
```

#### Install PostgreSQL (Optional - for local development)
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib -y
sudo systemctl start postgresql
sudo systemctl enable postgresql
psql --version  # Verify installation
```

#### Install Make (Optional)
```bash
sudo apt-get install build-essential -y
make --version  # Verify installation
```

#### Install golangci-lint (Optional)
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

</details>

<details>
<summary><b>🐧 Linux (Fedora/RHEL/CentOS)</b></summary>

#### Install Go
```bash
# Download and install
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin

# Reload shell configuration
source ~/.bashrc

# Verify installation
go version
```

#### Install Git
```bash
sudo dnf install git -y
git --version  # Verify installation
```

#### Install Docker (Optional)
```bash
sudo dnf install docker docker-compose -y
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
newgrp docker
docker --version
```

#### Install PostgreSQL (Optional)
```bash
sudo dnf install postgresql postgresql-server -y
sudo postgresql-setup --initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
psql --version
```

#### Install Make (Optional)
```bash
sudo dnf groupinstall "Development Tools" -y
make --version
```

#### Install golangci-lint (Optional)
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

</details>

### Verify Your Installation

After installing the required tools, verify everything is set up correctly:

```bash
# Check Go installation
go version
# Expected: go version go1.25.3 or later

# Check Git installation
git --version
# Expected: git version 2.x.x or later

# Check Docker installation (if using Docker)
docker --version
docker compose version
# Expected: Docker version 20.x.x or later

# Check PostgreSQL installation (if using local PostgreSQL)
psql --version
# Expected: psql (PostgreSQL) 14.x or later

# Check Go environment
go env GOPATH
go env GOROOT
# Verify these paths are set correctly
```

### Clone the Repository

```bash
# Clone via HTTPS
git clone https://github.com/YOUR_USERNAME/TodoList.git
cd TodoList

# Or clone via SSH (if you have SSH keys set up)
git clone git@github.com:YOUR_USERNAME/TodoList.git
cd TodoList
```

### Install Go Dependencies

```bash
# Download all dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy up (optional)
go mod tidy
```

## Quick Start

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

#### Option 2: With In-Memory Storage (No Authentication)

For quick testing without PostgreSQL or authentication:

```bash
# Build first
go build -o todolist-api ./cmd/server

# Run in memory mode (no auth required)
USE_MEMORY_STORAGE=true ./todolist-api
```

**Important:** In-memory mode runs **without authentication** - all endpoints are public.
- ✅ Perfect for quick testing and demos
- ⚠️ Data lost on restart
- ❌ No user accounts or JWT tokens
- See [MEMORY_MODE.md](MEMORY_MODE.md) for complete documentation

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

- **Database Migrations**: Versioned schema management using golang-migrate (see [MIGRATIONS.md](MIGRATIONS.md))
- **Soft deletes**: Deleted records are marked as deleted (not physically removed)
- **Foreign keys**: Todos are linked to lists with cascade delete
- **Indexes**: Optimized queries with indexes on commonly searched fields
- **UUID primary keys**: Uses UUIDs for all entity IDs

### Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema versioning. Migrations ensure your database schema is tracked, versioned, and can be reliably applied across environments.

**Quick Start:**

```bash
# Apply all pending migrations
make migrate-up

# Check current version
make migrate-version

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_feature
```

For detailed migration documentation, see [MIGRATIONS.md](MIGRATIONS.md).

For cloud VM deployment guide, see [DEPLOYMENT.md](DEPLOYMENT.md).

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
- `USE_MEMORY_STORAGE`: Set to "true" to use in-memory storage **without authentication** instead of PostgreSQL (see [MEMORY_MODE.md](MEMORY_MODE.md) for details)

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

### Setting Up Your Development Environment

#### Install Development Tools

To match the CI/CD environment and ensure your code passes all checks before pushing:

**golangci-lint (v1.64.8 - matches CI/CD):**
```bash
# Download and install specific version
cd /tmp
curl -L https://github.com/golangci/golangci-lint/releases/download/v1.64.8/golangci-lint-1.64.8-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64.tar.gz -o golangci-lint.tar.gz
tar -xzf golangci-lint.tar.gz
chmod +x golangci-lint-1.64.8-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64/golangci-lint
mv golangci-lint-1.64.8-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64/golangci-lint $(go env GOPATH)/bin/
cd -

# Verify installation
golangci-lint --version
# Expected: golangci-lint has version 1.64.8
```

**goimports (for import formatting):**
```bash
go install golang.org/x/tools/cmd/goimports@latest

# Verify installation
goimports -h
```

**Add Go bin to PATH (if not already):**

Add to your `~/.zshrc` (macOS) or `~/.bashrc` (Linux):
```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload:
```bash
source ~/.zshrc  # or source ~/.bashrc
```

### Building

```bash
go build -o todolist-api ./cmd/server
```

### Code Quality Checks

Run these before committing to ensure your code passes CI/CD checks:

```bash
# Run linter (matches CI/CD environment)
golangci-lint run --timeout=5m

# Format code
gofmt -w .

# Fix imports
goimports -w .

# Run tests
go test ./... -v
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

## Health Checks

The API provides comprehensive health check endpoints for monitoring application and infrastructure health.

### Endpoints

#### Basic Health Check
```
GET /health
```

Simple health check that returns a basic status response. Use this for basic uptime monitoring.

**Response (200 OK):**
```json
{
  "status": "healthy"
}
```

#### Detailed Health Check
```
GET /health/detailed
```

Comprehensive health check that includes:
- Database connectivity and connection pool statistics
- Database migration status
- System metrics (goroutines, memory usage, garbage collection)
- Overall health status based on all components

**Response (200 OK when healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-25T22:00:00Z",
  "uptime": "2h 15m 30s",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection is healthy",
      "details": {
        "open_connections": 5,
        "in_use": 2,
        "idle": 3,
        "wait_count": 0,
        "wait_duration_ms": 0
      }
    },
    "migrations": {
      "status": "healthy",
      "message": "Migrations are up to date",
      "details": {
        "version": 1,
        "dirty": false
      }
    },
    "system": {
      "status": "info",
      "message": "System information",
      "details": {
        "goroutines": 12,
        "memory_alloc_mb": 8,
        "memory_sys_mb": 24,
        "num_gc": 3,
        "go_version": "go1.24.0"
      }
    }
  }
}
```

**Response (503 Service Unavailable when unhealthy):**
Returns the same structure but with `"status": "unhealthy"` and error details in component checks.

#### Readiness Probe
```
GET /health/ready
```

Kubernetes-style readiness probe that checks if the application can handle requests. Returns 200 if ready, 503 if not ready.

**Use case:** Load balancers and orchestrators use this to determine if traffic should be routed to this instance.

**Response (200 OK):**
```json
{
  "status": "ready"
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "not_ready",
  "reason": "database_unavailable",
  "message": "Database ping failed"
}
```

#### Liveness Probe
```
GET /health/live
```

Kubernetes-style liveness probe that checks if the application is running. Always returns 200 if the process is alive.

**Use case:** Orchestrators use this to determine if the application should be restarted.

**Response (200 OK):**
```json
{
  "status": "alive"
}
```

### Health Check Usage Examples

**Basic uptime monitoring:**
```bash
curl http://localhost:8080/health
```

**Comprehensive health status:**
```bash
curl http://localhost:8080/health/detailed
```

**Check if ready for traffic:**
```bash
curl http://localhost:8080/health/ready
```

**Check if process is alive:**
```bash
curl http://localhost:8080/health/live
```

### Kubernetes Integration

For Kubernetes deployments, configure readiness and liveness probes:

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
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

# For a specific domain
./scripts/generate-certs.sh example.com

# For an IP address (OCI/cloud deployments)
./scripts/generate-certs.sh 192.168.1.100

# With specific ownership (for systemd services running as non-root)
sudo ./scripts/generate-certs.sh 192.168.1.100 todolist
```

This creates:
- `certs/server.key` - Private key (2048-bit RSA, permissions 600)
- `certs/server.crt` - Self-signed certificate (valid for 365 days, permissions 644)

**Script features:**
- Auto-detects IP addresses vs domain names
- Creates proper Subject Alternative Names (SAN) for both IPs and domains
- Optionally sets ownership (second parameter) - useful for systemd services
- Sets secure file permissions automatically

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

## Oracle Cloud Infrastructure (OCI) Deployment

The project includes automated setup scripts for deploying the TodoList API on Oracle Cloud Infrastructure (OCI) using separate database and application VMs.

### Overview

The deployment uses two VMs:
- **Database VM**: PostgreSQL 15 server with optimized configuration
- **Application VM**: TodoList API with systemd service and automatic startup

Both VMs use **improved v2 scripts** that are:
- ✅ **Idempotent** - Safe to re-run multiple times without breaking existing setup
- ✅ **Resumable** - Can continue from where it left off after SSH disconnects
- ✅ **State-tracked** - Tracks completed steps in state files
- ✅ **Logged** - Detailed logging for troubleshooting

**Important for Oracle Cloud Free Tier (1GB RAM):**
- The `--auto-screen` flag is **NOT recommended** for free tier VMs due to memory constraints
- Screen installation may be killed by the OOM (Out of Memory) process
- Simply run the scripts without the flag: `sudo ./setup-database-vm-v2.sh`
- If SSH disconnects, re-run the script - it will automatically resume from where it left off
- Monitor progress in another SSH session: `sudo tail -f /var/log/todolist-db-setup.log`

### Prerequisites

1. **OCI Account** with appropriate permissions
2. **Two Compute Instances** (VM.Standard.E2.1.Micro eligible for Always Free tier)
   - Database VM: Oracle Linux 8 or Ubuntu 20.04+ (aarch64 or x86_64)
   - Application VM: Oracle Linux 8 or Ubuntu 20.04+ (aarch64 or x86_64)
   - Note: Scripts auto-detect architecture and OS
3. **VCN Configuration** (Security Lists):
   - **CRITICAL:** Security Lists must be configured or deployment will fail with timeouts
   - For Database VM subnet:
     - Ingress: TCP port 5432 from `10.0.0.0/24` (or your VCN CIDR)
   - For Application VM subnet:
     - Ingress: TCP port 8080 from `0.0.0.0/0` (or your IP for security)
     - Ingress: TCP ports 80, 443 from `0.0.0.0/0` (optional, for future HTTPS)
   - Go to: OCI Console → Networking → Virtual Cloud Networks → Your VCN → Security Lists
4. **SSH Access** to both VMs
5. **Your TodoList GitHub Repository URL** (for application setup)

### Phase 1: Database VM Setup

Upload and run the improved database setup script:

```bash
# On your local machine, upload the script to the database VM
scp scripts/oci/setup-database-vm-v2.sh opc@<database-vm-public-ip>:~

# SSH into the database VM
ssh opc@<database-vm-public-ip>

# Make executable and run (without --auto-screen for free tier VMs)
chmod +x setup-database-vm-v2.sh
sudo ./setup-database-vm-v2.sh
```

**What it does:**
- Detects OS (Oracle Linux or Ubuntu)
- Installs PostgreSQL 15
- Creates `todolist` database and `todolist` user with secure password
- Configures PostgreSQL for remote access from VCN
- Sets up firewall rules (firewalld or ufw)
- Saves credentials to `/root/db-credentials.txt`

**Output:**
The script will display connection details at the end:
```
Database Information:
  Database Name: todolist
  Database User: todolist
  Database Password: <your-password>

Connection Details:
  Host: 10.0.1.x (Private IP)
  Port: 5432

Connection String:
  postgresql://todolist:<password>@10.0.1.x:5432/todolist
```

**Save these credentials** - you'll need them for the application VM setup.

### Phase 2: Application VM Setup

Upload and run the improved application setup script:

```bash
# On your local machine, upload the script to the application VM
scp scripts/oci/setup-application-vm-v2.sh opc@<application-vm-public-ip>:~

# SSH into the application VM
ssh opc@<application-vm-public-ip>

# Make executable and run (without --auto-screen for free tier VMs)
chmod +x setup-application-vm-v2.sh
sudo ./setup-application-vm-v2.sh
```

**What it does:**
- Detects OS and architecture (Oracle Linux/Ubuntu, x86_64/aarch64)
- Installs Go 1.24.10 (matches project requirement)
- Installs Git and PostgreSQL client
- Clones your TodoList repository
- Prompts for database connection details (from Phase 1)
- Generates a secure JWT secret key
- Creates and configures `.env` file
- Builds the application binaries
- Tests database connectivity and authentication
- Creates systemd service with automatic restart
- Configures firewall rules (ports 8080, 80, 443)
- Configures SELinux (Oracle Linux)
- Starts the API service

**During setup**, you'll be prompted for:
- Git repository URL (your TodoList GitHub repository)
- Database host (private IP from Phase 1)
- Database port (default: 5432)
- Database name (default: todolist)
- Database user (default: todolist)
- Database password (from Phase 1)
- CORS allowed origins (use `*` for testing, specific domain for production)

**Note:** Database migrations run automatically when the application starts (GORM AutoMigrate), so you don't need to run them manually.

### Using the Auto-Screen Feature

The `--auto-screen` flag automatically runs the setup inside a `screen` session, which:
- Prevents SSH timeout from interrupting the installation
- Allows you to disconnect and reconnect without losing progress
- Logs all output to a file for review

**If SSH disconnects during setup:**
```bash
# Reconnect to the VM
ssh opc@<vm-public-ip>

# Re-attach to the screen session
sudo screen -r dbsetup    # For database VM
sudo screen -r appsetup   # For application VM

# If screen session ended, check the log
sudo tail -f /var/log/todolist-db-setup.log       # Database VM
sudo tail -f /var/log/todolist-app-setup.log      # Application VM
```

### Resuming After Interruption

Both v2 scripts are idempotent and track completed steps. If a script is interrupted:

```bash
# Simply re-run the script - it will skip completed steps
sudo ./setup-database-vm-v2.sh
# or
sudo ./setup-application-vm-v2.sh
```

The script will display:
```
Step 'install_postgresql' already complete, skipping...
Step 'initialize_postgresql' already complete, skipping...
Continuing from step 'configure_postgresql'...
```

### Checking Setup Progress

```bash
# View state file to see completed steps
sudo cat /var/lib/todolist-db-setup.state        # Database VM
sudo cat /var/lib/todolist-app-setup.state       # Application VM

# View detailed logs
sudo tail -f /var/log/todolist-db-setup.log      # Database VM
sudo tail -f /var/log/todolist-app-setup.log     # Application VM

# Check service status (Application VM)
sudo systemctl status todolist-api
```

### Resetting Setup (Start Fresh)

If you need to completely reset and start over:

```bash
# Database VM
sudo systemctl stop postgresql-15    # or postgresql
sudo rm -rf /var/lib/pgsql/15/data   # or /var/lib/postgresql/15/main
sudo rm /var/lib/todolist-db-setup.state
sudo rm /var/log/todolist-db-setup.log

# Application VM
sudo systemctl stop todolist-api
sudo rm -rf /opt/todolist
sudo rm /var/lib/todolist-app-setup.state
sudo rm /var/log/todolist-app-setup.log
sudo rm /etc/systemd/system/todolist-api.service
sudo systemctl daemon-reload
```

Then re-run the setup scripts.

### Updating the Application (Deploying New Changes)

After making code changes and pushing to your Git repository, use the automated update script to deploy changes to the application VM.

#### Quick Update

```bash
# SSH to application VM
ssh opc@<application-vm-ip>

# Run update script (requires sudo)
sudo /opt/todolist-api/scripts/oci/update-application.sh
```

The update script will:
1. ✅ Stop the service
2. ✅ Backup the current binary
3. ✅ Pull latest code from Git
4. ✅ Show commit changes (current → target)
5. ✅ Regenerate Swagger documentation
6. ✅ Rebuild the application
7. ✅ Configure SELinux (if enabled)
8. ✅ Start the service
9. ✅ Verify service is running
10. ✅ Automatically rollback if deployment fails

#### Update to Specific Branch

```bash
# Update to a different branch
sudo /opt/todolist-api/scripts/oci/update-application.sh develop
```

#### Manual Update Steps

If you prefer manual control or need to troubleshoot:

```bash
# 1. Stop the service
sudo systemctl stop todolist-api

# 2. Navigate to application directory
cd /opt/todolist-api

# 3. Pull latest changes
sudo -u todolist git pull origin main

# 4. Regenerate Swagger docs (important!)
sudo -u todolist /usr/local/go/bin/swag init -g cmd/server/main.go -o docs

# 5. Rebuild application
sudo -u todolist bash -c "export PATH=/usr/local/go/bin:\$PATH && go build -o todolist-api cmd/server/main.go"

# 6. Set SELinux context (if needed)
sudo chcon -t bin_t /opt/todolist-api/todolist-api

# 7. Start the service
sudo systemctl start todolist-api

# 8. Verify
sudo systemctl status todolist-api
sudo journalctl -u todolist-api -f
```

#### First-Time Setup of Update Script

The update script should be included when you first run the setup script. If it's missing:

```bash
# Ensure script is present
cd /opt/todolist-api
sudo -u todolist git pull origin main

# Make executable
sudo chmod +x /opt/todolist-api/scripts/oci/update-application.sh
```

#### Rollback on Failure

If an update fails, the script automatically attempts to restore the previous backup. You can also manually rollback:

```bash
# List available backups
ls -lh /opt/todolist-api/todolist-api.backup.*

# Restore a specific backup
sudo systemctl stop todolist-api
sudo -u todolist cp /opt/todolist-api/todolist-api.backup.20250126_143022 /opt/todolist-api/todolist-api
sudo chcon -t bin_t /opt/todolist-api/todolist-api
sudo systemctl start todolist-api
```

The update script keeps the 5 most recent backups automatically.

### Verifying Deployment

After successful setup, test the API:

```bash
# From the application VM
curl http://localhost:8080/health

# From your local machine (using public IP)
curl http://<application-vm-public-ip>:8080/health

# Expected response:
{"status":"ok"}

# Test registration and authentication
curl -X POST http://<application-vm-public-ip>:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"SecurePass123!"}'
```

### Troubleshooting

**Screen installation killed (Out of Memory):**
```
sudo dnf install -y screen
Killed
```
- **Solution:** Don't use `--auto-screen` on 1GB RAM free tier VMs
- Simply run the script without the flag: `sudo ./setup-database-vm-v2.sh`
- The script is idempotent - if SSH disconnects, just re-run it

**PostgreSQL GPG signature error:**
```
Error: Failed to download metadata for repo 'pgdg-common': repomd.xml GPG signature verification error: Bad GPG signature
```
- **Solution:** Script now uses `--nogpgcheck` to bypass this known issue
- Re-upload the latest script version if you see this error

**PostgreSQL connection timeout:**
```
Ncat: TIMEOUT when testing nc -z -v <db-host> 5432
```
- **Solution:** Add OCI Security List ingress rule for port 5432
  1. Go to OCI Console → Networking → Virtual Cloud Networks
  2. Select your VCN → Security Lists → Default Security List
  3. Add Ingress Rule: Source CIDR `10.0.0.0/24`, Protocol TCP, Port `5432`
- Also verify database VM firewall allows 5432:
  ```bash
  sudo firewall-cmd --permanent --add-port=5432/tcp
  sudo firewall-cmd --reload
  ```

**Database authentication failed but network OK:**
```
Network is reachable but database authentication failed
```
- **Solution:** Verify password matches on both VMs:
  ```bash
  # On database VM
  sudo cat /root/db-credentials.txt

  # On application VM
  sudo grep DB_PASSWORD /opt/todolist-api/.env
  ```
- Reset password if needed:
  ```bash
  # On database VM
  DB_PASSWORD=$(sudo grep "Database Password:" /root/db-credentials.txt | awk '{print $3}')
  sudo -u postgres psql -c "ALTER USER todolist WITH PASSWORD '$DB_PASSWORD';"
  sudo systemctl reload postgresql-15
  ```

**API not accessible from internet (timeout):**
```
curl: (28) Failed to connect to <public-ip> port 8080 after timeout
```
- **Solution:** Add OCI Security List ingress rule for port 8080
  1. Go to OCI Console → Networking → Virtual Cloud Networks
  2. Select your VCN → Security Lists → Security List for application subnet
  3. Add Ingress Rule: Source CIDR `0.0.0.0/0`, Protocol TCP, Port `8080`
- Verify application VM firewall (script does this automatically):
  ```bash
  sudo firewall-cmd --list-all | grep 8080
  ```

**SELinux permission denied:**
```
Failed at step EXEC spawning /opt/todolist-api/todolist-api: Permission denied
```
- **Solution:** Script now automatically sets SELinux context
- If you see this error, manually fix with:
  ```bash
  sudo chcon -t bin_t /opt/todolist-api/todolist-api
  sudo setenforce 1  # Re-enable SELinux
  sudo systemctl restart todolist-api
  ```

**Service fails to start:**
```
Job for todolist-api.service failed
```
- Check logs for specific error: `sudo journalctl -u todolist-api -n 50`
- Common causes and solutions:
  - **Database connection failed:** Check network and credentials (see above)
  - **GORM migration error:** Database may need to be reset (see below)
  - **Port already in use:** Check if another process is using port 8080

**GORM migration constraint error:**
```
ERROR: constraint "uni_users_email" of relation "users" does not exist
```
- **Solution:** Reset database and let GORM AutoMigrate create schema:
  ```bash
  # On database VM - drop and recreate database
  sudo -u postgres psql -c "DROP DATABASE todolist;"
  sudo -u postgres psql -c "CREATE DATABASE todolist OWNER todolist;"

  # On application VM - restart service (migrations run automatically)
  sudo systemctl restart todolist-api
  ```

**Go download fails (wrong architecture):**
```
Failed to download Go 1.24.10
```
- **Solution:** Script now auto-detects architecture (arm64 vs amd64)
- Verify with: `uname -m` (should show aarch64 or x86_64)
- Re-upload latest script if you see this error

### SSH Keep-Alive Configuration

To prevent SSH timeouts during manual operations, add to your local `~/.ssh/config`:

```
Host oci-db
    HostName <database-vm-public-ip>
    User opc
    ServerAliveInterval 60
    ServerAliveCountMax 10

Host oci-app
    HostName <application-vm-public-ip>
    User opc
    ServerAliveInterval 60
    ServerAliveCountMax 10
```

Then connect using:
```bash
ssh oci-db
ssh oci-app
```

### Security Recommendations for Production

1. **Database VM**:
   - Only allow port 5432 from application VM's private IP (not entire VCN)
   - Disable password authentication, use key-based auth only
   - Regular PostgreSQL updates and security patches
   - Enable PostgreSQL SSL/TLS connections

2. **Application VM**:
   - Enable HTTPS with proper certificates (Let's Encrypt)
   - Set strict CORS origins (not `*`)
   - Use strong JWT secret key
   - Enable rate limiting
   - Regular application and OS updates
   - Consider using Oracle Cloud Load Balancer with SSL termination

3. **Both VMs**:
   - Keep SSH keys secure
   - Disable root login
   - Enable automatic security updates
   - Set up monitoring and alerts
   - Regular backups (DB data and application state)

### Cost Optimization

Oracle Cloud offers Always Free tier resources:
- 2x AMD-based Compute instances (VM.Standard.E2.1.Micro)
- 200GB block storage
- 10GB Object Storage

This is sufficient for running the TodoList API with database on separate VMs at no cost.

## Next Steps

- ✅ ~~Add database persistence (PostgreSQL/MongoDB)~~ - **COMPLETED**
- ✅ ~~Add unit and integration tests~~ - **COMPLETED**
- ✅ ~~Add rate limiting~~ - **COMPLETED**
- ✅ ~~Add request logging~~ - **COMPLETED**
- ✅ ~~Add security hardening (XSS, CORS, headers, size limits)~~ - **COMPLETED**
- ✅ ~~Add HTTPS/TLS support~~ - **COMPLETED**
- ✅ ~~Add JWT authentication and authorization~~ - **COMPLETED**
- ✅ ~~Add health check with database connectivity status~~ - **COMPLETED**
- ✅ ~~Add API documentation UI (Swagger/ReDoc)~~ - **COMPLETED**
- Add metrics and monitoring (Prometheus)
- Add database connection pooling tuning
- Add Let's Encrypt ACME support for automatic certificate management
- Add email verification for new user accounts
- Add password reset functionality
- Add multi-factor authentication (MFA/2FA)

## License

MIT
