package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPassword = "SecurePassword123!"

func TestHashPassword(t *testing.T) {
	t.Run("hashes password successfully", func(t *testing.T) {
		password := testPassword
		hash, err := HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
	})

	t.Run("generates different hashes for same password", func(t *testing.T) {
		password := testPassword
		hash1, err := HashPassword(password)
		require.NoError(t, err)

		hash2, err := HashPassword(password)
		require.NoError(t, err)

		// bcrypt uses random salt, so hashes should be different
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("rejects password too short", func(t *testing.T) {
		password := "Short1!"
		_, err := HashPassword(password)
		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("rejects password too long", func(t *testing.T) {
		// Create a password longer than 72 characters
		password := strings.Repeat("a", 73)
		_, err := HashPassword(password)
		assert.ErrorIs(t, err, ErrPasswordTooLong)
	})

	t.Run("accepts minimum length password", func(t *testing.T) {
		password := "Pass123!"
		hash, err := HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("accepts maximum length password", func(t *testing.T) {
		// Create a password exactly 72 characters
		password := strings.Repeat("a", 72)
		hash, err := HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestVerifyPassword(t *testing.T) {
	password := testPassword
	hash, err := HashPassword(password)
	require.NoError(t, err)

	t.Run("matches correct password", func(t *testing.T) {
		err := VerifyPassword(password, hash)
		assert.NoError(t, err)
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		err := VerifyPassword("WrongPassword123!", hash)
		assert.ErrorIs(t, err, ErrInvalidPassword)
	})

	t.Run("rejects empty password", func(t *testing.T) {
		err := VerifyPassword("", hash)
		assert.ErrorIs(t, err, ErrInvalidPassword)
	})

	t.Run("rejects invalid hash", func(t *testing.T) {
		err := VerifyPassword(password, "invalid-hash")
		assert.Error(t, err)
	})
}

func TestValidatePasswordRequirements(t *testing.T) {
	t.Run("accepts valid password", func(t *testing.T) {
		passwords := []string{
			"SecurePass123!",
			"MyP@ssw0rd",
			"Test1234",
			"abcdefgh",
		}

		for _, password := range passwords {
			err := ValidatePasswordRequirements(password)
			assert.NoError(t, err, "Password %s should be valid", password)
		}
	})

	t.Run("rejects password too short", func(t *testing.T) {
		err := ValidatePasswordRequirements("Short1!")
		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("rejects password too long", func(t *testing.T) {
		password := strings.Repeat("a", 73)
		err := ValidatePasswordRequirements(password)
		assert.ErrorIs(t, err, ErrPasswordTooLong)
	})

	t.Run("accepts boundary length passwords", func(t *testing.T) {
		// Minimum length (8 characters)
		err := ValidatePasswordRequirements("12345678")
		assert.NoError(t, err)

		// Maximum length (72 characters)
		err = ValidatePasswordRequirements(strings.Repeat("a", 72))
		assert.NoError(t, err)
	})
}

func TestPasswordSecurity(t *testing.T) {
	t.Run("hash is not reversible", func(t *testing.T) {
		password := testPassword
		hash, err := HashPassword(password)
		require.NoError(t, err)

		// Hash should not contain the original password
		assert.NotContains(t, hash, password)
	})

	t.Run("hash starts with bcrypt identifier", func(t *testing.T) {
		password := testPassword
		hash, err := HashPassword(password)
		require.NoError(t, err)

		// bcrypt hashes start with $2a$, $2b$, or $2y$
		assert.True(t, strings.HasPrefix(hash, "$2a$") ||
			strings.HasPrefix(hash, "$2b$") ||
			strings.HasPrefix(hash, "$2y$"))
	})

	t.Run("uses correct cost factor", func(t *testing.T) {
		password := testPassword
		hash, err := HashPassword(password)
		require.NoError(t, err)

		// bcrypt hash format: $2a$12$... where 12 is the cost factor
		parts := strings.Split(hash, "$")
		require.Len(t, parts, 4)
		assert.Equal(t, "12", parts[2])
	})
}
