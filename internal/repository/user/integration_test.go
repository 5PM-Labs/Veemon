//go:build integration

// Integration tests that require a real PostgreSQL (run with:
//
//	go test -tags integration ./internal/repository/user/...
//
// with DB_* env vars pointing at a database that has the migrations applied).
package user_repository_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/5PM-Labs/veemon/internal/domain"
	"github.com/5PM-Labs/veemon/internal/infra/database"
	user_repository "github.com/5PM-Labs/veemon/internal/repository/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	port, _ := strconv.Atoi(envOr("DB_PORT", "5432"))
	pool, err := database.New(context.Background(), database.Config{
		Host:     envOr("DB_HOST", "localhost"),
		Port:     port,
		User:     envOr("DB_USER", "postgres"),
		Password: envOr("DB_PASSWORD", "postgres"),
		Name:     envOr("DB_NAME", "veemon_db"),
		SSLMode:  envOr("DB_SSL_MODE", "disable"),
		Timezone: envOr("DB_TIMEZONE", "UTC"),
	}, zap.NewNop())
	require.NoError(t, err)
	return pool
}

func newUser(email string) *domain.User {
	return &domain.User{
		Email:        email,
		PasswordHash: "hash",
		Name:         "Integration User",
		Status:       domain.UserStatusActive,
		AuthProvider: domain.AuthProviderLocal,
		Roles:        []string{"user"},
	}
}

// The regression that motivated this test: Roles ([]string over text[]) and
// Metadata (jsonb) had to round-trip through a real Postgres via pgx.
func TestIntegration_UserRolesRoundTrip(t *testing.T) {
	repo := user_repository.New(testDB(t))
	ctx := context.Background()

	email := "int-" + uuid.NewString() + "@example.com"
	u := newUser(email)
	u.Roles = []string{"admin", "user"}
	require.NoError(t, repo.Create(ctx, u))
	t.Cleanup(func() { _ = repo.Delete(ctx, u.ID) })

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"admin", "user"}, got.Roles)
	require.True(t, got.EmailVerified == false)

	// FindByEmail is the Login read path.
	byEmail, err := repo.FindByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, u.ID, byEmail.ID)

	// FindAll is the ListUsers read path.
	list, total, err := repo.FindAll(ctx, user_repository.ListParams{Page: 1, Size: 50, Search: "Integration User"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, list)
}

// Proves the partial unique index: a soft-deleted user's email can be reused.
func TestIntegration_SoftDeletedEmailReusable(t *testing.T) {
	repo := user_repository.New(testDB(t))
	ctx := context.Background()

	email := "reuse-" + uuid.NewString() + "@example.com"
	first := newUser(email)
	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Delete(ctx, first.ID)) // soft delete

	second := newUser(email)
	require.NoError(t, repo.Create(ctx, second), "re-registering a soft-deleted email should succeed")
	t.Cleanup(func() { _ = repo.Delete(ctx, second.ID) })
}

// A duplicate live email must surface as domain.ErrDuplicateEntry, driving the
// usecase's register-race handling.
func TestIntegration_DuplicateEmail(t *testing.T) {
	repo := user_repository.New(testDB(t))
	ctx := context.Background()

	email := "dup-" + uuid.NewString() + "@example.com"
	first := newUser(email)
	require.NoError(t, repo.Create(ctx, first))
	t.Cleanup(func() { _ = repo.Delete(ctx, first.ID) })

	second := newUser(email)
	err := repo.Create(ctx, second)
	require.ErrorIs(t, err, domain.ErrDuplicateEntry)
}
