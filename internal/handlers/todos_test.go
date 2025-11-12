package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"todolist-api/internal/models"
	"todolist-api/internal/storage"
	"todolist-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTodoHandler() (*TodoHandler, storage.Store, uuid.UUID) {
	store := storage.NewStorage()
	handler := NewTodoHandler(store)

	// Create a test list for todos
	list, err := store.CreateList(testUserID, models.CreateTodoListRequest{
		Name: "Test List",
	})
	if err != nil {
		panic(err)
	}

	return handler, store, list.ID
}

func TestGetTodosByList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully retrieves all todos", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create test todos
		_, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "Todo 1",
			Priority:    models.PriorityHigh,
		})
		require.NoError(t, err)

		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "Todo 2",
			Priority:    models.PriorityMedium,
		})
		require.NoError(t, err)

		// Create request
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		assert.Len(t, todos, 2)
	})

	t.Run("filters by priority", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create todos with different priorities
		_, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "High Priority",
			Priority:    models.PriorityHigh,
		})
		require.NoError(t, err)

		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "Low Priority",
			Priority:    models.PriorityLow,
		})
		require.NoError(t, err)

		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "Medium Priority",
			Priority:    models.PriorityMedium,
		})
		require.NoError(t, err)

		// Filter for high priority
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?priority=high", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		assert.Len(t, todos, 1)
		assert.Equal(t, models.PriorityHigh, todos[0].Priority)
	})

	t.Run("filters by completed status - true", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create completed todo
		completedTodo, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Completed Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		// Mark as completed
		now := time.Now()
		_, err = store.UpdateTodo(testUserID, listID, completedTodo.ID, models.UpdateTodoRequest{
			Completed: &[]bool{true}[0],
		})
		require.NoError(t, err)

		// Create incomplete todo
		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Incomplete Todo",
			Priority: models.PriorityLow,
		})
		require.NoError(t, err)

		// Filter for completed
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?completed=true", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		assert.Len(t, todos, 1)
		assert.True(t, todos[0].Completed)
		assert.NotNil(t, todos[0].CompletedAt)
		assert.True(t, todos[0].CompletedAt.After(now.Add(-time.Second)))
	})

	t.Run("filters by completed status - false", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create completed todo
		completedTodo, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Completed Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		_, err = store.UpdateTodo(testUserID, listID, completedTodo.ID, models.UpdateTodoRequest{
			Completed: &[]bool{true}[0],
		})
		require.NoError(t, err)

		// Create incomplete todo
		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Incomplete Todo",
			Priority: models.PriorityLow,
		})
		require.NoError(t, err)

		// Filter for incomplete
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?completed=false", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		assert.Len(t, todos, 1)
		assert.False(t, todos[0].Completed)
		assert.Nil(t, todos[0].CompletedAt)
	})

	t.Run("sorts by createdAt ascending", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create todos with slight delays
		todo1, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "First Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		todo2, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Second Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		// Sort by createdAt asc (default)
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?sortBy=createdAt&sortOrder=asc", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		require.Len(t, todos, 2)
		assert.Equal(t, todo1.ID, todos[0].ID)
		assert.Equal(t, todo2.ID, todos[1].ID)
	})

	t.Run("sorts by createdAt descending", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create todos
		todo1, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "First Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		todo2, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Second Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		// Sort by createdAt desc
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?sortBy=createdAt&sortOrder=desc", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		require.Len(t, todos, 2)
		assert.Equal(t, todo2.ID, todos[0].ID)
		assert.Equal(t, todo1.ID, todos[1].ID)
	})

	t.Run("sorts by priority", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create todos with different priorities
		_, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Medium Priority",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "High Priority",
			Priority: models.PriorityHigh,
		})
		require.NoError(t, err)

		_, err = store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Low Priority",
			Priority: models.PriorityLow,
		})
		require.NoError(t, err)

		// Sort by priority asc
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?sortBy=priority&sortOrder=asc", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		require.Len(t, todos, 3)
		assert.Equal(t, models.PriorityHigh, todos[0].Priority)
		assert.Equal(t, models.PriorityMedium, todos[1].Priority)
		assert.Equal(t, models.PriorityLow, todos[2].Priority)
	})

	t.Run("returns empty list when no todos exist", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []models.Todo
		testutil.ParseJSONResponse(t, w, &todos)

		assert.Len(t, todos, 0)
	})

	t.Run("returns error for invalid list ID format", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/invalid-uuid/todos", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: "invalid-uuid"}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})

	t.Run("returns error for non-existent list", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", "/lists/"+nonExistentID.String()+"/todos", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: nonExistentID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for invalid priority", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?priority=invalid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_PRIORITY", errResp.Code)
	})

	t.Run("returns error for invalid completed value", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?completed=invalid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_COMPLETED", errResp.Code)
	})

	t.Run("returns error for invalid sortBy value", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?sortBy=invalid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_SORT_BY", errResp.Code)
	})

	t.Run("returns error for invalid sortOrder value", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos?sortOrder=invalid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.GetTodosByList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_SORT_ORDER", errResp.Code)
	})
}

func TestCreateTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully creates todo with all fields", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		dueDate := time.Now().Add(24 * time.Hour)
		reqBody := models.CreateTodoRequest{
			Description: "New Todo",
			Priority:    models.PriorityHigh,
			DueDate:     &dueDate,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists/"+listID.String()+"/todos", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.NotEqual(t, uuid.Nil, todo.ID)
		assert.Equal(t, "New Todo", todo.Description)
		assert.Equal(t, models.PriorityHigh, todo.Priority)
		assert.False(t, todo.Completed)
		assert.Nil(t, todo.CompletedAt)
		assert.NotNil(t, todo.DueDate)
	})

	t.Run("successfully creates todo with minimal fields", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		reqBody := models.CreateTodoRequest{
			Description:    "Minimal Todo",
			Priority: models.PriorityMedium,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists/"+listID.String()+"/todos", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.Equal(t, "Minimal Todo", todo.Description)
		assert.Equal(t, models.PriorityMedium, todo.Priority)
		assert.Nil(t, todo.DueDate)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("POST", "/lists/"+listID.String()+"/todos", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_INPUT", errResp.Code)
	})

	t.Run("returns error for missing title", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		reqBody := models.CreateTodoRequest{
			Priority: models.PriorityMedium,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists/"+listID.String()+"/todos", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: listID.String()}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_INPUT", errResp.Code)
	})

	t.Run("returns error for non-existent list", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		nonExistentID := uuid.New()
		reqBody := models.CreateTodoRequest{
			Description:    "Todo",
			Priority: models.PriorityMedium,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists/"+nonExistentID.String()+"/todos", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: nonExistentID.String()}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for invalid list ID format", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		reqBody := models.CreateTodoRequest{
			Description:    "Todo",
			Priority: models.PriorityMedium,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists/invalid-uuid/todos", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: "invalid-uuid"}}

		handler.CreateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

func TestGetTodoByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully retrieves todo by ID", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description: "Test Todo",
			Priority:    models.PriorityMedium,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos/"+created.ID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.GetTodoByID(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.Equal(t, created.ID, todo.ID)
		assert.Equal(t, "Test Todo", todo.Description)
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos/"+nonExistentID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: nonExistentID.String()},
		}

		handler.GetTodoByID(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "TODO_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for non-existent list", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		nonExistentListID := uuid.New()
		todoID := uuid.New()

		req := httptest.NewRequest("GET", "/lists/"+nonExistentListID.String()+"/todos/"+todoID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: nonExistentListID.String()},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.GetTodoByID(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for invalid todo ID format", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("GET", "/lists/"+listID.String()+"/todos/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: "invalid-uuid"},
		}

		handler.GetTodoByID(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_TODO_ID", errResp.Code)
	})

	t.Run("returns error for invalid list ID format", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		todoID := uuid.New()
		req := httptest.NewRequest("GET", "/lists/invalid-uuid/todos/"+todoID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: "invalid-uuid"},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.GetTodoByID(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

func TestUpdateTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully updates todo title", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Original Title",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Title"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.Equal(t, "Updated Title", todo.Description)
	})

	t.Run("successfully marks todo as completed", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Todo to Complete",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		reqBody := models.UpdateTodoRequest{
			Completed: boolPtr(true),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.True(t, todo.Completed)
		assert.NotNil(t, todo.CompletedAt)
	})

	t.Run("successfully marks todo as incomplete", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		// Create and mark as completed
		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Completed Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		_, err = store.UpdateTodo(testUserID, listID, created.ID, models.UpdateTodoRequest{
			Completed: boolPtr(true),
		})
		require.NoError(t, err)

		// Mark as incomplete
		reqBody := models.UpdateTodoRequest{
			Completed: boolPtr(false),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.False(t, todo.Completed)
		assert.Nil(t, todo.CompletedAt)
	})

	t.Run("successfully updates multiple fields", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Original Todo",
			Priority: models.PriorityLow,
		})
		require.NoError(t, err)

		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Todo"),
			Priority:    priorityPtr(models.PriorityHigh),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todo models.Todo
		testutil.ParseJSONResponse(t, w, &todo)

		assert.Equal(t, "Updated Todo", todo.Description)
		assert.Equal(t, models.PriorityHigh, todo.Priority)
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		nonExistentID := uuid.New()
		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Title"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/"+nonExistentID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: nonExistentID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "TODO_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for non-existent list", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		nonExistentListID := uuid.New()
		todoID := uuid.New()
		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Title"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+nonExistentListID.String()+"/todos/"+todoID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: nonExistentListID.String()},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Todo",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/lists/"+listID.String()+"/todos/"+created.ID.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_INPUT", errResp.Code)
	})

	t.Run("returns error for invalid todo ID format", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Title"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+listID.String()+"/todos/invalid-uuid", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: "invalid-uuid"},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_TODO_ID", errResp.Code)
	})

	t.Run("returns error for invalid list ID format", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		todoID := uuid.New()
		reqBody := models.UpdateTodoRequest{
			Description: strPtr("Updated Title"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/invalid-uuid/todos/"+todoID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: "invalid-uuid"},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.UpdateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

func TestDeleteTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully deletes todo", func(t *testing.T) {
		handler, store, listID := setupTodoHandler()

		created, err := store.CreateTodo(testUserID, listID, models.CreateTodoRequest{
			Description:    "Todo to Delete",
			Priority: models.PriorityMedium,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("DELETE", "/lists/"+listID.String()+"/todos/"+created.ID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: created.ID.String()},
		}

		handler.DeleteTodo(c)

		// Accept both 200 and 204 (Gin behavior difference)
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, w.Code)

		// Verify todo is deleted
		_, err = store.GetTodoByID(testUserID, listID, created.ID)
		assert.ErrorIs(t, err, storage.ErrTodoNotFound)
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		nonExistentID := uuid.New()
		req := httptest.NewRequest("DELETE", "/lists/"+listID.String()+"/todos/"+nonExistentID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: nonExistentID.String()},
		}

		handler.DeleteTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "TODO_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for non-existent list", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		nonExistentListID := uuid.New()
		todoID := uuid.New()

		req := httptest.NewRequest("DELETE", "/lists/"+nonExistentListID.String()+"/todos/"+todoID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: nonExistentListID.String()},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.DeleteTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns error for invalid todo ID format", func(t *testing.T) {
		handler, _, listID := setupTodoHandler()

		req := httptest.NewRequest("DELETE", "/lists/"+listID.String()+"/todos/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: listID.String()},
			{Key: "todoId", Value: "invalid-uuid"},
		}

		handler.DeleteTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_TODO_ID", errResp.Code)
	})

	t.Run("returns error for invalid list ID format", func(t *testing.T) {
		handler, _, _ := setupTodoHandler()

		todoID := uuid.New()
		req := httptest.NewRequest("DELETE", "/lists/invalid-uuid/todos/"+todoID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{
			{Key: "listId", Value: "invalid-uuid"},
			{Key: "todoId", Value: todoID.String()},
		}

		handler.DeleteTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

// Helper functions for pointer creation
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func priorityPtr(p models.Priority) *models.Priority {
	return &p
}
