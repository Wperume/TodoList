# In-Memory Mode (No Authentication)

The Todo List API can run in **in-memory mode** without authentication for quick testing and development.

## Overview

When running with `USE_MEMORY_STORAGE=true`, the API operates without requiring authentication:

- ✅ All CRUD operations for lists and todos work
- ✅ No database required - everything stored in RAM
- ✅ Perfect for quick testing, demos, and development
- ⚠️  No authentication - all endpoints are public
- ⚠️  No user isolation - all data is shared
- ⚠️  Data is lost when the API restarts
- ❌ No user registration/login endpoints
- ❌ No JWT tokens or password management

## Quick Start

### 1. Stop Docker (if running)

```bash
docker-compose down
```

### 2. Build the Application

```bash
go build -o todolist-api ./cmd/server
```

### 3. Start in Memory Mode

```bash
USE_MEMORY_STORAGE=true ./todolist-api
```

The API will start on `http://localhost:8080` without authentication.

### 4. Run Tests

Use the in-memory test script:

```bash
bash test-api-memory.sh
```

Or test manually with curl (no Authorization header needed):

```bash
# Create a list
curl -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My List",
    "description": "Test list"
  }'

# Get all lists
curl http://localhost:8080/api/v1/lists

# Create a todo
curl -X POST http://localhost:8080/api/v1/lists/{listId}/todos \
  -H "Content-Type: application/json" \
  -d '{
    "description": "My first todo",
    "priority": "high"
  }'
```

## Available Endpoints (No Auth Required)

All endpoints work without the `Authorization` header:

### Todo Lists
- `GET /api/v1/lists` - Get all todo lists
- `POST /api/v1/lists` - Create a new todo list
- `GET /api/v1/lists/{listId}` - Get a specific list
- `PUT /api/v1/lists/{listId}` - Update a list
- `DELETE /api/v1/lists/{listId}` - Delete a list

### Todos
- `GET /api/v1/lists/{listId}/todos` - Get all todos
- `POST /api/v1/lists/{listId}/todos` - Create a todo
- `GET /api/v1/lists/{listId}/todos/{todoId}` - Get a todo
- `PUT /api/v1/lists/{listId}/todos/{todoId}` - Update a todo
- `DELETE /api/v1/lists/{listId}/todos/{todoId}` - Delete a todo

### Health Check
- `GET /health` - Health check endpoint

## Comparison: In-Memory vs PostgreSQL Mode

| Feature | In-Memory Mode | PostgreSQL Mode |
|---------|----------------|-----------------|
| **Authentication** | ❌ Disabled | ✅ JWT Required |
| **Data Persistence** | ❌ Lost on restart | ✅ Persists in database |
| **User Isolation** | ❌ All data shared | ✅ Per-user data |
| **Setup Required** | ✅ None | ⚠️  PostgreSQL + Docker |
| **User Management** | ❌ No users | ✅ Full user accounts |
| **Use Case** | Quick testing, demos | Production, development |

## Switching Between Modes

### Start In-Memory Mode (No Auth)

```bash
# Stop any running instances
pkill todolist-api
docker-compose down

# Start in-memory
USE_MEMORY_STORAGE=true ./todolist-api
```

### Start PostgreSQL Mode (With Auth)

```bash
# Stop any running instances
pkill todolist-api

# Start with Docker
docker-compose up -d

# Or start locally with PostgreSQL
./todolist-api  # Requires PostgreSQL connection
```

## Testing

### Automated Testing

**In-Memory Mode:**
```bash
bash test-api-memory.sh
```

**PostgreSQL Mode:**
```bash
bash test-api.sh
```

### Manual Testing Examples

#### Create and Manage Lists (No Auth)

```bash
# Create a list
LIST_ID=$(curl -s -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Work Tasks",
    "description": "My work todo list"
  }' | jq -r '.id')

echo "Created list: $LIST_ID"

# Get all lists
curl -s http://localhost:8080/api/v1/lists | jq '.'

# Create a todo
TODO_ID=$(curl -s -X POST "http://localhost:8080/api/v1/lists/$LIST_ID/todos" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Complete project documentation",
    "priority": "high",
    "dueDate": "2025-12-31T23:59:59Z"
  }' | jq -r '.id')

echo "Created todo: $TODO_ID"

# Get all todos
curl -s "http://localhost:8080/api/v1/lists/$LIST_ID/todos" | jq '.'

# Mark todo as completed
curl -s -X PUT "http://localhost:8080/api/v1/lists/$LIST_ID/todos/$TODO_ID" \
  -H "Content-Type: application/json" \
  -d '{"completed": true}' | jq '.'

# Filter completed todos
curl -s "http://localhost:8080/api/v1/lists/$LIST_ID/todos?completed=true" | jq '.'

# Delete todo
curl -X DELETE "http://localhost:8080/api/v1/lists/$LIST_ID/todos/$TODO_ID"

# Delete list
curl -X DELETE "http://localhost:8080/api/v1/lists/$LIST_ID"
```

## Technical Details

### How It Works

When `USE_MEMORY_STORAGE=true` is set:

1. **No Database Connection**: The API skips PostgreSQL initialization
2. **In-Memory Storage**: Uses Go maps/slices to store data in RAM
3. **No Auth Handler**: Authentication endpoints are not registered
4. **Default User ID**: All operations use a default UUID (`00000000-0000-0000-0000-000000000000`)
5. **No Middleware Auth**: Authentication middleware is not applied to routes

### Code Changes

The key modifications that enable no-auth mode:

**Middleware Helper** ([internal/middleware/auth.go](internal/middleware/auth.go:215-224)):
```go
func GetUserIDOrDefault(c *gin.Context) uuid.UUID {
    userID, err := GetUserID(c)
    if err != nil {
        // Return default user ID for unauthenticated access
        return uuid.MustParse("00000000-0000-0000-0000-000000000000")
    }
    return userID
}
```

**Handler Updates**: All handlers use `GetUserIDOrDefault` instead of `GetUserID`

**Main Server** ([cmd/server/main.go](cmd/server/main.go:41-46)):
```go
if useInMemory {
    logging.Logger.Info("Using in-memory storage")
    logging.Logger.Warn("In-memory storage does not support authentication - API will run without auth")
    // Skip auth initialization, use in-memory storage
}
```

## Limitations

### Security
- **No authentication**: Anyone can access all endpoints
- **No authorization**: No user-based access control
- **Shared data**: All users see the same data
- ⚠️  **DO NOT use in production or expose to the internet**

### Functionality
- No user accounts or profiles
- No password management
- No token refresh/logout
- No user-specific data isolation

### Data
- All data lost on restart
- No database backups
- No transaction support
- No data persistence

## Use Cases

### ✅ Good For
- Quick local testing
- API exploration and learning
- Integration testing without database
- Demos and presentations
- Development without Docker/PostgreSQL
- CI/CD testing (fast, no dependencies)

### ❌ Not Good For
- Production deployments
- Multi-user applications
- Data that needs persistence
- Security-sensitive applications
- Internet-facing APIs

## Environment Variables

In-memory mode respects these environment variables:

```bash
# Storage
USE_MEMORY_STORAGE=true           # Enable in-memory mode

# Logging
LOG_FILE_ENABLED=false            # Disable file logging (recommended)
LOG_LEVEL=info                    # Log level (debug, info, warn, error)

# Rate Limiting
RATE_LIMIT_ENABLED=true           # Still enabled in memory mode
RATE_LIMIT_REQUESTS_PER_MIN=60    # Requests per minute per IP

# CORS
CORS_ALLOWED_ORIGINS=*            # Allow all origins (dev only)

# Server
HTTP_PORT=8080                    # HTTP port (default)
```

## FAQ

**Q: Can I use in-memory mode with authentication?**
A: No, in-memory mode is designed to run without authentication for simplicity.

**Q: Is data shared between different clients?**
A: Yes, all data is shared since there's no user isolation.

**Q: Can I save data between restarts?**
A: No, all data is lost when the API stops. Use PostgreSQL mode for persistence.

**Q: Is it safe to use in production?**
A: No, in-memory mode has no authentication and should only be used for local testing.

**Q: Can I switch from in-memory to PostgreSQL without losing data?**
A: No, data is not compatible. In-memory data is lost when switching modes.

**Q: Does rate limiting still work?**
A: Yes, rate limiting, security headers, and other middleware still apply.

## See Also

- [README.md](README.md) - Main documentation
- [test-api-memory.sh](test-api-memory.sh) - Automated test script for in-memory mode
- [test-api.sh](test-api.sh) - Automated test script for PostgreSQL mode with auth
