package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Valid 32-byte raw secrets for tests (NewTokenService rejects anything shorter).
const (
	testSecretA = "test-secret-key-0123456789abcdef"
	testSecretB = "test-secret-key-abcdef0123456789"
)

func mustNewTokenService(t *testing.T, secret string, expHours int) *TokenService {
	t.Helper()
	ts, err := NewTokenService(secret, expHours)
	require.NoError(t, err)
	return ts
}

func TestTokenService_GenerateAndValidateToken(t *testing.T) {
	ts := mustNewTokenService(t, testSecretA, 24)

	userID := "user123"
	email := "test@example.com"
	roles := []string{"admin", "user"}
	companyCode := "COMP001"

	// Generate token
	token, err := ts.GenerateToken(userID, email, roles, companyCode)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := ts.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, roles, claims.Roles)
	assert.Equal(t, companyCode, claims.CompanyCode)
}

func TestTokenService_InvalidToken(t *testing.T) {
	ts := mustNewTokenService(t, testSecretA, 24)

	// Test with invalid token
	_, err := ts.ValidateToken("invalid-token")
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Test with empty token
	_, err = ts.ValidateToken("")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenService_ExpiredToken(t *testing.T) {
	// Create service with a valid key but sub-millisecond expiration.
	base := mustNewTokenService(t, testSecretA, 1)
	ts := &TokenService{
		secretKey:  base.secretKey,
		expiration: 1 * time.Millisecond,
	}

	token, err := ts.GenerateToken("user123", "test@example.com", []string{"user"}, "COMP001")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired token
	_, err = ts.ValidateToken(token)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestTokenService_DifferentSecretKeys(t *testing.T) {
	ts1 := mustNewTokenService(t, testSecretA, 24)
	ts2 := mustNewTokenService(t, testSecretB, 24)

	// Generate token with first service
	token, err := ts1.GenerateToken("user123", "test@example.com", []string{"user"}, "COMP001")
	require.NoError(t, err)

	// Try to validate with different secret key
	_, err = ts2.ValidateToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenService_WeakSecretRejected(t *testing.T) {
	// Secrets shorter than 32 bytes must be rejected, not padded.
	for _, weak := range []string{"", "short", "your-secret-key-change-in-production"[:31]} {
		_, err := NewTokenService(weak, 24)
		assert.ErrorIs(t, err, ErrWeakSecret, "secret %q should be rejected", weak)
	}
}

func TestTokenService_HexSecretAccepted(t *testing.T) {
	// A 64-char hex secret (e.g. from GenerateSecretKey) is accepted.
	hexKey := GenerateSecretKey()
	require.Len(t, hexKey, 64)

	ts, err := NewTokenService(hexKey, 24)
	require.NoError(t, err)

	token, err := ts.GenerateToken("user123", "test@example.com", []string{"user"}, "COMP001")
	require.NoError(t, err)

	claims, err := ts.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
}

func TestTokenService_LongSecretKey(t *testing.T) {
	// A secret longer than 32 bytes is truncated to the first 32 bytes.
	longKey := "this-is-a-very-long-secret-key-that-exceeds-32-bytes-and-should-be-truncated"
	ts := mustNewTokenService(t, longKey, 24)

	token, err := ts.GenerateToken("user123", "test@example.com", []string{"user"}, "COMP001")
	require.NoError(t, err)

	claims, err := ts.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
}

func TestTokenService_EmptyRoles(t *testing.T) {
	ts := mustNewTokenService(t, testSecretA, 24)

	// Generate token with empty roles
	token, err := ts.GenerateToken("user123", "test@example.com", []string{}, "COMP001")
	require.NoError(t, err)

	claims, err := ts.ValidateToken(token)
	require.NoError(t, err)
	assert.Empty(t, claims.Roles)
}

func TestTokenService_MultipleRoles(t *testing.T) {
	ts := mustNewTokenService(t, testSecretA, 24)

	roles := []string{"admin", "user", "moderator", "viewer"}
	token, err := ts.GenerateToken("user123", "test@example.com", roles, "COMP001")
	require.NoError(t, err)

	claims, err := ts.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, roles, claims.Roles)
}
