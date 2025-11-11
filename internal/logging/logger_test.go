package logging

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogConfigFromEnv(t *testing.T) {
	// Save original environment
	origEnabled := os.Getenv("LOG_FILE_ENABLED")
	origPath := os.Getenv("LOG_FILE_PATH")
	origMaxSize := os.Getenv("LOG_MAX_SIZE_MB")
	origMaxBackups := os.Getenv("LOG_MAX_BACKUPS")
	origMaxAge := os.Getenv("LOG_MAX_AGE_DAYS")
	origCompress := os.Getenv("LOG_COMPRESS")
	origLevel := os.Getenv("LOG_LEVEL")
	origJSON := os.Getenv("LOG_JSON_FORMAT")

	// Restore environment after test
	defer func() {
		os.Setenv("LOG_FILE_ENABLED", origEnabled)
		os.Setenv("LOG_FILE_PATH", origPath)
		os.Setenv("LOG_MAX_SIZE_MB", origMaxSize)
		os.Setenv("LOG_MAX_BACKUPS", origMaxBackups)
		os.Setenv("LOG_MAX_AGE_DAYS", origMaxAge)
		os.Setenv("LOG_COMPRESS", origCompress)
		os.Setenv("LOG_LEVEL", origLevel)
		os.Setenv("LOG_JSON_FORMAT", origJSON)
	}()

	t.Run("uses default values when env vars not set", func(t *testing.T) {
		os.Unsetenv("LOG_FILE_ENABLED")
		os.Unsetenv("LOG_FILE_PATH")
		os.Unsetenv("LOG_MAX_SIZE_MB")
		os.Unsetenv("LOG_MAX_BACKUPS")
		os.Unsetenv("LOG_MAX_AGE_DAYS")
		os.Unsetenv("LOG_COMPRESS")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_JSON_FORMAT")

		config := NewLogConfigFromEnv()

		assert.True(t, config.Enabled, "Should be enabled by default")
		assert.Equal(t, "./logs/todolist-api.log", config.FilePath)
		assert.Equal(t, 100, config.MaxSize)
		assert.Equal(t, 3, config.MaxBackups)
		assert.Equal(t, 28, config.MaxAge)
		assert.True(t, config.Compress)
		assert.Equal(t, "info", config.Level)
		assert.False(t, config.JSONFormat)
	})

	t.Run("uses custom values from environment", func(t *testing.T) {
		os.Setenv("LOG_FILE_ENABLED", "false")
		os.Setenv("LOG_FILE_PATH", "/var/log/custom.log")
		os.Setenv("LOG_MAX_SIZE_MB", "50")
		os.Setenv("LOG_MAX_BACKUPS", "5")
		os.Setenv("LOG_MAX_AGE_DAYS", "7")
		os.Setenv("LOG_COMPRESS", "false")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("LOG_JSON_FORMAT", "true")

		config := NewLogConfigFromEnv()

		assert.False(t, config.Enabled)
		assert.Equal(t, "/var/log/custom.log", config.FilePath)
		assert.Equal(t, 50, config.MaxSize)
		assert.Equal(t, 5, config.MaxBackups)
		assert.Equal(t, 7, config.MaxAge)
		assert.False(t, config.Compress)
		assert.Equal(t, "debug", config.Level)
		assert.True(t, config.JSONFormat)
	})

	t.Run("handles invalid numeric values gracefully", func(t *testing.T) {
		// Set invalid values that will fail parsing
		os.Setenv("LOG_FILE_ENABLED", "true")
		os.Setenv("LOG_MAX_SIZE_MB", "invalid")
		os.Setenv("LOG_MAX_BACKUPS", "not-a-number")
		os.Setenv("LOG_MAX_AGE_DAYS", "abc")

		config := NewLogConfigFromEnv()

		// When strconv.Atoi fails, getEnvInt returns the default value
		// Defaults are 100, 3, 28
		assert.Equal(t, 100, config.MaxSize, "Should use default when parsing fails")
		assert.Equal(t, 3, config.MaxBackups, "Should use default when parsing fails")
		assert.Equal(t, 28, config.MaxAge, "Should use default when parsing fails")
	})

	t.Run("handles invalid boolean values gracefully", func(t *testing.T) {
		os.Setenv("LOG_FILE_ENABLED", "not-a-bool")
		os.Setenv("LOG_COMPRESS", "invalid")
		os.Setenv("LOG_JSON_FORMAT", "maybe")

		config := NewLogConfigFromEnv()

		// Should use default values when parsing fails
		assert.True(t, config.Enabled)
		assert.True(t, config.Compress)
		assert.False(t, config.JSONFormat)
	})
}

func TestInitLogger(t *testing.T) {
	t.Run("initializes with text format", func(t *testing.T) {
		config := &LogConfig{
			Enabled:    false, // Disable file logging for tests
			Level:      "info",
			JSONFormat: false,
		}

		logger := InitLogger(config)

		assert.NotNil(t, logger)
		assert.Equal(t, logrus.InfoLevel, logger.Level)
		assert.IsType(t, &logrus.TextFormatter{}, logger.Formatter)
	})

	t.Run("initializes with JSON format", func(t *testing.T) {
		config := &LogConfig{
			Enabled:    false,
			Level:      "debug",
			JSONFormat: true,
		}

		logger := InitLogger(config)

		assert.NotNil(t, logger)
		assert.Equal(t, logrus.DebugLevel, logger.Level)
		assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)
	})

	t.Run("handles invalid log level", func(t *testing.T) {
		config := &LogConfig{
			Enabled:    false,
			Level:      "invalid-level",
			JSONFormat: false,
		}

		logger := InitLogger(config)

		// Should default to Info level
		assert.NotNil(t, logger)
		assert.Equal(t, logrus.InfoLevel, logger.Level)
	})

	t.Run("accepts all valid log levels", func(t *testing.T) {
		levels := map[string]logrus.Level{
			"trace": logrus.TraceLevel,
			"debug": logrus.DebugLevel,
			"info":  logrus.InfoLevel,
			"warn":  logrus.WarnLevel,
			"error": logrus.ErrorLevel,
			"fatal": logrus.FatalLevel,
			"panic": logrus.PanicLevel,
		}

		for levelStr, expectedLevel := range levels {
			config := &LogConfig{
				Enabled:    false,
				Level:      levelStr,
				JSONFormat: false,
			}

			logger := InitLogger(config)
			assert.Equal(t, expectedLevel, logger.Level, "Level %s should be parsed correctly", levelStr)
		}
	})
}

func TestGetEnvHelpers(t *testing.T) {
	t.Run("getEnv returns value when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("getEnv returns default when not set", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR", "default_value")
		assert.Equal(t, "default_value", result)
	})

	t.Run("getEnvBool returns true when set", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "true")
		defer os.Unsetenv("TEST_BOOL")

		result := getEnvBool("TEST_BOOL", false)
		assert.True(t, result)
	})

	t.Run("getEnvBool returns default when not set", func(t *testing.T) {
		result := getEnvBool("NONEXISTENT_BOOL", true)
		assert.True(t, result)
	})

	t.Run("getEnvInt returns value when set", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 0)
		assert.Equal(t, 42, result)
	})

	t.Run("getEnvInt returns default when not set", func(t *testing.T) {
		result := getEnvInt("NONEXISTENT_INT", 99)
		assert.Equal(t, 99, result)
	})
}
