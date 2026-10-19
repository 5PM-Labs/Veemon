// Package token issues and validates PASETO v4 access tokens.
package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	// ErrWeakSecret is returned when the provided secret cannot yield a secure
	// 32-byte PASETO v4 key (i.e. it is neither a 64-char hex string nor at
	// least 32 raw bytes). Short secrets are rejected rather than padded, since
	// padding a guessable secret produces a guessable, forgeable key.
	ErrWeakSecret = errors.New("token secret must be a 64-character hex string or at least 32 bytes")
)

type Claims struct {
	UserID      string   `json:"userId"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	CompanyCode string   `json:"companyCode"`
	// TokenID (jti) uniquely identifies this token so it can be revoked.
	TokenID string `json:"jti"`
	// ExpiresAt is the token's natural expiry, used to bound how long a
	// revocation entry must be retained.
	ExpiresAt time.Time `json:"-"`
}

type TokenService struct {
	secretKey  paseto.V4SymmetricKey
	expiration time.Duration
}

// NewTokenService creates a new token service with PASETO v4.
//
// secretKeyString must be either a 64-character hex string (decoding to 32
// bytes, produced by GenerateSecretKey) or a raw string of at least 32 bytes.
// Weaker secrets are rejected with ErrWeakSecret — short secrets are never
// padded, because deriving a key from a guessable secret produces a guessable
// key and PASETO v4.local tokens would then be forgeable by anyone.
func NewTokenService(secretKeyString string, expirationHours int) (*TokenService, error) {
	key, err := deriveKey(secretKeyString)
	if err != nil {
		return nil, err
	}
	return &TokenService{
		secretKey:  key,
		expiration: time.Duration(expirationHours) * time.Hour,
	}, nil
}

// deriveKey turns a configured secret string into a PASETO v4 symmetric key.
func deriveKey(secretKeyString string) (paseto.V4SymmetricKey, error) {
	var zero paseto.V4SymmetricKey

	// 64 hex chars decode to exactly 32 bytes.
	if len(secretKeyString) == 64 {
		keyBytes, err := hex.DecodeString(secretKeyString)
		if err == nil && len(keyBytes) == 32 {
			return paseto.V4SymmetricKeyFromBytes(keyBytes)
		}
	}

	// Otherwise require at least 32 raw bytes and use the first 32.
	if len(secretKeyString) < 32 {
		return zero, ErrWeakSecret
	}
	return paseto.V4SymmetricKeyFromBytes([]byte(secretKeyString)[:32])
}

// GenerateSecretKey generates a new random 32-byte secret key and returns it as hex
func GenerateSecretKey() string {
	key := paseto.NewV4SymmetricKey()
	return hex.EncodeToString(key.ExportBytes())
}

// GenerateToken creates a new PASETO token with user claims. Each token is
// stamped with a unique jti so it can be individually revoked.
func (ts *TokenService) GenerateToken(userID, email string, roles []string, companyCode string) (string, error) {
	token := paseto.NewToken()

	now := time.Now()

	jti, err := newTokenID()
	if err != nil {
		return "", err
	}

	// Set registered claims
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(ts.expiration))
	token.SetJti(jti)

	// Set custom claims
	token.SetString("userId", userID)
	token.SetString("email", email)
	if err := token.Set("roles", roles); err != nil {
		return "", err
	}
	token.SetString("companyCode", companyCode)

	// Encrypt the token (v4.local)
	encrypted := token.V4Encrypt(ts.secretKey, nil)
	return encrypted, nil
}

// newTokenID returns a cryptographically random 128-bit token identifier (jti).
func newTokenID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// ValidateToken validates and decrypts a PASETO token
func (ts *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	parser := paseto.NewParser()

	// Add validation rules
	parser.AddRule(paseto.NotExpired())
	parser.AddRule(paseto.ValidAt(time.Now()))

	// Parse and decrypt the token
	token, err := parser.ParseV4Local(ts.secretKey, tokenString, nil)
	if err != nil {
		// Distinguish an expired token from an otherwise invalid one. The
		// paseto NotExpired rule reports "this token has expired"; match only
		// the unambiguous "expired" substring to avoid false positives from
		// words like "unexpected".
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims := &Claims{}

	// Get userId
	if err := token.Get("userId", &claims.UserID); err != nil {
		return nil, ErrInvalidToken
	}

	// Get email
	if err := token.Get("email", &claims.Email); err != nil {
		return nil, ErrInvalidToken
	}

	// Get companyCode
	if err := token.Get("companyCode", &claims.CompanyCode); err != nil {
		return nil, ErrInvalidToken
	}

	// Get roles array
	if err := token.Get("roles", &claims.Roles); err != nil {
		return nil, ErrInvalidToken
	}

	// Registered claims (best-effort).
	if jti, err := token.GetJti(); err == nil {
		claims.TokenID = jti
	}
	if exp, err := token.GetExpiration(); err == nil {
		claims.ExpiresAt = exp
	}

	return claims, nil
}

// GetSecretKeyHex exports the secret key as hex string (for backup/migration)
func (ts *TokenService) GetSecretKeyHex() string {
	return hex.EncodeToString(ts.secretKey.ExportBytes())
}
