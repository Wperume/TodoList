package storage

import (
	"testing"
	"time"

	"todolist-api/internal/models"
	"todolist-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresCreateList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	t.Run("successfully creates a list", func(t *testing.T) {
		req := models.CreateTodoListRequest{
			Name:        "Work Tasks",
			Description: "Tasks for work",
		}

		list, err := store.CreateList(req)
		require.NoError(t, err)
		assert.NotNil(t, list)
		assert.NotEqual(t, uuid.Nil, list.ID)
		assert.Equal(t, "Work Tasks", list.Name)
		assert.Equal(t, "Tasks for work", list.Description)
		assert.Equal(t, 0, list.TodoCount)
	})

	t.Run("fails when list name already exists", func(t *testing.T) {
		req := models.CreateTodoListRequest{
			Name:        "Duplicate List",
			Description: "First",
		}
		_, err := store.CreateList(req)
		require.NoError(t, err)

		// Try to create with same name
		req.Description = "Second"
		_, err = store.CreateList(req)
		assert.ErrorIs(t, err, ErrListNameExists)
	})
}

func TestPostgresGetAllLists(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	// Create test lists
	for i := 1; i <= 25; i++ {
		req := models.CreateTodoListRequest{
			Name: "List " + string(rune(i+64)),
		}
		_, err := store.CreateList(req)
		require.NoError(t, err)
	}

	t.Run("returns paginated lists", func(t *testing.T) {
		lists, pagination, err := store.GetAllLists(1, 10)
		require.NoError(t, err)
		assert.Len(t, lists, 10)
		assert.Equal(t, 1, pagination.Page)
		assert.Equal(t, 10, pagination.Limit)
		assert.Equal(t, 3, pagination.TotalPages)
		assert.Equal(t, 25, pagination.TotalItems)
	})

	t.Run("returns correct page", func(t *testing.T) {
		lists, pagination, err := store.GetAllLists(2, 10)
		require.NoError(t, err)
		assert.Len(t, lists, 10)
		assert.Equal(t, 2, pagination.Page)
	})
}

func TestPostgresGetListByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	created, err := store.CreateList(req)
	require.NoError(t, err)

	t.Run("successfully retrieves list", func(t *testing.T) {
		list, err := store.GetListByID(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, list.ID)
		assert.Equal(t, created.Name, list.Name)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		_, err := store.GetListByID(uuid.New())
		assert.ErrorIs(t, err, ErrListNotFound)
	})
}

func TestPostgresUpdateList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{
		Name:        "Original Name",
		Description: "Original Description",
	}
	created, err := store.CreateList(req)
	require.NoError(t, err)

	t.Run("successfully updates name", func(t *testing.T) {
		newName := "Updated Name"
		updateReq := models.UpdateTodoListRequest{Name: &newName}

		updated, err := store.UpdateList(created.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "Original Description", updated.Description)
	})

	t.Run("fails when new name conflicts", func(t *testing.T) {
		// Create another list
		req2 := models.CreateTodoListRequest{Name: "Another List"}
		_, err := store.CreateList(req2)
		require.NoError(t, err)

		// Try to update first list with second list's name
		conflictName := "Another List"
		updateReq := models.UpdateTodoListRequest{Name: &conflictName}
		_, err = store.UpdateList(created.ID, updateReq)
		assert.ErrorIs(t, err, ErrListNameExists)
	})
}

func TestPostgresDeleteList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	created, err := store.CreateList(req)
	require.NoError(t, err)

	// Add a todo to test cascade delete
	todoReq := models.CreateTodoRequest{
		Description: "Test Todo",
		Priority:    models.PriorityHigh,
	}
	_, err = store.CreateTodo(created.ID, todoReq)
	require.NoError(t, err)

	t.Run("successfully deletes list (soft delete)", func(t *testing.T) {
		err := store.DeleteList(created.ID)
		require.NoError(t, err)

		// Verify list is not found (soft deleted)
		_, err = store.GetListByID(created.ID)
		assert.ErrorIs(t, err, ErrListNotFound)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		err := store.DeleteList(uuid.New())
		assert.ErrorIs(t, err, ErrListNotFound)
	})
}

func TestPostgresCreateTodo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	t.Run("successfully creates todo", func(t *testing.T) {
		dueDate := time.Now().Add(24 * time.Hour)
		todoReq := models.CreateTodoRequest{
			Description: "Test Todo",
			Priority:    models.PriorityHigh,
			DueDate:     &dueDate,
		}

		todo, err := store.CreateTodo(list.ID, todoReq)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, todo.ID)
		assert.Equal(t, list.ID, todo.ListID)
		assert.Equal(t, "Test Todo", todo.Description)
		assert.Equal(t, models.PriorityHigh, todo.Priority)
		assert.NotNil(t, todo.DueDate)
		assert.False(t, todo.Completed)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		todoReq := models.CreateTodoRequest{
			Description: "Test",
			Priority:    models.PriorityLow,
		}
		_, err := store.CreateTodo(uuid.New(), todoReq)
		assert.ErrorIs(t, err, ErrListNotFound)
	})
}

func TestPostgresGetTodosByList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	// Create todos with different priorities
	todos := []models.CreateTodoRequest{
		{Description: "High priority", Priority: models.PriorityHigh},
		{Description: "Medium priority", Priority: models.PriorityMedium},
		{Description: "Low priority", Priority: models.PriorityLow},
	}

	for _, todoReq := range todos {
		_, err := store.CreateTodo(list.ID, todoReq)
		require.NoError(t, err)
	}

	t.Run("gets all todos", func(t *testing.T) {
		result, err := store.GetTodosByList(list.ID, nil, nil, "createdAt", "asc")
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("filters by priority", func(t *testing.T) {
		priority := models.PriorityHigh
		result, err := store.GetTodosByList(list.ID, &priority, nil, "createdAt", "asc")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, models.PriorityHigh, result[0].Priority)
	})

	t.Run("filters by completion status", func(t *testing.T) {
		completed := false
		result, err := store.GetTodosByList(list.ID, nil, &completed, "createdAt", "asc")
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("sorts by priority descending", func(t *testing.T) {
		result, err := store.GetTodosByList(list.ID, nil, nil, "priority", "desc")
		require.NoError(t, err)
		// Descending priority order: high -> medium -> low
		assert.Equal(t, models.PriorityHigh, result[0].Priority)
		assert.Equal(t, models.PriorityLow, result[2].Priority)
	})
}

func TestPostgresUpdateTodo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	todoReq := models.CreateTodoRequest{
		Description: "Original",
		Priority:    models.PriorityLow,
	}
	todo, err := store.CreateTodo(list.ID, todoReq)
	require.NoError(t, err)

	t.Run("marks todo as completed", func(t *testing.T) {
		completed := true
		updateReq := models.UpdateTodoRequest{Completed: &completed}

		updated, err := store.UpdateTodo(list.ID, todo.ID, updateReq)
		require.NoError(t, err)
		assert.True(t, updated.Completed)
		assert.NotNil(t, updated.CompletedAt)
	})

	t.Run("marks todo as incomplete", func(t *testing.T) {
		completed := false
		updateReq := models.UpdateTodoRequest{Completed: &completed}

		updated, err := store.UpdateTodo(list.ID, todo.ID, updateReq)
		require.NoError(t, err)
		assert.False(t, updated.Completed)
		assert.Nil(t, updated.CompletedAt)
	})

	t.Run("updates multiple fields", func(t *testing.T) {
		newDesc := "Updated"
		newPriority := models.PriorityHigh
		completed := true
		updateReq := models.UpdateTodoRequest{
			Description: &newDesc,
			Priority:    &newPriority,
			Completed:   &completed,
		}

		updated, err := store.UpdateTodo(list.ID, todo.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Description)
		assert.Equal(t, models.PriorityHigh, updated.Priority)
		assert.True(t, updated.Completed)
	})
}

func TestPostgresDeleteTodo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	todoReq := models.CreateTodoRequest{
		Description: "Test",
		Priority:    models.PriorityMedium,
	}
	todo, err := store.CreateTodo(list.ID, todoReq)
	require.NoError(t, err)

	t.Run("successfully deletes todo", func(t *testing.T) {
		err := store.DeleteTodo(list.ID, todo.ID)
		require.NoError(t, err)

		_, err = store.GetTodoByID(list.ID, todo.ID)
		assert.ErrorIs(t, err, ErrTodoNotFound)
	})
}

func TestPostgresTodoCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	store := NewPostgresStorage(db)

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	// Create multiple todos
	for i := 0; i < 5; i++ {
		todoReq := models.CreateTodoRequest{
			Description: "Todo " + string(rune(i+48)),
			Priority:    models.PriorityMedium,
		}
		_, err := store.CreateTodo(list.ID, todoReq)
		require.NoError(t, err)
	}

	t.Run("todo count is accurate", func(t *testing.T) {
		updated, err := store.GetListByID(list.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.TodoCount)
	})
}
