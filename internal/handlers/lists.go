package handlers

import (
	"net/http"
	"strconv"

	"todolist-api/internal/models"
	"todolist-api/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListHandler handles todo list operations
type ListHandler struct {
	storage *storage.Storage
}

// NewListHandler creates a new list handler
func NewListHandler(storage *storage.Storage) *ListHandler {
	return &ListHandler{storage: storage}
}

// GetAllLists handles GET /lists
func (h *ListHandler) GetAllLists(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	lists, pagination, err := h.storage.GetAllLists(page, limit)
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
func (h *ListHandler) CreateList(c *gin.Context) {
	var req models.CreateTodoListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	list, err := h.storage.CreateList(req)
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
func (h *ListHandler) GetListByID(c *gin.Context) {
	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	list, err := h.storage.GetListByID(listID)
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
func (h *ListHandler) UpdateList(c *gin.Context) {
	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	var req models.UpdateTodoListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	list, err := h.storage.UpdateList(listID, req)
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
func (h *ListHandler) DeleteList(c *gin.Context) {
	listID, err := uuid.Parse(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_LIST_ID",
			Message: "Invalid list ID format",
		})
		return
	}

	err = h.storage.DeleteList(listID)
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
