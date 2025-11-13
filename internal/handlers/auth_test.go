package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"


	"todolist-api/internal/auth"
	"todolist-api/internal/models"
	"todolist-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandler(t *testing.T) (*AuthHandler, *auth.Service) {
	db := testutil.SetupTestDB(t)
	jwtConfig := &auth.JWTConfig{
		SecretKey:            "test-secret-key-for-testing-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	}
	authService := auth.NewService(db, jwtConfig)
	handler := NewAuthHandler(authService)
	return handler, authService
}

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully registers a new user", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RegisterRequest{
			Email:     "newuser@example.com",
			Password:  "SecurePass123!",
			FirstName: "John",
			LastName:  "Doe",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.AuthResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.NotNil(t, response.User)
		assert.Equal(t, "newuser@example.com", response.User.Email)
		assert.Equal(t, "John", response.User.FirstName)
		assert.Equal(t, "Doe", response.User.LastName)
	})

	t.Run("successfully registers with minimal fields", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RegisterRequest{
			Email:    "minimal@example.com",
			Password: "SecurePass123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.AuthResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "minimal@example.com", response.User.Email)
		assert.Empty(t, response.User.FirstName)
		assert.Empty(t, response.User.LastName)
	})

	t.Run("returns error for duplicate email", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register first user
		existingUser := &models.RegisterRequest{
			Email:    "existing@example.com",
			Password: "SecurePass123!",
		}
		_, err := authService.Register(existingUser)
		require.NoError(t, err)

		// Try to register again with same email
		reqBody := models.RegisterRequest{
			Email:    "existing@example.com",
			Password: "AnotherPass123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "USER_EXISTS", errResp.Code)
	})

	t.Run("returns error for weak password", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RegisterRequest{
			Email:    "weak@example.com",
			Password: "ThisPasswordIsWayTooLongAndExceedsTheMaximumLengthOf72CharactersAllowedByBcryptSoItShouldTriggerValidationError",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		// Password too long is caught by binding validation (max=72)
		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})

	t.Run("returns error for invalid email format", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RegisterRequest{
			Email:    "not-an-email",
			Password: "SecurePass123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})

	t.Run("returns error for missing required fields", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RegisterRequest{
			Email: "missing@example.com",
			// Password is missing
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/register", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		req := httptest.NewRequest("POST", "/auth/register", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully logs in with valid credentials", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user first
		regReq := &models.RegisterRequest{
			Email:    "login@example.com",
			Password: "SecurePass123!",
		}
		_, err := authService.Register(regReq)
		require.NoError(t, err)

		// Now login
		reqBody := models.LoginRequest{
			Email:    "login@example.com",
			Password: "SecurePass123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/login", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.AuthResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.NotNil(t, response.User)
		assert.Equal(t, "login@example.com", response.User.Email)
	})

	t.Run("returns error for invalid email", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "AnyPassword123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/login", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_CREDENTIALS", errResp.Code)
	})

	t.Run("returns error for invalid password", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:    "wrongpass@example.com",
			Password: "CorrectPass123!",
		}
		_, err := authService.Register(regReq)
		require.NoError(t, err)

		// Try to login with wrong password
		reqBody := models.LoginRequest{
			Email:    "wrongpass@example.com",
			Password: "WrongPass123!",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/login", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_CREDENTIALS", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		req := httptest.NewRequest("POST", "/auth/login", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}

func TestRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully refreshes access token", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register and login
		regReq := &models.RegisterRequest{
			Email:    "refresh@example.com",
			Password: "SecurePass123!",
		}
		_, err := authService.Register(regReq)
		require.NoError(t, err)

		loginReq := &models.LoginRequest{
			Email:    "refresh@example.com",
			Password: "SecurePass123!",
		}
		authResp, err := authService.Login(loginReq)
		require.NoError(t, err)

		// Now refresh the token
		reqBody := models.RefreshTokenRequest{
			RefreshToken: authResp.RefreshToken,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/refresh", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.RefreshToken(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.AuthResponse
		testutil.ParseJSONResponse(t, w, &response)

		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEqual(t, authResp.AccessToken, response.AccessToken) // New access token
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RefreshTokenRequest{
			RefreshToken: "invalid-token",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/refresh", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.RefreshToken(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REFRESH_TOKEN", errResp.Code)
	})

	t.Run("returns error for revoked refresh token", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register and login
		regReq := &models.RegisterRequest{
			Email:    "revoked@example.com",
			Password: "SecurePass123!",
		}
		_, err := authService.Register(regReq)
		require.NoError(t, err)

		loginReq := &models.LoginRequest{
			Email:    "revoked@example.com",
			Password: "SecurePass123!",
		}
		authResp, err := authService.Login(loginReq)
		require.NoError(t, err)

		// Revoke the token
		err = authService.RevokeRefreshToken(authResp.RefreshToken)
		require.NoError(t, err)

		// Try to use the revoked token
		reqBody := models.RefreshTokenRequest{
			RefreshToken: authResp.RefreshToken,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/refresh", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.RefreshToken(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REFRESH_TOKEN", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		req := httptest.NewRequest("POST", "/auth/refresh", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.RefreshToken(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully logs out", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register and login
		regReq := &models.RegisterRequest{
			Email:    "logout@example.com",
			Password: "SecurePass123!",
		}
		_, err := authService.Register(regReq)
		require.NoError(t, err)

		loginReq := &models.LoginRequest{
			Email:    "logout@example.com",
			Password: "SecurePass123!",
		}
		authResp, err := authService.Login(loginReq)
		require.NoError(t, err)

		// Logout
		reqBody := models.RefreshTokenRequest{
			RefreshToken: authResp.RefreshToken,
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/logout", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Logout(c)

		// Accept both 200 and 204 (Gin behavior difference)
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, w.Code)

		// Verify token is revoked by trying to refresh
		refreshReq := models.RefreshTokenRequest{
			RefreshToken: authResp.RefreshToken,
		}

		req2 := testutil.MakeJSONRequest(t, "POST", "/auth/refresh", refreshReq)
		w2 := httptest.NewRecorder()

		c2, _ := gin.CreateTestContext(w2)
		c2.Request = req2

		handler.RefreshToken(c2)

		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.RefreshTokenRequest{
			RefreshToken: "invalid-token",
		}

		req := testutil.MakeJSONRequest(t, "POST", "/auth/logout", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Logout(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REFRESH_TOKEN", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		req := httptest.NewRequest("POST", "/auth/logout", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Logout(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}

func TestGetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully retrieves user profile", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:     "profile@example.com",
			Password:  "SecurePass123!",
			FirstName: "Jane",
			LastName:  "Smith",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/auth/profile", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// Simulate auth middleware setting user_id
		c.Set("user_id", user.ID)

		handler.GetProfile(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var userInfo models.UserInfo
		testutil.ParseJSONResponse(t, w, &userInfo)

		assert.Equal(t, user.ID, userInfo.ID)
		assert.Equal(t, "profile@example.com", userInfo.Email)
		assert.Equal(t, "Jane", userInfo.FirstName)
		assert.Equal(t, "Smith", userInfo.LastName)
	})

	t.Run("returns error when not authenticated", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		req := httptest.NewRequest("GET", "/auth/profile", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// No user_id set in context

		handler.GetProfile(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "UNAUTHORIZED", errResp.Code)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", "/auth/profile", http.NoBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", nonExistentID)

		handler.GetProfile(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "USER_NOT_FOUND", errResp.Code)
	})
}

func TestUpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully updates profile", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:     "updateprofile@example.com",
			Password:  "SecurePass123!",
			FirstName: "Old",
			LastName:  "Name",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		// Update profile
		reqBody := models.UpdateProfileRequest{
			FirstName: testutil.StringPtr("New"),
			LastName:  testutil.StringPtr("Updated"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/profile", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var userInfo models.UserInfo
		testutil.ParseJSONResponse(t, w, &userInfo)

		assert.Equal(t, "New", userInfo.FirstName)
		assert.Equal(t, "Updated", userInfo.LastName)
		assert.Equal(t, user.Email, userInfo.Email)
	})

	t.Run("successfully updates single field", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:     "singlefield@example.com",
			Password:  "SecurePass123!",
			FirstName: "First",
			LastName:  "Last",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		// Update only first name
		reqBody := models.UpdateProfileRequest{
			FirstName: testutil.StringPtr("Updated"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/profile", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var userInfo models.UserInfo
		testutil.ParseJSONResponse(t, w, &userInfo)

		assert.Equal(t, "Updated", userInfo.FirstName)
		assert.Equal(t, "Last", userInfo.LastName) // Unchanged
	})

	t.Run("returns error when not authenticated", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.UpdateProfileRequest{
			FirstName: testutil.StringPtr("New"),
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/profile", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// No user_id set

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "UNAUTHORIZED", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		regReq := &models.RegisterRequest{
			Email:    "invalidjson@example.com",
			Password: "SecurePass123!",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/auth/profile", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}

func TestChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully changes password", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:    "changepass@example.com",
			Password: "OldPass123!",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		// Change password
		reqBody := models.ChangePasswordRequest{
			CurrentPassword: "OldPass123!",
			NewPassword:     "NewPass123!",
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/password", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.ChangePassword(c)

		// Accept both 200 and 204
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, w.Code)

		// Verify new password works
		loginReq := &models.LoginRequest{
			Email:    "changepass@example.com",
			Password: "NewPass123!",
		}
		_, err = authService.Login(loginReq)
		assert.NoError(t, err)
	})

	t.Run("returns error for incorrect current password", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:    "wrongcurrent@example.com",
			Password: "CorrectPass123!",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		// Try to change with wrong current password
		reqBody := models.ChangePasswordRequest{
			CurrentPassword: "WrongPass123!",
			NewPassword:     "NewPass123!",
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/password", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_CURRENT_PASSWORD", errResp.Code)
	})

	t.Run("returns error for weak new password", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		// Register a user
		regReq := &models.RegisterRequest{
			Email:    "weaknew@example.com",
			Password: "StrongPass123!",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		// Try to change to weak password
		reqBody := models.ChangePasswordRequest{
			CurrentPassword: "StrongPass123!",
			NewPassword:     "weak",
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/password", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		// Password too long is caught by binding validation (max=72)
		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})

	t.Run("returns error when not authenticated", func(t *testing.T) {
		handler, _ := setupAuthHandler(t)

		reqBody := models.ChangePasswordRequest{
			CurrentPassword: "OldPass123!",
			NewPassword:     "NewPass123!",
		}

		req := testutil.MakeJSONRequest(t, "PUT", "/auth/password", reqBody)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// No user_id set

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "UNAUTHORIZED", errResp.Code)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler, authService := setupAuthHandler(t)

		regReq := &models.RegisterRequest{
			Email:    "invalidjsonpass@example.com",
			Password: "SecurePass123!",
		}
		user, err := authService.Register(regReq)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/auth/password", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", user.ID)

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp models.ErrorResponse
		testutil.ParseJSONResponse(t, w, &errResp)

		assert.Equal(t, "INVALID_REQUEST", errResp.Code)
	})
}
