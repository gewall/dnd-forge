package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/identity"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

type memStore struct {
	accounts     map[uuid.UUID]*Changes
	statusCalls  int
	statusErr    error
	revoked      []uuid.UUID
	audits       []audit.Entry
	lastFilter   Filter
	listedResult []Account
}

func newMemStore(ids ...uuid.UUID) *memStore {
	m := &memStore{accounts: map[uuid.UUID]*Changes{}}
	for _, id := range ids {
		m.accounts[id] = &Changes{Role: identity.RoleUser}
	}
	return m
}

func (m *memStore) Accounts(_ context.Context, f Filter) ([]Account, error) {
	m.lastFilter = f
	return m.listedResult, nil
}

func (m *memStore) Account(_ context.Context, id uuid.UUID) (Detail, error) {
	c, ok := m.accounts[id]
	if !ok {
		return Detail{}, ErrNotFound
	}
	return Detail{Account: Account{ID: id, Role: c.Role, IsBanned: c.IsBanned, PortraitQuota: c.PortraitQuota, LoreQuota: c.LoreQuota}}, nil
}

func (m *memStore) Status(_ context.Context, id uuid.UUID) (Status, error) {
	m.statusCalls++
	if m.statusErr != nil {
		return Status{}, m.statusErr
	}
	c, ok := m.accounts[id]
	if !ok {
		return Status{}, ErrNotFound
	}
	return Status{Role: c.Role, Banned: c.IsBanned}, nil
}

func (m *memStore) InTx(_ context.Context, fn func(TxStore) error) error { return fn(m) }

func (m *memStore) Lock(_ context.Context, id uuid.UUID) (Changes, error) {
	c, ok := m.accounts[id]
	if !ok {
		return Changes{}, ErrNotFound
	}
	return *c, nil
}

func (m *memStore) Update(_ context.Context, id uuid.UUID, c Changes) error {
	*m.accounts[id] = c
	return nil
}

func (m *memStore) RevokeSessions(_ context.Context, id uuid.UUID) error {
	m.revoked = append(m.revoked, id)
	return nil
}

func (m *memStore) RecordAudit(_ context.Context, e audit.Entry) error {
	m.audits = append(m.audits, e)
	return nil
}

type meter struct{ used int }

func (m meter) Quota(context.Context, uuid.UUID) (quota.Usage, error) {
	return quota.NewUsage(5, m.used, time.Time{}), nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newService(store *memStore) (*Service, *clock) {
	c := &clock{t: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	return NewService(Deps{Store: store, Portrait: meter{2}, Lore: meter{7}, Now: c.now}), c
}

func ptr(n int) *int { return &n }

func TestUpdateValidation(t *testing.T) {
	admin, user := uuid.New(), uuid.New()
	store := newMemStore(admin, user)
	store.accounts[admin].Role = identity.RoleAdmin
	svc, _ := newService(store)
	ctx := context.Background()

	tests := []struct {
		name string
		id   uuid.UUID
		c    Changes
		want error
	}{
		{"unknown role", user, Changes{Role: "OWNER"}, ErrInvalidRole},
		{"negative quota", user, Changes{Role: identity.RoleUser, PortraitQuota: ptr(-1)}, quota.ErrInvalidLimit},
		{"quota above max", user, Changes{Role: identity.RoleUser, LoreQuota: ptr(quota.MaxLimit + 1)}, quota.ErrInvalidLimit},
		{"self block", admin, Changes{Role: identity.RoleAdmin, IsBanned: true}, ErrSelfChange},
		{"self demote", admin, Changes{Role: identity.RoleUser}, ErrSelfChange},
		{"unknown user", uuid.New(), Changes{Role: identity.RoleUser}, ErrNotFound},
	}
	for _, tt := range tests {
		if _, err := svc.Update(ctx, admin, tt.id, tt.c); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
	if len(store.audits) != 0 {
		t.Errorf("rejected updates wrote %d audit entries", len(store.audits))
	}

	// An admin may still change their own limits.
	d, err := svc.Update(ctx, admin, admin, Changes{Role: identity.RoleAdmin, LoreQuota: ptr(50)})
	if err != nil || *d.LoreQuota != 50 {
		t.Errorf("own quota = %+v, %v", d.LoreQuota, err)
	}
}

func TestUpdateBlocksAndAudits(t *testing.T) {
	admin, user := uuid.New(), uuid.New()
	store := newMemStore(admin, user)
	svc, _ := newService(store)
	ctx := context.Background()

	d, err := svc.Update(ctx, admin, user, Changes{Role: identity.RoleUser, IsBanned: true, PortraitQuota: ptr(0)})
	if err != nil || !d.IsBanned || *d.PortraitQuota != 0 {
		t.Fatalf("block = %+v, %v", d, err)
	}
	if d.PortraitUsage.Used != 2 || d.LoreUsage.Used != 7 {
		t.Errorf("usage = %+v / %+v", d.PortraitUsage, d.LoreUsage)
	}
	if len(store.revoked) != 1 || store.revoked[0] != user {
		t.Errorf("revoked = %v, want the blocked user", store.revoked)
	}
	if len(store.audits) != 1 || store.audits[0].Entity != entityUser || store.audits[0].ActorID != admin {
		t.Fatalf("audits = %+v", store.audits)
	}
	diff := store.audits[0].Diff.(map[string][2]any)
	if _, ok := diff["isBanned"]; !ok || len(diff) != 2 {
		t.Errorf("diff = %v, want isBanned and portraitQuota", diff)
	}

	// Saving the same state again changes nothing and logs nothing.
	if _, err := svc.Update(ctx, admin, user, Changes{Role: identity.RoleUser, IsBanned: true, PortraitQuota: ptr(0)}); err != nil {
		t.Fatal(err)
	}
	// Unblocking does not revoke again.
	if _, err := svc.Update(ctx, admin, user, Changes{Role: identity.RoleUser}); err != nil {
		t.Fatal(err)
	}
	if len(store.audits) != 2 || len(store.revoked) != 1 {
		t.Errorf("after no-op and unblock: %d audits, %d revokes; want 2, 1", len(store.audits), len(store.revoked))
	}
}

func TestCurrent(t *testing.T) {
	admin, user := uuid.New(), uuid.New()
	store := newMemStore(admin, user)
	store.accounts[admin].Role = identity.RoleAdmin
	svc, clk := newService(store)
	ctx := context.Background()
	asUser := identity.Principal{UserID: user, Role: identity.RoleUser}

	if p, err := svc.Current(ctx, asUser); err != nil || p.Role != identity.RoleUser {
		t.Fatalf("Current = %+v, %v", p, err)
	}
	// A change outside this service shows up once the cache expires.
	store.accounts[user].Role = identity.RoleAdmin
	if p, _ := svc.Current(ctx, asUser); p.Role != identity.RoleUser || store.statusCalls != 1 {
		t.Errorf("cached role = %s after %d lookups", p.Role, store.statusCalls)
	}
	clk.t = clk.t.Add(DefaultStatusTTL)
	if p, _ := svc.Current(ctx, asUser); p.Role != identity.RoleAdmin {
		t.Errorf("role after TTL = %s, want ADMIN", p.Role)
	}

	// A change through the service applies at once.
	if _, err := svc.Update(ctx, admin, user, Changes{Role: identity.RoleAdmin, IsBanned: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Current(ctx, asUser); !errors.Is(err, ErrBanned) {
		t.Errorf("blocked user err = %v", err)
	}

	if _, err := svc.Current(ctx, identity.Principal{UserID: uuid.New()}); !errors.Is(err, ErrGone) {
		t.Errorf("deleted user err = %v, want ErrGone", err)
	}

	// Lookup failures are not cached.
	store.statusErr = errors.New("db down")
	ghost := identity.Principal{UserID: uuid.New()}
	calls := store.statusCalls
	for range 2 {
		if _, err := svc.Current(ctx, ghost); err == nil || errors.Is(err, ErrGone) {
			t.Errorf("db error = %v", err)
		}
	}
	if store.statusCalls != calls+2 {
		t.Errorf("failed lookups were cached")
	}
}

func TestList(t *testing.T) {
	store := newMemStore()
	svc, _ := newService(store)
	ctx := context.Background()
	if _, err := svc.List(ctx, Filter{Query: "  lia ", Limit: 500}); err != nil {
		t.Fatal(err)
	}
	if store.lastFilter.Limit != maxPageSize || store.lastFilter.Query != "lia" {
		t.Errorf("filter = %+v", store.lastFilter)
	}
	if _, err := svc.List(ctx, Filter{}); err != nil || store.lastFilter.Limit != defaultPageSize {
		t.Errorf("default limit = %d, %v", store.lastFilter.Limit, err)
	}
	if _, err := svc.List(ctx, Filter{Role: "GOD"}); !errors.Is(err, ErrInvalidRole) {
		t.Errorf("bad role err = %v", err)
	}
}
