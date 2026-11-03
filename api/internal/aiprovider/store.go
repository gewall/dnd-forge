package aiprovider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// imageSize is stored for schema compatibility; the size is chosen per
// request from the aspect ratio.
const imageSize = "auto"

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresStore) InTx(ctx context.Context, fn func(Store) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&PostgresStore{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func (s *PostgresStore) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, s.q, e)
}

func (s *PostgresStore) Providers(ctx context.Context) ([]Provider, error) {
	rows, err := s.q.ListAIProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	out := make([]Provider, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProvider(r))
	}
	return out, nil
}

func (s *PostgresStore) ProviderByID(ctx context.Context, id uuid.UUID) (Provider, error) {
	return one(s.q.AIProviderByID(ctx, id))
}

func (s *PostgresStore) ActiveProvider(ctx context.Context, c ai.Capability) (Provider, error) {
	return one(s.q.ActiveAIProvider(ctx, string(c)))
}

func (s *PostgresStore) CreateProvider(ctx context.Context, in Input, sealedKey []byte, last4 string) (Provider, error) {
	return one(s.q.CreateAIProvider(ctx, sqlc.CreateAIProviderParams{
		Type: string(in.Type), Capability: string(in.Capability), Label: in.Label,
		EncryptedApiKey: sealedKey, KeyLast4: last4,
		DefaultModel: in.DefaultModel, ImageSize: imageSize,
	}))
}

func (s *PostgresStore) UpdateProvider(ctx context.Context, id uuid.UUID, in Input, sealedKey []byte, last4 string) (Provider, error) {
	p := sqlc.UpdateAIProviderParams{ID: id, Label: in.Label, DefaultModel: in.DefaultModel, ImageSize: imageSize}
	if sealedKey != nil {
		p.EncryptedApiKey, p.KeyLast4 = sealedKey, &last4
	}
	return one(s.q.UpdateAIProvider(ctx, p))
}

func (s *PostgresStore) Activate(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeactivateAIProviders(ctx, id); err != nil {
		return fmt.Errorf("deactivate providers: %w", err)
	}
	n, err := s.q.ActivateAIProvider(ctx, id)
	if err != nil {
		return fmt.Errorf("activate provider: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteAIProvider(ctx, id)
	switch {
	case db.IsForeignKeyViolation(err):
		return ErrInUse
	case err != nil:
		return fmt.Errorf("delete provider: %w", err)
	case n == 0:
		return ErrNotFound
	}
	return nil
}

func one(r sqlc.AiProvider, err error) (Provider, error) {
	if db.IsNoRows(err) {
		return Provider{}, ErrNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("provider: %w", err)
	}
	return toProvider(r), nil
}

func toProvider(r sqlc.AiProvider) Provider {
	return Provider{
		ID: r.ID, Type: ai.ProviderType(r.Type), Capability: ai.Capability(r.Capability),
		Label: r.Label, KeyLast4: r.KeyLast4,
		DefaultModel: r.DefaultModel, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		sealedKey: r.EncryptedApiKey,
	}
}
