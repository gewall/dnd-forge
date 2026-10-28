package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

func newTestTokens(t *testing.T) *Tokens {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return NewTokens(key, "test-issuer", "test-aud", 15*time.Minute)
}

func TestTokensRoundTrip(t *testing.T) {
	tok := newTestTokens(t)
	want := identity.Principal{UserID: uuid.New(), Role: identity.RoleAdmin}

	raw, exp, err := tok.Issue(want)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if d := time.Until(exp); d < 14*time.Minute || d > 15*time.Minute {
		t.Errorf("expiry in %v, want ~15m", d)
	}
	got, err := tok.Verify(raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != want {
		t.Errorf("Verify = %+v, want %+v", got, want)
	}
}

func TestTokensVerifyRejects(t *testing.T) {
	tok := newTestTokens(t)
	p := identity.Principal{UserID: uuid.New(), Role: identity.RoleUser}
	valid, _, _ := tok.Issue(p)

	expired := newTestTokens(t)
	expired.key = tok.key
	expired.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expiredRaw, _, _ := expired.Issue(p)

	otherKey := newTestTokens(t)
	foreignRaw, _, _ := otherKey.Issue(p)

	otherAud := NewTokens(tok.key, "test-issuer", "other-aud", time.Minute)
	otherAudRaw, _, _ := otherAud.Issue(p)

	tests := map[string]string{
		"garbage":        "not-a-jwt",
		"tampered":       valid[:len(valid)-2] + "xx",
		"expired":        expiredRaw,
		"foreign key":    foreignRaw,
		"wrong audience": otherAudRaw,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := tok.Verify(raw); !errors.Is(err, ErrInvalidAccessToken) {
				t.Errorf("Verify error = %v, want ErrInvalidAccessToken", err)
			}
		})
	}
}

func TestPublicKeyPEM(t *testing.T) {
	pem, err := newTestTokens(t).PublicKeyPEM()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pem, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("unexpected PEM: %s", pem)
	}
}
