package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPriorityConstants(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		expected string
	}{
		{"Low priority", PriorityLow, "low"},
		{"Medium priority", PriorityMedium, "medium"},
		{"High priority", PriorityHigh, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.priority))
		})
	}
}

func TestTodoListBeforeCreate(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate without default values that are PostgreSQL-specific
	err = db.Exec(`CREATE TABLE IF NOT EXISTS todo_lists (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error
	assert.NoError(t, err)

	t.Run("generates UUID if not set", func(t *testing.T) {
		list := &TodoList{
			UserID:      uuid.New(),
			Name:        "Test List",
			Description: "Test Description",
		}

		err := db.Create(list).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, list.ID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		list := &TodoList{
			ID:          existingID,
			UserID:      uuid.New(),
			Name:        "Test List 2",
			Description: "Test Description",
		}

		err := db.Create(list).Error
		assert.NoError(t, err)
		assert.Equal(t, existingID, list.ID)
	})
}

func TestTodoBeforeCreate(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Create tables manually
	err = db.Exec(`CREATE TABLE IF NOT EXISTS todo_lists (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error
	assert.NoError(t, err)

	err = db.Exec(`CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		list_id TEXT NOT NULL,
		description TEXT NOT NULL,
		priority TEXT NOT NULL,
		due_date DATETIME,
		completed INTEGER DEFAULT 0,
		completed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		FOREIGN KEY(list_id) REFERENCES todo_lists(id)
	)`).Error
	assert.NoError(t, err)

	// Create a list first
	list := &TodoList{
		UserID: uuid.New(),
		Name:   "Test List",
	}
	err = db.Create(list).Error
	assert.NoError(t, err)

	t.Run("generates UUID if not set", func(t *testing.T) {
		todo := &Todo{
			ListID:      list.ID,
			Description: "Test Todo",
			Priority:    PriorityHigh,
		}

		err := db.Create(todo).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, todo.ID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		todo := &Todo{
			ID:          existingID,
			ListID:      list.ID,
			Description: "Test Todo 2",
			Priority:    PriorityMedium,
		}

		err := db.Create(todo).Error
		assert.NoError(t, err)
		assert.Equal(t, existingID, todo.ID)
	})
}

func TestTodoListModel(t *testing.T) {
	list := TodoList{
		ID:          uuid.New(),
		Name:        "Work Tasks",
		Description: "Tasks for work",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		TodoCount:   5,
	}

	assert.NotEqual(t, uuid.Nil, list.ID)
	assert.Equal(t, "Work Tasks", list.Name)
	assert.Equal(t, "Tasks for work", list.Description)
	assert.Equal(t, 5, list.TodoCount)
}

func TestTodoModel(t *testing.T) {
	listID := uuid.New()
	dueDate := time.Now().Add(24 * time.Hour)

	todo := Todo{
		ID:          uuid.New(),
		ListID:      listID,
		Description: "Complete project",
		Priority:    PriorityHigh,
		DueDate:     &dueDate,
		Completed:   false,
		CompletedAt: nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, todo.ID)
	assert.Equal(t, listID, todo.ListID)
	assert.Equal(t, "Complete project", todo.Description)
	assert.Equal(t, PriorityHigh, todo.Priority)
	assert.NotNil(t, todo.DueDate)
	assert.False(t, todo.Completed)
	assert.Nil(t, todo.CompletedAt)
}

func TestCreateTodoListRequest(t *testing.T) {
	req := CreateTodoListRequest{
		Name:        "Shopping List",
		Description: "Groceries to buy",
	}

	assert.Equal(t, "Shopping List", req.Name)
	assert.Equal(t, "Groceries to buy", req.Description)
}

func TestUpdateTodoListRequest(t *testing.T) {
	newName := "Updated Name"
	newDesc := "Updated Description"

	req := UpdateTodoListRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	assert.NotNil(t, req.Name)
	assert.Equal(t, "Updated Name", *req.Name)
	assert.NotNil(t, req.Description)
	assert.Equal(t, "Updated Description", *req.Description)
}

func TestCreateTodoRequest(t *testing.T) {
	dueDate := time.Now().Add(48 * time.Hour)

	req := CreateTodoRequest{
		Description: "Buy milk",
		Priority:    PriorityMedium,
		DueDate:     &dueDate,
	}

	assert.Equal(t, "Buy milk", req.Description)
	assert.Equal(t, PriorityMedium, req.Priority)
	assert.NotNil(t, req.DueDate)
}

func TestUpdateTodoRequest(t *testing.T) {
	newDesc := "Updated description"
	newPriority := PriorityLow
	completed := true
	dueDate := time.Now().Add(24 * time.Hour)

	req := UpdateTodoRequest{
		Description: &newDesc,
		Priority:    &newPriority,
		DueDate:     &dueDate,
		Completed:   &completed,
	}

	assert.NotNil(t, req.Description)
	assert.Equal(t, "Updated description", *req.Description)
	assert.NotNil(t, req.Priority)
	assert.Equal(t, PriorityLow, *req.Priority)
	assert.NotNil(t, req.DueDate)
	assert.NotNil(t, req.Completed)
	assert.True(t, *req.Completed)
}

func TestPagination(t *testing.T) {
	pagination := Pagination{
		Page:       2,
		Limit:      20,
		TotalPages: 5,
		TotalItems: 95,
	}

	assert.Equal(t, 2, pagination.Page)
	assert.Equal(t, 20, pagination.Limit)
	assert.Equal(t, 5, pagination.TotalPages)
	assert.Equal(t, 95, pagination.TotalItems)
}

func TestErrorResponse(t *testing.T) {
	err := ErrorResponse{
		Code:    "NOT_FOUND",
		Message: "Resource not found",
		Details: map[string]interface{}{
			"resource": "list",
			"id":       "123",
		},
	}

	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "Resource not found", err.Message)
	assert.NotNil(t, err.Details)
	assert.Equal(t, "list", err.Details["resource"])
}
