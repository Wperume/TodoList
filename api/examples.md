# API Examples

This document provides example requests and responses for the Todo List API.

## Base URL

```
http://localhost:8080/api/v1
```

## Todo Lists

### Create a Todo List

**Request:**
```bash
POST /api/v1/lists
Content-Type: application/json

{
  "name": "Work Tasks",
  "description": "Tasks for work projects"
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Work Tasks",
  "description": "Tasks for work projects",
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T10:00:00Z",
  "todoCount": 0
}
```

### Get All Lists

**Request:**
```bash
GET /api/v1/lists?page=1&limit=20
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Work Tasks",
      "description": "Tasks for work projects",
      "createdAt": "2025-11-09T10:00:00Z",
      "updatedAt": "2025-11-09T10:00:00Z",
      "todoCount": 3
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "totalPages": 1,
    "totalItems": 1
  }
}
```

### Get a Specific List

**Request:**
```bash
GET /api/v1/lists/{listId}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Work Tasks",
  "description": "Tasks for work projects",
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T10:00:00Z",
  "todoCount": 3
}
```

### Update a List

**Request:**
```bash
PUT /api/v1/lists/{listId}
Content-Type: application/json

{
  "name": "Updated Work Tasks",
  "description": "Updated description"
}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Updated Work Tasks",
  "description": "Updated description",
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T15:30:00Z",
  "todoCount": 3
}
```

### Delete a List

**Request:**
```bash
DELETE /api/v1/lists/{listId}
```

**Response (204 No Content)**

## Todos

### Create a Todo

**Request:**
```bash
POST /api/v1/lists/{listId}/todos
Content-Type: application/json

{
  "description": "Complete project documentation",
  "priority": "high",
  "dueDate": "2025-11-15T23:59:59Z"
}
```

**Response (201 Created):**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "listId": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Complete project documentation",
  "priority": "high",
  "dueDate": "2025-11-15T23:59:59Z",
  "completed": false,
  "completedAt": null,
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T10:00:00Z"
}
```

### Get All Todos in a List

**Request:**
```bash
GET /api/v1/lists/{listId}/todos
```

**Response (200 OK):**
```json
[
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "listId": "550e8400-e29b-41d4-a716-446655440000",
    "description": "Complete project documentation",
    "priority": "high",
    "dueDate": "2025-11-15T23:59:59Z",
    "completed": false,
    "completedAt": null,
    "createdAt": "2025-11-09T10:00:00Z",
    "updatedAt": "2025-11-09T10:00:00Z"
  }
]
```

### Filter and Sort Todos

**Get high priority todos:**
```bash
GET /api/v1/lists/{listId}/todos?priority=high
```

**Get incomplete todos:**
```bash
GET /api/v1/lists/{listId}/todos?completed=false
```

**Get todos sorted by due date:**
```bash
GET /api/v1/lists/{listId}/todos?sortBy=dueDate&sortOrder=asc
```

**Combined filters:**
```bash
GET /api/v1/lists/{listId}/todos?priority=high&completed=false&sortBy=dueDate&sortOrder=asc
```

### Get a Specific Todo

**Request:**
```bash
GET /api/v1/lists/{listId}/todos/{todoId}
```

**Response (200 OK):**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "listId": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Complete project documentation",
  "priority": "high",
  "dueDate": "2025-11-15T23:59:59Z",
  "completed": false,
  "completedAt": null,
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T10:00:00Z"
}
```

### Update a Todo

**Mark as completed:**
```bash
PUT /api/v1/lists/{listId}/todos/{todoId}
Content-Type: application/json

{
  "completed": true
}
```

**Update multiple fields:**
```bash
PUT /api/v1/lists/{listId}/todos/{todoId}
Content-Type: application/json

{
  "description": "Updated task description",
  "priority": "medium",
  "dueDate": "2025-11-20T23:59:59Z",
  "completed": true
}
```

**Response (200 OK):**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "listId": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Updated task description",
  "priority": "medium",
  "dueDate": "2025-11-20T23:59:59Z",
  "completed": true,
  "completedAt": "2025-11-09T12:30:00Z",
  "createdAt": "2025-11-09T10:00:00Z",
  "updatedAt": "2025-11-09T12:30:00Z"
}
```

### Delete a Todo

**Request:**
```bash
DELETE /api/v1/lists/{listId}/todos/{todoId}
```

**Response (204 No Content)**

## Error Responses

### Resource Not Found (404)

```json
{
  "code": "LIST_NOT_FOUND",
  "message": "The requested todo list was not found"
}
```

### Invalid Input (400)

```json
{
  "code": "INVALID_INPUT",
  "message": "Invalid request body",
  "details": {
    "error": "Key: 'CreateTodoRequest.Priority' Error:Field validation for 'Priority' failed on the 'required' tag"
  }
}
```

### Conflict (409)

```json
{
  "code": "LIST_NAME_EXISTS",
  "message": "A list with this name already exists"
}
```

### Internal Server Error (500)

```json
{
  "code": "INTERNAL_ERROR",
  "message": "Failed to create list"
}
```

## Health Check

**Request:**
```bash
GET /health
```

**Response (200 OK):**
```json
{
  "status": "healthy"
}
```
