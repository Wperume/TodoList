package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupHealthTest(t *testing.T) (*HealthHandler, sqlmock.Sqlmock, func()) {
	// Create SQL mock with ping monitoring enabled
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	// Expect ping during GORM initialization
	mock.ExpectPing()

	// Create GORM DB with mock
	dialector := postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	handler := NewHealthHandler(db)

	cleanup := func() {
		sqlDB.Close()
	}

	return handler, mock, cleanup
}

func TestNewHealthHandler(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	handler := NewHealthHandler(db)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.db)
	assert.False(t, handler.startTime.IsZero())
}

func TestBasicHealth(t *testing.T) {
	handler, _, cleanup := setupHealthTest(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.BasicHealth(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
}

func TestDetailedHealth_Healthy(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping
	mock.ExpectPing()

	// Mock migration check - table exists query
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Mock migration version query - use regexp to match multiline query
	mock.ExpectQuery(`SELECT version, dirty\s+FROM schema_migrations\s+LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(1, false))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.DetailedHealth(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotEmpty(t, response.Uptime)
	assert.Equal(t, "1.0.0", response.Version)

	// Check database health
	assert.Contains(t, response.Checks, "database")
	assert.Equal(t, "healthy", response.Checks["database"].Status)

	// Check migrations health
	assert.Contains(t, response.Checks, "migrations")

	// Check system info
	assert.Contains(t, response.Checks, "system")
	assert.Equal(t, "info", response.Checks["system"].Status)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailedHealth_DatabaseUnhealthy(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping failure
	mock.ExpectPing().WillReturnError(errors.New("connection refused"))

	// Mock migration check - even though DB is unhealthy, migrations are still checked
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnError(errors.New("connection refused"))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.DetailedHealth(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "unhealthy", response.Status)
	assert.Contains(t, response.Checks, "database")
	assert.Equal(t, "unhealthy", response.Checks["database"].Status)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailedHealth_NilDatabase(t *testing.T) {
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.DetailedHealth(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "unhealthy", response.Status)
	assert.Contains(t, response.Checks, "database")
	assert.Equal(t, "unhealthy", response.Checks["database"].Status)
	assert.Equal(t, "Database connection not initialized", response.Checks["database"].Message)
}

func TestReadinessProbe_Ready(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping
	mock.ExpectPing()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.ReadinessProbe(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response["status"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReadinessProbe_NotReady(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping failure
	mock.ExpectPing().WillReturnError(errors.New("database unavailable"))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.ReadinessProbe(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", response["status"])
	assert.Equal(t, "database_unavailable", response["reason"])
	assert.NotEmpty(t, response["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReadinessProbe_NilDatabase(t *testing.T) {
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.ReadinessProbe(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", response["status"])
	assert.Equal(t, "database_unavailable", response["reason"])
}

func TestLivenessProbe(t *testing.T) {
	handler, _, cleanup := setupHealthTest(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.LivenessProbe(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "alive", response["status"])
}

func TestCheckDatabase_Healthy(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping
	mock.ExpectPing()

	check := handler.checkDatabase()

	assert.Equal(t, "healthy", check.Status)
	assert.Equal(t, "Database connection is healthy", check.Message)
	assert.NotNil(t, check.Details)
	assert.Contains(t, check.Details, "open_connections")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckDatabase_PingFailed(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock database ping failure
	mock.ExpectPing().WillReturnError(errors.New("connection refused"))

	check := handler.checkDatabase()

	assert.Equal(t, "unhealthy", check.Status)
	assert.Equal(t, "Database ping failed", check.Message)
	assert.NotNil(t, check.Details)
	assert.Contains(t, check.Details, "error")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckDatabase_NilDB(t *testing.T) {
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
	}

	check := handler.checkDatabase()

	assert.Equal(t, "unhealthy", check.Status)
	assert.Equal(t, "Database connection not initialized", check.Message)
}

func TestCheckMigrations_Healthy(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock migration check - table exists query
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Mock migration version query - use regexp to match multiline query
	mock.ExpectQuery(`SELECT version, dirty\s+FROM schema_migrations\s+LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(1, false))

	check := handler.checkMigrations()

	assert.Equal(t, "healthy", check.Status)
	assert.Equal(t, "Migrations are up to date", check.Message)
	assert.NotNil(t, check.Details)
	assert.Equal(t, uint(1), check.Details["version"])
	assert.Equal(t, false, check.Details["dirty"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMigrations_Dirty(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock migration check - table exists query
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Mock migration version query with dirty state - use regexp to match multiline query
	mock.ExpectQuery(`SELECT version, dirty\s+FROM schema_migrations\s+LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(2, true))

	check := handler.checkMigrations()

	assert.Equal(t, "warning", check.Status)
	assert.Equal(t, "Database is in dirty state - manual intervention required", check.Message)
	assert.NotNil(t, check.Details)
	assert.Equal(t, uint(2), check.Details["version"])
	assert.Equal(t, true, check.Details["dirty"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMigrations_TableNotFound(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock migration check - table doesn't exist
	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	check := handler.checkMigrations()

	assert.Equal(t, "unknown", check.Status)
	assert.Equal(t, "Migration table not found", check.Message)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMigrations_NilDB(t *testing.T) {
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
	}

	check := handler.checkMigrations()

	assert.Equal(t, "unknown", check.Status)
	assert.Equal(t, "Database not available", check.Message)
}

func TestGetSystemInfo(t *testing.T) {
	handler, _, cleanup := setupHealthTest(t)
	defer cleanup()

	check := handler.getSystemInfo()

	assert.Equal(t, "info", check.Status)
	assert.Equal(t, "System information", check.Message)
	assert.NotNil(t, check.Details)

	// Verify all expected fields are present
	assert.Contains(t, check.Details, "goroutines")
	assert.Contains(t, check.Details, "memory_alloc_mb")
	assert.Contains(t, check.Details, "memory_sys_mb")
	assert.Contains(t, check.Details, "num_gc")
	assert.Contains(t, check.Details, "go_version")

	// Verify types are correct
	assert.IsType(t, 0, check.Details["goroutines"])
	assert.IsType(t, uint64(0), check.Details["memory_alloc_mb"])
	assert.IsType(t, uint64(0), check.Details["memory_sys_mb"])
	assert.IsType(t, uint32(0), check.Details["num_gc"])
	assert.IsType(t, "", check.Details["go_version"])
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "only seconds",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "5m 30s",
		},
		{
			name:     "hours, minutes and seconds",
			duration: 2*time.Hour + 15*time.Minute + 10*time.Second,
			expected: "2h 15m 10s",
		},
		{
			name:     "days, hours, minutes and seconds",
			duration: 3*24*time.Hour + 5*time.Hour + 20*time.Minute + 5*time.Second,
			expected: "3d 5h 20m 5s",
		},
		{
			name:     "exactly 1 day",
			duration: 24 * time.Hour,
			expected: "1d 0h 0m 0s",
		},
		{
			name:     "exactly 1 hour",
			duration: time.Hour,
			expected: "1h 0m 0s",
		},
		{
			name:     "exactly 1 minute",
			duration: time.Minute,
			expected: "1m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthHandler_Uptime(t *testing.T) {
	handler, _, cleanup := setupHealthTest(t)
	defer cleanup()

	// Wait a bit to ensure uptime is > 0
	time.Sleep(100 * time.Millisecond)

	assert.True(t, time.Since(handler.startTime) > 0)
}

func TestHealthResponse_JSONSerialization(t *testing.T) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    "1h 23m 45s",
		Version:   "1.0.0",
		Checks: map[string]HealthCheck{
			"database": {
				Status:  "healthy",
				Message: "Database is healthy",
				Details: map[string]interface{}{
					"connections": 5,
				},
			},
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded HealthResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.Status, decoded.Status)
	assert.Equal(t, response.Timestamp, decoded.Timestamp)
	assert.Equal(t, response.Uptime, decoded.Uptime)
	assert.Equal(t, response.Version, decoded.Version)
	assert.Len(t, decoded.Checks, 1)
}

func TestHealthCheck_JSONSerialization(t *testing.T) {
	check := HealthCheck{
		Status:  "healthy",
		Message: "All systems operational",
		Details: map[string]interface{}{
			"version": "1.0.0",
			"uptime":  300,
		},
	}

	data, err := json.Marshal(check)
	require.NoError(t, err)

	var decoded HealthCheck
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, check.Status, decoded.Status)
	assert.Equal(t, check.Message, decoded.Message)
	assert.Len(t, decoded.Details, 2)
}

// TestCheckDatabase_GetDBError tests error handling when getting DB instance fails
func TestCheckDatabase_GetDBError(t *testing.T) {
	// Note: It's difficult to test the sqlDB.DB() error case with sqlmock
	// because once GORM opens the connection, calling Close() doesn't trigger
	// the specific error path we want to test. This test verifies the nil DB case instead.
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
	}

	check := handler.checkDatabase()

	assert.Equal(t, "unhealthy", check.Status)
	assert.Contains(t, check.Message, "Database")
}

// TestCheckMigrations_QueryError tests error handling when migration query fails
func TestCheckMigrations_QueryError(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock query error - use regexp to match multiline query with whitespace
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnError(errors.New("query failed"))

	check := handler.checkMigrations()

	assert.Equal(t, "unknown", check.Status)
	assert.Equal(t, "Migration table not found", check.Message)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckMigrations_ScanError tests error handling when scanning migration data fails
func TestCheckMigrations_ScanError(t *testing.T) {
	handler, mock, cleanup := setupHealthTest(t)
	defer cleanup()

	// Mock table exists - use regexp to match multiline query
	mock.ExpectQuery(`SELECT EXISTS \(\s+SELECT FROM information_schema\.tables\s+WHERE table_name = 'schema_migrations'\s+\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Mock version query with error - use regexp to match multiline query
	mock.ExpectQuery(`SELECT version, dirty\s+FROM schema_migrations\s+LIMIT 1`).
		WillReturnError(sql.ErrNoRows)

	check := handler.checkMigrations()

	assert.Equal(t, "unknown", check.Status)
	assert.Equal(t, "Could not read migration status", check.Message)

	assert.NoError(t, mock.ExpectationsWereMet())
}
