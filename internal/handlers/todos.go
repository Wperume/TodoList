package handlers

import (
	"net/http"

	"todolist-api/internal/middleware"
	"todolist-api/internal/models"
	"todolist-api/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TodoHandler handles todo operations
type TodoHandler struct {
	storage storage.Store
}

// NewTodoHandler creates a new todo handler
func NewTodoHandler(store storage.Store) *TodoHandler {
	return &TodoHandler{storage: store}
}

// GetTodosByList handles GET /lists/:listId/todos
func (h *TodoHandler) GetTodosByList(c *gin.Context) {
	// Get authenticated user ID
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	// Parse and validate query parameters
	priority, ok := parsePriorityFilter(c)
	if !ok {
		return
	}

	completed, ok := parseCompletedFilter(c)
	if !ok {
		return
	}

	sortBy, sortOrder, ok := parseSortParams(c)
	if !ok {
		return
	}

	todos, err := h.storage.GetTodosByList(userID, listID, priority, completed, sortBy, sortOrder)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve todos",
		})
		return
	}

	c.JSON(http.StatusOK, todos)
}

// CreateTodo handles POST /lists/:listId/todos
func (h *TodoHandler) CreateTodo(c *gin.Context) {
	// Get authenticated user ID
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	var req models.CreateTodoRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": bindErr.Error()},
		})
		return
	}

	todo, err := h.storage.CreateTodo(userID, listID, req)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create todo",
		})
		return
	}

	c.JSON(http.StatusCreated, todo)
}

// GetTodoByID handles GET /lists/:listId/todos/:todoId
func (h *TodoHandler) GetTodoByID(c *gin.Context) {
	// Get authenticated user ID
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	todoID, err := uuid.Parse(c.Param("todoId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_TODO_ID",
			Message: "Invalid todo ID format",
		})
		return
	}

	todo, err := h.storage.GetTodoByID(userID, listID, todoID)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		if err == storage.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "TODO_NOT_FOUND",
				Message: "The requested todo was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve todo",
		})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// UpdateTodo handles PUT /lists/:listId/todos/:todoId
func (h *TodoHandler) UpdateTodo(c *gin.Context) {
	// Get authenticated user ID
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	todoID, err := uuid.Parse(c.Param("todoId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_TODO_ID",
			Message: "Invalid todo ID format",
		})
		return
	}

	var req models.UpdateTodoRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": bindErr.Error()},
		})
		return
	}

	todo, err := h.storage.UpdateTodo(userID, listID, todoID, req)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		if err == storage.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "TODO_NOT_FOUND",
				Message: "The requested todo was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update todo",
		})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// DeleteTodo handles DELETE /lists/:listId/todos/:todoId
func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	// Get authenticated user ID
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	todoID, err := uuid.Parse(c.Param("todoId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_TODO_ID",
			Message: "Invalid todo ID format",
		})
		return
	}

	err = h.storage.DeleteTodo(userID, listID, todoID)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		if err == storage.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "TODO_NOT_FOUND",
				Message: "The requested todo was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to delete todo",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper functions for query parameter validation

func parsePriorityFilter(c *gin.Context) (*models.Priority, bool) {
	priorityStr := c.Query("priority")
	if priorityStr == "" {
		return nil, true
	}

	p := models.Priority(priorityStr)
	if p != models.PriorityLow && p != models.PriorityMedium && p != models.PriorityHigh {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_PRIORITY",
			Message: "Priority must be one of: low, medium, high",
		})
		return nil, false
	}
	return &p, true
}

func parseCompletedFilter(c *gin.Context) (*bool, bool) {
	completedStr := c.Query("completed")
	if completedStr == "" {
		return nil, true
	}

	switch completedStr {
	case "true":
		t := true
		return &t, true
	case "false":
		f := false
		return &f, true
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_COMPLETED",
			Message: "Completed must be true or false",
		})
		return nil, false
	}
}

func parseSortParams(c *gin.Context) (string, string, bool) {
	sortBy := c.DefaultQuery("sortBy", "createdAt")
	sortOrder := c.DefaultQuery("sortOrder", "asc")

	if sortBy != "dueDate" && sortBy != "priority" && sortBy != "createdAt" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_SORT_BY",
			Message: "sortBy must be one of: dueDate, priority, createdAt",
		})
		return "", "", false
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_SORT_ORDER",
			Message: "sortOrder must be asc or desc",
		})
		return "", "", false
	}

	return sortBy, sortOrder, true
}
