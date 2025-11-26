package handlers

import (
	"net/http"
	"strconv"

	"todolist-api/internal/middleware"
	"todolist-api/internal/models"
	"todolist-api/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListHandler handles todo list operations
type ListHandler struct {
	storage storage.Store
}

// NewListHandler creates a new list handler
func NewListHandler(store storage.Store) *ListHandler {
	return &ListHandler{storage: store}
}

// GetAllLists handles GET /lists
// @Summary Get all todo lists
// @Description Get all todo lists for the authenticated user with pagination
// @Tags Lists
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} models.PaginatedListsResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists [get]
func (h *ListHandler) GetAllLists(c *gin.Context) {
	// Get user ID (or default for unauthenticated access)
	userID := middleware.GetUserIDOrDefault(c)

	// Parse query parameters
	page, pageErr := strconv.Atoi(c.DefaultQuery("page", "1"))
	if pageErr != nil || page < 1 {
		page = 1
	}

	limit, limitErr := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limitErr != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	lists, pagination, err := h.storage.GetAllLists(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve lists",
		})
		return
	}

	c.JSON(http.StatusOK, models.PaginatedListsResponse{
		Data:       lists,
		Pagination: pagination,
	})
}

// CreateList handles POST /lists
// @Summary Create a new todo list
// @Description Create a new todo list for the authenticated user
// @Tags Lists
// @Accept json
// @Produce json
// @Param request body models.CreateTodoListRequest true "List details"
// @Success 201 {object} models.TodoList
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists [post]
func (h *ListHandler) CreateList(c *gin.Context) {
	// Get user ID (or default for unauthenticated access)
	userID := middleware.GetUserIDOrDefault(c)

	var req models.CreateTodoListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	list, err := h.storage.CreateList(userID, req)
	if err != nil {
		if err == storage.ErrListNameExists {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Code:    "LIST_NAME_EXISTS",
				Message: "A list with this name already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create list",
		})
		return
	}

	c.JSON(http.StatusCreated, list)
}

// GetListByID handles GET /lists/:listId
// @Summary Get a specific todo list
// @Description Get a specific todo list by ID
// @Tags Lists
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Success 200 {object} models.TodoList
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId} [get]
func (h *ListHandler) GetListByID(c *gin.Context) {
	// Get user ID (or default for unauthenticated access)
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	list, err := h.storage.GetListByID(userID, listID)
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
			Message: "Failed to retrieve list",
		})
		return
	}

	c.JSON(http.StatusOK, list)
}

// UpdateList handles PUT /lists/:listId
// @Summary Update a todo list
// @Description Update a specific todo list
// @Tags Lists
// @Accept json
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Param request body models.UpdateTodoListRequest true "Updated list details"
// @Success 200 {object} models.TodoList
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId} [put]
func (h *ListHandler) UpdateList(c *gin.Context) {
	// Get user ID (or default for unauthenticated access)
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	var req models.UpdateTodoListRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": bindErr.Error()},
		})
		return
	}

	list, err := h.storage.UpdateList(userID, listID, req)
	if err != nil {
		if err == storage.ErrListNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "LIST_NOT_FOUND",
				Message: "The requested todo list was not found",
			})
			return
		}
		if err == storage.ErrListNameExists {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Code:    "LIST_NAME_EXISTS",
				Message: "A list with this name already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update list",
		})
		return
	}

	c.JSON(http.StatusOK, list)
}

// DeleteList handles DELETE /lists/:listId
// @Summary Delete a todo list
// @Description Delete a specific todo list and all its todos
// @Tags Lists
// @Produce json
// @Param listId path string true "List ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security Bearer
// @Router /lists/{listId} [delete]
func (h *ListHandler) DeleteList(c *gin.Context) {
	// Get user ID (or default for unauthenticated access)
	userID := middleware.GetUserIDOrDefault(c)

	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	err = h.storage.DeleteList(userID, listID)
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
			Message: "Failed to delete list",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
