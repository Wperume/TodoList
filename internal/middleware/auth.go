package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"todolist-api/internal/auth"
	"todolist-api/internal/models"
)

const (
	// ContextKeyUserID is the context key for storing user ID
	ContextKeyUserID = "user_id"
	// ContextKeyUserEmail is the context key for storing user email
	ContextKeyUserEmail = "user_email"
	// ContextKeyUserRole is the context key for storing user role
	ContextKeyUserRole = "user_role"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(jwtConfig *auth.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "INVALID_TOKEN",
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := auth.ValidateAccessToken(tokenString, jwtConfig)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    "TOKEN_EXPIRED",
					Message: "Access token has expired. Please refresh your token.",
				})
			} else {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    "INVALID_TOKEN",
					Message: "Invalid access token",
				})
			}
			c.Abort()
			return
		}

		// Store user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserEmail, claims.Email)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context (set by AuthMiddleware)
		roleValue, exists := c.Get(ContextKeyUserRole)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Invalid role type in context",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		// Admin has access to everything
		if role == models.RoleAdmin {
			c.Next()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth is a middleware that validates JWT if present, but doesn't require it
func OptionalAuth(jwtConfig *auth.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without authentication
			c.Next()
			return
		}

		// Extract Bearer token
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		// Validate token
		claims, err := auth.ValidateAccessToken(tokenString, jwtConfig)
		if err != nil {
			// Invalid or expired token, continue without authentication
			c.Next()
			return
		}

		// Store user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserEmail, claims.Email)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

// Helper functions to extract user information from context

// GetUserID retrieves the user ID from the Gin context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type in context")
	}

	return id, nil
}

// GetUserEmail retrieves the user email from the Gin context
func GetUserEmail(c *gin.Context) (string, error) {
	email, exists := c.Get(ContextKeyUserEmail)
	if !exists {
		return "", errors.New("user email not found in context")
	}

	emailStr, ok := email.(string)
	if !ok {
		return "", errors.New("invalid email type in context")
	}

	return emailStr, nil
}

// GetUserRole retrieves the user role from the Gin context
func GetUserRole(c *gin.Context) (models.UserRole, error) {
	role, exists := c.Get(ContextKeyUserRole)
	if !exists {
		return "", errors.New("user role not found in context")
	}

	roleValue, ok := role.(models.UserRole)
	if !ok {
		return "", errors.New("invalid role type in context")
	}

	return roleValue, nil
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(ContextKeyUserID)
	return exists
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *gin.Context) bool {
	role, err := GetUserRole(c)
	if err != nil {
		return false
	}
	return role == models.RoleAdmin
}
