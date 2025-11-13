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
	setupTest()
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

		// Invalid values should fall back to defaults
		assert.Equal(t, int64(60), config.RequestsPerMin)
		assert.Equal(t, int64(1000), config.RequestsPerHour)
		assert.Equal(t, int64(10), config.BurstSize)
	})
}

func TestGlobalRateLimiter(t *testing.T) {
	setupTest()
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
	setupTest()
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
	setupTest()
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
	setupTest()
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

func TestPerUserRateLimiter(t *testing.T) {
	setupTest()
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when disabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: false,
		}

		router := gin.New()
		router.Use(PerUserRateLimiter(config))
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

	t.Run("enforces per-user rate limit for authenticated users", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 5, // Very low limit for testing
		}

		router := gin.New()
		router.Use(PerUserRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			// Simulate auth middleware setting user_id
			c.Set("user_id", "user-123")
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		successCount := 0
		rateLimitedCount := 0

		// Make more requests than the limit from the same user
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"

			// Create a new context and set user_id before the middleware runs
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", "user-123")

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

	t.Run("different users have independent rate limits", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 3, // Low limit for testing
		}

		router := gin.New()
		router.Use(func(c *gin.Context) {
			// Simulate auth middleware - get user from header
			userID := c.GetHeader("X-User-ID")
			if userID != "" {
				c.Set("user_id", userID)
			}
			c.Next()
		})
		router.Use(PerUserRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// User 1 makes requests
		user1Success := 0
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				user1Success++
			}
		}

		// User 2 makes requests - should have their own limit
		user2Success := 0
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-User-ID", "user-2")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				user2Success++
			}
		}

		// Both users should get similar number of successful requests
		assert.LessOrEqual(t, user1Success, 3, "User 1 should be rate limited")
		assert.LessOrEqual(t, user2Success, 3, "User 2 should be rate limited")
		assert.Greater(t, user1Success, 0, "User 1 should have some successful requests")
		assert.Greater(t, user2Success, 0, "User 2 should have some successful requests")
	})

	t.Run("falls back to IP-based limiting for unauthenticated requests", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 3,
		}

		router := gin.New()
		router.Use(PerUserRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		successCount := 0
		rateLimitedCount := 0

		// Make requests without user_id (unauthenticated)
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345" // Same IP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		// Should enforce rate limit based on IP
		assert.LessOrEqual(t, successCount, 3, "Should not exceed the rate limit")
		assert.Greater(t, rateLimitedCount, 0, "Should have rate-limited some requests")
	})

	t.Run("returns correct error format when rate limited", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 1,
		}

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-456")
			c.Next()
		})
		router.Use(PerUserRateLimiter(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should be rate limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		require.Equal(t, http.StatusTooManyRequests, w2.Code)

		// Verify error response format
		assert.Contains(t, w2.Body.String(), "RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w2.Body.String(), "Too many requests")
		assert.Contains(t, w2.Body.String(), "retryAfter")
		assert.Contains(t, w2.Body.String(), "limit")
	})
}

func TestPerUserAuthRateLimiter(t *testing.T) {
	setupTest()
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when disabled", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: false,
		}

		router := gin.New()
		router.Use(PerUserAuthRateLimiter(config))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Make multiple requests - all should succeed
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("enforces strict auth rate limit (5 attempts per 15 minutes)", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60, // This is ignored for auth limiter
		}

		router := gin.New()
		router.Use(PerUserAuthRateLimiter(config))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		successCount := 0
		rateLimitedCount := 0

		// Make more than 5 attempts from same IP
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.50:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		// Should allow at most 5 successful attempts
		assert.LessOrEqual(t, successCount, 5, "Should not exceed 5 auth attempts")
		assert.Greater(t, rateLimitedCount, 0, "Should have rate-limited some requests")
	})

	t.Run("different IPs have independent auth rate limits", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: true,
		}

		router := gin.New()
		router.Use(PerUserAuthRateLimiter(config))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// IP 1 makes requests
		ip1Success := 0
		for i := 0; i < 7; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.10:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				ip1Success++
			}
		}

		// IP 2 makes requests - should have their own limit
		ip2Success := 0
		for i := 0; i < 7; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.20:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				ip2Success++
			}
		}

		// Both IPs should have independent limits
		assert.LessOrEqual(t, ip1Success, 5, "IP 1 should be rate limited to 5 attempts")
		assert.LessOrEqual(t, ip2Success, 5, "IP 2 should be rate limited to 5 attempts")
		assert.Greater(t, ip1Success, 0, "IP 1 should have some successful requests")
		assert.Greater(t, ip2Success, 0, "IP 2 should have some successful requests")
	})

	t.Run("returns correct error format when auth rate limited", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled: true,
		}

		router := gin.New()
		router.Use(PerUserAuthRateLimiter(config))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Make 5 successful requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.200:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 6th request should be rate limited
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "192.168.1.200:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusTooManyRequests, w.Code)

		// Verify error response format
		assert.Contains(t, w.Body.String(), "AUTH_RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "authentication attempts")
		assert.Contains(t, w.Body.String(), "retryAfter")
		assert.Contains(t, w.Body.String(), "limit")
	})
}
