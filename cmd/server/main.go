package main

import (
	"log"
	"os"

	"todolist-api/internal/handlers"
	"todolist-api/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize storage
	store := storage.NewStorage()

	// Initialize handlers
	listHandler := handlers.NewListHandler(store)
	todoHandler := handlers.NewTodoHandler(store)

	// Set up Gin router
	router := gin.Default()

	// API version 1 routes
	v1 := router.Group("/api/v1")
	{
		// Todo List routes
		lists := v1.Group("/lists")
		{
			lists.GET("", listHandler.GetAllLists)
			lists.POST("", listHandler.CreateList)
			lists.GET("/:listId", listHandler.GetListByID)
			lists.PUT("/:listId", listHandler.UpdateList)
			lists.DELETE("/:listId", listHandler.DeleteList)

			// Todo routes (nested under lists)
			lists.GET("/:listId/todos", todoHandler.GetTodosByList)
			lists.POST("/:listId/todos", todoHandler.CreateTodo)
			lists.GET("/:listId/todos/:todoId", todoHandler.GetTodoByID)
			lists.PUT("/:listId/todos/:todoId", todoHandler.UpdateTodo)
			lists.DELETE("/:listId/todos/:todoId", todoHandler.DeleteTodo)
		}
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Start server
	log.Printf("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
