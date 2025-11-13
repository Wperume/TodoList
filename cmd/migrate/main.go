package main

import (
	"fmt"
	"os"
	"strconv"

	"todolist-api/internal/migration"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Create migrator
	migrator, err := migration.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer migrator.Close()

	// Run command and exit with appropriate code
	if err := runCommand(migrator, command, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(migrator *migration.Migrator, command string, args []string) error {
	switch command {
	case "up":
		if err := migrator.Up(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		fmt.Println("✅ Migrations applied successfully")

	case "down":
		if err := migrator.Down(); err != nil {
			return fmt.Errorf("failed to rollback migration: %w", err)
		}
		fmt.Println("✅ Migration rolled back successfully")

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		if dirty {
			fmt.Printf("Current version: %d (dirty)\n", version)
			fmt.Println("⚠️  Warning: Database is in a dirty state. Use 'force' command to fix.")
		} else {
			fmt.Printf("Current version: %d\n", version)
		}

	case "steps":
		if len(args) < 1 {
			printUsage()
			return fmt.Errorf("'steps' command requires number of steps")
		}
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid number of steps: %w", err)
		}
		if err := migrator.Steps(n); err != nil {
			return fmt.Errorf("failed to run %d steps: %w", n, err)
		}
		fmt.Printf("✅ Successfully ran %d migration steps\n", n)

	case "force":
		if len(args) < 1 {
			printUsage()
			return fmt.Errorf("'force' command requires version number")
		}
		version, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid version number: %w", err)
		}
		if err := migrator.Force(version); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
		fmt.Printf("✅ Forced migration version to %d\n", version)
		fmt.Println("⚠️  Warning: This does not run migrations. Make sure database state matches the forced version.")

	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

func printUsage() {
	fmt.Print(`Database Migration Tool

Usage:
  migrate <command> [arguments]

Commands:
  up              Apply all pending migrations
  down            Rollback the last migration
  version         Show current migration version
  steps <n>       Run n migrations (positive = up, negative = down)
  force <version> Force set the migration version (use with caution)

Examples:
  migrate up                # Apply all pending migrations
  migrate down              # Rollback last migration
  migrate version           # Show current version
  migrate steps 2           # Run next 2 migrations
  migrate steps -1          # Rollback 1 migration
  migrate force 1           # Force version to 1 (for dirty state recovery)

Environment Variables:
  DB_HOST         Database host (default: localhost)
  DB_PORT         Database port (default: 5432)
  DB_USER         Database user (default: postgres)
  DB_PASSWORD     Database password (default: postgres)
  DB_NAME         Database name (default: todolist)
  DB_SSL_MODE     SSL mode (default: disable)
`)
}
