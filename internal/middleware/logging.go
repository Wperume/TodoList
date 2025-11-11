package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"todolist-api/internal/logging"
)

// RequestLogger is a middleware that logs HTTP requests with detailed information
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Get client IP
		clientIP := c.ClientIP()

		// Create log entry with initial fields
		logEntry := logging.Logger.WithFields(logrus.Fields{
			"client_ip": clientIP,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"query":     c.Request.URL.RawQuery,
		})

		// Add API key if present (for future authentication)
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			// Log only first 8 characters for security
			if len(apiKey) > 8 {
				logEntry = logEntry.WithField("api_key_prefix", apiKey[:8]+"...")
			} else {
				logEntry = logEntry.WithField("api_key_prefix", apiKey)
			}
		}

		// Add User-Agent
		if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
			logEntry = logEntry.WithField("user_agent", userAgent)
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get response status
		statusCode := c.Writer.Status()

		// Add response fields
		logEntry = logEntry.WithFields(logrus.Fields{
			"status":       statusCode,
			"latency_ms":   latency.Milliseconds(),
			"response_size": c.Writer.Size(),
		})

		// Add error if present
		if len(c.Errors) > 0 {
			logEntry = logEntry.WithField("errors", c.Errors.String())
		}

		// Check if request was rate limited
		if statusCode == 429 {
			logEntry = logEntry.WithField("rate_limited", true)
		}

		// Log at appropriate level based on status code
		switch {
		case statusCode >= 500:
			logEntry.Error("Server error")
		case statusCode >= 400:
			logEntry.Warn("Client error")
		case statusCode >= 300:
			logEntry.Info("Redirect")
		default:
			logEntry.Info("Request completed")
		}
	}
}

// StructuredLogger returns a middleware that logs requests in JSON format
// This is useful for log aggregation systems
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Build structured log entry
		entry := map[string]interface{}{
			"timestamp":     startTime.Format(time.RFC3339),
			"client_ip":     c.ClientIP(),
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"query":         c.Request.URL.RawQuery,
			"status":        c.Writer.Status(),
			"latency_ms":    latency.Milliseconds(),
			"latency":       latency.String(),
			"response_size": c.Writer.Size(),
		}

		// Add API key if present
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			if len(apiKey) > 8 {
				entry["api_key_prefix"] = apiKey[:8] + "..."
			} else {
				entry["api_key_prefix"] = apiKey
			}
		}

		// Add user agent
		if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
			entry["user_agent"] = userAgent
		}

		// Add errors if present
		if len(c.Errors) > 0 {
			entry["errors"] = c.Errors.String()
		}

		// Check if rate limited
		if c.Writer.Status() == 429 {
			entry["rate_limited"] = true
		}

		// Log with appropriate level
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			logging.Logger.WithFields(entry).Error("Server error")
		case statusCode >= 400:
			logging.Logger.WithFields(entry).Warn("Client error")
		case statusCode >= 300:
			logging.Logger.WithFields(entry).Info("Redirect")
		default:
			logging.Logger.WithFields(entry).Info("Request completed")
		}
	}
}
