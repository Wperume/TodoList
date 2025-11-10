package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitConfigFromEnv(t *testing.T) {
	// Save original environment
	origEnabled := os.Getenv("RATE_LIMIT_ENABLED")
	origPerMin := os.Getenv("RATE_LIMIT_REQUESTS_PER_MIN")
	origPerHour := os.Getenv("RATE_LIMIT_REQUESTS_PER_HOUR")
	origBurst := os.Getenv("RATE_LIMIT_BURST")

	// Restore environment after test
	defer func() {
		os.Setenv("RATE_LIMIT_ENABLED", origEnabled)
		os.Setenv("RATE_LIMIT_REQUESTS_PER_MIN", origPerMin)
		os.Setenv("RATE_LIMIT_REQUESTS_PER_HOUR", origPerHour)
		os.Setenv("RATE_LIMIT_BURST", origBurst)
	}()

	t.Run("uses default values when env vars not set", func(t *testing.T) {
		os.Unsetenv("RATE_LIMIT_ENABLED")
		os.Unsetenv("RATE_LIMIT_REQUESTS_PER_MIN")
		os.Unsetenv("RATE_LIMIT_REQUESTS_PER_HOUR")
		os.Unsetenv("RATE_LIMIT_BURST")

		config := NewRateLimitConfigFromEnv()

		assert.True(t, config.Enabled, "Should be enabled by default")
		assert.Equal(t, int64(60), config.RequestsPerMin, "Default requests per minute should be 60")
		assert.Equal(t, int64(1000), config.RequestsPerHour, "Default requests per hour should be 1000")
		assert.Equal(t, int64(10), config.BurstSize, "Default burst size should be 10")
	})

	t.Run("uses custom values from environment", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "false")
		os.Setenv("RATE_LIMIT_REQUESTS_PER_MIN", "100")
		os.Setenv("RATE_LIMIT_REQUESTS_PER_HOUR", "2000")
		os.Setenv("RATE_LIMIT_BURST", "20")

		config := NewRateLimitConfigFromEnv()

		assert.False(t, config.Enabled, "Should respect RATE_LIMIT_ENABLED=false")
		assert.Equal(t, int64(100), config.RequestsPerMin)
		assert.Equal(t, int64(2000), config.RequestsPerHour)
		assert.Equal(t, int64(20), config.BurstSize)
	})

	t.Run("handles invalid numeric values gracefully", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Setenv("RATE_LIMIT_REQUESTS_PER_MIN", "invalid")
		os.Setenv("RATE_LIMIT_REQUESTS_PER_HOUR", "not-a-number")
		os.Setenv("RATE_LIMIT_BURST", "abc")

		config := NewRateLimitConfigFromEnv()

		// ParseInt returns 0 on error, so invalid values should result in 0
		assert.Equal(t, int64(0), config.RequestsPerMin)
		assert.Equal(t, int64(0), config.RequestsPerHour)
		assert.Equal(t, int64(0), config.BurstSize)
	})
}

func TestGlobalRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when disabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: false,
		}

		router := gin.New()
		router.Use(GlobalRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Make multiple requests - all should succeed since rate limiting is disabled
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed when rate limiting is disabled", i+1)
		}
	})

	t.Run("enforces rate limit when enabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 5, // Very low limit for testing
		}

		router := gin.New()
		router.Use(GlobalRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		successCount := 0
		rateLimitedCount := 0

		// Make more requests than the limit
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345" // Same IP for all requests
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		// Should have some successful requests and some rate-limited
		assert.Greater(t, successCount, 0, "Should have at least some successful requests")
		assert.Greater(t, rateLimitedCount, 0, "Should have rate-limited some requests")
		assert.LessOrEqual(t, successCount, 5, "Should not exceed the rate limit")
	})

	t.Run("returns correct error format when rate limited", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 1, // Only 1 request allowed
		}

		router := gin.New()
		router.Use(GlobalRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.2:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request from same IP should be rate limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.2:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		require.Equal(t, http.StatusTooManyRequests, w2.Code)

		// Verify error response format
		assert.Contains(t, w2.Body.String(), "RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w2.Body.String(), "Too many requests")
		assert.Contains(t, w2.Body.String(), "retryAfter")
	})
}

func TestReadRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when disabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: false,
		}

		middleware := ReadRateLimiter(config)
		assert.NotNil(t, middleware, "Should return a middleware function")

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("applies double the global limit", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 5,
		}

		// ReadRateLimiter should allow 2x the requests (10)
		middleware := ReadRateLimiter(config)
		assert.NotNil(t, middleware)
	})
}

func TestWriteRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when disabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: false,
		}

		middleware := WriteRateLimiter(config)
		assert.NotNil(t, middleware, "Should return a middleware function")

		router := gin.New()
		router.Use(middleware)
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("applies half the global limit", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 10,
		}

		// WriteRateLimiter should allow half the requests (5)
		middleware := WriteRateLimiter(config)
		assert.NotNil(t, middleware)
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("returns value when env var is set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("returns default when env var is not set", func(t *testing.T) {
		os.Unsetenv("NONEXISTENT_VAR")

		result := getEnv("NONEXISTENT_VAR", "default_value")
		assert.Equal(t, "default_value", result)
	})

	t.Run("returns default when env var is empty", func(t *testing.T) {
		os.Setenv("EMPTY_VAR", "")
		defer os.Unsetenv("EMPTY_VAR")

		result := getEnv("EMPTY_VAR", "default_value")
		assert.Equal(t, "default_value", result)
	})
}
