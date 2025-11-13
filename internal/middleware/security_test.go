package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"todolist-api/internal/logging"
)

var setupOnce sync.Once

func setupTest() {
	setupOnce.Do(func() {
		// Initialize logger for tests
		logging.InitLogger(&logging.LogConfig{
			Enabled:    false,
			Level:      "info",
			JSONFormat: false,
		})
		gin.SetMode(gin.TestMode)
	})
}

func TestSecurityHeaders(t *testing.T) {
	setupTest()
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check security headers
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "", w.Header().Get("X-Powered-By"))
	assert.Equal(t, "", w.Header().Get("Server"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "no-store")
}

func TestRequestSizeLimit(t *testing.T) {
	setupTest()
	maxSize := int64(100) // 100 bytes

	t.Run("allows requests under limit", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestSizeLimit(maxSize))
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		body := strings.NewReader(`{"name":"test"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestSizeLimit(maxSize))
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create body larger than limit
		largeBody := strings.Repeat("a", 200)
		req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(largeBody))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
		assert.Contains(t, w.Body.String(), "REQUEST_TOO_LARGE")
	})
}

func TestValidateUUID(t *testing.T) {
	setupTest()
	tests := []struct {
		name  string
		uuid  string
		valid bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID lowercase", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"valid UUID uppercase", "6BA7B810-9DAD-11D1-80B4-00C04FD430C8", true},
		{"invalid - too short", "550e8400-e29b-41d4-a716", false},
		{"invalid - too long", "550e8400-e29b-41d4-a716-446655440000-extra", false},
		{"invalid - wrong format", "550e8400e29b41d4a716446655440000", false},
		{"invalid - invalid chars", "550e8400-e29b-41d4-a716-44665544zzzz", false},
		{"empty string", "", false},
		{"non-UUID string", "not-a-uuid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUUID(tt.uuid)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestUUIDValidator(t *testing.T) {
	setupTest()
	t.Run("accepts valid UUID", func(t *testing.T) {
		router := gin.New()
		router.GET("/item/:id", UUIDValidator("id"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
		})

		req := httptest.NewRequest("GET", "/item/550e8400-e29b-41d4-a716-446655440000", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects invalid UUID", func(t *testing.T) {
		router := gin.New()
		router.GET("/item/:id", UUIDValidator("id"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
		})

		req := httptest.NewRequest("GET", "/item/not-a-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_UUID")
	})

	t.Run("validates multiple UUID parameters", func(t *testing.T) {
		router := gin.New()
		router.GET("/list/:listId/item/:itemId", UUIDValidator("listId", "itemId"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"listId": c.Param("listId"),
				"itemId": c.Param("itemId"),
			})
		})

		// Both valid
		req := httptest.NewRequest("GET", "/list/550e8400-e29b-41d4-a716-446655440000/item/6ba7b810-9dad-11d1-80b4-00c04fd430c8", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// First invalid
		req = httptest.NewRequest("GET", "/list/invalid/item/6ba7b810-9dad-11d1-80b4-00c04fd430c8", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Second invalid
		req = httptest.NewRequest("GET", "/list/550e8400-e29b-41d4-a716-446655440000/item/invalid", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSanitizeValue(t *testing.T) {
	setupTest()
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "simple string with HTML",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "map with HTML values",
			input:    map[string]interface{}{"name": "<b>test</b>"},
			expected: map[string]interface{}{"name": "&lt;b&gt;test&lt;/b&gt;"},
		},
		{
			name:     "nested map",
			input:    map[string]interface{}{"user": map[string]interface{}{"name": "<script>"}},
			expected: map[string]interface{}{"user": map[string]interface{}{"name": "&lt;script&gt;"}},
		},
		{
			name:     "array of strings",
			input:    []interface{}{"<div>", "<span>"},
			expected: []interface{}{"&lt;div&gt;", "&lt;span&gt;"},
		},
		{
			name:     "number - unchanged",
			input:    42,
			expected: 42,
		},
		{
			name:     "boolean - unchanged",
			input:    true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	setupTest()
	t.Run("sanitizes JSON POST request", func(t *testing.T) {
		router := gin.New()
		router.Use(SanitizeInput())
		router.POST("/test", func(c *gin.Context) {
			// Get sanitized input from context
			sanitized, exists := c.Get("sanitized_input")
			assert.True(t, exists)

			data := sanitized.(map[string]interface{})
			// Should be HTML escaped
			assert.Equal(t, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;", data["name"])

			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		body := bytes.NewBufferString(`{"name":"<script>alert('xss')</script>"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skips non-JSON requests", func(t *testing.T) {
		router := gin.New()
		router.Use(SanitizeInput())
		router.POST("/test", func(c *gin.Context) {
			_, exists := c.Get("sanitized_input")
			assert.False(t, exists, "Should not sanitize non-JSON requests")
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("POST", "/test", strings.NewReader("plain text"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skips GET requests", func(t *testing.T) {
		router := gin.New()
		router.Use(SanitizeInput())
		router.GET("/test", func(c *gin.Context) {
			_, exists := c.Get("sanitized_input")
			assert.False(t, exists, "Should not process GET requests")
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestErrorSanitizer(t *testing.T) {
	setupTest()
	t.Run("sanitizes 5xx errors", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorSanitizer())
		router.GET("/test", func(c *gin.Context) {
			c.Error(assert.AnError)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		// The last JSON response wins, so check for generic error
		assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	})

	t.Run("allows 4xx errors through", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorSanitizer())
		router.GET("/test", func(c *gin.Context) {
			c.Error(assert.AnError)
			c.JSON(http.StatusBadRequest, gin.H{"error": "validation error"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation error")
	})
}

func TestParseTrustedProxies(t *testing.T) {
	setupTest()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"single IP", "192.168.1.1", []string{"192.168.1.1"}},
		{"multiple IPs", "192.168.1.1,10.0.0.1", []string{"192.168.1.1", "10.0.0.1"}},
		{"with spaces", "192.168.1.1, 10.0.0.1 , 172.16.0.1", []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"}},
		{"trailing comma", "192.168.1.1,", []string{"192.168.1.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTrustedProxies(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
