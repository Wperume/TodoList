package storage

import (
	"errors"

	"todolist-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PostgresStorage implements storage using PostgreSQL with GORM
type PostgresStorage struct {
	db *gorm.DB
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(db *gorm.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// CreateList creates a new todo list
func (s *PostgresStorage) CreateList(userID uuid.UUID, req models.CreateTodoListRequest) (*models.TodoList, error) {
	// Check if list with same name exists for this user
	var existing models.TodoList
	result := s.db.Where("user_id = ? AND name = ?", userID, req.Name).First(&existing)
	if result.Error == nil {
		return nil, ErrListNameExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	list := &models.TodoList{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.db.Create(list).Error; err != nil {
		return nil, err
	}

	list.TodoCount = 0
	return list, nil
}

// GetAllLists retrieves all todo lists with pagination for a specific user
func (s *PostgresStorage) GetAllLists(userID uuid.UUID, page, limit int) ([]models.TodoList, *models.Pagination, error) {
	var lists []models.TodoList
	var totalItems int64

	// Count total items for this user
	if err := s.db.Model(&models.TodoList{}).Where("user_id = ?", userID).Count(&totalItems).Error; err != nil {
		return nil, nil, err
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	// Fetch paginated lists for this user
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&lists).Error; err != nil {
		return nil, nil, err
	}

	// Get todo counts for each list
	for i := range lists {
		var count int64
		s.db.Model(&models.Todo{}).Where("list_id = ?", lists[i].ID).Count(&count)
		lists[i].TodoCount = int(count)
	}

	pagination := &models.Pagination{
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
	}

	return lists, pagination, nil
}

// GetListByID retrieves a todo list by ID for a specific user
func (s *PostgresStorage) GetListByID(userID, listID uuid.UUID) (*models.TodoList, error) {
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	// Get todo count
	var count int64
	s.db.Model(&models.Todo{}).Where("list_id = ?", list.ID).Count(&count)
	list.TodoCount = int(count)

	return &list, nil
}

// UpdateList updates an existing todo list for a specific user
func (s *PostgresStorage) UpdateList(userID, listID uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error) {
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	// Check if new name conflicts with existing list for this user
	if req.Name != nil && *req.Name != list.Name {
		var existing models.TodoList
		result := s.db.Where("user_id = ? AND name = ? AND id != ?", userID, *req.Name, listID).First(&existing)
		if result.Error == nil {
			return nil, ErrListNameExists
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}
		list.Name = *req.Name
	}

	if req.Description != nil {
		list.Description = *req.Description
	}

	if err := s.db.Save(&list).Error; err != nil {
		return nil, err
	}

	// Get todo count
	var count int64
	s.db.Model(&models.Todo{}).Where("list_id = ?", list.ID).Count(&count)
	list.TodoCount = int(count)

	return &list, nil
}

// DeleteList deletes a todo list and all its todos for a specific user
func (s *PostgresStorage) DeleteList(userID, listID uuid.UUID) error {
	result := s.db.Where("id = ? AND user_id = ?", listID, userID).Delete(&models.TodoList{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrListNotFound
	}
	return nil
}

// CreateTodo creates a new todo in a list owned by a specific user
func (s *PostgresStorage) CreateTodo(userID, listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	// Check if list exists and belongs to user
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	todo := &models.Todo{
		ListID:      listID,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		Completed:   false,
	}

	if err := s.db.Create(todo).Error; err != nil {
		return nil, err
	}

	return todo, nil
}

// GetTodosByList retrieves all todos in a list owned by a specific user with filtering and sorting
func (s *PostgresStorage) GetTodosByList(userID, listID uuid.UUID, priority *models.Priority, completed *bool, sortBy, sortOrder string) ([]models.Todo, error) {
	// Check if list exists and belongs to user
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	// Build query
	query := s.db.Where("list_id = ?", listID)

	// Apply filters
	if priority != nil {
		query = query.Where("priority = ?", *priority)
	}
	if completed != nil {
		query = query.Where("completed = ?", *completed)
	}

	// Apply sorting
	orderClause := buildOrderClause(sortBy, sortOrder)
	query = query.Order(orderClause)

	var todos []models.Todo
	if err := query.Find(&todos).Error; err != nil {
		return nil, err
	}

	return todos, nil
}

// GetTodoByID retrieves a specific todo from a list owned by a specific user
func (s *PostgresStorage) GetTodoByID(userID, listID, todoID uuid.UUID) (*models.Todo, error) {
	// Check if list exists and belongs to user
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	var todo models.Todo
	if err := s.db.Where("id = ? AND list_id = ?", todoID, listID).First(&todo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTodoNotFound
		}
		return nil, err
	}

	return &todo, nil
}

// UpdateTodo updates an existing todo in a list owned by a specific user
func (s *PostgresStorage) UpdateTodo(userID, listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	// Check if list exists and belongs to user
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	var todo models.Todo
	if err := s.db.Where("id = ? AND list_id = ?", todoID, listID).First(&todo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTodoNotFound
		}
		return nil, err
	}

	// Update fields
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Priority != nil {
		todo.Priority = *req.Priority
	}
	if req.DueDate != nil {
		todo.DueDate = req.DueDate
	}
	if req.Completed != nil {
		wasCompleted := todo.Completed
		todo.Completed = *req.Completed

		// Update CompletedAt timestamp
		if *req.Completed && !wasCompleted {
			now := s.db.NowFunc()
			todo.CompletedAt = &now
		} else if !*req.Completed && wasCompleted {
			todo.CompletedAt = nil
		}
	}

	if err := s.db.Save(&todo).Error; err != nil {
		return nil, err
	}

	return &todo, nil
}

// DeleteTodo deletes a todo from a list owned by a specific user
func (s *PostgresStorage) DeleteTodo(userID, listID, todoID uuid.UUID) error {
	// Check if list exists and belongs to user
	var list models.TodoList
	if err := s.db.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrListNotFound
		}
		return err
	}

	result := s.db.Where("id = ? AND list_id = ?", todoID, listID).Delete(&models.Todo{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTodoNotFound
	}
	return nil
}

// buildOrderClause creates the ORDER BY clause for sorting
func buildOrderClause(sortBy, sortOrder string) string {
	// Map sort fields to actual column names
	var column string
	switch sortBy {
	case "dueDate":
		column = "due_date"
	case "priority":
		// PostgreSQL sorting with CASE for priority ordering
		if sortOrder == "desc" {
			return "CASE priority WHEN 'low' THEN 1 WHEN 'medium' THEN 2 WHEN 'high' THEN 3 END DESC"
		}
		return "CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END ASC"
	case "createdAt", "":
		column = "created_at"
	default:
		column = "created_at"
	}

	if sortOrder == "desc" {
		return column + " DESC"
	}
	return column + " ASC"
}
