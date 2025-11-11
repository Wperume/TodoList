package storage

import (
	"errors"
	"sort"
	"sync"
	"time"

	"todolist-api/internal/models"

	"github.com/google/uuid"
)

var (
	ErrListNotFound      = errors.New("todo list not found")
	ErrTodoNotFound      = errors.New("todo not found")
	ErrListNameExists    = errors.New("list with this name already exists")
	ErrInvalidPriority   = errors.New("invalid priority value")
	ErrInvalidSortField  = errors.New("invalid sort field")
)

// Storage provides in-memory storage for todo lists and todos
type Storage struct {
	mu    sync.RWMutex
	lists map[uuid.UUID]*models.TodoList // maps list ID to list
	todos map[uuid.UUID]*models.Todo     // maps todo ID to todo
}

// NewStorage creates a new in-memory storage instance
func NewStorage() *Storage {
	return &Storage{
		lists: make(map[uuid.UUID]*models.TodoList),
		todos: make(map[uuid.UUID]*models.Todo),
	}
}

// CreateList creates a new todo list for a specific user
func (s *Storage) CreateList(userID uuid.UUID, req models.CreateTodoListRequest) (*models.TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if list with same name exists for this user
	for _, list := range s.lists {
		if list.UserID == userID && list.Name == req.Name {
			return nil, ErrListNameExists
		}
	}

	now := time.Now()
	list := &models.TodoList{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
		TodoCount:   0,
	}

	s.lists[list.ID] = list
	return list, nil
}

// GetAllLists retrieves all todo lists for a specific user with pagination
func (s *Storage) GetAllLists(userID uuid.UUID, page, limit int) ([]models.TodoList, *models.Pagination, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice, filtering by user
	allLists := make([]models.TodoList, 0, len(s.lists))
	for _, list := range s.lists {
		if list.UserID == userID {
			listCopy := *list
			listCopy.TodoCount = s.countTodosInList(list.ID)
			allLists = append(allLists, listCopy)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(allLists, func(i, j int) bool {
		return allLists[i].CreatedAt.After(allLists[j].CreatedAt)
	})

	// Calculate pagination
	totalItems := len(allLists)
	totalPages := (totalItems + limit - 1) / limit
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	paginatedLists := allLists[start:end]
	pagination := &models.Pagination{
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		TotalItems: totalItems,
	}

	return paginatedLists, pagination, nil
}

// GetListByID retrieves a todo list by ID for a specific user
func (s *Storage) GetListByID(userID, listID uuid.UUID) (*models.TodoList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	listCopy := *list
	listCopy.TodoCount = s.countTodosInList(listID)
	return &listCopy, nil
}

// UpdateList updates an existing todo list for a specific user
func (s *Storage) UpdateList(userID, listID uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	// Check if new name conflicts with existing list for this user
	if req.Name != nil && *req.Name != list.Name {
		for _, l := range s.lists {
			if l.UserID == userID && l.ID != listID && l.Name == *req.Name {
				return nil, ErrListNameExists
			}
		}
		list.Name = *req.Name
	}

	if req.Description != nil {
		list.Description = *req.Description
	}

	list.UpdatedAt = time.Now()

	listCopy := *list
	listCopy.TodoCount = s.countTodosInList(listID)
	return &listCopy, nil
}

// DeleteList deletes a todo list and all its todos for a specific user
func (s *Storage) DeleteList(userID, listID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return ErrListNotFound
	}

	// Delete all todos in this list
	for todoID, todo := range s.todos {
		if todo.ListID == listID {
			delete(s.todos, todoID)
		}
	}

	delete(s.lists, listID)
	return nil
}

// CreateTodo creates a new todo in a list owned by a specific user
func (s *Storage) CreateTodo(userID, listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	now := time.Now()
	todo := &models.Todo{
		ID:          uuid.New(),
		ListID:      listID,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		Completed:   false,
		CompletedAt: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.todos[todo.ID] = todo
	return todo, nil
}

// GetTodosByList retrieves all todos in a list owned by a specific user with filtering and sorting
func (s *Storage) GetTodosByList(userID, listID uuid.UUID, priority *models.Priority, completed *bool, sortBy, sortOrder string) ([]models.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	// Filter todos
	result := make([]models.Todo, 0)
	for _, todo := range s.todos {
		if todo.ListID != listID {
			continue
		}

		// Apply filters
		if priority != nil && todo.Priority != *priority {
			continue
		}
		if completed != nil && todo.Completed != *completed {
			continue
		}

		result = append(result, *todo)
	}

	// Sort todos
	if err := sortTodos(result, sortBy, sortOrder); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTodoByID retrieves a specific todo from a list owned by a specific user
func (s *Storage) GetTodoByID(userID, listID, todoID uuid.UUID) (*models.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	todo, exists := s.todos[todoID]
	if !exists || todo.ListID != listID {
		return nil, ErrTodoNotFound
	}

	todoCopy := *todo
	return &todoCopy, nil
}

// UpdateTodo updates an existing todo in a list owned by a specific user
func (s *Storage) UpdateTodo(userID, listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return nil, ErrListNotFound
	}

	todo, exists := s.todos[todoID]
	if !exists || todo.ListID != listID {
		return nil, ErrTodoNotFound
	}

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

		// Set or clear CompletedAt timestamp
		if *req.Completed && !wasCompleted {
			now := time.Now()
			todo.CompletedAt = &now
		} else if !*req.Completed && wasCompleted {
			todo.CompletedAt = nil
		}
	}

	todo.UpdatedAt = time.Now()

	todoCopy := *todo
	return &todoCopy, nil
}

// DeleteTodo deletes a todo from a list owned by a specific user
func (s *Storage) DeleteTodo(userID, listID, todoID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.lists[listID]
	if !exists || list.UserID != userID {
		return ErrListNotFound
	}

	todo, exists := s.todos[todoID]
	if !exists || todo.ListID != listID {
		return ErrTodoNotFound
	}

	delete(s.todos, todoID)
	return nil
}

// countTodosInList counts todos in a list (must be called with lock held)
func (s *Storage) countTodosInList(listID uuid.UUID) int {
	count := 0
	for _, todo := range s.todos {
		if todo.ListID == listID {
			count++
		}
	}
	return count
}

// sortTodos sorts todos based on the specified field and order
func sortTodos(todos []models.Todo, sortBy, sortOrder string) error {
	// Validate sort field before creating the comparison function
	if sortBy != "dueDate" && sortBy != "priority" && sortBy != "createdAt" && sortBy != "" {
		return ErrInvalidSortField
	}

	less := func(i, j int) bool {
		var result bool
		switch sortBy {
		case "dueDate":
			// Handle nil due dates (put them at the end)
			if todos[i].DueDate == nil && todos[j].DueDate == nil {
				result = todos[i].CreatedAt.Before(todos[j].CreatedAt)
			} else if todos[i].DueDate == nil {
				result = false
			} else if todos[j].DueDate == nil {
				result = true
			} else {
				result = todos[i].DueDate.Before(*todos[j].DueDate)
			}
		case "priority":
			priorityOrder := map[models.Priority]int{
				models.PriorityHigh:   3,
				models.PriorityMedium: 2,
				models.PriorityLow:    1,
			}
			result = priorityOrder[todos[i].Priority] > priorityOrder[todos[j].Priority]
		case "createdAt", "":
			result = todos[i].CreatedAt.Before(todos[j].CreatedAt)
		}

		if sortOrder == "desc" {
			return !result
		}
		return result
	}

	sort.Slice(todos, less)
	return nil
}
