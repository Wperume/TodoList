package storage

import (
	"todolist-api/internal/models"

	"github.com/google/uuid"
)

// Store defines the interface for storage operations
type Store interface {
	// List operations
	CreateList(userID uuid.UUID, req models.CreateTodoListRequest) (*models.TodoList, error)
	GetAllLists(userID uuid.UUID, page, limit int) ([]models.TodoList, *models.Pagination, error)
	GetListByID(userID, listID uuid.UUID) (*models.TodoList, error)
	UpdateList(userID, listID uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error)
	DeleteList(userID, listID uuid.UUID) error

	// Todo operations
	CreateTodo(userID, listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error)
	GetTodosByList(userID, listID uuid.UUID, priority *models.Priority, completed *bool, sortBy, sortOrder string) ([]models.Todo, error)
	GetTodoByID(userID, listID, todoID uuid.UUID) (*models.Todo, error)
	UpdateTodo(userID, listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error)
	DeleteTodo(userID, listID, todoID uuid.UUID) error
}
