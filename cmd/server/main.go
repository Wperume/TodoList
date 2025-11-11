package main

import (
	"os"

	"todolist-api/internal/database"
	"todolist-api/internal/handlers"
	"todolist-api/internal/logging"
	"todolist-api/internal/middleware"
	"todolist-api/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logging first
	logConfig := logging.NewLogConfigFromEnv()
	logging.InitLogger(logConfig)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Check if we should use in-memory storage (for development)
	useInMemory := os.Getenv("USE_MEMORY_STORAGE") == "true"

	var listHandler *handlers.ListHandler
	var todoHandler *handlers.TodoHandler

	if useInMemory {
		logging.Logger.Info("Using in-memory storage")
		store := storage.NewStorage()
		listHandler = handlers.NewListHandler(store)
		todoHandler = handlers.NewTodoHandler(store)
	} else {
		// Initialize PostgreSQL connection
		dbConfig := database.NewConfigFromEnv()
		db, err := database.Connect(dbConfig)
		if err != nil {
			logging.Logger.Fatalf("Failed to connect to database: %v", err)
		}

		// Run migrations
		if err := database.AutoMigrate(db); err != nil {
			logging.Logger.Fatalf("Failed to run migrations: %v", err)
		}

		logging.Logger.Info("PostgreSQL storage initialized successfully")
		// Initialize PostgreSQL storage
		store := storage.NewPostgresStorage(db)
		listHandler = handlers.NewListHandler(store)
		todoHandler = handlers.NewTodoHandler(store)
	}

	// Set up Gin router (without default logger since we'll use our own)
	router := gin.New()
	router.Use(gin.Recovery()) // Add recovery middleware

	// Add request logging middleware
	router.Use(middleware.RequestLogger())

	// Initialize rate limiting
	rateLimitConfig := middleware.NewRateLimitConfigFromEnv()
	router.Use(middleware.GlobalRateLimiter(rateLimitConfig))

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
	logging.Logger.Infof("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		logging.Logger.Fatalf("Failed to start server: %v", err)
	}
}
