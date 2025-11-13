package auth

import (
	"testing"
	"time"

	"todolist-api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// For SQLite, we need to disable the default UUID generation
	err = db.Exec("PRAGMA foreign_keys = ON").Error
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.TodoList{}, &models.Todo{})
	require.NoError(t, err)

	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	jwtConfig := &JWTConfig{
		SecretKey:            "test-secret-key-32-characters!!",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-todolist-api",
	}
	service := NewService(db, jwtConfig)
	return service, db
}

func TestRegister(t *testing.T) {
	service, _ := setupTestService(t)

	t.Run("successful registration", func(t *testing.T) {
		req := &models.RegisterRequest{
			Email:     "test@example.com",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}

		user, err := service.Register(req)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.FirstName, user.FirstName)
		assert.Equal(t, req.LastName, user.LastName)
		assert.Equal(t, models.RoleUser, user.Role)
		assert.True(t, user.IsActive)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, req.Password, user.PasswordHash)
	})

	t.Run("duplicate email", func(t *testing.T) {
		req := &models.RegisterRequest{
			Email:     "duplicate@example.com",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}

		// Register first user
		_, err := service.Register(req)
		require.NoError(t, err)

		// Try to register with same email
		_, err = service.Register(req)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})

	t.Run("password too short", func(t *testing.T) {
		req := &models.RegisterRequest{
			Email:     "short@example.com",
			Password:  "Short1!",
			FirstName: "Test",
			LastName:  "User",
		}

		_, err := service.Register(req)
		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})
}

func TestLogin(t *testing.T) {
	service, _ := setupTestService(t)

	// Create a test user
	registerReq := &models.RegisterRequest{
		Email:     "login@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful login", func(t *testing.T) {
		loginReq := &models.LoginRequest{
			Email:    "login@example.com",
			Password: "SecurePass123!",
		}

		authResponse, err := service.Login(loginReq)
		require.NoError(t, err)
		assert.NotEmpty(t, authResponse.AccessToken)
		assert.NotEmpty(t, authResponse.RefreshToken)
		assert.Equal(t, loginReq.Email, authResponse.User.Email)
	})

	t.Run("invalid email", func(t *testing.T) {
		loginReq := &models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "SecurePass123!",
		}

		_, err := service.Login(loginReq)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("invalid password", func(t *testing.T) {
		loginReq := &models.LoginRequest{
			Email:    "login@example.com",
			Password: "WrongPassword123!",
		}

		_, err := service.Login(loginReq)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

func TestRefreshAccessToken(t *testing.T) {
	service, _ := setupTestService(t)

	// Create and login a test user
	registerReq := &models.RegisterRequest{
		Email:     "refresh@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	loginReq := &models.LoginRequest{
		Email:    "refresh@example.com",
		Password: "SecurePass123!",
	}
	authResponse, err := service.Login(loginReq)
	require.NoError(t, err)

	t.Run("successful refresh", func(t *testing.T) {
		newAuthResponse, err := service.RefreshAccessToken(authResponse.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAuthResponse.AccessToken)
		assert.NotEmpty(t, newAuthResponse.RefreshToken)
		assert.NotEqual(t, authResponse.AccessToken, newAuthResponse.AccessToken)
		assert.NotEqual(t, authResponse.RefreshToken, newAuthResponse.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := service.RefreshAccessToken("invalid-token")
		assert.ErrorIs(t, err, ErrRefreshTokenInvalid)
	})
}

func TestRevokeRefreshToken(t *testing.T) {
	service, _ := setupTestService(t)

	// Create and login a test user
	registerReq := &models.RegisterRequest{
		Email:     "revoke@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	loginReq := &models.LoginRequest{
		Email:    "revoke@example.com",
		Password: "SecurePass123!",
	}
	authResponse, err := service.Login(loginReq)
	require.NoError(t, err)

	t.Run("successful revoke", func(t *testing.T) {
		err := service.RevokeRefreshToken(authResponse.RefreshToken)
		assert.NoError(t, err)

		// Try to use revoked token
		_, err = service.RefreshAccessToken(authResponse.RefreshToken)
		assert.ErrorIs(t, err, ErrRefreshTokenInvalid)
	})

	t.Run("invalid token", func(t *testing.T) {
		err := service.RevokeRefreshToken("invalid-token")
		assert.ErrorIs(t, err, ErrRefreshTokenInvalid)
	})
}

func TestGetUserByID(t *testing.T) {
	service, _ := setupTestService(t)

	// Create a test user
	registerReq := &models.RegisterRequest{
		Email:     "getuser@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	user, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful get", func(t *testing.T) {
		foundUser, err := service.GetUserByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundUser.ID)
		assert.Equal(t, user.Email, foundUser.Email)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := service.GetUserByID(uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUpdateProfile(t *testing.T) {
	service, _ := setupTestService(t)

	// Create a test user
	registerReq := &models.RegisterRequest{
		Email:     "update@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	user, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("update first and last name", func(t *testing.T) {
		newFirstName := "Updated"
		newLastName := "Name"
		updateReq := &models.UpdateProfileRequest{
			FirstName: &newFirstName,
			LastName:  &newLastName,
		}

		updatedUser, err := service.UpdateProfile(user.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, newFirstName, updatedUser.FirstName)
		assert.Equal(t, newLastName, updatedUser.LastName)
	})

	t.Run("user not found", func(t *testing.T) {
		name := "Test"
		updateReq := &models.UpdateProfileRequest{
			FirstName: &name,
		}

		_, err := service.UpdateProfile(uuid.New(), updateReq)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestChangePassword(t *testing.T) {
	service, _ := setupTestService(t)

	// Create a test user
	registerReq := &models.RegisterRequest{
		Email:     "password@example.com",
		Password:  "OldPassword123!",
		FirstName: "Test",
		LastName:  "User",
	}
	user, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful password change", func(t *testing.T) {
		changeReq := &models.ChangePasswordRequest{
			CurrentPassword: "OldPassword123!",
			NewPassword:     "NewPassword123!",
		}

		err := service.ChangePassword(user.ID, changeReq)
		require.NoError(t, err)

		// Verify new password works
		loginReq := &models.LoginRequest{
			Email:    "password@example.com",
			Password: "NewPassword123!",
		}
		_, err = service.Login(loginReq)
		assert.NoError(t, err)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		changeReq := &models.ChangePasswordRequest{
			CurrentPassword: "WrongPassword123!",
			NewPassword:     "NewPassword456!",
		}

		err := service.ChangePassword(user.ID, changeReq)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("user not found", func(t *testing.T) {
		changeReq := &models.ChangePasswordRequest{
			CurrentPassword: "OldPassword123!",
			NewPassword:     "NewPassword123!",
		}

		err := service.ChangePassword(uuid.New(), changeReq)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestInactiveUser(t *testing.T) {
	service, db := setupTestService(t)

	// Create a test user
	registerReq := &models.RegisterRequest{
		Email:     "inactive@example.com",
		Password:  "SecurePass123!",
		FirstName: "Test",
		LastName:  "User",
	}
	user, err := service.Register(registerReq)
	require.NoError(t, err)

	// Deactivate the user
	db.Model(&user).Update("is_active", false)

	t.Run("inactive user cannot login", func(t *testing.T) {
		loginReq := &models.LoginRequest{
			Email:    "inactive@example.com",
			Password: "SecurePass123!",
		}

		_, err := service.Login(loginReq)
		assert.ErrorIs(t, err, ErrUserInactive)
	})
}
