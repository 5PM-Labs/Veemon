// Package middleware provides HTTP and gRPC middleware (auth, logging, tracing, rate limiting).
package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/5PM-Labs/veemon/internal/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

// ctxKey is an unexported type for context keys defined in this package,
// preventing collisions with keys defined elsewhere (an untyped string key
// like "auth" can silently clash across packages).
type ctxKey int

const authCtxKey ctxKey = iota

// WithAuthContext returns a new context carrying the authenticated user.
func WithAuthContext(ctx context.Context, a *AuthContext) context.Context {
	return context.WithValue(ctx, authCtxKey, a)
}

// AuthFromContext extracts the authenticated user from the context, if present.
func AuthFromContext(ctx context.Context) (*AuthContext, bool) {
	a, ok := ctx.Value(authCtxKey).(*AuthContext)
	return a, ok
}

type AuthContext struct {
	UserID      string
	Email       string
	Roles       []string
	Permissions []string
	CompanyCode string
	Token       string
	// TokenID is the token's jti, used to revoke this specific token.
	TokenID string
	// ExpiresAt is the token's natural expiry, used to bound revocation TTL.
	ExpiresAt time.Time
}

// AuthConfig is one route's/one RPC's auth policy. AllowedPerms and
// AllowedRoles are OR-ed together: holding either satisfies the policy.
type AuthConfig struct {
	NeedAuth     bool
	AllowedRoles []string
	AllowedPerms []string
}

type TokenValidator func(token string) (*AuthContext, error)

func AuthMiddleware(validator TokenValidator, config AuthConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !config.NeedAuth {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return errors.Unauthorized("missing authorization header").FiberError(c)
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return errors.Unauthorized("invalid authorization format").FiberError(c)
		}

		token := parts[1]

		authCtx, err := validator(token)
		if err != nil {
			return errors.Unauthorized("invalid token").FiberError(c)
		}

		if !authorized(authCtx, config) {
			return errors.Forbidden("insufficient permissions").FiberError(c)
		}

		c.Locals("auth", authCtx)

		return c.Next()
	}
}

// GetAuthContext returns the authenticated user attached by AuthMiddleware.
// Handlers must treat !ok as unauthenticated (401) rather than assume the
// middleware ran, so a route wired without auth fails closed.
func GetAuthContext(c fiber.Ctx) (*AuthContext, bool) {
	auth, ok := c.Locals("auth").(*AuthContext)
	return auth, ok
}

func hasAnyRole(userRoles, allowedRoles []string) bool {
	roleSet := make(map[string]bool)
	for _, role := range userRoles {
		roleSet[role] = true
	}
	for _, allowed := range allowedRoles {
		if roleSet[allowed] {
			return true
		}
	}
	return false
}

// authorized returns true if the context satisfies the role OR permission
// requirements of config. When neither is set, any authenticated caller passes.
func authorized(authCtx *AuthContext, config AuthConfig) bool {
	if len(config.AllowedRoles) > 0 && hasAnyRole(authCtx.Roles, config.AllowedRoles) {
		return true
	}
	if len(config.AllowedPerms) > 0 && hasAnyPermission(authCtx.Permissions, config.AllowedPerms) {
		return true
	}
	return len(config.AllowedRoles) == 0 && len(config.AllowedPerms) == 0
}

func hasAnyPermission(userPerms, allowedPerms []string) bool {
	permSet := make(map[string]bool)
	for _, p := range userPerms {
		permSet[p] = true
	}
	if permSet["*"] { // wildcard grants everything
		return true
	}
	for _, allowed := range allowedPerms {
		if permSet[allowed] {
			return true
		}
	}
	return false
}
