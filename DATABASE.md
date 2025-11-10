# Database Guide

## PostgreSQL Setup

### With Docker Compose (Recommended)

The easiest way to get started:

```bash
docker-compose up -d
```

This starts both PostgreSQL and the API server. The database is automatically created and migrated.

### Local PostgreSQL Setup

If you want to run PostgreSQL locally without Docker:

```bash
# Create database and user
createdb -U postgres todolist
createuser -U postgres todouser
psql -U postgres -c "ALTER USER todouser WITH PASSWORD 'todopass';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE todolist TO todouser;"
```

## Database Migrations

Migrations run **automatically** when the application starts. GORM's AutoMigrate feature:

- Creates tables if they don't exist
- Adds new columns if needed
- Creates indexes
- Does NOT delete columns or tables (safe for production)

## Database Operations

### Connect to Database Shell

With Docker Compose:
```bash
make db-shell
# or
docker-compose exec postgres psql -U todouser -d todolist
```

Locally:
```bash
psql -U todouser -d todolist
```

### Useful SQL Queries

**View all lists:**
```sql
SELECT id, name, created_at FROM todo_lists WHERE deleted_at IS NULL;
```

**View all todos:**
```sql
SELECT id, description, priority, completed, list_id
FROM todos
WHERE deleted_at IS NULL;
```

**Count todos by priority:**
```sql
SELECT priority, COUNT(*)
FROM todos
WHERE deleted_at IS NULL
GROUP BY priority;
```

**Find overdue todos:**
```sql
SELECT id, description, due_date, list_id
FROM todos
WHERE deleted_at IS NULL
  AND completed = false
  AND due_date < NOW();
```

**List stats:**
```sql
SELECT
    l.id,
    l.name,
    COUNT(t.id) as todo_count,
    COUNT(CASE WHEN t.completed THEN 1 END) as completed_count
FROM todo_lists l
LEFT JOIN todos t ON t.list_id = l.id AND t.deleted_at IS NULL
WHERE l.deleted_at IS NULL
GROUP BY l.id, l.name;
```

## Soft Deletes

The application uses GORM's soft delete feature:

- When you delete a list or todo via the API, it's not physically removed
- The `deleted_at` column is set to the current timestamp
- Soft-deleted records are automatically excluded from queries
- This allows for data recovery if needed

**View soft-deleted records:**
```sql
SELECT * FROM todo_lists WHERE deleted_at IS NOT NULL;
SELECT * FROM todos WHERE deleted_at IS NOT NULL;
```

**Permanently delete soft-deleted records:**
```sql
DELETE FROM todos WHERE deleted_at IS NOT NULL;
DELETE FROM todo_lists WHERE deleted_at IS NOT NULL;
```

**Restore a soft-deleted record:**
```sql
UPDATE todo_lists SET deleted_at = NULL WHERE id = 'your-uuid-here';
UPDATE todos SET deleted_at = NULL WHERE id = 'your-uuid-here';
```

## Database Backup and Restore

### Backup

**Full database backup:**
```bash
docker-compose exec postgres pg_dump -U todouser todolist > backup.sql
```

**Backup with Docker volume:**
```bash
docker run --rm \
  -v todolist_postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/db-backup.tar.gz /data
```

### Restore

**Restore from SQL dump:**
```bash
docker-compose exec -T postgres psql -U todouser todolist < backup.sql
```

## Environment Variables

All database configuration is done via environment variables:

```bash
DB_HOST=localhost       # Database host
DB_PORT=5432           # Database port
DB_USER=todouser       # Database user
DB_PASSWORD=todopass   # Database password
DB_NAME=todolist       # Database name
DB_SSLMODE=disable     # SSL mode (disable/require/verify-full)
DB_LOG_LEVEL=info      # Logging level (info/silent)
```

## Performance Optimization

### Indexes

The following indexes are automatically created:

- `todo_lists.name` (unique index)
- `todo_lists.deleted_at`
- `todos.list_id`
- `todos.completed`
- `todos.deleted_at`

### Connection Pool Settings

Configure in [internal/database/database.go](internal/database/database.go:61):

```go
sqlDB.SetMaxIdleConns(10)      // Max idle connections
sqlDB.SetMaxOpenConns(100)     // Max open connections
sqlDB.SetConnMaxLifetime(time.Hour)  // Connection lifetime
```

Adjust these based on your workload.

## Troubleshooting

### Connection Refused

**Problem:** `connection refused` error when starting the API

**Solution:**
1. Ensure PostgreSQL is running: `docker-compose ps`
2. Check database host in environment variables
3. Wait for PostgreSQL to be fully ready (health check)

### Database Does Not Exist

**Problem:** `database "todolist" does not exist`

**Solution:**
```bash
docker-compose exec postgres createdb -U todouser todolist
```

### Permission Denied

**Problem:** `permission denied for database`

**Solution:**
```bash
docker-compose exec postgres psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE todolist TO todouser;"
```

### Migration Errors

**Problem:** Migration fails due to existing data/schema

**Solution:**
1. Backup your data first
2. Drop and recreate the database:
```bash
docker-compose down -v
docker-compose up -d
```

## Switching Between Storage Backends

### PostgreSQL (Default)
```bash
# Normal startup
docker-compose up
# or locally
go run cmd/server/main.go
```

### In-Memory Storage
```bash
# Useful for testing
USE_MEMORY_STORAGE=true go run cmd/server/main.go
# or
make run-memory
```

Note: In-memory storage is wiped on restart and doesn't support persistence.

## Database Schema Visualization

```
┌─────────────────────────┐
│     todo_lists          │
├─────────────────────────┤
│ id (PK, UUID)          │
│ name (UNIQUE)          │
│ description            │
│ created_at             │
│ updated_at             │
│ deleted_at             │
└───────────┬─────────────┘
            │
            │ 1:N
            │
┌───────────▼─────────────┐
│       todos             │
├─────────────────────────┤
│ id (PK, UUID)          │
│ list_id (FK)           │◄── ON DELETE CASCADE
│ description            │
│ priority               │
│ due_date               │
│ completed              │
│ completed_at           │
│ created_at             │
│ updated_at             │
│ deleted_at             │
└─────────────────────────┘
```

## Production Considerations

1. **SSL/TLS**: Enable SSL for production: `DB_SSLMODE=require`
2. **Connection Pooling**: Tune based on your traffic
3. **Backups**: Set up automated backups
4. **Monitoring**: Monitor connection pool utilization
5. **Read Replicas**: Consider read replicas for high traffic
6. **Indexes**: Add custom indexes based on query patterns
7. **Vacuum**: PostgreSQL autovacuum should be enabled (default)

## Resources

- [GORM Documentation](https://gorm.io/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker PostgreSQL Image](https://hub.docker.com/_/postgres)
