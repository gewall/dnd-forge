package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

// Tokens signs and verifies access tokens with an Ed25519 key. The Next.js
// middleware verifies the same tokens with the public half.
type Tokens struct {
	key      ed25519.PrivateKey
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

// NewTokens returns a Tokens that issues access tokens valid for ttl.
func NewTokens(key ed25519.PrivateKey, issuer, audience string, ttl time.Duration) *Tokens {
	return &Tokens{key: key, issuer: issuer, audience: audience, ttl: ttl, now: time.Now}
}

type claims struct {
	Role identity.Role `json:"role"`
	jwt.RegisteredClaims
}

// Issue returns a signed access token for p and its expiry.
func (t *Tokens) Issue(p identity.Principal) (string, time.Time, error) {
	now := t.now()
	exp := now.Add(t.ttl)
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims{
		Role: p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.UserID.String(),
			Issuer:    t.issuer,
			Audience:  jwt.ClaimStrings{t.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	signed, err := tok.SignedString(t.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}
	return signed, exp, nil
}

// Verify parses an access token and returns the principal it carries.
func (t *Tokens) Verify(raw string) (identity.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c,
		func(*jwt.Token) (any, error) { return t.key.Public(), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithAudience(t.audience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return identity.Principal{}, ErrInvalidAccessToken
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil || (c.Role != identity.RoleUser && c.Role != identity.RoleAdmin) {
		return identity.Principal{}, ErrInvalidAccessToken
	}
	return identity.Principal{UserID: id, Role: c.Role}, nil
}

// PublicKeyPEM returns the verification key as a PKIX PEM block.
func (t *Tokens) PublicKeyPEM() (string, error) {
	der, err := x509.MarshalPKIXPublicKey(t.key.Public())
	if err != nil {
		return "", fmt.Errorf("auth: marshal public key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

// newRefreshToken returns a random refresh token and the hash to store.
func newRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth: refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashRefreshToken(raw), nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
