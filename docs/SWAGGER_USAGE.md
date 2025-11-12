# Swagger API Documentation Usage

## Overview

The TodoList API includes Swagger/OpenAPI documentation for interactive API exploration. **Swagger UI is protected and requires authentication** when running with PostgreSQL (database mode).

## Security Model

### Database Mode (PostgreSQL) - **SECURED** 🔒
- ✅ **Authentication Required**: Must provide valid JWT token
- ✅ **User must be logged in**: Register and login first
- ✅ **Rate Limited**: Subject to API rate limits
- ✅ **Audit Logged**: All access is logged

### In-Memory Mode - **OPEN** ⚠️
- ⚠️ **No Authentication**: Anyone can access
- ⚠️ **Development Only**: Not recommended for production

## Accessing Swagger UI

### Step 1: Start the Server

```bash
# With PostgreSQL (authenticated mode)
go run cmd/server/main.go

# Or with Docker Compose
docker-compose up
```

### Step 2: Register a User (First Time Only)

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your.email@example.com",
    "password": "YourSecurePassword",
    "first_name": "Your",
    "last_name": "Name"
  }'
```

### Step 3: Login to Get JWT Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your.email@example.com",
    "password": "YourSecurePassword"
  }'
```

**Response:**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "...",
  "tokenType": "Bearer",
  "expiresIn": 900,
  "user": {
    "id": "...",
    "email": "your.email@example.com",
    "role": "user"
  }
}
```

**Save the `accessToken` - you'll need it!**

### Step 4: Access Swagger UI with Authentication

#### Option A: Using Browser with ModHeader Extension

1. Install ModHeader browser extension
2. Add header: `Authorization: Bearer YOUR_ACCESS_TOKEN`
3. Navigate to: `http://localhost:8080/swagger/index.html`

#### Option B: Using curl

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/swagger/index.html
```

#### Option C: Using Swagger UI's Built-in Authentication

1. Try accessing: `http://localhost:8080/swagger/index.html`
2. You'll get 401 Unauthorized (expected)
3. **Current Limitation**: Swagger UI doesn't have a built-in login form

**Recommended**: Use Option A (browser extension) for interactive exploration.

## Using the API from Swagger UI

Once authenticated and Swagger UI loads:

1. **Explore Endpoints**: Browse all available API endpoints
2. **View Schemas**: See request/response data structures
3. **Try It Out**: Execute API calls directly from the browser
4. **Authentication**: Your JWT is already in the header (via browser extension)

### Example: Creating a Todo List

1. Find `POST /api/v1/lists` endpoint
2. Click "Try it out"
3. Enter request body:
```json
{
  "name": "My Todo List",
  "description": "Things to do today"
}
```
4. Click "Execute"
5. See the response!

## Token Expiration

Access tokens expire after 15 minutes (default). When you see 401 errors:

### Option 1: Refresh Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

### Option 2: Login Again

Re-run the login command to get a new token.

## Accessing Swagger JSON/YAML

The OpenAPI spec files are also protected:

```bash
# Get JSON spec (requires auth)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/swagger/doc.json

# Get YAML spec (requires auth)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/swagger/swagger.yaml
```

## Security Features

### What's Protected

✅ **Rate Limited**: 60 requests/minute per IP
✅ **Request Size Limits**: Prevents buffer overflow attacks
✅ **Security Headers**: XSS, clickjacking protection
✅ **JWT Validation**: Only valid tokens accepted
✅ **Audit Logging**: All access logged
✅ **CORS Protection**: Configurable origin restrictions

### What's Validated

When you access Swagger:
1. **Token Presence**: Must include `Authorization: Bearer <token>` header
2. **Token Validity**: Token must be properly signed
3. **Token Expiration**: Token must not be expired
4. **User Existence**: User in token must exist in database

## Troubleshooting

### 401 Unauthorized

**Problem**: Cannot access Swagger UI

**Solutions:**
1. ✅ Make sure you're logged in and have a valid token
2. ✅ Check token hasn't expired (15-minute default)
3. ✅ Verify `Authorization: Bearer <token>` header format
4. ✅ Ensure token includes full JWT (not truncated)

### Token Expired

**Problem**: Token worked before, now getting 401

**Solution:** Login again or use refresh token to get new access token.

### Cannot Login

**Problem**: Login endpoint returns 401

**Solutions:**
1. ✅ Verify email/password are correct
2. ✅ Check user is registered
3. ✅ Review server logs for errors

### In-Memory Mode

**Problem**: Want to test Swagger without database

**Solution:**
```bash
USE_MEMORY_STORAGE=true ./todolist-api
```

Then access: `http://localhost:8080/swagger/index.html` (no auth required)

## Development vs Production

### Development
- Swagger enabled by default
- Use in-memory mode for quick testing (no auth)
- Access token expiration: 15 minutes

### Production
- Swagger requires authentication (database mode)
- **Recommendation**: Consider disabling Swagger in production
- Or: Restrict to admin role only (future enhancement)

## Alternative: Use Postman or Insomnia

Instead of Swagger UI, you can import the OpenAPI spec into:

1. **Postman**:
   - Import → Link → `http://localhost:8080/swagger/doc.json`
   - Add Bearer token in Authorization tab

2. **Insomnia**:
   - Import → From URL → `http://localhost:8080/swagger/doc.json`
   - Set Bearer token in Auth settings

These tools handle authentication better than browser-based Swagger UI.

## Example Workflow

Complete example from start to finish:

```bash
# 1. Start server
docker-compose up -d

# 2. Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"api@test.com","password":"Test1234","first_name":"API","last_name":"User"}'

# 3. Login and save token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"api@test.com","password":"Test1234"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['accessToken'])")

# 4. Access Swagger JSON
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/swagger/doc.json | jq .

# 5. For browser access, use ModHeader with token:
echo "Add this to ModHeader:"
echo "Authorization: Bearer $TOKEN"
```

## Additional Resources

- **OpenAPI Specification**: `/swagger/doc.json`
- **Swagger YAML**: `/swagger/swagger.yaml`
- **API Endpoints**: All documented in Swagger UI
- **Authentication Guide**: See `AUTHENTICATION.md`

---

**Last Updated**: 2025-11-12
**API Version**: 1.0
**Swagger Version**: 2.0
