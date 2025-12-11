package security_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"todolist-api/internal/models"
	"todolist-api/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAccessOtherUsersLists tests that users cannot access other users' lists
func TestAccessOtherUsersLists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	// Create two users
	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	// User1 creates a list
	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)

	// User2 tries to access User1's list
	req, err := http.NewRequest("GET", "/api/v1/lists/"+list1.ID.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or 403 (not found/forbidden)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
		"User2 should not be able to access User1's list")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response, "error")
}

// TestModifyOtherUsersList tests that users cannot modify other users' lists
func TestModifyOtherUsersList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	// User1 creates a list
	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)

	// User2 tries to update User1's list
	updateReq := models.UpdateTodoListRequest{
		Name: stringPtr("Hacked List"),
	}

	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/v1/lists/"+list1.ID.String(), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be denied
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
		"User2 should not be able to update User1's list")
}

// TestDeleteOtherUsersList tests that users cannot delete other users' lists
func TestDeleteOtherUsersList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	// User1 creates a list
	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)

	// User2 tries to delete User1's list
	req, err := http.NewRequest("DELETE", "/api/v1/lists/"+list1.ID.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be denied
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
		"User2 should not be able to delete User1's list")

	// Verify list still exists for User1
	req, err = http.NewRequest("GET", "/api/v1/lists/"+list1.ID.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token1)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "User1 should still be able to access their list")
}

// TestAccessOtherUsersTodos tests that users cannot access other users' todos
func TestAccessOtherUsersTodos(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	// User1 creates a list and todo
	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)
	todo1 := testutil.CreateTestTodoViaAPI(t, router, token1, list1.ID)

	// User2 tries to access User1's todo
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/lists/%s/todos/%s", list1.ID, todo1.ID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be denied
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
		"User2 should not be able to access User1's todo")
}

// TestModifyOtherUsersTodo tests that users cannot modify other users' todos
func TestModifyOtherUsersTodo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)
	todo1 := testutil.CreateTestTodoViaAPI(t, router, token1, list1.ID)

	// User2 tries to update User1's todo
	updateReq := models.UpdateTodoRequest{
		Description: stringPtr("Hacked todo"),
	}

	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/api/v1/lists/%s/todos/%s", list1.ID, todo1.ID), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be denied
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
		"User2 should not be able to update User1's todo")
}

// TestInsecureDirectObjectReference tests IDOR vulnerabilities
func TestInsecureDirectObjectReference(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	user1, token1 := CreateTestUserWithToken(t, router)
	_, token2 := CreateTestUserWithToken(t, router)

	// Create resources for user1
	list1 := testutil.CreateTestListViaAPI(t, router, token1, user1.ID)

	// Try various IDOR attacks
	idorAttempts := []struct {
		name     string
		method   string
		endpoint string
	}{
		{"GET list", "GET", "/api/v1/lists/" + list1.ID.String()},
		{"UPDATE list", "PUT", "/api/v1/lists/" + list1.ID.String()},
		{"DELETE list", "DELETE", "/api/v1/lists/" + list1.ID.String()},
		{"GET todos", "GET", "/api/v1/lists/" + list1.ID.String() + "/todos"},
	}

	for _, attempt := range idorAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			var body []byte
			if attempt.method == "PUT" {
				body = []byte(`{"name":"hacked"}`)
			}

			req, err := http.NewRequest(attempt.method, attempt.endpoint, bytes.NewBuffer(body))
			require.NoError(t, err)

			if len(body) > 0 {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer "+token2)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// All attempts should be denied
			assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, w.Code,
				"IDOR attempt should be blocked: %s", attempt.name)
		})
	}
}

// TestMassAssignment tests mass assignment vulnerabilities
func TestMassAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	_, token := CreateTestUserWithToken(t, router)

	// Try to set fields that shouldn't be settable via mass assignment
	maliciousReq := map[string]interface{}{
		"name":        "Test List",
		"description": "Test",
		"user_id":     "00000000-0000-0000-0000-000000000000", // Try to assign to different user
		"id":          "11111111-1111-1111-1111-111111111111", // Try to set ID
		"created_at":  "2020-01-01T00:00:00Z",                 // Try to set created_at
		"updated_at":  "2020-01-01T00:00:00Z",                 // Try to set updated_at
	}

	body, err := json.Marshal(maliciousReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/lists", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		var list models.TodoList
		err = json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)

		// Verify that protected fields were not set from request
		assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", list.UserID.String(),
			"Should not allow setting user_id via mass assignment")
		assert.NotEqual(t, "11111111-1111-1111-1111-111111111111", list.ID.String(),
			"Should not allow setting ID via mass assignment")
	}
}

// TestPrivilegeEscalation tests for privilege escalation vulnerabilities
func TestPrivilegeEscalation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	_, token := CreateTestUserWithToken(t, router)

	// Try to escalate privileges via profile update
	escalationAttempts := []map[string]interface{}{
		{"role": "admin"},
		{"role": "superuser"},
		{"is_admin": true},
		{"permissions": []string{"admin", "superuser"}},
	}

	for _, attempt := range escalationAttempts {
		t.Run(fmt.Sprintf("%v", attempt), func(t *testing.T) {
			body, err := json.Marshal(attempt)
			require.NoError(t, err)

			req, err := http.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Even if request succeeds, role should not change
			if w.Code == http.StatusOK {
				var user models.User
				err = json.Unmarshal(w.Body.Bytes(), &user)
				require.NoError(t, err)

				assert.Equal(t, models.RoleUser, user.Role,
					"Should not allow privilege escalation via profile update")
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
