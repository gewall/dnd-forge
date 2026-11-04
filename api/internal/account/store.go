package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresStore) InTx(ctx context.Context, fn func(TxStore) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(txStore{q: s.q.WithTx(tx)})
	})
}

func (s *PostgresStore) Accounts(ctx context.Context, f Filter) ([]Account, error) {
	arg := sqlc.ListAccountsParams{Banned: f.Banned, Before: f.Before, MaxResults: int32(f.Limit)} //nolint:gosec // G115: limit is capped by the service
	if f.Query != "" {
		p := "%" + likeEscaper.Replace(f.Query) + "%"
		arg.Pattern = &p
	}
	if f.Role != "" {
		r := sqlc.UserRole(f.Role)
		arg.Role = &r
	}
	rows, err := s.q.ListAccounts(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("accounts: %w", err)
	}
	out := make([]Account, len(rows))
	for i, r := range rows {
		out[i] = Account{
			ID: r.ID, Email: r.Email, Name: r.Name, Role: identity.Role(r.Role), IsBanned: r.IsBanned,
			PortraitQuota: limit(r.DailyQuota), LoreQuota: limit(r.DailyLoreQuota),
			HasPassword: r.HasPassword, HasGoogle: r.HasGoogle, Characters: int(r.Characters),
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return out, nil
}

func (s *PostgresStore) Account(ctx context.Context, id uuid.UUID) (Detail, error) {
	r, err := s.q.AccountByID(ctx, id)
	if db.IsNoRows(err) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("account: %w", err)
	}
	return Detail{
		Account: Account{
			ID: r.ID, Email: r.Email, Name: r.Name, Role: identity.Role(r.Role), IsBanned: r.IsBanned,
			PortraitQuota: limit(r.DailyQuota), LoreQuota: limit(r.DailyLoreQuota),
			HasPassword: r.HasPassword, HasGoogle: r.HasGoogle, Characters: int(r.Characters),
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		},
		Portraits: int(r.Portraits),
		Lores:     int(r.Lores),
	}, nil
}

func (s *PostgresStore) Status(ctx context.Context, id uuid.UUID) (Status, error) {
	r, err := s.q.AccountStatus(ctx, id)
	if db.IsNoRows(err) {
		return Status{}, ErrNotFound
	}
	if err != nil {
		return Status{}, fmt.Errorf("account status: %w", err)
	}
	return Status{Role: identity.Role(r.Role), Banned: r.IsBanned}, nil
}

type txStore struct{ q *sqlc.Queries }

func (s txStore) Lock(ctx context.Context, id uuid.UUID) (Changes, error) {
	r, err := s.q.LockAccount(ctx, id)
	if db.IsNoRows(err) {
		return Changes{}, ErrNotFound
	}
	if err != nil {
		return Changes{}, fmt.Errorf("lock account: %w", err)
	}
	return Changes{
		Role: identity.Role(r.Role), IsBanned: r.IsBanned,
		PortraitQuota: limit(r.DailyQuota), LoreQuota: limit(r.DailyLoreQuota),
	}, nil
}

func (s txStore) Update(ctx context.Context, id uuid.UUID, c Changes) error {
	err := s.q.UpdateAccount(ctx, sqlc.UpdateAccountParams{
		ID: id, Role: sqlc.UserRole(c.Role), IsBanned: c.IsBanned,
		DailyQuota: limit32(c.PortraitQuota), DailyLoreQuota: limit32(c.LoreQuota),
	})
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}

func (s txStore) RevokeSessions(ctx context.Context, userID uuid.UUID) error {
	if err := s.q.RevokeUserRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

func (s txStore) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, s.q, e)
}

// likeEscaper makes user text match literally inside an ILIKE pattern.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func limit(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func limit32(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n) //nolint:gosec // G115: limits are validated to 0..quota.MaxLimit
	return &v
}
