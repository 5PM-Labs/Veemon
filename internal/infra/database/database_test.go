package database

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOperationSummary_NeverLeaksInterpolatedValues(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "select",
			sql:  `SELECT * FROM "users" WHERE email = 'user@example.com' LIMIT 1`,
			want: "SELECT users",
		},
		{
			name: "insert with sensitive literals in VALUES",
			sql:  `INSERT INTO "users" ("email","password") VALUES ('user@example.com','$2a$10$abc')`,
			want: "INSERT users",
		},
		{
			name: "update with sensitive literal in SET",
			sql:  `UPDATE "users" SET "password" = '$2a$10$abc' WHERE "id" = 42`,
			want: "UPDATE users",
		},
		{
			name: "delete",
			sql:  `DELETE FROM "users" WHERE "id" = 42`,
			want: "DELETE users",
		},
		{
			name: "unrecognized statement falls back safely",
			sql:  `EXPLAIN ANALYZE SELECT 1`,
			want: "unknown",
		},
		{
			name: "empty string falls back safely",
			sql:  "",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OperationSummary(tt.sql)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "user@example.com")
			assert.NotContains(t, got, "$2a$10$")
		})
	}
}

// unusedTCPAddr reserves a free local port, then releases it, so the dial fails
// with a real "connection refused".
func unusedTCPAddr(t *testing.T) (string, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	require.NoError(t, l.Close())
	return addr.IP.String(), addr.Port
}

// New pings on open, so it fails fast on an unreachable host rather than
// deferring the failure to the first query.
func TestNew_UnreachableHostReturnsWrappedError(t *testing.T) {
	host, port := unusedTCPAddr(t)

	_, err := New(context.Background(), Config{
		Host: host, Port: port, User: "u", Name: "d", SSLMode: "disable", Timezone: "UTC",
	}, zap.NewNop())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to database")
}
