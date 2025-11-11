package middleware

import (
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"todolist-api/internal/logging"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	MaxRequestBodySize int64  // Maximum request body size in bytes
	EnableXSSProtection bool  // Enable XSS input sanitization
	TrustedProxies     []string // List of trusted proxy IPs
}

// NewSecurityConfigFromEnv creates security config from environment variables
func NewSecurityConfigFromEnv() *SecurityConfig {
	maxSize := getEnvInt("MAX_REQUEST_BODY_SIZE", 1048576) // Default 1MB

	return &SecurityConfig{
		MaxRequestBodySize:  int64(maxSize),
		EnableXSSProtection: getEnvBool("ENABLE_XSS_PROTECTION", true),
		TrustedProxies:      parseTrustedProxies(getEnv("TRUSTED_PROXIES", "")),
	}
}

// parseTrustedProxies parses comma-separated list of proxy IPs
func parseTrustedProxies(proxies string) []string {
	if proxies == "" {
		return nil
	}
	parts := strings.Split(proxies, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection in browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Prevent information leakage
		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		// Content Security Policy (strict for API)
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Referrer policy
		c.Header("Referrer-Policy", "no-referrer")

		// Prevent browsers from caching sensitive data
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// RequestSizeLimit limits the size of incoming request bodies
func RequestSizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			logging.Logger.WithFields(map[string]interface{}{
				"client_ip":      c.ClientIP(),
				"content_length": c.Request.ContentLength,
				"max_size":       maxSize,
			}).Warn("Request body too large")

			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":    "REQUEST_TOO_LARGE",
				"message": "Request body too large",
				"max_size_bytes": maxSize,
			})
			c.Abort()
			return
		}

		// Set a hard limit on the request body reader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// SanitizeInput sanitizes user input to prevent XSS attacks
// This middleware processes JSON request bodies and HTML-escapes string values
func SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process POST, PUT, PATCH requests with JSON content
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.Next()
			return
		}

		// Get the raw body
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			// Let the handler deal with invalid JSON
			c.Next()
			return
		}

		// Recursively sanitize all string values
		sanitized := sanitizeValue(body)

		// Store sanitized data in context for handlers to use
		c.Set("sanitized_input", sanitized)

		c.Next()
	}
}

// sanitizeValue recursively sanitizes all string values in a data structure
func sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// HTML escape to prevent XSS
		return html.EscapeString(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = sanitizeValue(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = sanitizeValue(val)
		}
		return result
	default:
		return v
	}
}

// ErrorSanitizer catches errors and returns sanitized error messages
func ErrorSanitizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Log the full error details
			logging.Logger.WithFields(map[string]interface{}{
				"client_ip": c.ClientIP(),
				"path":      c.Request.URL.Path,
				"method":    c.Request.Method,
				"error":     err.Error(),
			}).Error("Request error")

			// Don't expose internal error details to client
			// The handler should have already set an appropriate response
			// This is just a safety net
			if c.Writer.Status() >= 500 {
				// For 5xx errors, return generic message
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "An internal error occurred. Please try again later.",
				})
			}
		}
	}
}

// ValidateUUID is a helper function to validate UUID strings
func ValidateUUID(uuidStr string) bool {
	// Basic UUID format validation (8-4-4-4-12)
	if len(uuidStr) != 36 {
		return false
	}

	// Check format
	parts := strings.Split(uuidStr, "-")
	if len(parts) != 5 {
		return false
	}

	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 ||
	   len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}

	// Check that all characters are valid hex
	validChars := "0123456789abcdefABCDEF-"
	for _, char := range uuidStr {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

// UUIDValidator validates UUID path parameters
func UUIDValidator(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, param := range params {
			uuidStr := c.Param(param)
			if uuidStr != "" && !ValidateUUID(uuidStr) {
				logging.Logger.WithFields(map[string]interface{}{
					"client_ip":  c.ClientIP(),
					"path":       c.Request.URL.Path,
					"param":      param,
					"value":      uuidStr,
				}).Warn("Invalid UUID format")

				c.JSON(http.StatusBadRequest, gin.H{
					"code":    "INVALID_UUID",
					"message": "Invalid UUID format",
					"field":   param,
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
