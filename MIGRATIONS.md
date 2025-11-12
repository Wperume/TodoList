# Database Migrations Guide

This guide explains how to use database migrations in the TodoList API project.

## Overview

The TodoList API uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema versioning and migrations. This ensures that database changes are tracked, versioned, and can be reliably applied across different environments.

## Benefits

- **Version Control**: Database schema changes are versioned alongside code
- **Repeatability**: Migrations can be applied consistently across environments
- **Rollback Support**: Easily revert changes if something goes wrong
- **Team Collaboration**: Everyone stays in sync with database changes
- **Deployment Safety**: Automated, tested migrations reduce human error

## Quick Start

### Local Development

```bash
# Apply all pending migrations
make migrate-up

# Check current version
make migrate-version

# Rollback last migration
make migrate-down
```

### Cloud VM Deployment

```bash
# Run the deployment script (handles migrations automatically)
./deploy.sh production
```

## Migration Commands

### Apply Migrations

```bash
# Apply all pending migrations
make migrate-up

# Apply next 2 migrations only
make migrate-steps N=2
```

### Rollback Migrations

```bash
# Rollback last migration
make migrate-down

# Rollback last 3 migrations
make migrate-steps N=-3
```

### Check Status

```bash
# Show current migration version
make migrate-version
```

### Create New Migration

```bash
# Create new migration files
make migrate-create NAME=add_user_preferences

# This creates two files:
# - internal/migration/migrations/NNNNNN_add_user_preferences.up.sql
# - internal/migration/migrations/NNNNNN_add_user_preferences.down.sql
```

### Force Version (Recovery)

If the database is in a "dirty" state (migration partially applied):

```bash
# Force to a specific version
make migrate-force V=1

# Then manually fix the database and re-run migrations
```

## Migration File Structure

Migration files are located in `internal/migration/migrations/`:

```
internal/migration/migrations/
├── 000001_initial_schema.up.sql     # Initial database schema
├── 000001_initial_schema.down.sql   # Rollback for initial schema
├── 000002_add_indexes.up.sql        # Next migration (up)
└── 000002_add_indexes.down.sql      # Rollback for indexes (down)
```

### File Naming Convention

```
NNNNNN_description.{up|down}.sql
```

- `NNNNNN`: 6-digit sequential number
- `description`: Snake_case description of the change
- `up`: Apply the migration
- `down`: Rollback the migration

## Writing Migrations

### UP Migration (Apply Changes)

File: `000002_add_user_preferences.up.sql`

```sql
-- Add user preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    theme VARCHAR(20) DEFAULT 'light',
    notifications_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add index
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);
```

### DOWN Migration (Rollback)

File: `000002_add_user_preferences.down.sql`

```sql
-- Remove user preferences table
DROP TABLE IF EXISTS user_preferences CASCADE;
```

### Best Practices

1. **Always include IF EXISTS/IF NOT EXISTS**
   ```sql
   CREATE TABLE IF NOT EXISTS ...
   DROP TABLE IF EXISTS ... CASCADE;
   ```

2. **Use transactions where possible**
   ```sql
   BEGIN;
   -- Your changes here
   COMMIT;
   ```

3. **Make migrations reversible**
   - Every UP migration must have a corresponding DOWN
   - Test both directions

4. **Keep migrations small and focused**
   - One logical change per migration
   - Easier to debug and rollback

5. **Never modify existing migrations**
   - Once applied to production, treat as immutable
   - Create new migrations to fix issues

6. **Add comments**
   ```sql
   -- Add tags column to support categorization feature (Ticket #123)
   ALTER TABLE todos ADD COLUMN tags TEXT[];
   ```

## Environment Variables

The migration tool uses the same database configuration as the main application:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=todolist
DB_SSL_MODE=disable
```

## Cloud VM Setup

### Initial Setup

1. **Install PostgreSQL**
   ```bash
   sudo apt-get update
   sudo apt-get install -y postgresql postgresql-contrib
   ```

2. **Create Database**
   ```bash
   sudo -u postgres createdb todolist
   sudo -u postgres createuser todolist
   ```

3. **Clone Repository**
   ```bash
   sudo mkdir -p /opt/todolist-api
   sudo git clone https://github.com/yourusername/todolist-api.git /opt/todolist-api
   cd /opt/todolist-api
   ```

4. **Configure Environment**
   ```bash
   sudo cp .env.example .env
   sudo nano .env  # Set your database credentials
   ```

5. **Run Migrations**
   ```bash
   make migrate-up
   ```

6. **Install Service**
   ```bash
   sudo cp todolist-api.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable todolist-api
   sudo systemctl start todolist-api
   ```

### Deployment Workflow

Use the deployment script for automated deployments:

```bash
# Deploy to production
./deploy.sh production

# Deploy to staging
./deploy.sh staging
```

The deployment script will:
1. ✅ Backup current binary
2. ✅ Pull latest code
3. ✅ Build application
4. ✅ Run database migrations
5. ✅ Restart service
6. ✅ Run health checks
7. ✅ Rollback on failure

## Migration Tracking

Migrations are tracked in the `schema_migrations` table:

```sql
-- Check applied migrations
SELECT * FROM schema_migrations;

-- Example output:
 version | dirty
---------+-------
  000001 | f
```

- `version`: Current migration version
- `dirty`: `true` if migration failed mid-execution

## Common Scenarios

### Scenario 1: Adding a New Column

**Create migration:**
```bash
make migrate-create NAME=add_todo_priority
```

**UP migration:**
```sql
ALTER TABLE todos ADD COLUMN priority VARCHAR(10) DEFAULT 'medium';
ALTER TABLE todos ADD CONSTRAINT check_priority
    CHECK (priority IN ('low', 'medium', 'high'));
```

**DOWN migration:**
```sql
ALTER TABLE todos DROP CONSTRAINT IF EXISTS check_priority;
ALTER TABLE todos DROP COLUMN IF EXISTS priority;
```

### Scenario 2: Changing a Column Type

**UP migration:**
```sql
-- Step 1: Add new column
ALTER TABLE users ADD COLUMN email_new VARCHAR(320);

-- Step 2: Copy data
UPDATE users SET email_new = email;

-- Step 3: Drop old column
ALTER TABLE users DROP COLUMN email;

-- Step 4: Rename new column
ALTER TABLE users RENAME COLUMN email_new TO email;

-- Step 5: Add constraints
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
```

### Scenario 3: Data Migration

**UP migration:**
```sql
-- Populate default values for existing records
UPDATE todos
SET priority = 'medium'
WHERE priority IS NULL;

-- Make column NOT NULL
ALTER TABLE todos ALTER COLUMN priority SET NOT NULL;
```

## Troubleshooting

### Dirty State

If you see `dirty: true`:

```bash
# 1. Check what went wrong
psql -U todolist -d todolist -c "SELECT version, dirty FROM schema_migrations;"

# 2. Manually fix the database issue

# 3. Force the version
make migrate-force V=1

# 4. Try migration again
make migrate-up
```

### Migration Fails

1. **Check error message carefully**
2. **Verify database connection** (`psql -U todolist -d todolist`)
3. **Check SQL syntax** in migration file
4. **Ensure tables/columns don't already exist**
5. **Review migration order** (dependencies)

### Rollback Doesn't Work

- Ensure DOWN migration SQL is correct
- Some operations can't be rolled back (e.g., DROP TABLE with data loss)
- May need to restore from backup

### Can't Connect to Database

```bash
# Check environment variables
env | grep DB_

# Test connection
psql postgresql://todolist:password@localhost:5432/todolist

# Check PostgreSQL is running
sudo systemctl status postgresql
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Deploy to Production
        run: |
          ssh production << 'EOF'
            cd /opt/todolist-api
            git pull
            make migrate-up
            sudo systemctl restart todolist-api
          EOF
```

## Backup Strategy

Before major migrations:

```bash
# Backup database
pg_dump todolist > backup_$(date +%Y%m%d_%H%M%S).sql

# Run migration
make migrate-up

# If something goes wrong:
psql todolist < backup_20231112_143000.sql
```

## Additional Resources

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL ALTER TABLE](https://www.postgresql.org/docs/current/sql-altertable.html)
- [Database Migration Best Practices](https://www.prisma.io/dataguide/types/relational/migration-strategies)

## Support

For issues or questions:
- Check the [Troubleshooting](#troubleshooting) section
- Review migration logs: `sudo journalctl -u todolist-api -n 100`
- Open an issue on GitHub
