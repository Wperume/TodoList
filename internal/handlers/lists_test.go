package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"todolist-api/internal/models"
	"todolist-api/internal/storage"
	"todolist-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user ID for all handler tests
var testUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

func setupListHandler() (*ListHandler, storage.Store) {
	store := storage.NewStorage()
	handler := NewListHandler(store)
	return handler, store
}

func TestGetAllLists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully retrieves paginated lists", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create test lists
		for i := 1; i <= 25; i++ {
			_, err := store.CreateList(testUserID, models.CreateTodoListRequest{
				Name: "List " + string(rune(i+64)),
			})
			require.NoError(t, err)
		}

		// Create request
		req := httptest.NewRequest("GET", "/lists?page=1&limit=10", http.NoBody)
		w := httptest.NewRecorder()

		// Create test context
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Call handler
		handler.GetAllLists(c)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response models.PaginatedListsResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.Len(t, response.Data, 10)
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 10, response.Pagination.Limit)
		assert.Equal(t, 3, response.Pagination.TotalPages)
		assert.Equal(t, 25, response.Pagination.TotalItems)
	})

	t.Run("returns empty list when no lists exist", func(t *testing.T) {
		handler, _ := setupListHandler()

		req := httptest.NewRequest("GET", "/lists", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.GetAllLists(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.PaginatedListsResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.Len(t, response.Data, 0)
		assert.Equal(t, 0, response.Pagination.TotalItems)
	})

	t.Run("validates pagination parameters", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create one list
		_, err := store.CreateList(testUserID, models.CreateTodoListRequest{Name: "Test"})
		require.NoError(t, err)

		// Test page < 1 (should default to 1)
		req := httptest.NewRequest("GET", "/lists?page=0&limit=10", http.NoBody)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.GetAllLists(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response models.PaginatedListsResponse
		testutil.ParseJSONResponse(t, w, &response)
		assert.Equal(t, 1, response.Pagination.Page)
	})

	t.Run("limits page size to maximum", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create one list
		_, err := store.CreateList(testUserID, models.CreateTodoListRequest{Name: "Test"})
		require.NoError(t, err)

		// Test limit > 100 (should cap at 20)
		req := httptest.NewRequest("GET", "/lists?page=1&limit=200", http.NoBody)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.GetAllLists(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response models.PaginatedListsResponse
		testutil.ParseJSONResponse(t, w, &response)
		assert.Equal(t, 20, response.Pagination.Limit)
	})
}

func TestCreateList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully creates a list", func(t *testing.T) {
		handler, _ := setupListHandler()

		reqBody := models.CreateTodoListRequest{
			Name:        "Work Tasks",
			Description: "Tasks for work",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.CreateList(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var list models.TodoList
		testutil.ParseJSONResponse(t, w, &list)

		assert.NotEqual(t, uuid.Nil, list.ID)
		assert.Equal(t, "Work Tasks", list.Name)
		assert.Equal(t, "Tasks for work", list.Description)
		assert.Equal(t, 0, list.TodoCount)
		assert.False(t, list.CreatedAt.IsZero())
		assert.False(t, list.UpdatedAt.IsZero())
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler, _ := setupListHandler()

		req := httptest.NewRequest("POST", "/lists", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.CreateList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "INVALID_INPUT", errResp.Code)
	})

	t.Run("returns 409 when list name already exists", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create first list
		_, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name: "Duplicate Name",
		})
		require.NoError(t, err)

		// Try to create second list with same name
		reqBody := models.CreateTodoListRequest{
			Name: "Duplicate Name",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.CreateList(c)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "LIST_NAME_EXISTS", errResp.Code)
	})

	t.Run("validates required fields", func(t *testing.T) {
		handler, _ := setupListHandler()

		// Empty name should fail validation
		reqBody := models.CreateTodoListRequest{
			Name: "",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/lists", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.CreateList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetListByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully retrieves existing list", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create test list
		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name:        "Test List",
			Description: "Test Description",
		})
		require.NoError(t, err)

		// Create request
		req := httptest.NewRequest("GET", "/lists/"+created.ID.String(), http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.GetListByID(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var list models.TodoList
		testutil.ParseJSONResponse(t, w, &list)

		assert.Equal(t, created.ID, list.ID)
		assert.Equal(t, created.Name, list.Name)
		assert.Equal(t, created.Description, list.Description)
	})

	t.Run("returns 404 for non-existent list", func(t *testing.T) {
		handler, _ := setupListHandler()

		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", "/lists/"+nonExistentID.String(), http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: nonExistentID.String()}}

		handler.GetListByID(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns 400 for invalid UUID format", func(t *testing.T) {
		handler, _ := setupListHandler()

		req := httptest.NewRequest("GET", "/lists/invalid-uuid", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: "invalid-uuid"}}

		handler.GetListByID(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

func TestUpdateList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully updates list name", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create test list
		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name:        "Original Name",
			Description: "Original Description",
		})
		require.NoError(t, err)

		// Update name
		newName := "Updated Name"
		reqBody := models.UpdateTodoListRequest{
			Name: &newName,
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var list models.TodoList
		testutil.ParseJSONResponse(t, w, &list)

		assert.Equal(t, "Updated Name", list.Name)
		assert.Equal(t, "Original Description", list.Description)
	})

	t.Run("successfully updates list description", func(t *testing.T) {
		handler, store := setupListHandler()

		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name:        "Test Name",
			Description: "Original Description",
		})
		require.NoError(t, err)

		newDesc := "Updated Description"
		reqBody := models.UpdateTodoListRequest{
			Description: &newDesc,
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var list models.TodoList
		testutil.ParseJSONResponse(t, w, &list)

		assert.Equal(t, "Test Name", list.Name)
		assert.Equal(t, "Updated Description", list.Description)
	})

	t.Run("handles partial updates", func(t *testing.T) {
		handler, store := setupListHandler()

		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name:        "Test Name",
			Description: "Test Description",
		})
		require.NoError(t, err)

		// Update only name
		newName := "Only Name Changed"
		reqBody := models.UpdateTodoListRequest{
			Name: &newName,
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+created.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var list models.TodoList
		testutil.ParseJSONResponse(t, w, &list)

		assert.Equal(t, "Only Name Changed", list.Name)
		assert.Equal(t, "Test Description", list.Description) // Unchanged
	})

	t.Run("returns 404 for non-existent list", func(t *testing.T) {
		handler, _ := setupListHandler()

		nonExistentID := uuid.New()
		newName := "New Name"
		reqBody := models.UpdateTodoListRequest{
			Name: &newName,
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+nonExistentID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: nonExistentID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns 409 on name conflict", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create two lists
		list1, err := store.CreateList(testUserID, models.CreateTodoListRequest{Name: "List 1"})
		require.NoError(t, err)

		_, err = store.CreateList(testUserID, models.CreateTodoListRequest{Name: "List 2"})
		require.NoError(t, err)

		// Try to update list1 with list2's name
		conflictName := "List 2"
		reqBody := models.UpdateTodoListRequest{
			Name: &conflictName,
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/"+list1.ID.String(), reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: list1.ID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "LIST_NAME_EXISTS", errResp.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler, store := setupListHandler()

		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{Name: "Test"})
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/lists/"+created.ID.String(), http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		handler, _ := setupListHandler()

		newName := "Test"
		reqBody := models.UpdateTodoListRequest{Name: &newName}

		req := testutil.MakeJSONRequest(t, "PUT", "/lists/invalid-uuid", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: "invalid-uuid"}}

		handler.UpdateList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})
}

func TestDeleteList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully deletes list", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create test list
		created, err := store.CreateList(testUserID, models.CreateTodoListRequest{
			Name: "Test List",
		})
		require.NoError(t, err)

		// Delete request
		req := httptest.NewRequest("DELETE", "/lists/"+created.ID.String(), http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: created.ID.String()}}

		handler.DeleteList(c)

		// Accept both 200 and 204 (Gin behavior difference)
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, w.Code)

		// Verify list is deleted
		_, err = store.GetListByID(testUserID, created.ID)
		assert.ErrorIs(t, err, storage.ErrListNotFound)
	})

	t.Run("returns 404 for non-existent list", func(t *testing.T) {
		handler, _ := setupListHandler()

		nonExistentID := uuid.New()
		req := httptest.NewRequest("DELETE", "/lists/"+nonExistentID.String(), http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: nonExistentID.String()}}

		handler.DeleteList(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)
		assert.Equal(t, "LIST_NOT_FOUND", errResp.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		handler, _ := setupListHandler()

		req := httptest.NewRequest("DELETE", "/lists/invalid-uuid", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: "invalid-uuid"}}

		handler.DeleteList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_LIST_ID", errResp.Code)
	})

	t.Run("cascades delete to todos", func(t *testing.T) {
		handler, store := setupListHandler()

		// Create list
		list, err := store.CreateList(testUserID, models.CreateTodoListRequest{Name: "Test List"})
		require.NoError(t, err)

		// Create todo in list
		_, err = store.CreateTodo(testUserID, list.ID, models.CreateTodoRequest{
			Description: "Test Todo",
			Priority:    models.PriorityHigh,
		})
		require.NoError(t, err)

		// Delete list
		req := httptest.NewRequest("DELETE", "/lists/"+list.ID.String(), http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "listId", Value: list.ID.String()}}

		handler.DeleteList(c)

		// Accept both 200 and 204 (Gin behavior difference)
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, w.Code)

		// Verify todos are also deleted (indirectly through list not found)
		_, err = store.GetTodosByList(testUserID, list.ID, nil, nil, "createdAt", "asc")
		assert.ErrorIs(t, err, storage.ErrListNotFound)
	})
}
