# Database Migration Quick Reference

## Common Commands

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check current version
make migrate-version

# Apply next 2 migrations
make migrate-steps N=2

# Rollback last 3 migrations
make migrate-steps N=-3

# Force version (dirty state recovery)
make migrate-force V=1

# Create new migration
make migrate-create NAME=add_user_preferences
```

## Cloud VM Deployment

```bash
# Automated deployment (includes migrations)
./deploy.sh production

# Manual migration only
./bin/migrate up
```

## Migration File Template

### UP Migration: `NNNNNN_description.up.sql`

```sql
-- Add your schema changes here
CREATE TABLE IF NOT EXISTS example (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_example_name ON example(name);
```

### DOWN Migration: `NNNNNN_description.down.sql`

```sql
-- Reverse the changes
DROP TABLE IF EXISTS example CASCADE;
```

## Troubleshooting

### Dirty State

```bash
# 1. Check status
make migrate-version

# 2. Force to last known good version
make migrate-force V=1

# 3. Try again
make migrate-up
```

### Connection Error

```bash
# Check environment variables
env | grep DB_

# Test connection
psql -U todolist -d todolist -h localhost
```

### Migration Fails

1. Check error in output
2. Review SQL syntax in migration file
3. Ensure dependencies are in order
4. Check table/column doesn't already exist

## Best Practices

✅ **DO:**
- Keep migrations small and focused
- Test both UP and DOWN
- Use IF EXISTS / IF NOT EXISTS
- Add comments explaining changes
- Version control all migrations

❌ **DON'T:**
- Modify existing migrations (create new ones)
- Skip DOWN migrations
- Run manual SQL on production
- Force version without understanding why

## Environment Variables

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=todolist
DB_PASSWORD=your_password
DB_NAME=todolist
DB_SSL_MODE=disable
```

## Files

- Migration files: `internal/migration/migrations/`
- Migration code: `internal/migration/migrate.go`
- CLI tool: `cmd/migrate/main.go`
- Deploy script: `deploy.sh`

## Documentation

- Full guide: [MIGRATIONS.md](MIGRATIONS.md)
- Deployment: [DEPLOYMENT.md](DEPLOYMENT.md)
- Main README: [README.md](README.md)
