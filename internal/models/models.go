package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Priority represents the priority level of a todo
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// TodoList represents a named list containing todos
type TodoList struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null;size:100" json:"name" binding:"required,min=1,max=100"`
	Description string         `gorm:"size:500" json:"description,omitempty" binding:"max=500"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Todos       []Todo         `gorm:"foreignKey:ListID;constraint:OnDelete:CASCADE" json:"-"`
	TodoCount   int            `gorm:"-" json:"todoCount"`
}

// BeforeCreate hook to generate UUID if not set
func (t *TodoList) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// CreateTodoListRequest represents the request to create a new todo list
type CreateTodoListRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty" binding:"max=500"`
}

// UpdateTodoListRequest represents the request to update a todo list
type UpdateTodoListRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500"`
}

// Todo represents a todo item within a list
type Todo struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ListID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"listId"`
	Description string         `gorm:"not null;size:500" json:"description" binding:"required,min=1,max=500"`
	Priority    Priority       `gorm:"type:varchar(10);not null" json:"priority" binding:"required,oneof=low medium high"`
	DueDate     *time.Time     `gorm:"type:timestamp" json:"dueDate,omitempty"`
	Completed   bool           `gorm:"default:false;index" json:"completed"`
	CompletedAt *time.Time     `gorm:"type:timestamp" json:"completedAt,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook to generate UUID if not set
func (t *Todo) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// CreateTodoRequest represents the request to create a new todo
type CreateTodoRequest struct {
	Description string     `json:"description" binding:"required,min=1,max=500"`
	Priority    Priority   `json:"priority" binding:"required,oneof=low medium high"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
}

// UpdateTodoRequest represents the request to update a todo
type UpdateTodoRequest struct {
	Description *string    `json:"description,omitempty" binding:"omitempty,min=1,max=500"`
	Priority    *Priority  `json:"priority,omitempty" binding:"omitempty,oneof=low medium high"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
}

// PaginatedListsResponse represents a paginated response of todo lists
type PaginatedListsResponse struct {
	Data       []TodoList  `json:"data"`
	Pagination *Pagination `json:"pagination"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}
