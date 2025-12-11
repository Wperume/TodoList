package security_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"todolist-api/internal/auth"
	"todolist-api/internal/handlers"
	"todolist-api/internal/logging"
	"todolist-api/internal/middleware"
	"todolist-api/internal/models"
	"todolist-api/internal/storage"
	"todolist-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// SetupTestRouter creates a test router with all handlers initialized
func SetupTestRouter(t *testing.T) (*gin.Engine, func()) {
	// Initialize logger for tests (required by middleware)
	logging.InitLogger(&logging.LogConfig{
		Level: "error", // Suppress logs during tests
	})

	// Setup test database
	db := testutil.SetupTestDB(t)

	// Import required packages for handlers
	var (
		authHandler   interface{}
		listHandler   interface{}
		todoHandler   interface{}
		healthHandler interface{}
		jwtConfig     interface{}
	)

	// Initialize JWT configuration for testing
	jwtConfig = &auth.JWTConfig{
		SecretKey:            "test-secret-key-min-32-characters-long!",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "todolist-api-test",
	}

	// Initialize authentication service and handler
	authService := auth.NewService(db, jwtConfig.(*auth.JWTConfig))
	authHandler = handlers.NewAuthHandler(authService)

	// Initialize storage and handlers
	store := storage.NewPostgresStorage(db)
	listHandler = handlers.NewListHandler(store)
	todoHandler = handlers.NewTodoHandler(store)
	healthHandler = handlers.NewHealthHandler(db)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup router with middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.ErrorSanitizer())

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes
		if authHandler != nil {
			authGroup := v1.Group("/auth")
			ah := authHandler.(*handlers.AuthHandler)
			{
				authGroup.POST("/register", ah.Register)
				authGroup.POST("/login", ah.Login)
				authGroup.POST("/refresh", ah.RefreshToken)
				authGroup.POST("/logout", ah.Logout)

				// Protected auth routes
				protected := authGroup.Group("")
				protected.Use(middleware.AuthMiddleware(jwtConfig.(*auth.JWTConfig)))
				protected.GET("/profile", ah.GetProfile)
				protected.PUT("/profile", ah.UpdateProfile)
				protected.PUT("/password", ah.ChangePassword)
			}
		}

		// List routes (protected)
		lists := v1.Group("/lists")
		lists.Use(middleware.AuthMiddleware(jwtConfig.(*auth.JWTConfig)))
		lh := listHandler.(*handlers.ListHandler)
		th := todoHandler.(*handlers.TodoHandler)
		{
			lists.GET("", lh.GetAllLists)
			lists.POST("", lh.CreateList)
			lists.GET("/:listId", middleware.UUIDValidator("listId"), lh.GetListByID)
			lists.PUT("/:listId", middleware.UUIDValidator("listId"), lh.UpdateList)
			lists.DELETE("/:listId", middleware.UUIDValidator("listId"), lh.DeleteList)

			// Todo routes
			lists.GET("/:listId/todos", middleware.UUIDValidator("listId"), th.GetTodosByList)
			lists.POST("/:listId/todos", middleware.UUIDValidator("listId"), th.CreateTodo)
			lists.GET("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), th.GetTodoByID)
			lists.PUT("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), th.UpdateTodo)
			lists.DELETE("/:listId/todos/:todoId", middleware.UUIDValidator("listId", "todoId"), th.DeleteTodo)
		}
	}

	// Health endpoints
	if healthHandler != nil {
		hh := healthHandler.(*handlers.HealthHandler)
		router.GET("/health", hh.BasicHealth)
		router.GET("/health/detailed", hh.DetailedHealth)
		router.GET("/health/ready", hh.ReadinessProbe)
		router.GET("/health/live", hh.LivenessProbe)
	}

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return router, cleanup
}

// CreateTestUserWithToken creates a test user and returns the user and an access token
func CreateTestUserWithToken(t *testing.T, router *gin.Engine) (*models.User, string) {
	// Create a unique email for this test user
	email := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())

	// Register the user via API
	registerReq := models.RegisterRequest{
		Email:     email,
		Password:  "TestPassword123!",
		FirstName: "Test",
		LastName:  "User",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "Failed to register test user: %s", w.Body.String())

	// Parse the response to get user and token
	var response models.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Convert UserInfo to User (AuthResponse returns UserInfo, but we need User for tests)
	user := &models.User{
		ID:        response.User.ID,
		Email:     response.User.Email,
		FirstName: response.User.FirstName,
		LastName:  response.User.LastName,
		Role:      response.User.Role,
	}

	return user, response.AccessToken
}
