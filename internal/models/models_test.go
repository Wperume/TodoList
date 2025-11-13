package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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

func TestUserBeforeCreate(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Create users table
	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		is_active INTEGER DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error
	assert.NoError(t, err)

	t.Run("generates UUID if not set", func(t *testing.T) {
		user := &User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			FirstName:    "Test",
			LastName:     "User",
			Role:         "user",
		}

		err := db.Create(user).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		user := &User{
			ID:           existingID,
			Email:        "test2@example.com",
			PasswordHash: "hashed_password",
			FirstName:    "Test",
			LastName:     "User",
			Role:         "user",
		}

		err := db.Create(user).Error
		assert.NoError(t, err)
		assert.Equal(t, existingID, user.ID)
	})
}

func TestRefreshTokenBeforeCreate(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Create users table first (for foreign key)
	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		is_active INTEGER DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error
	assert.NoError(t, err)

	// Create refresh_tokens table
	err = db.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME,
		revoked_at DATETIME,
		deleted_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id)
	)`).Error
	assert.NoError(t, err)

	// Create a user first
	user := &User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "user",
	}
	err = db.Create(user).Error
	assert.NoError(t, err)

	t.Run("generates UUID if not set", func(t *testing.T) {
		token := &RefreshToken{
			UserID:    user.ID,
			Token:     "hashed_token_1",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err := db.Create(token).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, token.ID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		token := &RefreshToken{
			ID:        existingID,
			UserID:    user.ID,
			Token:     "hashed_token_2",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err := db.Create(token).Error
		assert.NoError(t, err)
		assert.Equal(t, existingID, token.ID)
	})
}

func TestRefreshTokenIsValid(t *testing.T) {
	now := time.Now()

	t.Run("returns true for valid token", func(t *testing.T) {
		token := &RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "valid_token",
			ExpiresAt: now.Add(24 * time.Hour), // Expires in the future
			RevokedAt: nil,                     // Not revoked
		}

		assert.True(t, token.IsValid())
	})

	t.Run("returns false for expired token", func(t *testing.T) {
		token := &RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "expired_token",
			ExpiresAt: now.Add(-1 * time.Hour), // Expired in the past
			RevokedAt: nil,
		}

		assert.False(t, token.IsValid())
	})

	t.Run("returns false for revoked token", func(t *testing.T) {
		revokedTime := now.Add(-1 * time.Hour)
		token := &RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "revoked_token",
			ExpiresAt: now.Add(24 * time.Hour), // Not expired
			RevokedAt: &revokedTime,            // But revoked
		}

		assert.False(t, token.IsValid())
	})

	t.Run("returns false for revoked and expired token", func(t *testing.T) {
		revokedTime := now.Add(-2 * time.Hour)
		token := &RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "revoked_expired_token",
			ExpiresAt: now.Add(-1 * time.Hour), // Expired
			RevokedAt: &revokedTime,            // And revoked
		}

		assert.False(t, token.IsValid())
	})
}
