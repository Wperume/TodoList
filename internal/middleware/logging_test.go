package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"todolist-api/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRequestLogger(t *testing.T) {
	setupTest()
	// Initialize logger for tests
	logging.InitLogger(&logging.LogConfig{
		Enabled:    false,
		Level:      "info",
		JSONFormat: false,
	})

	gin.SetMode(gin.TestMode)

	t.Run("logs successful requests", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "Request completed")
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/test")
		assert.Contains(t, logOutput, "status=200")
	})

	t.Run("logs client errors", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "Client error")
		assert.Contains(t, logOutput, "status=400")
	})

	t.Run("logs server errors", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "Server error")
		assert.Contains(t, logOutput, "status=500")
	})

	t.Run("logs rate limited requests", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "rate_limited=true")
		assert.Contains(t, logOutput, "status=429")
	})

	t.Run("logs API key prefix when present", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		req.Header.Set("X-API-Key", "my-secret-api-key-12345")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "api_key_prefix")
		assert.Contains(t, logOutput, "my-secre...")
		assert.NotContains(t, logOutput, "my-secret-api-key-12345") // Full key should not be logged
	})

	t.Run("logs short API key without truncation", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		req.Header.Set("X-API-Key", "short")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "api_key_prefix=short")
	})

	t.Run("logs user agent when present", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		req.Header.Set("User-Agent", "TestClient/1.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "user_agent")
		assert.Contains(t, logOutput, "TestClient/1.0")
	})

	t.Run("logs query parameters", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test?foo=bar&baz=qux", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "query")
		assert.Contains(t, logOutput, "foo=bar")
	})

	t.Run("includes latency information", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "latency_ms")
	})
}

func TestStructuredLogger(t *testing.T) {
	setupTest()
	// Initialize logger with JSON format for structured logging
	logging.InitLogger(&logging.LogConfig{
		Enabled:    false,
		Level:      "info",
		JSONFormat: true,
	})

	gin.SetMode(gin.TestMode)

	t.Run("logs in structured format", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(StructuredLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()

		// JSON format should include these fields
		assert.Contains(t, logOutput, "\"method\":\"GET\"")
		assert.Contains(t, logOutput, "\"path\":\"/test\"")
		assert.Contains(t, logOutput, "\"status\":200")
		assert.Contains(t, logOutput, "\"latency_ms\":")
	})

	t.Run("includes all relevant fields", func(t *testing.T) {
		var buf bytes.Buffer
		logging.Logger.SetOutput(&buf)

		router := gin.New()
		router.Use(StructuredLogger())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test?param=value", http.NoBody)
		req.Header.Set("X-API-Key", "test-key-123")
		req.Header.Set("User-Agent", "TestAgent/1.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()

		assert.Contains(t, logOutput, "\"query\":\"param=value\"")
		assert.Contains(t, logOutput, "\"api_key_prefix\":\"test-key...\"") // Truncated to first 8 chars
		assert.Contains(t, logOutput, "\"user_agent\":\"TestAgent/1.0\"")
		assert.Contains(t, logOutput, "\"timestamp\":")
		assert.Contains(t, logOutput, "\"client_ip\":")
	})
}

func TestLogLevels(t *testing.T) {
	setupTest()
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		statusCode int
		logLevel   logrus.Level
		levelStr   string
	}{
		{"2xx success", 200, logrus.InfoLevel, "info"},
		{"3xx redirect", 301, logrus.InfoLevel, "info"},
		{"400 bad request", 400, logrus.WarnLevel, "warning"},
		{"404 not found", 404, logrus.WarnLevel, "warning"},
		{"429 rate limited", 429, logrus.WarnLevel, "warning"},
		{"500 server error", 500, logrus.ErrorLevel, "error"},
		{"503 unavailable", 503, logrus.ErrorLevel, "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logging.InitLogger(&logging.LogConfig{
				Enabled:    false,
				Level:      "trace", // Set to trace to capture all levels
				JSONFormat: false,
			})
			logging.Logger.SetOutput(&buf)

			router := gin.New()
			router.Use(RequestLogger())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(tc.statusCode, gin.H{"status": tc.statusCode})
			})

			req := httptest.NewRequest("GET", "/test", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.statusCode, w.Code)
			logOutput := buf.String()
			assert.Contains(t, logOutput, "level="+tc.levelStr)
		})
	}
}
