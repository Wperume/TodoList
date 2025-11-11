package auth

import (
	"encoding/base64"
	"testing"
	"time"

	"todolist-api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:            "test-secret-key-32-characters!!",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-todolist-api",
	}
}

func getTestUser() *models.User {
	return &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
}

func TestGenerateAccessToken(t *testing.T) {
	config := getTestJWTConfig()
	user := getTestUser()

	t.Run("generates valid token", func(t *testing.T) {
		token, err := GenerateAccessToken(user, config)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("token contains correct claims", func(t *testing.T) {
		token, err := GenerateAccessToken(user, config)
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, config)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, config.Issuer, claims.Issuer)
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("generates valid token", func(t *testing.T) {
		token, err := GenerateRefreshToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := GenerateRefreshToken()
		require.NoError(t, err)

		token2, err := GenerateRefreshToken()
		require.NoError(t, err)

		// Tokens should be different
		assert.NotEqual(t, token1, token2)
	})

	t.Run("generates base64 URL-encoded tokens", func(t *testing.T) {
		token, err := GenerateRefreshToken()
		require.NoError(t, err)

		// Should be able to decode as base64 URL
		_, err = base64.URLEncoding.DecodeString(token)
		assert.NoError(t, err)
	})
}

func TestValidateAccessToken(t *testing.T) {
	config := getTestJWTConfig()
	user := getTestUser()

	t.Run("validates correct token", func(t *testing.T) {
		token, err := GenerateAccessToken(user, config)
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, config)
		require.NoError(t, err)
		assert.NotNil(t, claims)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		_, err := ValidateAccessToken("invalid-token", config)
		assert.Error(t, err)
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		token, err := GenerateAccessToken(user, config)
		require.NoError(t, err)

		wrongConfig := &JWTConfig{
			SecretKey:            "wrong-secret-key-32-characters!",
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			Issuer:               "test-todolist-api",
		}

		_, err = ValidateAccessToken(token, wrongConfig)
		assert.Error(t, err)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		expiredConfig := &JWTConfig{
			SecretKey:            config.SecretKey,
			AccessTokenDuration:  -1 * time.Hour, // Negative duration = already expired
			RefreshTokenDuration: 7 * 24 * time.Hour,
			Issuer:               config.Issuer,
		}

		token, err := GenerateAccessToken(user, expiredConfig)
		require.NoError(t, err)

		_, err = ValidateAccessToken(token, config)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})
}

func TestExtractTokenFromHeader(t *testing.T) {
	t.Run("extracts valid Bearer token", func(t *testing.T) {
		header := "Bearer test-token-123"
		token, err := ExtractTokenFromHeader(header)
		require.NoError(t, err)
		assert.Equal(t, "test-token-123", token)
	})

	t.Run("rejects empty header", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("")
		assert.Error(t, err)
	})

	t.Run("rejects header without Bearer prefix", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("test-token-123")
		assert.Error(t, err)
	})

	t.Run("rejects header with only Bearer", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("Bearer ")
		assert.Error(t, err)
	})

	t.Run("rejects header with wrong case", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("bearer test-token-123")
		assert.Error(t, err)
	})
}

func TestNewJWTConfigFromEnv(t *testing.T) {
	t.Run("uses default values when env vars not set", func(t *testing.T) {
		config := NewJWTConfigFromEnv()
		assert.NotEmpty(t, config.SecretKey)
		assert.Equal(t, 15*time.Minute, config.AccessTokenDuration)
		assert.Equal(t, 7*24*time.Hour, config.RefreshTokenDuration)
		assert.Equal(t, "todolist-api", config.Issuer)
	})
}

func TestAdminRole(t *testing.T) {
	config := getTestJWTConfig()
	adminUser := &models.User{
		ID:        uuid.New(),
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      models.RoleAdmin,
		IsActive:  true,
	}

	t.Run("admin role in access token", func(t *testing.T) {
		token, err := GenerateAccessToken(adminUser, config)
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, config)
		require.NoError(t, err)
		assert.Equal(t, models.RoleAdmin, claims.Role)
	})
}
