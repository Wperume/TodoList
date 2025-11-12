package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db        *gorm.DB
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Version   string                 `json:"version"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// BasicHealth is a simple health check (backwards compatible)
// @Summary Basic health check
// @Description Returns a simple health status
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *HealthHandler) BasicHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// DetailedHealth provides comprehensive health information
// @Summary Detailed health check
// @Description Returns detailed health status including database connectivity, uptime, and system info
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health/detailed [get]
func (h *HealthHandler) DetailedHealth(c *gin.Context) {
	checks := make(map[string]HealthCheck)
	overallStatus := "healthy"

	// Check database connectivity
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check migration status
	migrationCheck := h.checkMigrations()
	checks["migrations"] = migrationCheck

	// System information
	checks["system"] = h.getSystemInfo()

	// Calculate uptime
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    formatDuration(uptime),
		Version:   "1.0.0", // TODO: Get from build info
		Checks:    checks,
	}

	// Return 503 if unhealthy
	if overallStatus == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessProbe checks if the application is ready to serve traffic
// @Summary Readiness probe
// @Description Kubernetes-style readiness probe - checks if app can handle requests
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /health/ready [get]
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	// Check if database is accessible
	dbCheck := h.checkDatabase()

	if dbCheck.Status != "healthy" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not_ready",
			"reason":  "database_unavailable",
			"message": dbCheck.Message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// LivenessProbe checks if the application is alive
// @Summary Liveness probe
// @Description Kubernetes-style liveness probe - checks if app is running
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// checkDatabase verifies database connectivity
func (h *HealthHandler) checkDatabase() HealthCheck {
	if h.db == nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	// Get underlying SQL database
	sqlDB, err := h.db.DB()
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Failed to get database instance",
		}
	}

	// Ping database with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database ping failed",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Get database stats
	stats := sqlDB.Stats()

	return HealthCheck{
		Status:  "healthy",
		Message: "Database connection is healthy",
		Details: map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration_ms":    stats.WaitDuration.Milliseconds(),
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		},
	}
}

// checkMigrations verifies migration status
func (h *HealthHandler) checkMigrations() HealthCheck {
	if h.db == nil {
		return HealthCheck{
			Status:  "unknown",
			Message: "Database not available",
		}
	}

	// Check if schema_migrations table exists
	var exists bool
	err := h.db.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'schema_migrations'
		)
	`).Scan(&exists).Error

	if err != nil || !exists {
		return HealthCheck{
			Status:  "unknown",
			Message: "Migration table not found",
		}
	}

	// Get current migration version
	var version uint
	var dirty bool
	err = h.db.Raw(`
		SELECT version, dirty
		FROM schema_migrations
		LIMIT 1
	`).Row().Scan(&version, &dirty)

	if err != nil {
		return HealthCheck{
			Status:  "unknown",
			Message: "Could not read migration status",
		}
	}

	status := "healthy"
	message := "Migrations are up to date"
	if dirty {
		status = "warning"
		message = "Database is in dirty state - manual intervention required"
	}

	return HealthCheck{
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"version": version,
			"dirty":   dirty,
		},
	}
}

// getSystemInfo returns system information
func (h *HealthHandler) getSystemInfo() HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return HealthCheck{
		Status:  "info",
		Message: "System information",
		Details: map[string]interface{}{
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": m.Alloc / 1024 / 1024,
			"memory_sys_mb":   m.Sys / 1024 / 1024,
			"num_gc":          m.NumGC,
			"go_version":      runtime.Version(),
		},
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
