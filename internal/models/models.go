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

// UserRole represents the role of a user
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User represents a registered user
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	PasswordHash string         `gorm:"not null;size:255" json:"-"` // Never expose password hash
	FirstName    string         `gorm:"size:100" json:"firstName,omitempty"`
	LastName     string         `gorm:"size:100" json:"lastName,omitempty"`
	Role         UserRole       `gorm:"type:varchar(20);not null;default:'user'" json:"role"`
	IsActive     bool           `gorm:"default:true;index" json:"isActive"`
	LastLoginAt  *time.Time     `gorm:"type:timestamp" json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	TodoLists    []TodoList     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// BeforeCreate hook to generate UUID if not set
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	Token     string         `gorm:"uniqueIndex;not null;size:255" json:"-"` // Hashed token
	ExpiresAt time.Time      `gorm:"not null;index" json:"expiresAt"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	RevokedAt *time.Time     `gorm:"type:timestamp;index" json:"revokedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// BeforeCreate hook to generate UUID if not set
func (rt *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}

// TodoList represents a named list containing todos
type TodoList struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	Name        string         `gorm:"not null;size:100;index:idx_user_list_name,unique" json:"name" binding:"required,min=1,max=100"`
	Description string         `gorm:"size:500" json:"description,omitempty" binding:"max=500"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	User        User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Todos       []Todo         `gorm:"foreignKey:ListID;constraint:OnDelete:CASCADE" json:"-"`
	TodoCount   int            `gorm:"-" json:"todoCount"`
}

// BeforeCreate hook to generate UUID if not set
func (t *TodoList) BeforeCreate(_ *gorm.DB) error {
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
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
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
func (t *Todo) BeforeCreate(_ *gorm.DB) error {
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

// Authentication DTOs

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email,max=255"`
	Password  string `json:"password" binding:"required,min=8,max=72"` // bcrypt max is 72 bytes
	FirstName string `json:"firstName,omitempty" binding:"max=100"`
	LastName  string `json:"lastName,omitempty" binding:"max=100"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"` // Always "Bearer"
	ExpiresIn    int       `json:"expiresIn"` // Access token expiry in seconds
	User         *UserInfo `json:"user"`
}

// UserInfo represents public user information (safe to expose)
type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	Role      UserRole  `json:"role"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName *string `json:"firstName,omitempty" binding:"omitempty,max=100"`
	LastName  *string `json:"lastName,omitempty" binding:"omitempty,max=100"`
}
