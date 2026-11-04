package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/identity"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// Store reads accounts.
type Store interface {
	Accounts(ctx context.Context, f Filter) ([]Account, error)
	Account(ctx context.Context, id uuid.UUID) (Detail, error) // ErrNotFound; usage is left empty
	Status(ctx context.Context, id uuid.UUID) (Status, error)  // ErrNotFound
	InTx(ctx context.Context, fn func(TxStore) error) error
}

// TxStore changes an account inside a transaction.
type TxStore interface {
	Lock(ctx context.Context, id uuid.UUID) (Changes, error) // ErrNotFound
	Update(ctx context.Context, id uuid.UUID, c Changes) error
	RevokeSessions(ctx context.Context, userID uuid.UUID) error
	RecordAudit(ctx context.Context, e audit.Entry) error
}

const (
	defaultPageSize = 50
	maxPageSize     = 100
	// DefaultStatusTTL bounds how long a request may run on a cached
	// account status. Changes made through this service apply at once.
	DefaultStatusTTL = 30 * time.Second
	maxCachedStatus  = 10_000
)

// Deps are the Service's collaborators.
type Deps struct {
	Store    Store
	Portrait quota.Meter
	Lore     quota.Meter
	// StatusTTL defaults to DefaultStatusTTL.
	StatusTTL time.Duration
	Now       func() time.Time
}

// Service manages accounts and checks callers.
type Service struct {
	Deps

	mu     sync.Mutex
	status map[uuid.UUID]cachedStatus
}

type cachedStatus struct {
	status Status
	err    error // only ErrNotFound is cached
	until  time.Time
}

// NewService returns a Service.
func NewService(d Deps) *Service {
	if d.StatusTTL <= 0 {
		d.StatusTTL = DefaultStatusTTL
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{Deps: d, status: map[uuid.UUID]cachedStatus{}}
}

// List returns accounts matching f, newest first.
func (s *Service) List(ctx context.Context, f Filter) ([]Account, error) {
	if f.Limit <= 0 {
		f.Limit = defaultPageSize
	}
	f.Limit = min(f.Limit, maxPageSize)
	f.Query = strings.TrimSpace(f.Query)
	if f.Role != "" && !validRole(f.Role) {
		return nil, ErrInvalidRole
	}
	accounts, err := s.Store.Accounts(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("account: list: %w", err)
	}
	return accounts, nil
}

// Get returns one account with its totals and today's usage.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Detail, error) {
	d, err := s.Store.Account(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("account: %w", err)
	}
	if d.PortraitUsage, err = s.Portrait.Quota(ctx, id); err != nil {
		return Detail{}, fmt.Errorf("account: portrait usage: %w", err)
	}
	if d.LoreUsage, err = s.Lore.Quota(ctx, id); err != nil {
		return Detail{}, fmt.Errorf("account: lore usage: %w", err)
	}
	return d, nil
}

// Update replaces the editable state of an account. Blocking signs the user
// out everywhere. Admins cannot block or demote themselves, which also
// keeps at least one admin around.
func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, c Changes) (Detail, error) {
	if !validRole(c.Role) {
		return Detail{}, ErrInvalidRole
	}
	for _, q := range []*int{c.PortraitQuota, c.LoreQuota} {
		if q != nil && !quota.ValidLimit(*q) {
			return Detail{}, quota.ErrInvalidLimit
		}
	}
	if actor == id && (c.IsBanned || c.Role != identity.RoleAdmin) {
		return Detail{}, ErrSelfChange
	}

	err := s.Store.InTx(ctx, func(tx TxStore) error {
		old, err := tx.Lock(ctx, id)
		if err != nil {
			return err
		}
		diff := changesDiff(old, c)
		if len(diff) == 0 {
			return nil
		}
		if err := tx.Update(ctx, id, c); err != nil {
			return err
		}
		if c.IsBanned && !old.IsBanned {
			if err := tx.RevokeSessions(ctx, id); err != nil {
				return err
			}
		}
		return tx.RecordAudit(ctx, audit.Entry{
			ActorID: actor, Action: audit.ActionUpdate, Entity: entityUser, EntityID: id.String(), Diff: diff,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("account: update: %w", err)
	}
	s.forget(id)
	return s.Get(ctx, id)
}

// Current returns p with the role stored now, or ErrBanned / ErrGone. It is
// called on every authenticated request, so statuses are cached briefly.
func (s *Service) Current(ctx context.Context, p identity.Principal) (identity.Principal, error) {
	st, err := s.lookup(ctx, p.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		return identity.Principal{}, ErrGone
	case err != nil:
		return identity.Principal{}, fmt.Errorf("account: status: %w", err)
	case st.Banned:
		return identity.Principal{}, ErrBanned
	}
	p.Role = st.Role
	return p, nil
}

func (s *Service) lookup(ctx context.Context, id uuid.UUID) (Status, error) {
	now := s.Now()
	s.mu.Lock()
	c, ok := s.status[id]
	s.mu.Unlock()
	if ok && now.Before(c.until) {
		return c.status, c.err
	}

	st, err := s.Store.Status(ctx, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Status{}, err
	}
	s.mu.Lock()
	if len(s.status) >= maxCachedStatus {
		clear(s.status)
	}
	s.status[id] = cachedStatus{status: st, err: err, until: now.Add(s.StatusTTL)}
	s.mu.Unlock()
	return st, err
}

func (s *Service) forget(id uuid.UUID) {
	s.mu.Lock()
	delete(s.status, id)
	s.mu.Unlock()
}

func validRole(r identity.Role) bool { return r == identity.RoleUser || r == identity.RoleAdmin }

// changesDiff lists the changed fields as [old, new] pairs.
func changesDiff(old, c Changes) map[string][2]any {
	d := map[string][2]any{}
	if old.Role != c.Role {
		d["role"] = [2]any{old.Role, c.Role}
	}
	if old.IsBanned != c.IsBanned {
		d["isBanned"] = [2]any{old.IsBanned, c.IsBanned}
	}
	if !sameLimit(old.PortraitQuota, c.PortraitQuota) {
		d["portraitQuota"] = [2]any{old.PortraitQuota, c.PortraitQuota}
	}
	if !sameLimit(old.LoreQuota, c.LoreQuota) {
		d["loreQuota"] = [2]any{old.LoreQuota, c.LoreQuota}
	}
	return d
}

func sameLimit(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
