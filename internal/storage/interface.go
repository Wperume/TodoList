package storage

import (
	"todolist-api/internal/models"

	"github.com/google/uuid"
)

// Store defines the interface for storage operations
type Store interface {
	// List operations
	CreateList(req models.CreateTodoListRequest) (*models.TodoList, error)
	GetAllLists(page, limit int) ([]models.TodoList, *models.Pagination, error)
	GetListByID(id uuid.UUID) (*models.TodoList, error)
	UpdateList(id uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error)
	DeleteList(id uuid.UUID) error

	// Todo operations
	CreateTodo(listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error)
	GetTodosByList(listID uuid.UUID, priority *models.Priority, completed *bool, sortBy, sortOrder string) ([]models.Todo, error)
	GetTodoByID(listID, todoID uuid.UUID) (*models.Todo, error)
	UpdateTodo(listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error)
	DeleteTodo(listID, todoID uuid.UUID) error
}
