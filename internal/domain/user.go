// Package domain holds the domain models and sentinel errors shared by the
// repository, usecase, and handler layers. It depends on nothing in this
// module, so every layer can import it without a cycle.
package domain

import (
	"errors"
	"time"
)

// Sentinel errors returned by repositories. Callers map them to HTTP/gRPC
// status codes; they are deliberately infrastructure-agnostic so no caller
// needs to import the driver.
var (
	ErrNotFound       = errors.New("record not found")
	ErrDuplicateEntry = errors.New("duplicate entry")
)

// UserStatus is the account lifecycle state.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusPending  UserStatus = "pending"
	UserStatusLocked   UserStatus = "locked"
)

// Valid reports whether s is a recognised status.
func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusPending, UserStatusLocked:
		return true
	}
	return false
}

// AuthProvider identifies how an account authenticates. Local accounts carry a
// password hash; Auth0 accounts are federated and carry a subject instead.
type AuthProvider string

const (
	AuthProviderLocal AuthProvider = "local"
	AuthProviderAuth0 AuthProvider = "auth0"
)

// User is a users row. Indexes, defaults, and constraints are owned by the SQL
// migrations (golang-migrate is the schema source of truth), so this struct
// carries no schema tags. PasswordHash is empty for federated accounts.
type User struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	PasswordHash     string         `json:"-"`
	Name             string         `json:"name"`
	Phone            string         `json:"phone,omitempty"`
	AvatarURL        string         `json:"avatarUrl,omitempty"`
	Locale           string         `json:"locale,omitempty"`
	Timezone         string         `json:"timezone,omitempty"`
	Status           UserStatus     `json:"status"`
	EmailVerified    bool           `json:"emailVerified"`
	MFAEnabled       bool           `json:"mfaEnabled"`
	AuthProvider     AuthProvider   `json:"authProvider"`
	Auth0Subject     string         `json:"-"`
	CompanyCode      string         `json:"companyCode,omitempty"`
	Roles            []string       `json:"roles"`
	Permissions      []string       `json:"permissions,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	LastLoginAt      *time.Time     `json:"lastLoginAt,omitempty"`
	FailedLoginCount int            `json:"-"`
	LockedUntil      *time.Time     `json:"-"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        *time.Time     `json:"-"`
}

// HasRole reports whether the user holds role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission reports whether the user holds permission. The wildcard
// "resource:*" and the superadmin role both satisfy any permission check.
func (u *User) HasPermission(permission string) bool {
	if u.HasRole(RoleSuperAdmin) {
		return true
	}
	for _, p := range u.Permissions {
		if p == PermissionWildcard || p == permission {
			return true
		}
	}
	return false
}
