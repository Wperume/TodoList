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

func TestNewStorage(t *testing.T) {
	store := NewStorage()
	assert.NotNil(t, store)
	assert.NotNil(t, store.lists)
	assert.NotNil(t, store.todos)
}

func TestCreateList(t *testing.T) {
	store := NewStorage()

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
		assert.False(t, list.CreatedAt.IsZero())
		assert.False(t, list.UpdatedAt.IsZero())
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

func TestGetAllLists(t *testing.T) {
	store := NewStorage()

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

	t.Run("returns last page with remaining items", func(t *testing.T) {
		lists, pagination, err := store.GetAllLists(3, 10)
		require.NoError(t, err)
		assert.Len(t, lists, 5)
		assert.Equal(t, 3, pagination.Page)
	})
}

func TestGetListByID(t *testing.T) {
	store := NewStorage()

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

func TestUpdateList(t *testing.T) {
	store := NewStorage()

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

	t.Run("successfully updates description", func(t *testing.T) {
		newDesc := "Updated Description"
		updateReq := models.UpdateTodoListRequest{Description: &newDesc}

		updated, err := store.UpdateList(created.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", updated.Description)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		updateReq := models.UpdateTodoListRequest{Name: testutil.StringPtr("Name")}
		_, err := store.UpdateList(uuid.New(), updateReq)
		assert.ErrorIs(t, err, ErrListNotFound)
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

func TestDeleteList(t *testing.T) {
	store := NewStorage()

	req := models.CreateTodoListRequest{Name: "Test List"}
	created, err := store.CreateList(req)
	require.NoError(t, err)

	// Add some todos to the list
	todoReq := models.CreateTodoRequest{
		Description: "Test Todo",
		Priority:    models.PriorityHigh,
	}
	_, err = store.CreateTodo(created.ID, todoReq)
	require.NoError(t, err)

	t.Run("successfully deletes list and todos", func(t *testing.T) {
		err := store.DeleteList(created.ID)
		require.NoError(t, err)

		// Verify list is deleted
		_, err = store.GetListByID(created.ID)
		assert.ErrorIs(t, err, ErrListNotFound)

		// Verify todos are deleted
		todos, err := store.GetTodosByList(created.ID, nil, nil, "createdAt", "asc")
		assert.ErrorIs(t, err, ErrListNotFound)
		assert.Nil(t, todos)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		err := store.DeleteList(uuid.New())
		assert.ErrorIs(t, err, ErrListNotFound)
	})
}

func TestCreateTodo(t *testing.T) {
	store := NewStorage()

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
		assert.Nil(t, todo.CompletedAt)
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

func TestGetTodosByList(t *testing.T) {
	store := NewStorage()

	req := models.CreateTodoListRequest{Name: "Test List"}
	list, err := store.CreateList(req)
	require.NoError(t, err)

	// Create todos with different priorities and completion status
	todos := []models.CreateTodoRequest{
		{Description: "High priority incomplete", Priority: models.PriorityHigh},
		{Description: "Medium priority incomplete", Priority: models.PriorityMedium},
		{Description: "Low priority incomplete", Priority: models.PriorityLow},
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

	t.Run("sorts by priority", func(t *testing.T) {
		result, err := store.GetTodosByList(list.ID, nil, nil, "priority", "asc")
		require.NoError(t, err)
		assert.Equal(t, models.PriorityHigh, result[0].Priority)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		_, err := store.GetTodosByList(uuid.New(), nil, nil, "createdAt", "asc")
		assert.ErrorIs(t, err, ErrListNotFound)
	})
}

func TestUpdateTodo(t *testing.T) {
	store := NewStorage()

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

	t.Run("updates description and priority", func(t *testing.T) {
		newDesc := "Updated"
		newPriority := models.PriorityHigh
		updateReq := models.UpdateTodoRequest{
			Description: &newDesc,
			Priority:    &newPriority,
		}

		updated, err := store.UpdateTodo(list.ID, todo.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Description)
		assert.Equal(t, models.PriorityHigh, updated.Priority)
	})

	t.Run("fails when list not found", func(t *testing.T) {
		updateReq := models.UpdateTodoRequest{}
		_, err := store.UpdateTodo(uuid.New(), todo.ID, updateReq)
		assert.ErrorIs(t, err, ErrListNotFound)
	})

	t.Run("fails when todo not found", func(t *testing.T) {
		updateReq := models.UpdateTodoRequest{}
		_, err := store.UpdateTodo(list.ID, uuid.New(), updateReq)
		assert.ErrorIs(t, err, ErrTodoNotFound)
	})
}

func TestDeleteTodo(t *testing.T) {
	store := NewStorage()

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

	t.Run("fails when list not found", func(t *testing.T) {
		err := store.DeleteTodo(uuid.New(), uuid.New())
		assert.ErrorIs(t, err, ErrListNotFound)
	})

	t.Run("fails when todo not found", func(t *testing.T) {
		err := store.DeleteTodo(list.ID, uuid.New())
		assert.ErrorIs(t, err, ErrTodoNotFound)
	})
}
