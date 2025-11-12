package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"todolist-api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to open test database")

	// Run migrations manually for SQLite compatibility
	// SQLite doesn't support PostgreSQL UUID functions
	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		first_name TEXT,
		last_name TEXT,
		role TEXT NOT NULL DEFAULT 'user',
		is_active INTEGER DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error
	require.NoError(t, err, "Failed to create users table")

	err = db.Exec(`CREATE TABLE IF NOT EXISTS todo_lists (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	)`).Error
	require.NoError(t, err, "Failed to create todo_lists table")

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
		FOREIGN KEY(list_id) REFERENCES todo_lists(id) ON DELETE CASCADE
	)`).Error
	require.NoError(t, err, "Failed to create todos table")

	err = db.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME,
		revoked_at DATETIME,
		deleted_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	)`).Error
	require.NoError(t, err, "Failed to create refresh_tokens table")

	// Create indexes
	db.Exec(`CREATE INDEX idx_users_deleted_at ON users(deleted_at)`)
	db.Exec(`CREATE INDEX idx_users_is_active ON users(is_active)`)
	db.Exec(`CREATE INDEX idx_todo_lists_user_id ON todo_lists(user_id)`)
	db.Exec(`CREATE INDEX idx_todo_lists_deleted_at ON todo_lists(deleted_at)`)
	db.Exec(`CREATE UNIQUE INDEX idx_user_list_name ON todo_lists(user_id, name) WHERE deleted_at IS NULL`)
	db.Exec(`CREATE INDEX idx_todos_list_id ON todos(list_id)`)
	db.Exec(`CREATE INDEX idx_todos_completed ON todos(completed)`)
	db.Exec(`CREATE INDEX idx_todos_deleted_at ON todos(deleted_at)`)
	db.Exec(`CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id)`)
	db.Exec(`CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token)`)
	db.Exec(`CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at)`)
	db.Exec(`CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at)`)
	db.Exec(`CREATE INDEX idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at)`)

	// Insert test user (used by postgres_test.go)
	err = db.Exec(`INSERT INTO users (id, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com', '$2a$10$test', 'user', 1, datetime('now'), datetime('now'))`).Error
	require.NoError(t, err, "Failed to create test user")

	return db
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}

// CreateTestList creates a test TodoList
func CreateTestList(name, description string) *models.TodoList {
	return &models.TodoList{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		TodoCount:   0,
	}
}

// CreateTestTodo creates a test Todo
func CreateTestTodo(listID uuid.UUID, description string, priority models.Priority) *models.Todo {
	return &models.Todo{
		ID:          uuid.New(),
		ListID:      listID,
		Description: description,
		Priority:    priority,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestTodoWithDueDate creates a test Todo with a due date
func CreateTestTodoWithDueDate(listID uuid.UUID, description string, priority models.Priority, dueDate time.Time) *models.Todo {
	todo := CreateTestTodo(listID, description, priority)
	todo.DueDate = &dueDate
	return todo
}

// MakeJSONRequest creates an HTTP request with JSON body
func MakeJSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ParseJSONResponse parses a JSON response into a target structure
func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), target)
	require.NoError(t, err, "Failed to parse JSON response")
}

// AssertJSONEqual asserts that two JSON objects are equal
func AssertJSONEqual(t *testing.T, expected, actual interface{}) {
	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err, "Failed to marshal expected value")

	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err, "Failed to marshal actual value")

	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// TimePtr returns a pointer to a time.Time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// PriorityPtr returns a pointer to a Priority value
func PriorityPtr(p models.Priority) *models.Priority {
	return &p
}
