package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows all origins with wildcard", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("allows specific origin", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("blocks disallowed origin", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("handles preflight OPTIONS request", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         3600,
		}

		router := gin.New()
		router.Use(CORS(config))
		router.OPTIONS("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	})

	t.Run("allows credentials when configured", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("exposes headers when configured", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET"},
			AllowedHeaders: []string{"Content-Type"},
			ExposeHeaders:  []string{"X-Total-Count", "X-Page"},
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Total-Count")
	})

	t.Run("disabled CORS does nothing", func(t *testing.T) {
		config := &CORSConfig{
			Enabled: false,
		}

		router := gin.New()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{"wildcard allows all", "https://example.com", []string{"*"}, true},
		{"exact match", "https://example.com", []string{"https://example.com"}, true},
		{"no match", "https://evil.com", []string{"https://example.com"}, false},
		{"multiple allowed, matches first", "https://app.com", []string{"https://app.com", "https://api.com"}, true},
		{"multiple allowed, matches second", "https://api.com", []string{"https://app.com", "https://api.com"}, true},
		{"subdomain wildcard matches", "https://sub.example.com", []string{"*.example.com"}, true},
		{"subdomain wildcard matches deep", "https://a.b.example.com", []string{"*.example.com"}, true},
		{"subdomain wildcard no match", "https://evil.com", []string{"*.example.com"}, false},
		{"empty allowed list", "https://example.com", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single value", "GET", []string{"GET"}},
		{"multiple values", "GET,POST,PUT", []string{"GET", "POST", "PUT"}},
		{"with spaces", "GET, POST , PUT", []string{"GET", "POST", "PUT"}},
		{"trailing comma", "GET,POST,", []string{"GET", "POST"}},
		{"leading comma", ",GET,POST", []string{"GET", "POST"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
