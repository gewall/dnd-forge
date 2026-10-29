//go:build integration

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

func newIntegrationStore(t *testing.T) *PostgresStore {
	t.Helper()
	return NewPostgresStore(dbtest.Pool(t, "users"))
}

func TestPostgresStoreUsers(t *testing.T) {
	ctx := context.Background()
	s := newIntegrationStore(t)

	u, err := s.CreateUser(ctx, NewUser{Email: "tiefling@example.com", Name: "Zara", Role: identity.RoleUser, PasswordHash: "h"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := s.CreateUser(ctx, NewUser{Email: "tiefling@example.com", Role: identity.RoleUser}); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate CreateUser error = %v, want ErrEmailTaken", err)
	}

	got, err := s.UserByEmail(ctx, "tiefling@example.com")
	if err != nil || got.ID != u.ID || got.PasswordHash != "h" || got.GoogleSub != "" {
		t.Errorf("UserByEmail = %+v, %v", got, err)
	}
	if _, err := s.UserByEmail(ctx, "nobody@example.com"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("missing UserByEmail error = %v, want ErrUserNotFound", err)
	}

	linked, err := s.LinkGoogle(ctx, u.ID, "sub-1")
	if err != nil || linked.GoogleSub != "sub-1" {
		t.Errorf("LinkGoogle = %+v, %v", linked, err)
	}
	if bySub, err := s.UserByGoogleSub(ctx, "sub-1"); err != nil || bySub.ID != u.ID {
		t.Errorf("UserByGoogleSub = %+v, %v", bySub, err)
	}
	if admin, err := s.SetRole(ctx, u.ID, identity.RoleAdmin); err != nil || admin.Role != identity.RoleAdmin {
		t.Errorf("SetRole = %+v, %v", admin, err)
	}
}

func TestPostgresStoreRefreshTokens(t *testing.T) {
	ctx := context.Background()
	s := newIntegrationStore(t)
	u, err := s.CreateUser(ctx, NewUser{Email: "paladin@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}

	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	if err := s.SaveRefreshToken(ctx, RefreshToken{UserID: u.ID, Hash: "hash-1", ExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}

	err = s.InTx(ctx, func(ts TokenStore) error {
		tok, err := ts.RefreshTokenForUpdate(ctx, "hash-1")
		if err != nil {
			return err
		}
		if tok.UserID != u.ID || !tok.ExpiresAt.Equal(exp) || tok.RevokedAt != nil {
			t.Errorf("RefreshTokenForUpdate = %+v", tok)
		}
		if err := ts.RevokeRefreshToken(ctx, tok.ID); err != nil {
			return err
		}
		return ts.SaveRefreshToken(ctx, RefreshToken{UserID: u.ID, Hash: "hash-2", ExpiresAt: exp})
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}

	old, _ := s.RefreshTokenForUpdate(ctx, "hash-1")
	if old.RevokedAt == nil {
		t.Error("old token not revoked")
	}
	if err := s.RevokeAllRefreshTokens(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if next, _ := s.RefreshTokenForUpdate(ctx, "hash-2"); next.RevokedAt == nil {
		t.Error("RevokeAllRefreshTokens left a token active")
	}
	if _, err := s.RefreshTokenForUpdate(ctx, "missing"); !errors.Is(err, errRefreshTokenNotFound) {
		t.Errorf("missing token error = %v", err)
	}
}

func TestPostgresStoreInTxRollsBack(t *testing.T) {
	ctx := context.Background()
	s := newIntegrationStore(t)
	u, _ := s.CreateUser(ctx, NewUser{Email: "warlock@example.com", Role: identity.RoleUser})

	boom := errors.New("boom")
	err := s.InTx(ctx, func(ts TokenStore) error {
		if err := ts.SaveRefreshToken(ctx, RefreshToken{UserID: u.ID, Hash: "rolled-back", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx error = %v, want boom", err)
	}
	if _, err := s.RefreshTokenForUpdate(ctx, "rolled-back"); !errors.Is(err, errRefreshTokenNotFound) {
		t.Errorf("token survived rollback: %v", err)
	}
}
