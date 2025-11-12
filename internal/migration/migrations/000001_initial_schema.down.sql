-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS todos CASCADE;
DROP TABLE IF EXISTS todo_lists CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop extension (optional - may be used by other databases)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
