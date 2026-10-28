package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

type serviceFixture struct {
	svc    *Service
	store  *fakeStore
	tokens *Tokens
}

func newServiceFixture(t *testing.T, cfg Config) serviceFixture {
	t.Helper()
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 24 * time.Hour
	}
	store := newFakeStore()
	tokens := newTestTokens(t)
	return serviceFixture{svc: NewService(store, tokens, cfg), store: store, tokens: tokens}
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})

	s, err := f.svc.Register(ctx, "  Elf@Example.COM ", "password-panjang", " Lia ")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if s.User.Email != "elf@example.com" || s.User.Name != "Lia" || s.User.Role != identity.RoleUser {
		t.Errorf("unexpected user %+v", s.User)
	}
	if p, err := f.tokens.Verify(s.AccessToken); err != nil || p.UserID != s.User.ID {
		t.Errorf("access token invalid: %v", err)
	}

	if _, err := f.svc.Register(ctx, "elf@example.com", "password-lain", ""); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate Register error = %v, want ErrEmailTaken", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"ok, email case-insensitive", "ELF@example.com", "password-panjang", nil},
		{"wrong password", "elf@example.com", "salah", ErrInvalidCredentials},
		{"unknown email", "orc@example.com", "password-panjang", ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.svc.Login(ctx, tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Login error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginBannedUser(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})
	s, _ := f.svc.Register(ctx, "orc@example.com", "password-panjang", "")
	f.store.setBanned(s.User.ID)

	if _, err := f.svc.Login(ctx, "orc@example.com", "password-panjang"); !errors.Is(err, ErrBanned) {
		t.Errorf("Login error = %v, want ErrBanned", err)
	}
}

func TestAdminEmailsArePromoted(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{AdminEmails: []string{"dm@example.com"}})

	s, err := f.svc.Register(ctx, "DM@example.com", "password-panjang", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.User.Role != identity.RoleAdmin {
		t.Errorf("role = %s, want ADMIN", s.User.Role)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})
	first, _ := f.svc.Register(ctx, "bard@example.com", "password-panjang", "")

	second, err := f.svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}
	if n := f.store.activeTokens(first.User.ID); n != 1 {
		t.Errorf("active tokens = %d, want 1", n)
	}
}

func TestRefreshReuseRevokesAllSessions(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})
	first, _ := f.svc.Register(ctx, "rogue@example.com", "password-panjang", "")
	if _, err := f.svc.Login(ctx, "rogue@example.com", "password-panjang"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Refresh(ctx, first.RefreshToken); err != nil {
		t.Fatal(err)
	}

	// Replaying the already rotated token signals theft.
	if _, err := f.svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("replay error = %v, want ErrInvalidRefreshToken", err)
	}
	if n := f.store.activeTokens(first.User.ID); n != 0 {
		t.Errorf("active tokens after reuse = %d, want 0", n)
	}
}

func TestRefreshRejects(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown token", func(t *testing.T) {
		f := newServiceFixture(t, Config{})
		if _, err := f.svc.Refresh(ctx, "tidak-ada"); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Errorf("error = %v, want ErrInvalidRefreshToken", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		f := newServiceFixture(t, Config{RefreshTTL: time.Hour})
		s, _ := f.svc.Register(ctx, "monk@example.com", "password-panjang", "")
		f.svc.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
		if _, err := f.svc.Refresh(ctx, s.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Errorf("error = %v, want ErrInvalidRefreshToken", err)
		}
	})

	t.Run("banned user", func(t *testing.T) {
		f := newServiceFixture(t, Config{})
		s, _ := f.svc.Register(ctx, "cleric@example.com", "password-panjang", "")
		f.store.setBanned(s.User.ID)
		if _, err := f.svc.Refresh(ctx, s.RefreshToken); !errors.Is(err, ErrBanned) {
			t.Errorf("error = %v, want ErrBanned", err)
		}
		if n := f.store.activeTokens(s.User.ID); n != 0 {
			t.Errorf("active tokens = %d, want 0", n)
		}
	})
}

func TestLogoutIsIdempotent(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})
	s, _ := f.svc.Register(ctx, "druid@example.com", "password-panjang", "")

	for range 2 {
		if err := f.svc.Logout(ctx, s.RefreshToken); err != nil {
			t.Fatalf("Logout: %v", err)
		}
	}
	if _, err := f.svc.Refresh(ctx, s.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("Refresh after logout error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestLoginWithGoogle(t *testing.T) {
	ctx := context.Background()
	profile := GoogleProfile{Subject: "g-123", Email: "Ranger@Example.com", EmailVerified: true, Name: "Ranger"}

	t.Run("disabled", func(t *testing.T) {
		f := newServiceFixture(t, Config{})
		if _, err := f.svc.LoginWithGoogle(ctx, "code", ""); !errors.Is(err, ErrGoogleDisabled) {
			t.Errorf("error = %v, want ErrGoogleDisabled", err)
		}
	})

	t.Run("unverified email", func(t *testing.T) {
		p := profile
		p.EmailVerified = false
		f := newServiceFixture(t, Config{Google: fakeGoogle{profile: p}})
		if _, err := f.svc.LoginWithGoogle(ctx, "code", ""); !errors.Is(err, ErrGoogleUnverified) {
			t.Errorf("error = %v, want ErrGoogleUnverified", err)
		}
	})

	t.Run("creates then reuses account", func(t *testing.T) {
		f := newServiceFixture(t, Config{Google: fakeGoogle{profile: profile}})
		a, err := f.svc.LoginWithGoogle(ctx, "code", "verifier")
		if err != nil {
			t.Fatal(err)
		}
		if a.User.Email != "ranger@example.com" || a.User.GoogleSub != "g-123" || a.User.PasswordHash != "" {
			t.Errorf("unexpected user %+v", a.User)
		}
		b, err := f.svc.LoginWithGoogle(ctx, "code", "verifier")
		if err != nil {
			t.Fatal(err)
		}
		if a.User.ID != b.User.ID {
			t.Error("second Google login created a new account")
		}
	})

	t.Run("links existing password account", func(t *testing.T) {
		f := newServiceFixture(t, Config{Google: fakeGoogle{profile: profile}})
		reg, _ := f.svc.Register(ctx, "ranger@example.com", "password-panjang", "")
		s, err := f.svc.LoginWithGoogle(ctx, "code", "")
		if err != nil {
			t.Fatal(err)
		}
		if s.User.ID != reg.User.ID || s.User.GoogleSub != "g-123" {
			t.Errorf("account not linked: %+v", s.User)
		}
	})
}

func TestEnsureAdmin(t *testing.T) {
	ctx := context.Background()
	f := newServiceFixture(t, Config{})

	u, err := f.svc.EnsureAdmin(ctx, "Owner@example.com", "password-panjang")
	if err != nil || u.Role != identity.RoleAdmin {
		t.Fatalf("EnsureAdmin new = %+v, %v", u, err)
	}
	u2, err := f.svc.EnsureAdmin(ctx, "owner@example.com", "password-baru-123")
	if err != nil || u2.ID != u.ID {
		t.Fatalf("EnsureAdmin existing = %+v, %v", u2, err)
	}
	if _, err := f.svc.Login(ctx, "owner@example.com", "password-baru-123"); err != nil {
		t.Errorf("login with reset password: %v", err)
	}
}
