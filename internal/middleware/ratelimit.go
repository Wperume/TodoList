package middleware

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"todolist-api/internal/logging"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool
	RequestsPerMin  int64
	RequestsPerHour int64
	BurstSize       int64
}

// NewRateLimitConfigFromEnv creates rate limit config from environment variables
func NewRateLimitConfigFromEnv() *RateLimitConfig {
	enabled := getEnv("RATE_LIMIT_ENABLED", "true") == "true"

	requestsPerMin, _ := strconv.ParseInt(getEnv("RATE_LIMIT_REQUESTS_PER_MIN", "60"), 10, 64)
	requestsPerHour, _ := strconv.ParseInt(getEnv("RATE_LIMIT_REQUESTS_PER_HOUR", "1000"), 10, 64)
	burstSize, _ := strconv.ParseInt(getEnv("RATE_LIMIT_BURST", "10"), 10, 64)

	return &RateLimitConfig{
		Enabled:         enabled,
		RequestsPerMin:  requestsPerMin,
		RequestsPerHour: requestsPerHour,
		BurstSize:       burstSize,
	}
}

// GlobalRateLimiter creates a global rate limiter middleware
func GlobalRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	// If rate limiting is disabled, return a no-op middleware
	if !config.Enabled {
		logging.Logger.Info("Rate limiting is disabled")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Create rate limit: use per-minute limit
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  config.RequestsPerMin,
	}

	// Create in-memory store
	store := memory.NewStore()

	// Create limiter instance
	instance := limiter.New(store, rate)

	// Create middleware with custom error handler
	middleware := mgin.NewMiddleware(instance, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		// Log rate limit violation with client details
		logging.Logger.WithFields(map[string]interface{}{
			"client_ip":       c.ClientIP(),
			"path":            c.Request.URL.Path,
			"method":          c.Request.Method,
			"rate_limited":    true,
			"limit_per_min":   config.RequestsPerMin,
		}).Warn("Rate limit exceeded")

		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":    "RATE_LIMIT_EXCEEDED",
			"message": "Too many requests. Please try again later.",
			"retryAfter": int(rate.Period.Seconds()),
		})
		c.Abort()
	}))

	logging.Logger.Infof("Rate limiting enabled: %d requests per minute", config.RequestsPerMin)
	return middleware
}

// ReadRateLimiter creates a rate limiter for read operations (GET requests)
func ReadRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Read operations can have higher limits
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  config.RequestsPerMin * 2, // Double the limit for reads
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	middleware := mgin.NewMiddleware(instance, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		logging.Logger.WithFields(map[string]interface{}{
			"client_ip":       c.ClientIP(),
			"path":            c.Request.URL.Path,
			"method":          c.Request.Method,
			"rate_limited":    true,
			"limit_type":      "read",
			"limit_per_min":   rate.Limit,
		}).Warn("Read rate limit exceeded")

		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":       "RATE_LIMIT_EXCEEDED",
			"message":    "Too many read requests. Please try again later.",
			"retryAfter": int(rate.Period.Seconds()),
			"limit":      rate.Limit,
		})
		c.Abort()
	}))

	return middleware
}

// WriteRateLimiter creates a rate limiter for write operations (POST, PUT, DELETE)
func WriteRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Write operations have stricter limits
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  config.RequestsPerMin / 2, // Half the limit for writes
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	middleware := mgin.NewMiddleware(instance, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		logging.Logger.WithFields(map[string]interface{}{
			"client_ip":       c.ClientIP(),
			"path":            c.Request.URL.Path,
			"method":          c.Request.Method,
			"rate_limited":    true,
			"limit_type":      "write",
			"limit_per_min":   rate.Limit,
		}).Warn("Write rate limit exceeded")

		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":       "RATE_LIMIT_EXCEEDED",
			"message":    "Too many write requests. Please try again later.",
			"retryAfter": int(rate.Period.Seconds()),
			"limit":      rate.Limit,
		})
		c.Abort()
	}))

	return middleware
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
