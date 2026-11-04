//go:build integration

package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

func TestPostgresAccounts(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "audit_logs")
	users := auth.NewPostgresStore(pool)
	mk := func(email, name string, role identity.Role) auth.User {
		t.Helper()
		u, err := users.CreateUser(ctx, auth.NewUser{Email: email, Name: name, Role: role, PasswordHash: "x"})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	admin := mk("admin@example.com", "Admin", identity.RoleAdmin)
	lia := mk("lia@example.com", "Lia 100%", identity.RoleUser)
	bo := mk("bo_b@example.com", "Bob", identity.RoleUser)
	if err := users.SaveRefreshToken(ctx, auth.RefreshToken{UserID: lia.ID, Hash: "h1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	store := NewPostgresStore(pool)
	svc := NewService(Deps{Store: store, Portrait: meter{}, Lore: meter{}})

	list := func(f Filter) []string {
		t.Helper()
		as, err := svc.List(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		var emails []string
		for _, a := range as {
			emails = append(emails, a.Email)
		}
		return emails
	}
	if got := list(Filter{}); len(got) != 3 || got[0] != bo.Email {
		t.Errorf("all = %v, want 3 newest first", got)
	}
	if got := list(Filter{Query: "100%"}); len(got) != 1 || got[0] != lia.Email {
		t.Errorf("literal %% search = %v", got)
	}
	if got := list(Filter{Query: "o_b"}); len(got) != 1 || got[0] != bo.Email {
		t.Errorf("literal _ search = %v", got)
	}
	if got := list(Filter{Role: identity.RoleAdmin}); len(got) != 1 || got[0] != admin.Email {
		t.Errorf("admins = %v", got)
	}
	if got := list(Filter{Before: &bo.CreatedAt, Limit: 1}); len(got) != 1 || got[0] != lia.Email {
		t.Errorf("page after bob = %v", got)
	}

	d, err := svc.Update(ctx, admin.ID, lia.ID, Changes{Role: identity.RoleUser, IsBanned: true, LoreQuota: ptr(2)})
	if err != nil || !d.IsBanned || d.LoreQuota == nil || *d.LoreQuota != 2 || d.PortraitQuota != nil || !d.HasPassword {
		t.Fatalf("Update = %+v, %v", d, err)
	}
	banned := true
	if got := list(Filter{Banned: &banned}); len(got) != 1 || got[0] != lia.Email {
		t.Errorf("banned = %v", got)
	}
	tok, err := users.RefreshTokenForUpdate(ctx, "h1")
	if err != nil || tok.RevokedAt == nil {
		t.Errorf("refresh token after block = %+v, %v; want revoked", tok, err)
	}
	if _, err := svc.Current(ctx, identity.Principal{UserID: lia.ID}); !errors.Is(err, ErrBanned) {
		t.Errorf("Current(blocked) err = %v", err)
	}
	var entries int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE entity = 'user' AND entity_id = $1 AND actor_id = $2",
		lia.ID.String(), admin.ID).Scan(&entries); err != nil || entries != 1 {
		t.Errorf("audit entries = %d, %v", entries, err)
	}

	if _, err := svc.Update(ctx, admin.ID, uuid.New(), Changes{Role: identity.RoleUser}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user err = %v", err)
	}
	if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get unknown err = %v", err)
	}
	if _, err := store.Status(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Status unknown err = %v", err)
	}
}
