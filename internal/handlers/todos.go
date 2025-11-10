package handlers

import (
	"net/http"

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
	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	// Parse query parameters
	var priority *models.Priority
	if priorityStr := c.Query("priority"); priorityStr != "" {
		p := models.Priority(priorityStr)
		if p != models.PriorityLow && p != models.PriorityMedium && p != models.PriorityHigh {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    "INVALID_PRIORITY",
				Message: "Priority must be one of: low, medium, high",
			})
			return
		}
		priority = &p
	}

	var completed *bool
	if completedStr := c.Query("completed"); completedStr != "" {
		if completedStr == "true" {
			t := true
			completed = &t
		} else if completedStr == "false" {
			f := false
			completed = &f
		} else {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    "INVALID_COMPLETED",
				Message: "Completed must be true or false",
			})
			return
		}
	}

	sortBy := c.DefaultQuery("sortBy", "createdAt")
	sortOrder := c.DefaultQuery("sortOrder", "asc")

	// Validate sort parameters
	if sortBy != "dueDate" && sortBy != "priority" && sortBy != "createdAt" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_SORT_BY",
			Message: "sortBy must be one of: dueDate, priority, createdAt",
		})
		return
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_SORT_ORDER",
			Message: "sortOrder must be asc or desc",
		})
		return
	}

	todos, err := h.storage.GetTodosByList(listID, priority, completed, sortBy, sortOrder)
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
	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	var req models.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	todo, err := h.storage.CreateTodo(listID, req)
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

	todo, err := h.storage.GetTodoByID(listID, todoID)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	todo, err := h.storage.UpdateTodo(listID, todoID, req)
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

	err = h.storage.DeleteTodo(listID, todoID)
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
