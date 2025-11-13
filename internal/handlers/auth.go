package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"todolist-api/internal/auth"
	"todolist-api/internal/middleware"
	"todolist-api/internal/models"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	// Validate password requirements
	if err := auth.ValidatePasswordRequirements(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_PASSWORD",
			Message: err.Error(),
		})
		return
	}

	// Register user
	user, err := h.authService.Register(&req)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Code:    "USER_EXISTS",
				Message: "A user with this email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "REGISTRATION_FAILED",
			Message: "Failed to register user",
		})
		return
	}

	// Auto-login after registration
	loginReq := &models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	authResponse, err := h.authService.Login(loginReq)
	if err != nil {
		// Registration succeeded but login failed - still return success
		c.JSON(http.StatusCreated, models.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
		})
		return
	}

	c.JSON(http.StatusCreated, authResponse)
}

// Login handles user login
// @Summary Login
// @Description Authenticate a user and return JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	authResponse, err := h.authService.Login(&req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "INVALID_CREDENTIALS",
				Message: "Invalid email or password",
			})
			return
		}

		if errors.Is(err, auth.ErrUserInactive) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "USER_INACTIVE",
				Message: "User account is inactive",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "LOGIN_FAILED",
			Message: "Failed to login",
		})
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

// RefreshToken handles access token refresh
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	authResponse, err := h.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenInvalid) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "Refresh token is invalid or expired",
			})
			return
		}

		if errors.Is(err, auth.ErrUserInactive) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "USER_INACTIVE",
				Message: "User account is inactive",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "REFRESH_FAILED",
			Message: "Failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

// Logout handles user logout
// @Summary Logout
// @Description Revoke the current refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token to revoke"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	err := h.authService.RevokeRefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenInvalid) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "INVALID_REFRESH_TOKEN",
				Message: "Refresh token is invalid",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "LOGOUT_FAILED",
			Message: "Failed to logout",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProfile returns the current user's profile
// @Summary Get user profile
// @Description Get the authenticated user's profile information
// @Tags User
// @Produce json
// @Success 200 {object} models.UserInfo
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "PROFILE_FETCH_FAILED",
			Message: "Failed to fetch profile",
		})
		return
	}

	c.JSON(http.StatusOK, models.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	})
}

// UpdateProfile updates the current user's profile
// @Summary Update user profile
// @Description Update the authenticated user's profile information
// @Tags User
// @Accept json
// @Produce json
// @Param request body models.UpdateProfileRequest true "Profile updates"
// @Success 200 {object} models.UserInfo
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	var req models.UpdateProfileRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": bindErr.Error()},
		})
		return
	}

	user, err := h.authService.UpdateProfile(userID, &req)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "PROFILE_UPDATE_FAILED",
			Message: "Failed to update profile",
		})
		return
	}

	c.JSON(http.StatusOK, models.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	})
}

// ChangePassword handles password change
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags User
// @Accept json
// @Produce json
// @Param request body models.ChangePasswordRequest true "Password change details"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	var req models.ChangePasswordRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request payload",
			Details: map[string]interface{}{"error": bindErr.Error()},
		})
		return
	}

	// Validate new password requirements
	if validateErr := auth.ValidatePasswordRequirements(req.NewPassword); validateErr != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_PASSWORD",
			Message: validateErr.Error(),
		})
		return
	}

	err = h.authService.ChangePassword(userID, &req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "INVALID_CURRENT_PASSWORD",
				Message: "Current password is incorrect",
			})
			return
		}

		if errors.Is(err, auth.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "PASSWORD_CHANGE_FAILED",
			Message: "Failed to change password",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
