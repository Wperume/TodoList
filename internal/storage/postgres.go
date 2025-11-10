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
func (s *PostgresStorage) CreateList(req models.CreateTodoListRequest) (*models.TodoList, error) {
	// Check if list with same name exists
	var existing models.TodoList
	result := s.db.Where("name = ?", req.Name).First(&existing)
	if result.Error == nil {
		return nil, ErrListNameExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	list := &models.TodoList{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.db.Create(list).Error; err != nil {
		return nil, err
	}

	list.TodoCount = 0
	return list, nil
}

// GetAllLists retrieves all todo lists with pagination
func (s *PostgresStorage) GetAllLists(page, limit int) ([]models.TodoList, *models.Pagination, error) {
	var lists []models.TodoList
	var totalItems int64

	// Count total items
	if err := s.db.Model(&models.TodoList{}).Count(&totalItems).Error; err != nil {
		return nil, nil, err
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	// Fetch paginated lists
	if err := s.db.Order("created_at DESC").
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

// GetListByID retrieves a todo list by ID
func (s *PostgresStorage) GetListByID(id uuid.UUID) (*models.TodoList, error) {
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", id).Error; err != nil {
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

// UpdateList updates an existing todo list
func (s *PostgresStorage) UpdateList(id uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error) {
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	// Check if new name conflicts with existing list
	if req.Name != nil && *req.Name != list.Name {
		var existing models.TodoList
		result := s.db.Where("name = ? AND id != ?", *req.Name, id).First(&existing)
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

// DeleteList deletes a todo list and all its todos
func (s *PostgresStorage) DeleteList(id uuid.UUID) error {
	result := s.db.Delete(&models.TodoList{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrListNotFound
	}
	return nil
}

// CreateTodo creates a new todo in a list
func (s *PostgresStorage) CreateTodo(listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	// Check if list exists
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", listID).Error; err != nil {
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

// GetTodosByList retrieves all todos in a list with filtering and sorting
func (s *PostgresStorage) GetTodosByList(listID uuid.UUID, priority *models.Priority, completed *bool, sortBy, sortOrder string) ([]models.Todo, error) {
	// Check if list exists
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", listID).Error; err != nil {
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

// GetTodoByID retrieves a specific todo
func (s *PostgresStorage) GetTodoByID(listID, todoID uuid.UUID) (*models.Todo, error) {
	// Check if list exists
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", listID).Error; err != nil {
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

// UpdateTodo updates an existing todo
func (s *PostgresStorage) UpdateTodo(listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	// Check if list exists
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", listID).Error; err != nil {
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

// DeleteTodo deletes a todo
func (s *PostgresStorage) DeleteTodo(listID, todoID uuid.UUID) error {
	// Check if list exists
	var list models.TodoList
	if err := s.db.First(&list, "id = ?", listID).Error; err != nil {
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
