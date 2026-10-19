-- 000001_create_users_table.down.sql
-- Drop users table and related objects

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_company_code;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email_unique;
DROP TABLE IF EXISTS users;
