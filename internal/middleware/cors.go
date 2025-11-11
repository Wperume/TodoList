package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"todolist-api/internal/logging"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string // List of allowed origins, or ["*"] for all
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// NewCORSConfigFromEnv creates CORS config from environment variables
func NewCORSConfigFromEnv() *CORSConfig {
	enabled := getEnvBool("CORS_ENABLED", true)

	// Parse allowed origins
	originsStr := getEnv("CORS_ALLOWED_ORIGINS", "*")
	var origins []string
	if originsStr == "*" {
		origins = []string{"*"}
	} else {
		origins = parseCommaSeparated(originsStr)
	}

	// Parse allowed methods
	methodsStr := getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
	methods := parseCommaSeparated(methodsStr)

	// Parse allowed headers
	headersStr := getEnv("CORS_ALLOWED_HEADERS", "Origin,Content-Type,Accept,Authorization,X-API-Key")
	headers := parseCommaSeparated(headersStr)

	// Parse expose headers
	exposeStr := getEnv("CORS_EXPOSE_HEADERS", "Content-Length,Content-Type")
	expose := parseCommaSeparated(exposeStr)

	allowCredentials := getEnvBool("CORS_ALLOW_CREDENTIALS", false)
	maxAge := getEnvInt("CORS_MAX_AGE", 3600) // 1 hour default

	return &CORSConfig{
		Enabled:          enabled,
		AllowedOrigins:   origins,
		AllowedMethods:   methods,
		AllowedHeaders:   headers,
		ExposeHeaders:    expose,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	}
}

// parseCommaSeparated parses comma-separated string into slice
func parseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
			// Set CORS headers
			c.Header("Access-Control-Allow-Origin", origin)

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			if len(config.ExposeHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}

			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))

				logging.Logger.WithFields(map[string]interface{}{
					"client_ip": c.ClientIP(),
					"origin":    origin,
					"method":    c.Request.Method,
				}).Debug("CORS preflight request")

				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		} else if origin != "" {
			// Origin not allowed
			logging.Logger.WithFields(map[string]interface{}{
				"client_ip": c.ClientIP(),
				"origin":    origin,
				"path":      c.Request.URL.Path,
			}).Warn("CORS request from disallowed origin")
		}

		c.Next()
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowed []string) bool {
	// Check for wildcard
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if a == origin {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(a, "*.") {
			domain := strings.TrimPrefix(a, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
