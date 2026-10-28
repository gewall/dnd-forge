package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

// fakeStore is an in-memory Store for service tests.
type fakeStore struct {
	mu     sync.Mutex
	users  map[uuid.UUID]User
	tokens map[string]RefreshToken // by hash
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[uuid.UUID]User{}, tokens: map[string]RefreshToken{}}
}

func (f *fakeStore) InTx(_ context.Context, fn func(TokenStore) error) error {
	// The fake is not transactional; service tests run sequentially.
	return fn(f)
}

func (f *fakeStore) CreateUser(_ context.Context, u NewUser) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return User{}, ErrEmailTaken
		}
	}
	user := User{
		ID:           uuid.New(),
		Email:        u.Email,
		Name:         u.Name,
		Role:         u.Role,
		PasswordHash: u.PasswordHash,
		GoogleSub:    u.GoogleSub,
		CreatedAt:    time.Now(),
	}
	f.users[user.ID] = user
	return user, nil
}

func (f *fakeStore) UserByID(_ context.Context, id uuid.UUID) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeStore) UserByEmail(_ context.Context, email string) (User, error) {
	return f.find(func(u User) bool { return u.Email == email })
}

func (f *fakeStore) UserByGoogleSub(_ context.Context, sub string) (User, error) {
	return f.find(func(u User) bool { return u.GoogleSub == sub })
}

func (f *fakeStore) find(match func(User) bool) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if match(u) {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (f *fakeStore) update(id uuid.UUID, change func(*User)) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	change(&u)
	f.users[id] = u
	return u, nil
}

func (f *fakeStore) LinkGoogle(_ context.Context, id uuid.UUID, sub string) (User, error) {
	return f.update(id, func(u *User) { u.GoogleSub = sub })
}

func (f *fakeStore) SetRole(_ context.Context, id uuid.UUID, role identity.Role) (User, error) {
	return f.update(id, func(u *User) { u.Role = role })
}

func (f *fakeStore) SetPasswordHash(_ context.Context, id uuid.UUID, hash string) error {
	_, err := f.update(id, func(u *User) { u.PasswordHash = hash })
	return err
}

func (f *fakeStore) setBanned(id uuid.UUID) {
	_, _ = f.update(id, func(u *User) { u.IsBanned = true })
}

func (f *fakeStore) SaveRefreshToken(_ context.Context, t RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t.ID = uuid.New()
	f.tokens[t.Hash] = t
	return nil
}

func (f *fakeStore) RefreshTokenForUpdate(_ context.Context, hash string) (RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tokens[hash]
	if !ok {
		return RefreshToken{}, errRefreshTokenNotFound
	}
	return t, nil
}

func (f *fakeStore) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for h, t := range f.tokens {
		if t.ID == id && t.RevokedAt == nil {
			now := time.Now()
			t.RevokedAt = &now
			f.tokens[h] = t
		}
	}
	return nil
}

func (f *fakeStore) RevokeAllRefreshTokens(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for h, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			now := time.Now()
			t.RevokedAt = &now
			f.tokens[h] = t
		}
	}
	return nil
}

func (f *fakeStore) activeTokens(userID uuid.UUID) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			n++
		}
	}
	return n
}

// fakeGoogle returns a fixed profile or error.
type fakeGoogle struct {
	profile GoogleProfile
	err     error
}

func (g fakeGoogle) Exchange(context.Context, string, string) (GoogleProfile, error) {
	return g.profile, g.err
}
