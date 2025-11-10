#!/bin/bash

# Test script for Todo List API
# Make sure the server is running before executing this script

BASE_URL="http://localhost:8080/api/v1"

echo "========================================="
echo "Testing Todo List REST API"
echo "========================================="
echo ""

# Test 1: Health check
echo "1. Testing health check..."
curl -s http://localhost:8080/health | jq .
echo -e "\n"

# Test 2: Create a todo list
echo "2. Creating a new todo list..."
LIST_RESPONSE=$(curl -s -X POST "$BASE_URL/lists" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Work Tasks",
    "description": "Tasks for work projects"
  }')
echo "$LIST_RESPONSE" | jq .
LIST_ID=$(echo "$LIST_RESPONSE" | jq -r '.id')
echo "Created list with ID: $LIST_ID"
echo -e "\n"

# Test 3: Get all lists
echo "3. Getting all todo lists..."
curl -s "$BASE_URL/lists" | jq .
echo -e "\n"

# Test 4: Get specific list
echo "4. Getting specific list..."
curl -s "$BASE_URL/lists/$LIST_ID" | jq .
echo -e "\n"

# Test 5: Create a todo
echo "5. Creating a high priority todo..."
TODO_RESPONSE=$(curl -s -X POST "$BASE_URL/lists/$LIST_ID/todos" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Complete project documentation",
    "priority": "high",
    "dueDate": "2025-11-15T23:59:59Z"
  }')
echo "$TODO_RESPONSE" | jq .
TODO_ID=$(echo "$TODO_RESPONSE" | jq -r '.id')
echo "Created todo with ID: $TODO_ID"
echo -e "\n"

# Test 6: Create another todo
echo "6. Creating a medium priority todo..."
curl -s -X POST "$BASE_URL/lists/$LIST_ID/todos" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Review pull requests",
    "priority": "medium",
    "dueDate": "2025-11-12T17:00:00Z"
  }' | jq .
echo -e "\n"

# Test 7: Get all todos in list
echo "7. Getting all todos in the list..."
curl -s "$BASE_URL/lists/$LIST_ID/todos" | jq .
echo -e "\n"

# Test 8: Get high priority todos
echo "8. Getting only high priority todos..."
curl -s "$BASE_URL/lists/$LIST_ID/todos?priority=high" | jq .
echo -e "\n"

# Test 9: Update todo (mark as completed)
echo "9. Marking todo as completed..."
curl -s -X PUT "$BASE_URL/lists/$LIST_ID/todos/$TODO_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "completed": true
  }' | jq .
echo -e "\n"

# Test 10: Get completed todos
echo "10. Getting completed todos..."
curl -s "$BASE_URL/lists/$LIST_ID/todos?completed=true" | jq .
echo -e "\n"

# Test 11: Update list
echo "11. Updating list name..."
curl -s -X PUT "$BASE_URL/lists/$LIST_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Work Tasks"
  }' | jq .
echo -e "\n"

# Test 12: Delete todo
echo "12. Deleting a todo..."
curl -s -X DELETE "$BASE_URL/lists/$LIST_ID/todos/$TODO_ID"
echo "Todo deleted (204 No Content expected)"
echo -e "\n"

# Test 13: Verify deletion
echo "13. Verifying todo deletion..."
curl -s "$BASE_URL/lists/$LIST_ID/todos" | jq .
echo -e "\n"

# Test 14: Delete list
echo "14. Deleting the todo list..."
curl -s -X DELETE "$BASE_URL/lists/$LIST_ID"
echo "List deleted (204 No Content expected)"
echo -e "\n"

# Test 15: Verify list deletion
echo "15. Verifying list deletion (should return 404)..."
curl -s "$BASE_URL/lists/$LIST_ID" | jq .
echo -e "\n"

echo "========================================="
echo "API Testing Complete!"
echo "========================================="
