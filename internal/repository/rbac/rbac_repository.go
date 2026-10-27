// Package rbac_repository resolves roles and permissions from Postgres.
//
// Permissions are resolved on every authenticated request rather than embedded
// in the token, so a role or permission change takes effect immediately instead
// of at the next token refresh.
package rbac_repository

import (
	"context"

	"github.com/5PM-Labs/veemon/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	// RolesForUser returns the role names assigned to userID.
	RolesForUser(ctx context.Context, userID string) ([]string, error)
	// PermissionsForRoles returns the union of permission codes granted to
	// roles. Unknown role names are ignored.
	PermissionsForRoles(ctx context.Context, roles []string) ([]string, error)
	// ListRoles returns every role with its permissions.
	ListRoles(ctx context.Context) ([]domain.Role, error)
	// ListPermissions returns every permission code.
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) RolesForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.name FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]string, 0, 3)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}

func (r *repository) PermissionsForRoles(ctx context.Context, roles []string) ([]string, error) {
	if len(roles) == 0 {
		return []string{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN roles r ON r.id = rp.role_id
		WHERE r.name = ANY($1)
		ORDER BY p.code`, roles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := make([]string, 0, 8)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		perms = append(perms, code)
	}
	return perms, rows.Err()
}

func (r *repository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, r.description, r.is_system,
		       coalesce(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		GROUP BY r.id
		ORDER BY r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]domain.Role, 0)
	for rows.Next() {
		var role domain.Role
		var codes []string
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &codes); err != nil {
			return nil, err
		}
		for _, c := range codes {
			role.Permissions = append(role.Permissions, domain.Permission{Code: c})
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *repository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, description FROM permissions ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := make([]domain.Permission, 0)
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
