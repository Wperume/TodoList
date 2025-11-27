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
// @Summary Get todos in a list
// @Description Get all todos in a specific list with optional filtering and sorting
// @Tags Todos
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param priority query string false "Filter by priority (low, medium, high)"
// @Param completed query boolean false "Filter by completion status"
// @Param flagged query boolean false "Filter by flagged status"
// @Param sort_by query string false "Sort by field (due_date, priority, created_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {array} models.Todo
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId}/todos [get]
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

	flagged, ok := parseFlaggedFilter(c)
	if !ok {
		return
	}

	sortBy, sortOrder, ok := parseSortParams(c)
	if !ok {
		return
	}

	todos, err := h.storage.GetTodosByList(userID, listID, priority, completed, flagged, sortBy, sortOrder)
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
// @Summary Create a new todo
// @Description Create a new todo item in a specific list
// @Tags Todos
// @Accept json
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param request body models.CreateTodoRequest true "Todo details"
// @Success 201 {object} models.Todo
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId}/todos [post]
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
// @Summary Get a specific todo
// @Description Get a specific todo item by ID
// @Tags Todos
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param todoId path string true "Todo ID (UUID)"
// @Success 200 {object} models.Todo
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId}/todos/{todoId} [get]
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
// @Summary Update a todo
// @Description Update a specific todo item
// @Tags Todos
// @Accept json
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param todoId path string true "Todo ID (UUID)"
// @Param request body models.UpdateTodoRequest true "Updated todo details"
// @Success 200 {object} models.Todo
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId}/todos/{todoId} [put]
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
// @Summary Delete a todo
// @Description Delete a specific todo item
// @Tags Todos
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param todoId path string true "Todo ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId}/todos/{todoId} [delete]
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

func parseFlaggedFilter(c *gin.Context) (*bool, bool) {
	flaggedStr := c.Query("flagged")
	if flaggedStr == "" {
		return nil, true
	}

	switch flaggedStr {
	case "true":
		t := true
		return &t, true
	case "false":
		f := false
		return &f, true
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_FLAGGED",
			Message: "Flagged must be true or false",
		})
		return nil, false
	}
}

func parseSortParams(c *gin.Context) (sortBy, sortOrder string, ok bool) {
	sortBy = c.DefaultQuery("sortBy", "createdAt")
	sortOrder = c.DefaultQuery("sortOrder", "asc")

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
