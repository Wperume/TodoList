package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq" // PostgreSQL driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds migration configuration
type Config struct {
	DatabaseURL string
}

// Migrator handles database migrations
type Migrator struct {
	migrate *migrate.Migrate
	db      *sql.DB
}

// New creates a new Migrator instance
func New(cfg *Config) (*Migrator, error) {
	// Open database connection
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		migrate: m,
		db:      db,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	if err := m.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}
	return nil
}

// Steps runs n migrations (positive = up, negative = down)
func (m *Migrator) Steps(n int) error {
	if err := m.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
// Use with caution - primarily for fixing dirty state
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	return nil
}

// Close closes the database connection
func (m *Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// NewFromEnv creates a new Migrator from environment variables
func NewFromEnv() (*Migrator, error) {
	// Build database URL from environment or use default
	dbURL := buildDatabaseURL()

	cfg := &Config{
		DatabaseURL: dbURL,
	}

	return New(cfg)
}

// buildDatabaseURL constructs the database URL from environment variables
func buildDatabaseURL() string {
	// This will use the same database configuration as your main app
	// You can import from your database package or reconstruct here
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "todolist")
	sslmode := getEnv("DB_SSL_MODE", "disable")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode,
	)
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
