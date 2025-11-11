#!/bin/bash

# Todo List API Test Script - In-Memory Mode (No Authentication)
# This script tests the basic CRUD operations without authentication

set -e  # Exit on error

API_URL="http://localhost:8080/api/v1"

echo "🧪 Todo List API Testing Script (In-Memory Mode)"
echo "================================================"
echo "⚠️  Note: In-memory mode runs WITHOUT authentication"
echo "⚠️  No user management, no JWT tokens, data not persisted"
echo ""

# Check if API is running
echo "1️⃣  Checking if API is running..."
if ! curl -s "http://localhost:8080/health" | grep -q "healthy"; then
    echo "❌ API is not running"
    echo "Please start the API first with: USE_MEMORY_STORAGE=true ./todolist-api"
    exit 1
fi
echo "✅ API is healthy"
echo ""

# Create a todo list
echo "2️⃣  Creating a todo list..."
LIST_RESPONSE=$(curl -s -X POST "$API_URL/lists" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test List",
    "description": "Automated test list"
  }')

LIST_ID=$(echo $LIST_RESPONSE | jq -r '.id')

if [ "$LIST_ID" == "null" ] || [ -z "$LIST_ID" ]; then
    echo "❌ Failed to create list"
    echo "Response: $LIST_RESPONSE"
    exit 1
fi
echo "✅ List created"
echo "   List ID: $LIST_ID"
echo ""

# Get all lists
echo "3️⃣  Retrieving all lists..."
LISTS=$(curl -s "$API_URL/lists")
LIST_COUNT=$(echo $LISTS | jq '.data | length')
echo "✅ Retrieved lists: $LIST_COUNT list(s)"
echo ""

# Create todos
echo "4️⃣  Creating todos..."
TODO1=$(curl -s -X POST "$API_URL/lists/$LIST_ID/todos" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Test todo 1 - High priority",
    "priority": "high",
    "dueDate": "2025-12-31T23:59:59Z"
  }')

TODO1_ID=$(echo $TODO1 | jq -r '.id')

TODO2=$(curl -s -X POST "$API_URL/lists/$LIST_ID/todos" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Test todo 2 - Medium priority",
    "priority": "medium"
  }')

TODO2_ID=$(echo $TODO2 | jq -r '.id')

echo "✅ Created 2 todos"
echo "   Todo 1 ID: $TODO1_ID"
echo "   Todo 2 ID: $TODO2_ID"
echo ""

# Get all todos
echo "5️⃣  Retrieving todos..."
TODOS=$(curl -s "$API_URL/lists/$LIST_ID/todos")
TODO_COUNT=$(echo $TODOS | jq '. | length')
echo "✅ Retrieved todos: $TODO_COUNT items"
echo ""

# Update a todo
echo "6️⃣  Marking todo as completed..."
UPDATE_RESPONSE=$(curl -s -X PUT "$API_URL/lists/$LIST_ID/todos/$TODO1_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "completed": true
  }')

COMPLETED=$(echo $UPDATE_RESPONSE | jq -r '.completed')
if [ "$COMPLETED" == "true" ]; then
    echo "✅ Todo marked as completed"
else
    echo "❌ Failed to update todo"
    exit 1
fi
echo ""

# Filter completed todos
echo "7️⃣  Filtering completed todos..."
COMPLETED_TODOS=$(curl -s "$API_URL/lists/$LIST_ID/todos?completed=true")
COMPLETED_COUNT=$(echo $COMPLETED_TODOS | jq '. | length')
echo "✅ Found $COMPLETED_COUNT completed todo(s)"
echo ""

# Filter by priority
echo "8️⃣  Filtering high priority todos..."
HIGH_PRIORITY=$(curl -s "$API_URL/lists/$LIST_ID/todos?priority=high")
HIGH_COUNT=$(echo $HIGH_PRIORITY | jq '. | length')
echo "✅ Found $HIGH_COUNT high priority todo(s)"
echo ""

# Update list
echo "9️⃣  Updating list name..."
UPDATE_LIST=$(curl -s -X PUT "$API_URL/lists/$LIST_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Test List",
    "description": "Updated description"
  }')

UPDATED_NAME=$(echo $UPDATE_LIST | jq -r '.name')
echo "✅ List updated: $UPDATED_NAME"
echo ""

# Test pagination
echo "🔟 Testing pagination..."
PAGINATED=$(curl -s "$API_URL/lists?page=1&limit=10")
PAGINATION=$(echo $PAGINATED | jq -r '.pagination')
echo "✅ Pagination working"
echo "   $PAGINATION"
echo ""

# Cleanup - Delete todo
echo "🧹 Cleaning up..."
curl -s -X DELETE "$API_URL/lists/$LIST_ID/todos/$TODO1_ID" > /dev/null
echo "✅ Deleted todo 1"

curl -s -X DELETE "$API_URL/lists/$LIST_ID/todos/$TODO2_ID" > /dev/null
echo "✅ Deleted todo 2"

# Delete the list
curl -s -X DELETE "$API_URL/lists/$LIST_ID" > /dev/null
echo "✅ Test list deleted"
echo ""

echo "================================================"
echo "✅ All tests passed successfully!"
echo "================================================"
echo ""
echo "💡 What was tested (WITHOUT authentication):"
echo "   ✅ List creation, retrieval, update, deletion"
echo "   ✅ Todo creation, retrieval, update, deletion"
echo "   ✅ Filtering by completion status"
echo "   ✅ Filtering by priority"
echo "   ✅ Pagination"
echo "   ✅ Health check"
echo ""
echo "❌ What is NOT available in memory mode:"
echo "   ❌ User registration/login"
echo "   ❌ JWT authentication"
echo "   ❌ User isolation (all data is shared)"
echo "   ❌ Data persistence (lost on restart)"
echo "   ❌ Password management"
echo ""
