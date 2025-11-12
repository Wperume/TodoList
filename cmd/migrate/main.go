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

	switch command {
	case "up":
		if err := migrator.Up(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to run migrations: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Migrations applied successfully")

	case "down":
		if err := migrator.Down(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to rollback migration: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Migration rolled back successfully")

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to get version: %v\n", err)
			os.Exit(1)
		}
		if dirty {
			fmt.Printf("Current version: %d (dirty)\n", version)
			fmt.Println("⚠️  Warning: Database is in a dirty state. Use 'force' command to fix.")
		} else {
			fmt.Printf("Current version: %d\n", version)
		}

	case "steps":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: 'steps' command requires number of steps\n")
			printUsage()
			os.Exit(1)
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid number of steps: %v\n", err)
			os.Exit(1)
		}
		if err := migrator.Steps(n); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to run %d steps: %v\n", n, err)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully ran %d migration steps\n", n)

	case "force":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: 'force' command requires version number\n")
			printUsage()
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid version number: %v\n", err)
			os.Exit(1)
		}
		if err := migrator.Force(version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to force version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Forced migration version to %d\n", version)
		fmt.Println("⚠️  Warning: This does not run migrations. Make sure database state matches the forced version.")

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Database Migration Tool

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
