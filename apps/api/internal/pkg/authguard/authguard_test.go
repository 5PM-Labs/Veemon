package authguard

import (
	"context"
	"testing"
	"time"
)

// With no Redis client the guard must degrade to a safe no-op: never locked,
// never revoked, and mutating calls must not panic.
func TestGuard_NoRedis_IsNoop(t *testing.T) {
	ctx := context.Background()
	g := New(nil, 5, 15)

	if g.IsLocked(ctx, "user@example.com") {
		t.Error("IsLocked should be false without Redis")
	}
	if g.IsRevoked(ctx, "some-jti") {
		t.Error("IsRevoked should be false without Redis")
	}

	// These must be safe no-ops (no panic, no error surfaced).
	g.RecordFailure(ctx, "user@example.com")
	g.Reset(ctx, "user@example.com")
	if err := g.Revoke(ctx, "some-jti", time.Minute); err != nil {
		t.Errorf("Revoke without Redis should be a no-op, got %v", err)
	}
}

// A nil *Guard must also behave as a no-op so callers need not nil-check.
func TestGuard_Nil_IsNoop(t *testing.T) {
	ctx := context.Background()
	var g *Guard

	if g.IsLocked(ctx, "user@example.com") {
		t.Error("nil guard IsLocked should be false")
	}
	if g.IsRevoked(ctx, "jti") {
		t.Error("nil guard IsRevoked should be false")
	}
	g.RecordFailure(ctx, "user@example.com")
	g.Reset(ctx, "user@example.com")
	if err := g.Revoke(ctx, "jti", time.Minute); err != nil {
		t.Errorf("nil guard Revoke should be nil, got %v", err)
	}
}
