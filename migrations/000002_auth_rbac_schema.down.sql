-- 000002_auth_rbac_schema.down.sql

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;

ALTER TABLE users ADD COLUMN IF NOT EXISTS roles TEXT[] DEFAULT ARRAY['user']::TEXT[];
UPDATE users u SET roles = coalesce(
    (SELECT array_agg(r.name) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = u.id),
    ARRAY[]::TEXT[]);
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;

DROP INDEX IF EXISTS idx_users_metadata;
DROP INDEX IF EXISTS idx_users_auth0_subject;

ALTER TABLE users
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS failed_login_count,
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS auth0_subject,
    DROP COLUMN IF EXISTS auth_provider,
    DROP COLUMN IF EXISTS mfa_enabled,
    DROP COLUMN IF EXISTS email_verified,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS avatar_url;

ALTER TABLE users RENAME COLUMN password_hash TO password;
