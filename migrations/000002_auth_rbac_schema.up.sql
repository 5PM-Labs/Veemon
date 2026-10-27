-- 000002_auth_rbac_schema.up.sql
-- Detailed auth schema + RBAC.
--
-- users gains federated-identity (Auth0), MFA/verification state, profile
-- preferences, and login-tracking columns. Role assignments and permissions
-- live in their own tables and are resolved to a permission set at token-issue
-- time, so no request pays for a join on the auth path.

-- --- users: new columns -----------------------------------------------------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_url         TEXT,
    ADD COLUMN IF NOT EXISTS locale             VARCHAR(10),
    ADD COLUMN IF NOT EXISTS timezone           VARCHAR(64),
    ADD COLUMN IF NOT EXISTS email_verified     BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS mfa_enabled        BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS auth_provider      VARCHAR(20) NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS auth0_subject      VARCHAR(255),
    ADD COLUMN IF NOT EXISTS metadata           JSONB       NOT NULL DEFAULT '{}'::JSONB,
    ADD COLUMN IF NOT EXISTS last_login_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS failed_login_count INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until       TIMESTAMPTZ;

-- Federated accounts have no local password: rename and relax NOT NULL.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'users' AND column_name = 'password') THEN
        ALTER TABLE users RENAME COLUMN password TO password_hash;
    END IF;
END $$;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- A subject is unique per provider, but only for accounts that have one.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth0_subject
    ON users(auth0_subject) WHERE auth0_subject IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_metadata ON users USING GIN (metadata);

-- --- roles / permissions ----------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    is_system   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);

-- --- seed built-in roles and permissions ------------------------------------
INSERT INTO roles (name, description, is_system) VALUES
    ('superadmin', 'Unrestricted access to every resource', TRUE),
    ('admin',      'Manage users and read audit data',       TRUE),
    ('user',       'Default role for registered accounts',   TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (code, description) VALUES
    ('*',            'Every permission (superadmin only)'),
    ('users:read',   'Read users'),
    ('users:create', 'Create users'),
    ('users:update', 'Update users'),
    ('users:delete', 'Delete users'),
    ('roles:read',   'Read roles and permissions'),
    ('roles:create', 'Create roles'),
    ('roles:update', 'Update role permission sets'),
    ('roles:delete', 'Delete roles'),
    ('audit:read',   'Read audit log')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin' AND p.code = '*'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.code IN
    ('users:read','users:create','users:update','users:delete','roles:read','audit:read')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'user' AND p.code = 'users:read'
ON CONFLICT DO NOTHING;

-- --- migrate existing text[] roles into user_roles --------------------------
-- Unknown role names are created as non-system roles so no assignment is lost.
INSERT INTO roles (name, description)
SELECT DISTINCT role_name, 'Migrated from users.roles'
FROM users u, unnest(u.roles) AS role_name
WHERE role_name <> ''
ON CONFLICT (name) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u, unnest(u.roles) AS role_name
JOIN roles r ON r.name = role_name
ON CONFLICT DO NOTHING;

-- user_roles is now the single source of truth for role assignment; the
-- denormalized text[] column is dropped so the two can never disagree.
ALTER TABLE users DROP COLUMN roles;

-- --- audit log (SOC 2 / PDPA security events) -------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    resource_id TEXT,
    metadata    JSONB        NOT NULL DEFAULT '{}'::JSONB,
    ip          INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor   ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);

-- updated_at trigger reuses update_updated_at_column() from migration 1.
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
