// Package auth0 verifies Auth0-issued RS256 access tokens and maps a verified
// Auth0 identity onto the first-party user model (provider=auth0, subject = the
// Auth0 "sub" claim, e.g. "auth0|abc123").
//
// Signing keys are fetched from the tenant JWKS endpoint and cached briefly.
// Only the "aud" claim is matched against the configured API audience; all
// other claim checks (iss, exp, signature) are handled by jwt.ParseWithClaims.
package auth0

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// ErrInvalidToken is returned when a token fails structure, signature, or
// audience validation.
var ErrInvalidToken = errors.New("invalid token")

// keyCacheTTL bounds how long a JWKS set is trusted before refetch.
const keyCacheTTL = 5 * time.Minute

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

// Provider verifies Auth0 tokens using a cached JWKS.
type Provider struct {
	jwksURL  string
	issuer   string
	audience string
	algo     string

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// New builds a Provider. jwksURL is the tenant JWKS endpoint (typically
// "https://<tenant>.auth0.com/.well-known/jwks.json"); audience is the API
// identifier configured on the Auth0 API. issuer, when set, is additionally
// enforced against the token's "iss" claim.
func New(jwksURL, issuer, audience string) *Provider {
	return &Provider{
		jwksURL:  jwksURL,
		issuer:   issuer,
		audience: audience,
		algo:     "RS256",
		keys:     map[string]*rsa.PublicKey{},
	}
}

// Verify parses, signature-checks, and validates an Auth0 access token,
// returning its claims.
func (p *Provider) Verify(ctx context.Context, token string) (jwt.MapClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{p.algo}))

	parsed, err := parser.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return p.keyFor(ctx, kid)
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	if p.issuer != "" {
		iss, _ := claims["iss"].(string)
		if !strings.EqualFold(iss, p.issuer) {
			return nil, fmt.Errorf("token issuer mismatch: %w", ErrInvalidToken)
		}
	}

	if p.audience != "" {
		if !audienceContains(claims["aud"], p.audience) {
			return nil, fmt.Errorf("token audience mismatch: %w", ErrInvalidToken)
		}
	}

	return claims, nil
}

// Subject extracts the stable Auth0 subject ("auth0|...") from claims.
func Subject(claims jwt.MapClaims) string {
	if s, _ := claims["sub"].(string); s != "" {
		return s
	}
	return ""
}

// Email extracts the verified email claim, if present.
func Email(claims jwt.MapClaims) string {
	if e, _ := claims["email"].(string); e != "" {
		return e
	}
	return ""
}

func (p *Provider) keyFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.fetchedAt) < keyCacheTTL && p.keys[kid] != nil {
		return p.keys[kid], nil
	}
	if err := p.refreshKeys(ctx); err != nil {
		return nil, err
	}
	if key, ok := p.keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("no signing key for kid %q", kid)
}

func (p *Provider) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var set jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return err
	}

	fresh := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if key, err := rsaKey(k.N, k.E); err == nil {
			fresh[k.Kid] = key
		}
	}
	if len(fresh) == 0 {
		return errors.New("no usable RSA keys in jwks set")
	}

	p.keys = fresh
	p.fetchedAt = time.Now()
	return nil
}

func rsaKey(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	exp := 0
	for _, b := range eb {
		exp = exp<<8 | int(b)
	}
	if exp < 2 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: exp}, nil
}

// audienceContains matches aud whether the claim is a string or a string slice.
func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
