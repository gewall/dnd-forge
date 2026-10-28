package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

// UserStore persists accounts.
type UserStore interface {
	CreateUser(ctx context.Context, u NewUser) (User, error) // ErrEmailTaken on duplicate email
	UserByID(ctx context.Context, id uuid.UUID) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)       // ErrUserNotFound
	UserByGoogleSub(ctx context.Context, subject string) (User, error) // ErrUserNotFound
	LinkGoogle(ctx context.Context, id uuid.UUID, subject string) (User, error)
	SetRole(ctx context.Context, id uuid.UUID, role identity.Role) (User, error)
	SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error
}

// TokenStore persists refresh tokens.
type TokenStore interface {
	SaveRefreshToken(ctx context.Context, t RefreshToken) error
	// RefreshTokenForUpdate locks and returns the token with the given hash,
	// or errRefreshTokenNotFound.
	RefreshTokenForUpdate(ctx context.Context, hash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

// Store is everything the service needs from persistence. InTx runs fn with
// a TokenStore bound to a single transaction.
type Store interface {
	UserStore
	TokenStore
	InTx(ctx context.Context, fn func(TokenStore) error) error
}

// AccessTokenIssuer signs access tokens.
type AccessTokenIssuer interface {
	Issue(p identity.Principal) (token string, expiresAt time.Time, err error)
}

// GoogleIdentity exchanges an OAuth authorization code for a verified profile.
type GoogleIdentity interface {
	Exchange(ctx context.Context, code, codeVerifier string) (GoogleProfile, error)
}

// Config tunes the service.
type Config struct {
	RefreshTTL  time.Duration
	AdminEmails []string       // lowercase; promoted to ADMIN on sign-in
	Google      GoogleIdentity // nil disables Google sign-in
	Logger      *slog.Logger
}

// Service implements registration, sign-in and session refresh.
type Service struct {
	store  Store
	tokens AccessTokenIssuer
	google GoogleIdentity
	admins map[string]bool
	ttl    time.Duration
	log    *slog.Logger
	now    func() time.Time
}

// NewService returns a Service backed by store and tokens.
func NewService(store Store, tokens AccessTokenIssuer, cfg Config) *Service {
	admins := make(map[string]bool, len(cfg.AdminEmails))
	for _, e := range cfg.AdminEmails {
		admins[normalizeEmail(e)] = true
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store:  store,
		tokens: tokens,
		google: cfg.Google,
		admins: admins,
		ttl:    cfg.RefreshTTL,
		log:    log,
		now:    time.Now,
	}
}

// Register creates a password account and signs it in.
func (s *Service) Register(ctx context.Context, email, password, name string) (Session, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return Session{}, err
	}
	email = normalizeEmail(email)
	u, err := s.store.CreateUser(ctx, NewUser{
		Email:        email,
		Name:         strings.TrimSpace(name),
		Role:         s.roleFor(email),
		PasswordHash: hash,
	})
	if err != nil {
		return Session{}, fmt.Errorf("auth: register: %w", err)
	}
	return s.startSession(ctx, u)
}

// Login signs in with email and password.
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	u, err := s.store.UserByEmail(ctx, normalizeEmail(email))
	switch {
	case errors.Is(err, ErrUserNotFound):
		_, _ = VerifyPassword(dummyHash, password)
		return Session{}, ErrInvalidCredentials
	case err != nil:
		return Session{}, fmt.Errorf("auth: login: %w", err)
	case u.PasswordHash == "":
		_, _ = VerifyPassword(dummyHash, password)
		return Session{}, ErrInvalidCredentials
	}
	ok, err := VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return Session{}, fmt.Errorf("auth: login: %w", err)
	}
	if !ok {
		return Session{}, ErrInvalidCredentials
	}
	return s.signIn(ctx, u)
}

// LoginWithGoogle signs in (creating or linking the account) with a Google
// authorization code obtained by the frontend.
func (s *Service) LoginWithGoogle(ctx context.Context, code, codeVerifier string) (Session, error) {
	if s.google == nil {
		return Session{}, ErrGoogleDisabled
	}
	p, err := s.google.Exchange(ctx, code, codeVerifier)
	if err != nil {
		return Session{}, fmt.Errorf("auth: google: %w", err)
	}
	if !p.EmailVerified {
		return Session{}, ErrGoogleUnverified
	}

	u, err := s.store.UserByGoogleSub(ctx, p.Subject)
	if errors.Is(err, ErrUserNotFound) {
		u, err = s.linkOrCreateGoogleUser(ctx, p)
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth: google: %w", err)
	}
	return s.signIn(ctx, u)
}

func (s *Service) linkOrCreateGoogleUser(ctx context.Context, p GoogleProfile) (User, error) {
	email := normalizeEmail(p.Email)
	u, err := s.store.UserByEmail(ctx, email)
	switch {
	case err == nil:
		// Google verified the address, so it is safe to attach it to the
		// existing password account with the same email.
		return s.store.LinkGoogle(ctx, u.ID, p.Subject)
	case errors.Is(err, ErrUserNotFound):
		return s.store.CreateUser(ctx, NewUser{
			Email:     email,
			Name:      strings.TrimSpace(p.Name),
			Role:      s.roleFor(email),
			GoogleSub: p.Subject,
		})
	default:
		return User{}, err
	}
}

// Refresh rotates a refresh token and issues a new session. Presenting a
// token that was already rotated revokes every session of its owner, since
// it means the token leaked.
func (s *Service) Refresh(ctx context.Context, rawToken string) (Session, error) {
	var (
		userID uuid.UUID
		reused bool
		next   RefreshToken
		raw    string
	)
	err := s.store.InTx(ctx, func(ts TokenStore) error {
		old, err := ts.RefreshTokenForUpdate(ctx, hashRefreshToken(rawToken))
		if errors.Is(err, errRefreshTokenNotFound) {
			return ErrInvalidRefreshToken
		}
		if err != nil {
			return err
		}
		userID = old.UserID
		if old.RevokedAt != nil {
			reused = true
			return ts.RevokeAllRefreshTokens(ctx, old.UserID)
		}
		if !s.now().Before(old.ExpiresAt) {
			return ErrInvalidRefreshToken
		}
		if err := ts.RevokeRefreshToken(ctx, old.ID); err != nil {
			return err
		}
		raw, next, err = s.newRefreshToken(old.UserID)
		if err != nil {
			return err
		}
		return ts.SaveRefreshToken(ctx, next)
	})
	if err != nil {
		return Session{}, fmt.Errorf("auth: refresh: %w", err)
	}
	if reused {
		s.log.WarnContext(ctx, "refresh token reuse detected; all sessions revoked", "user_id", userID)
		return Session{}, ErrInvalidRefreshToken
	}

	u, err := s.store.UserByID(ctx, userID)
	if err != nil {
		return Session{}, fmt.Errorf("auth: refresh: %w", err)
	}
	if u.IsBanned {
		if err := s.store.RevokeAllRefreshTokens(ctx, u.ID); err != nil {
			return Session{}, fmt.Errorf("auth: refresh: %w", err)
		}
		return Session{}, ErrBanned
	}
	return s.session(u, raw, next.ExpiresAt)
}

// Logout revokes a refresh token. Unknown tokens are ignored so logout is
// idempotent.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	t, err := s.store.RefreshTokenForUpdate(ctx, hashRefreshToken(rawToken))
	if errors.Is(err, errRefreshTokenNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	if err := s.store.RevokeRefreshToken(ctx, t.ID); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	return nil
}

// User returns the account with the given id.
func (s *Service) User(ctx context.Context, id uuid.UUID) (User, error) {
	u, err := s.store.UserByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("auth: user: %w", err)
	}
	return u, nil
}

// EnsureAdmin creates an admin account, or promotes and resets the password
// of an existing one. It backs the seed-admin command.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	u, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		return s.store.CreateUser(ctx, NewUser{Email: email, Role: identity.RoleAdmin, PasswordHash: hash})
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: ensure admin: %w", err)
	}
	if err := s.store.SetPasswordHash(ctx, u.ID, hash); err != nil {
		return User{}, fmt.Errorf("auth: ensure admin: %w", err)
	}
	return s.store.SetRole(ctx, u.ID, identity.RoleAdmin)
}

// signIn checks the account state, applies ADMIN_EMAILS and starts a session.
func (s *Service) signIn(ctx context.Context, u User) (Session, error) {
	if u.IsBanned {
		return Session{}, ErrBanned
	}
	if u.Role != identity.RoleAdmin && s.admins[u.Email] {
		var err error
		if u, err = s.store.SetRole(ctx, u.ID, identity.RoleAdmin); err != nil {
			return Session{}, fmt.Errorf("auth: promote admin: %w", err)
		}
	}
	return s.startSession(ctx, u)
}

func (s *Service) startSession(ctx context.Context, u User) (Session, error) {
	raw, rt, err := s.newRefreshToken(u.ID)
	if err != nil {
		return Session{}, err
	}
	if err := s.store.SaveRefreshToken(ctx, rt); err != nil {
		return Session{}, fmt.Errorf("auth: save refresh token: %w", err)
	}
	return s.session(u, raw, rt.ExpiresAt)
}

func (s *Service) session(u User, rawRefresh string, refreshExp time.Time) (Session, error) {
	access, accessExp, err := s.tokens.Issue(u.Principal())
	if err != nil {
		return Session{}, err
	}
	return Session{
		User:             u,
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     rawRefresh,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (s *Service) newRefreshToken(userID uuid.UUID) (string, RefreshToken, error) {
	raw, hash, err := newRefreshToken()
	if err != nil {
		return "", RefreshToken{}, err
	}
	return raw, RefreshToken{UserID: userID, Hash: hash, ExpiresAt: s.now().Add(s.ttl)}, nil
}

func (s *Service) roleFor(email string) identity.Role {
	if s.admins[email] {
		return identity.RoleAdmin
	}
	return identity.RoleUser
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
