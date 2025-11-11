#!/bin/bash

# Todo List API Test Script
# This script tests the complete authentication flow

set -e  # Exit on error

API_URL="http://localhost:8080/api/v1"
EMAIL="test-$(date +%s)@example.com"
PASSWORD="SecureTestPassword123!"

echo "🧪 Todo List API Testing Script"
echo "================================"
echo ""

# Check if API is running
echo "1️⃣  Checking if API is running..."
if ! curl -s "http://localhost:8080/health" | grep -q "healthy"; then
    echo "❌ API is not running at $API_URL"
    echo "Please start the API first with: docker-compose up -d"
    exit 1
fi
echo "✅ API is healthy"
echo ""

# Register a new user
echo "2️⃣  Registering new user: $EMAIL"
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"firstName\": \"Test\",
    \"lastName\": \"User\"
  }")

ACCESS_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.refreshToken')

if [ "$ACCESS_TOKEN" == "null" ] || [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Registration failed"
    echo "Response: $REGISTER_RESPONSE"
    exit 1
fi
echo "✅ User registered successfully"
echo "   User ID: $(echo $REGISTER_RESPONSE | jq -r '.user.id')"
echo ""

# Get user profile
echo "3️⃣  Getting user profile..."
PROFILE=$(curl -s "$API_URL/auth/profile" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo $PROFILE | jq -e '.email' > /dev/null 2>&1; then
    echo "✅ Profile retrieved"
    echo "   Email: $(echo $PROFILE | jq -r '.email')"
    echo "   Name: $(echo $PROFILE | jq -r '.firstName') $(echo $PROFILE | jq -r '.lastName')"
else
    echo "❌ Failed to get profile"
    exit 1
fi
echo ""

# Create a todo list
echo "4️⃣  Creating a todo list..."
LIST_RESPONSE=$(curl -s -X POST "$API_URL/lists" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
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

# Create todos
echo "5️⃣  Creating todos..."
TODO1=$(curl -s -X POST "$API_URL/lists/$LIST_ID/todos" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Test todo 1 - High priority",
    "priority": "high",
    "dueDate": "2025-12-31T23:59:59Z"
  }')

TODO1_ID=$(echo $TODO1 | jq -r '.id')

TODO2=$(curl -s -X POST "$API_URL/lists/$LIST_ID/todos" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
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
echo "6️⃣  Retrieving todos..."
TODOS=$(curl -s "$API_URL/lists/$LIST_ID/todos" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

TODO_COUNT=$(echo $TODOS | jq '. | length')
echo "✅ Retrieved todos: $TODO_COUNT items"
echo ""

# Update a todo
echo "7️⃣  Marking todo as completed..."
UPDATE_RESPONSE=$(curl -s -X PUT "$API_URL/lists/$LIST_ID/todos/$TODO1_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
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
echo "8️⃣  Filtering completed todos..."
COMPLETED_TODOS=$(curl -s "$API_URL/lists/$LIST_ID/todos?completed=true" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

COMPLETED_COUNT=$(echo $COMPLETED_TODOS | jq '. | length')
echo "✅ Found $COMPLETED_COUNT completed todo(s)"
echo ""

# Test token refresh
echo "9️⃣  Testing token refresh..."
REFRESH_RESPONSE=$(curl -s -X POST "$API_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\": \"$REFRESH_TOKEN\"}")

NEW_ACCESS_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.accessToken')

if [ "$NEW_ACCESS_TOKEN" != "null" ] && [ -n "$NEW_ACCESS_TOKEN" ]; then
    echo "✅ Token refreshed successfully"
    ACCESS_TOKEN=$NEW_ACCESS_TOKEN
else
    echo "❌ Failed to refresh token"
    exit 1
fi
echo ""

# Cleanup - Delete the list
echo "🧹 Cleaning up..."
curl -s -X DELETE "$API_URL/lists/$LIST_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null

echo "✅ Test list deleted"
echo ""

# Logout
curl -s -X POST "$API_URL/auth/logout" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\": \"$REFRESH_TOKEN\"}" > /dev/null

echo "✅ Logged out"
echo ""

echo "================================"
echo "✅ All tests passed successfully!"
echo "================================"
