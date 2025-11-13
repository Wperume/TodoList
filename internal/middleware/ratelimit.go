package middleware

import (
	"net/http"
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

	requestsPerMin, err := strconv.ParseInt(getEnv("RATE_LIMIT_REQUESTS_PER_MIN", "60"), 10, 64)
	if err != nil {
		requestsPerMin = 60 // default value
	}

	requestsPerHour, err := strconv.ParseInt(getEnv("RATE_LIMIT_REQUESTS_PER_HOUR", "1000"), 10, 64)
	if err != nil {
		requestsPerHour = 1000 // default value
	}

	burstSize, err := strconv.ParseInt(getEnv("RATE_LIMIT_BURST", "10"), 10, 64)
	if err != nil {
		burstSize = 10 // default value
	}

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

// createOperationRateLimiter creates a rate limiter for specific operation types
func createOperationRateLimiter(limit int64, limitType, message string) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  limit,
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return mgin.NewMiddleware(instance, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		logging.Logger.WithFields(map[string]interface{}{
			"client_ip":     c.ClientIP(),
			"path":          c.Request.URL.Path,
			"method":        c.Request.Method,
			"rate_limited":  true,
			"limit_type":    limitType,
			"limit_per_min": rate.Limit,
		}).Warnf("%s rate limit exceeded", limitType)

		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":       "RATE_LIMIT_EXCEEDED",
			"message":    message,
			"retryAfter": int(rate.Period.Seconds()),
			"limit":      rate.Limit,
		})
		c.Abort()
	}))
}

// ReadRateLimiter creates a rate limiter for read operations (GET requests)
func ReadRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Read operations can have higher limits
	return createOperationRateLimiter(
		config.RequestsPerMin*2,
		"read",
		"Too many read requests. Please try again later.",
	)
}

// WriteRateLimiter creates a rate limiter for write operations (POST, PUT, DELETE)
func WriteRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Write operations have stricter limits
	return createOperationRateLimiter(
		config.RequestsPerMin/2,
		"write",
		"Too many write requests. Please try again later.",
	)
}

// PerUserRateLimiter creates a rate limiter that tracks limits per user
// Unauthenticated requests fall back to IP-based rate limiting
func PerUserRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		logging.Logger.Info("Per-user rate limiting is disabled")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Per-user rate limit
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  config.RequestsPerMin,
	}

	// Create in-memory store
	store := memory.NewStore()

	// Create limiter instance
	instance := limiter.New(store, rate)

	// Custom key generator function
	keyGetter := func(c *gin.Context) string {
		// Try to get user ID from context (set by auth middleware)
		if userIDValue, exists := c.Get("user_id"); exists {
			if userIDStr, ok := userIDValue.(string); ok {
				return "user:" + userIDStr
			}
		}

		// Fall back to IP-based limiting for unauthenticated requests
		return "ip:" + c.ClientIP()
	}

	// Create middleware with custom error handler and key generator
	middleware := mgin.NewMiddleware(instance,
		mgin.WithKeyGetter(keyGetter),
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			// Determine if this is a user or IP-based limit
			limitType := "ip"
			identifier := c.ClientIP()

			if userIDValue, exists := c.Get("user_id"); exists {
				if userIDStr, ok := userIDValue.(string); ok {
					limitType = "user"
					identifier = userIDStr
				}
			}

			// Log rate limit violation
			logging.Logger.WithFields(map[string]interface{}{
				"limit_type":      limitType,
				"identifier":      identifier,
				"client_ip":       c.ClientIP(),
				"path":            c.Request.URL.Path,
				"method":          c.Request.Method,
				"rate_limited":    true,
				"limit_per_min":   config.RequestsPerMin,
			}).Warn("Per-user rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":       "RATE_LIMIT_EXCEEDED",
				"message":    "Too many requests from this account. Please try again later.",
				"retryAfter": int(rate.Period.Seconds()),
				"limit":      rate.Limit,
			})
			c.Abort()
		}))

	logging.Logger.Infof("Per-user rate limiting enabled: %d requests per minute", config.RequestsPerMin)
	return middleware
}

// PerUserAuthRateLimiter creates a stricter rate limiter for authentication endpoints
// to prevent brute-force attacks
func PerUserAuthRateLimiter(config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Stricter limits for auth endpoints
	rate := limiter.Rate{
		Period: 15 * time.Minute,
		Limit:  5, // Only 5 attempts per 15 minutes
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	// Custom key generator - use IP for auth endpoints since we can't reliably parse body
	keyGetter := func(c *gin.Context) string {
		// Use IP for authentication rate limiting
		// This prevents brute-force attacks from a single IP
		return "auth:ip:" + c.ClientIP()
	}

	middleware := mgin.NewMiddleware(instance,
		mgin.WithKeyGetter(keyGetter),
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			logging.Logger.WithFields(map[string]interface{}{
				"client_ip":       c.ClientIP(),
				"path":            c.Request.URL.Path,
				"method":          c.Request.Method,
				"rate_limited":    true,
				"limit_type":      "auth",
				"limit":           rate.Limit,
				"period_minutes":  int(rate.Period.Minutes()),
			}).Warn("Authentication rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":       "AUTH_RATE_LIMIT_EXCEEDED",
				"message":    "Too many authentication attempts. Please try again later.",
				"retryAfter": int(rate.Period.Seconds()),
				"limit":      rate.Limit,
			})
			c.Abort()
		}))

	logging.Logger.Infof("Auth rate limiting enabled: %d attempts per 15 minutes", rate.Limit)
	return middleware
}
