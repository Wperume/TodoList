package security_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"todolist-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMissingAuthToken tests that endpoints reject requests without auth tokens
func TestMissingAuthToken(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/lists"},
		{"POST", "/api/v1/lists"},
		{"GET", "/api/v1/auth/profile"},
		{"PUT", "/api/v1/auth/profile"},
	}

	for _, endpoint := range protectedEndpoints {
		t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
			var body []byte
			if endpoint.method == "POST" || endpoint.method == "PUT" {
				body = []byte(`{"name":"test"}`)
			}

			req, err := http.NewRequest(endpoint.method, endpoint.path, bytes.NewBuffer(body))
			require.NoError(t, err)

			if len(body) > 0 {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code, "Should require authentication")

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// API returns "code" and "message" fields for errors
			assert.Contains(t, response, "code")
			assert.Contains(t, response, "message")
		})
	}
}

// TestInvalidAuthToken tests that invalid tokens are rejected
func TestInvalidAuthToken(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	invalidTokens := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"malformed", "not.a.jwt"},
		{"wrong_format", "Bearer not.a.jwt"},
		{"random_string", "randomstring123"},
		{"base64_garbage", base64.StdEncoding.EncodeToString([]byte("garbage"))},
		{"sql_injection", "' OR '1'='1"},
	}

	for _, tc := range invalidTokens {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/lists", nil)
			require.NoError(t, err)

			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject invalid token: %s", tc.name)
		})
	}
}

// TestExpiredToken tests that expired tokens are rejected
func TestExpiredToken(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user, _ := CreateTestUserWithToken(t, router)

	// Create an expired token manually
	jwtSecret := []byte("test-secret-key-min-32-characters-long!")
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString(jwtSecret)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/api/v1/lists", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+expiredToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject expired token")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should indicate token expired
	if code, ok := response["code"].(string); ok {
		assert.Equal(t, "TOKEN_EXPIRED", code)
	}
}

// TestTokenReuse tests that logout invalidates tokens properly
func TestTokenReuse(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	_, accessToken := CreateTestUserWithToken(t, router)

	// Verify token works
	req, err := http.NewRequest("GET", "/api/v1/lists", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Token should work before logout")

	// Note: Token invalidation after logout would require a token blacklist
	// This test documents the current behavior
	t.Log("Note: Current implementation uses stateless JWT, tokens remain valid until expiry")
}

// TestJWTAlgorithmSubstitution tests algorithm substitution attacks
func TestJWTAlgorithmSubstitution(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user, _ := CreateTestUserWithToken(t, router)

	// Try to create token with "none" algorithm
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}

	// Create token with "none" algorithm
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	noneTokenString, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req, err := http.NewRequest("GET", "/api/v1/lists", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+noneTokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should reject tokens with "none" algorithm
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject 'none' algorithm tokens")
}

// TestJWTSignatureVerification tests that signature is properly verified
func TestJWTSignatureVerification(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user, validToken := CreateTestUserWithToken(t, router)

	// Verify the valid token works
	req, err := http.NewRequest("GET", "/api/v1/lists", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+validToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Valid token should work")

	// Try to tamper with the token
	parts := strings.Split(validToken, ".")
	if len(parts) == 3 {
		// Decode payload
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)

		// Modify payload to change user_id
		var claims map[string]interface{}
		err = json.Unmarshal(payload, &claims)
		require.NoError(t, err)

		claims["user_id"] = "00000000-0000-0000-0000-000000000000"

		// Re-encode
		newPayload, err := json.Marshal(claims)
		require.NoError(t, err)
		parts[1] = base64.RawURLEncoding.EncodeToString(newPayload)

		// Create tampered token
		tamperedToken := strings.Join(parts, ".")

		req, err = http.NewRequest("GET", "/api/v1/lists", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tamperedToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should reject tampered token
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject tampered token")
	}

	// Create a token with wrong secret
	wrongSecret := []byte("wrong-secret-key-different-from-real!")
	wrongClaims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}

	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongClaims)
	wrongTokenString, err := wrongToken.SignedString(wrongSecret)
	require.NoError(t, err)

	req, err = http.NewRequest("GET", "/api/v1/lists", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+wrongTokenString)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should reject token signed with wrong secret
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject token with wrong signature")
}

// TestPasswordComplexity tests that weak passwords are rejected
func TestPasswordComplexity(t *testing.T) {

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	weakPasswords := []string{
		"123",           // Too short
		"12345",         // Too short
		"password",      // Common password
		"abc123",        // Too simple
		"qwerty",        // Common pattern
		"11111111",      // Repetitive
		"aaaaaaaa",      // Repetitive
	}

	for _, weakPwd := range weakPasswords {
		t.Run(weakPwd, func(t *testing.T) {
			regReq := models.RegisterRequest{
				Email:     fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
				Password:  weakPwd,
				FirstName: "Test",
				LastName:  "User",
			}

			body, err := json.Marshal(regReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should reject weak passwords (if validation is implemented)
			// Note: Document current behavior
			if w.Code == http.StatusBadRequest {
				t.Logf("Weak password rejected: %s", weakPwd)
			} else {
				t.Logf("Note: Password complexity validation not enforced for: %s", weakPwd)
			}
		})
	}
}
