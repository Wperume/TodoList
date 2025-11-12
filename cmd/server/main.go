package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todolist-api/internal/auth"
	"todolist-api/internal/database"
	"todolist-api/internal/handlers"
	"todolist-api/internal/logging"
	"todolist-api/internal/middleware"
	"todolist-api/internal/storage"
	tlsconfig "todolist-api/internal/tls"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "todolist-api/docs" // Import generated docs
)

// @title           TodoList API
// @version         1.0
// @description     A RESTful API for managing todo lists and tasks with user authentication
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@todolist-api.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Authentication
// @tag.description User registration, login, and profile management

// @tag.name Lists
// @tag.description Todo list management operations

// @tag.name Todos
// @tag.description Todo item management operations within lists

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
	var authHandler *handlers.AuthHandler
	var healthHandler *handlers.HealthHandler
	var jwtConfig *auth.JWTConfig
	var db *gorm.DB

	if useInMemory {
		logging.Logger.Info("Using in-memory storage")
		logging.Logger.Warn("In-memory storage does not support authentication - API will run without auth")
		store := storage.NewStorage()
		listHandler = handlers.NewListHandler(store)
		todoHandler = handlers.NewTodoHandler(store)
	} else {
		// Initialize PostgreSQL connection
		dbConfig := database.NewConfigFromEnv()
		var err error
		db, err = database.Connect(dbConfig)
		if err != nil {
			logging.Logger.Fatalf("Failed to connect to database: %v", err)
		}

		// Run migrations
		if err := database.AutoMigrate(db); err != nil {
			logging.Logger.Fatalf("Failed to run migrations: %v", err)
		}

		logging.Logger.Info("PostgreSQL storage initialized successfully")

		// Initialize JWT configuration
		jwtConfig = auth.NewJWTConfigFromEnv()

		// Initialize authentication service
		authService := auth.NewService(db, jwtConfig)
		authHandler = handlers.NewAuthHandler(authService)

		// Initialize PostgreSQL storage
		store := storage.NewPostgresStorage(db)
		listHandler = handlers.NewListHandler(store)
		todoHandler = handlers.NewTodoHandler(store)

		// Initialize health handler with database connection
		healthHandler = handlers.NewHealthHandler(db)
	}

	// Set up Gin router (without default logger since we'll use our own)
	router := gin.New()
	router.Use(gin.Recovery()) // Add recovery middleware

	// Add security headers (should be first)
	router.Use(middleware.SecurityHeaders())

	// Add CORS middleware
	corsConfig := middleware.NewCORSConfigFromEnv()
	router.Use(middleware.CORS(corsConfig))

	// Add request size limit
	securityConfig := middleware.NewSecurityConfigFromEnv()
	router.Use(middleware.RequestSizeLimit(securityConfig.MaxRequestBodySize))

	// Add request logging middleware
	router.Use(middleware.RequestLogger())

	// Add error sanitization (catches panics and sanitizes errors)
	router.Use(middleware.ErrorSanitizer())

	// Initialize rate limiting configuration
	rateLimitConfig := middleware.NewRateLimitConfigFromEnv()

	// Add global rate limiter as a fallback (applies to all routes)
	router.Use(middleware.GlobalRateLimiter(rateLimitConfig))

	// API version 1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (public - no auth required)
		if authHandler != nil {
			auth := v1.Group("/auth")
			// Apply stricter rate limiting to auth endpoints to prevent brute-force
			auth.Use(middleware.PerUserAuthRateLimiter(rateLimitConfig))
			{
				auth.POST("/register", authHandler.Register)
				auth.POST("/login", authHandler.Login)
				auth.POST("/refresh", authHandler.RefreshToken)
				auth.POST("/logout", authHandler.Logout)

				// Protected auth routes (require authentication)
				// These use per-user rate limiting after auth middleware sets user_id
				protected := auth.Group("")
				protected.Use(middleware.AuthMiddleware(jwtConfig))
				protected.Use(middleware.PerUserRateLimiter(rateLimitConfig))
				{
					protected.GET("/profile", authHandler.GetProfile)
					protected.PUT("/profile", authHandler.UpdateProfile)
					protected.PUT("/password", authHandler.ChangePassword)
				}
			}
		}

		// Todo List routes (protected - require authentication)
		lists := v1.Group("/lists")
		if jwtConfig != nil {
			lists.Use(middleware.AuthMiddleware(jwtConfig))
			// Apply per-user rate limiting after auth (user_id is set in context)
			lists.Use(middleware.PerUserRateLimiter(rateLimitConfig))
		}
		{
			lists.GET("", listHandler.GetAllLists)
			lists.POST("", listHandler.CreateList)

			// Routes with listId parameter - validate UUID
			lists.GET("/:listId", middleware.UUIDValidator("listId"), listHandler.GetListByID)
			lists.PUT("/:listId", middleware.UUIDValidator("listId"), listHandler.UpdateList)
			lists.DELETE("/:listId", middleware.UUIDValidator("listId"), listHandler.DeleteList)

			// Todo routes (nested under lists) - validate both listId and todoId
			lists.GET("/:listId/todos", middleware.UUIDValidator("listId"), todoHandler.GetTodosByList)
			lists.POST("/:listId/todos", middleware.UUIDValidator("listId"), todoHandler.CreateTodo)
			lists.GET("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), todoHandler.GetTodoByID)
			lists.PUT("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), todoHandler.UpdateTodo)
			lists.DELETE("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), todoHandler.DeleteTodo)
		}
	}

	// Health check endpoints
	if healthHandler != nil {
		router.GET("/health", healthHandler.BasicHealth)
		router.GET("/health/detailed", healthHandler.DetailedHealth)
		router.GET("/health/ready", healthHandler.ReadinessProbe)
		router.GET("/health/live", healthHandler.LivenessProbe)
	} else {
		// Fallback for in-memory mode
		router.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "healthy",
			})
		})
	}

	// Swagger documentation endpoint (requires authentication)
	// Only authenticated users can view API documentation
	if jwtConfig != nil {
		swagger := router.Group("/swagger")
		swagger.Use(middleware.AuthMiddleware(jwtConfig))
		{
			swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	} else {
		// Fallback for in-memory mode (no auth available)
		// In production with database, this branch won't execute
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Check if TLS is enabled
	tlsConf := tlsconfig.NewTLSConfigFromEnv()

	if tlsConf.Enabled {
		// Run with HTTPS
		startHTTPSServer(router, tlsConf, port, db)
	} else {
		// Run with HTTP only
		startHTTPServer(router, port, db)
	}
}

// startHTTPServer starts an HTTP-only server
func startHTTPServer(router *gin.Engine, port string, db *gorm.DB) {
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		logging.Logger.Infof("Starting HTTP server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	waitForShutdown(db, srv)
}

// startHTTPSServer starts an HTTPS server with optional HTTP redirect
func startHTTPSServer(router *gin.Engine, tlsConf *tlsconfig.TLSConfig, httpPort string, db *gorm.DB) {
	// Create TLS config
	tlsConfig, err := tlsConf.CreateTLSConfig()
	if err != nil {
		logging.Logger.Fatalf("Failed to create TLS config: %v", err)
	}

	// HTTPS server
	httpsSrv := &http.Server{
		Addr:           ":" + tlsConf.Port,
		Handler:        router,
		TLSConfig:      tlsConfig,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start HTTPS server in goroutine
	go func() {
		logging.Logger.Infof("Starting HTTPS server on port %s", tlsConf.Port)
		if err := httpsSrv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatalf("Failed to start HTTPS server: %v", err)
		}
	}()

	// Optional HTTP to HTTPS redirect server
	var httpSrv *http.Server
	if tlsConf.RedirectHTTP {
		httpSrv = &http.Server{
			Addr:           ":" + httpPort,
			Handler:        tlsconfig.HTTPSRedirectHandler(tlsConf.Port),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1MB
		}

		go func() {
			logging.Logger.Infof("Starting HTTP redirect server on port %s -> HTTPS port %s", httpPort, tlsConf.Port)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logging.Logger.Errorf("HTTP redirect server error: %v", err)
			}
		}()
	}

	// Wait for interrupt signal to gracefully shutdown both servers
	waitForShutdown(db, httpsSrv, httpSrv)
}

// waitForShutdown waits for interrupt signal and gracefully shuts down servers
func waitForShutdown(db *gorm.DB, servers ...*http.Server) {
	// Setup signal catching
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-quit
	logging.Logger.Infof("Received signal %v, shutting down gracefully...", sig)

	// Gracefully shutdown servers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, srv := range servers {
		if srv != nil {
			if err := srv.Shutdown(ctx); err != nil {
				logging.Logger.Errorf("Server shutdown error: %v", err)
			}
		}
	}

	// Close database connection if it exists
	if db != nil {
		logging.Logger.Info("Closing database connection...")
		sqlDB, err := db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logging.Logger.Errorf("Database close error: %v", err)
			} else {
				logging.Logger.Info("Database connection closed")
			}
		}
	}

	logging.Logger.Info("Server stopped")
}
