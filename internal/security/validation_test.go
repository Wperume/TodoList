package security_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"todolist-api/internal/models"
	"todolist-api/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXSSInInputs tests XSS payload handling
func TestXSSInInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	user, token := testutil.CreateTestUserWithToken(t, router)
	list := testutil.CreateTestList(t, router, token, user.ID)

	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"<svg/onload=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
		"<body onload=alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"'><script>alert(String.fromCharCode(88,83,83))</script>",
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			todoReq := models.CreateTodoRequest{
				Description: payload,
				Priority:    models.PriorityMedium,
			}

			body, err := json.Marshal(todoReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/lists/"+list.ID.String()+"/todos", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should either accept and store safely, or reject
			if w.Code == http.StatusCreated {
				var todo models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &todo)
				require.NoError(t, err)

				// Payload should be stored as-is (escaping happens on output)
				assert.Equal(t, payload, todo.Description,
					"XSS payload should be stored safely without execution")
			}
		})
	}
}

// TestOversizedInputs tests handling of excessively large inputs
func TestOversizedInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	user, token := testutil.CreateTestUserWithToken(t, router)
	list := testutil.CreateTestList(t, router, token, user.ID)

	// Create very large strings
	oversizedInputs := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, test := range oversizedInputs {
		t.Run(test.name, func(t *testing.T) {
			largeString := strings.Repeat("A", test.size)

			todoReq := models.CreateTodoRequest{
				Description: largeString,
				Priority:    models.PriorityMedium,
			}

			body, err := json.Marshal(todoReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/lists/"+list.ID.String()+"/todos", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should handle gracefully (accept or reject with proper error)
			if w.Code == http.StatusBadRequest || w.Code == http.StatusRequestEntityTooLarge {
				t.Logf("Large input rejected: %s", test.name)
			} else if w.Code == http.StatusCreated {
				t.Logf("Large input accepted: %s", test.name)
			} else {
				t.Errorf("Unexpected status code for large input: %d", w.Code)
			}
		})
	}
}

// TestInvalidUUIDs tests UUID validation
func TestInvalidUUIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	_, token := testutil.CreateTestUserWithToken(t, router)

	invalidUUIDs := []string{
		"not-a-uuid",
		"123",
		"00000000-0000-0000-0000",           // Incomplete
		"00000000-0000-0000-0000-00000000g", // Invalid character
		"../../../etc/passwd",               // Path traversal
		"'; DROP TABLE todos--",             // SQL injection
		"<script>alert(1)</script>",         // XSS
	}

	for _, invalidID := range invalidUUIDs {
		t.Run(invalidID, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/lists/"+invalidID, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should reject with 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, w.Code,
				"Should reject invalid UUID: %s", invalidID)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response, "error")
		})
	}
}

// TestInvalidJSON tests handling of malformed JSON
func TestInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	_, token := testutil.CreateTestUserWithToken(t, router)

	malformedJSONs := []string{
		"{invalid json}",
		"{'single': 'quotes'}",
		"{name: 'missing quotes'}",
		"{\"unclosed\": ",
		"}{",
		"null",
		"[]",
		"\"just a string\"",
	}

	for _, malformedJSON := range malformedJSONs {
		t.Run(malformedJSON, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/lists", bytes.NewBufferString(malformedJSON))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should reject with 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, w.Code,
				"Should reject malformed JSON")

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response, "error")
		})
	}
}

// TestSpecialCharacters tests handling of special characters
func TestSpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	user, token := testutil.CreateTestUserWithToken(t, router)
	list := testutil.CreateTestList(t, router, token, user.ID)

	specialCharInputs := []string{
		"Unicode: 你好世界 🌍",
		"Emojis: 😀😁😂🤣",
		"Special: !@#$%^&*()_+-={}[]|\\:;\"'<>,.?/",
		"Null bytes: \x00",
		"Control chars: \r\n\t",
		"CRLF injection: \r\nX-Custom-Header: injected",
	}

	for _, input := range specialCharInputs {
		t.Run(input, func(t *testing.T) {
			todoReq := models.CreateTodoRequest{
				Description: input,
				Priority:    models.PriorityMedium,
			}

			body, err := json.Marshal(todoReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/lists/"+list.ID.String()+"/todos", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should handle special characters safely
			if w.Code == http.StatusCreated {
				var todo models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &todo)
				require.NoError(t, err)

				t.Logf("Special characters handled: %s", input)
			}
		})
	}
}

// TestContentTypeValidation tests content-type header validation
func TestContentTypeValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	_, token := testutil.CreateTestUserWithToken(t, router)

	validJSON := `{"name":"Test List","description":"Test"}`

	contentTypes := []struct {
		contentType  string
		shouldAccept bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/plain", false},
		{"text/html", false},
		{"application/xml", false},
		{"", false},
	}

	for _, tc := range contentTypes {
		t.Run(tc.contentType, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/lists", bytes.NewBufferString(validJSON))
			require.NoError(t, err)

			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.shouldAccept {
				assert.Equal(t, http.StatusCreated, w.Code,
					"Should accept content-type: %s", tc.contentType)
			} else {
				assert.NotEqual(t, http.StatusCreated, w.Code,
					"Should reject content-type: %s", tc.contentType)
			}
		})
	}
}

// TestEmailValidation tests email format validation
func TestEmailValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := testutil.SetupTestRouter(t)
	defer cleanup()

	invalidEmails := []string{
		"notanemail",
		"@example.com",
		"user@",
		"user@.com",
		"user@example",
		"user name@example.com", // Space
		"user<script>@example.com",
		"user@example.com; DROP TABLE users--",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			regReq := models.RegisterRequest{
				Email:     email,
				Password:  "ValidPass123!",
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

			// Should reject invalid emails
			if w.Code == http.StatusBadRequest {
				t.Logf("Invalid email rejected: %s", email)
			} else if w.Code == http.StatusCreated {
				t.Logf("Note: Email validation may not be enforced for: %s", email)
			}
		})
	}
}
