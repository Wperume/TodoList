package logging

import (
	"io"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig holds configuration for logging
type LogConfig struct {
	Enabled    bool   // Enable/disable file logging
	FilePath   string // Path to log file
	MaxSize    int    // Maximum size in megabytes before rotation
	MaxBackups int    // Maximum number of old log files to retain
	MaxAge     int    // Maximum number of days to retain old log files
	Compress   bool   // Compress rotated log files
	Level      string // Log level (trace, debug, info, warn, error, fatal, panic)
	JSONFormat bool   // Use JSON format instead of text
}

// Logger is the global logger instance
var Logger *logrus.Logger

// InitLogger initializes the global logger with the provided configuration
func InitLogger(config *LogConfig) *logrus.Logger {
	Logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
		Logger.Warnf("Invalid log level '%s', using 'info'", config.Level)
	}
	Logger.SetLevel(level)

	// Set formatter
	if config.JSONFormat {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Configure output
	if config.Enabled && config.FilePath != "" {
		// Create lumberjack logger for log rotation
		logWriter := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// Write to both file and stdout
		multiWriter := io.MultiWriter(os.Stdout, logWriter)
		Logger.SetOutput(multiWriter)

		Logger.Infof("File logging enabled: %s (max size: %dMB, max backups: %d, max age: %d days)",
			config.FilePath, config.MaxSize, config.MaxBackups, config.MaxAge)
	} else {
		// Only write to stdout
		Logger.SetOutput(os.Stdout)
		Logger.Info("File logging disabled, logging to stdout only")
	}

	return Logger
}

// NewLogConfigFromEnv creates a LogConfig from environment variables
func NewLogConfigFromEnv() *LogConfig {
	return &LogConfig{
		Enabled:    getEnvBool("LOG_FILE_ENABLED", true),
		FilePath:   getEnv("LOG_FILE_PATH", "./logs/todolist-api.log"),
		MaxSize:    getEnvInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
		MaxAge:     getEnvInt("LOG_MAX_AGE_DAYS", 28),
		Compress:   getEnvBool("LOG_COMPRESS", true),
		Level:      getEnv("LOG_LEVEL", "info"),
		JSONFormat: getEnvBool("LOG_JSON_FORMAT", false),
	}
}

// Helper functions for environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
