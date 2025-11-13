package main

import (
	"fmt"
	"os"
	"strconv"

	"todolist-api/internal/migration"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		printUsage()
		return 1
	}

	command := os.Args[1]

	// Create migrator
	migrator, err := migration.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create migrator: %v\n", err)
		return 1
	}
	defer migrator.Close()

	// Run command and exit with appropriate code
	if err := runCommand(migrator, command, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func runCommand(migrator *migration.Migrator, command string, args []string) error {
	switch command {
	case "up":
		return runUp(migrator)
	case "down":
		return runDown(migrator)
	case "version":
		return runVersion(migrator)
	case "steps":
		return runSteps(migrator, args)
	case "force":
		return runForce(migrator, args)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runUp(migrator *migration.Migrator) error {
	if err := migrator.Up(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	fmt.Println("✅ Migrations applied successfully")
	return nil
}

func runDown(migrator *migration.Migrator) error {
	if err := migrator.Down(); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}
	fmt.Println("✅ Migration rolled back successfully")
	return nil
}

func runVersion(migrator *migration.Migrator) error {
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
	return nil
}

func runSteps(migrator *migration.Migrator, args []string) error {
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
	return nil
}

func runForce(migrator *migration.Migrator, args []string) error {
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
