package domain

// Built-in roles. They are seeded by migration and marked is_system so the
// admin API cannot delete them.
const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

// PermissionWildcard grants every permission; assigned to superadmin.
const PermissionWildcard = "*"

// Permission codes are "<resource>:<action>" strings. Route auth policies and
// the RBAC middleware reference these exact values, so the set is a compile-
// time contract rather than free-form strings.
const (
	PermissionUsersRead   = "users:read"
	PermissionUsersCreate = "users:create"
	PermissionUsersUpdate = "users:update"
	PermissionUsersDelete = "users:delete"

	PermissionRolesRead   = "roles:read"
	PermissionRolesCreate = "roles:create"
	PermissionRolesUpdate = "roles:update"
	PermissionRolesDelete = "roles:delete"

	PermissionAuditRead = "audit:read"
)

// Permission is a permissions row.
type Permission struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// Role is a roles row.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"isSystem"`
	Permissions []Permission `json:"permissions,omitempty"`
}
