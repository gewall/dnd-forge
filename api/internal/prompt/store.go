package prompt

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
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

func (s *PostgresStore) InTx(ctx context.Context, fn func(Store) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&PostgresStore{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func (s *PostgresStore) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, s.q, e)
}

func (s *PostgresStore) Templates(ctx context.Context) ([]Template, error) {
	rows, err := s.q.ListPromptTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	out := make([]Template, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTemplate(r))
	}
	return out, nil
}

func (s *PostgresStore) TemplateByID(ctx context.Context, id uuid.UUID) (Template, error) {
	return one(s.q.PromptTemplateByID(ctx, id))
}

func (s *PostgresStore) ActiveTemplate(ctx context.Context, k Kind) (Template, error) {
	t, err := one(s.q.ActivePromptTemplate(ctx, string(k)))
	if errors.Is(err, ErrNotFound) {
		return Template{}, ErrNoActive
	}
	return t, err
}

func (s *PostgresStore) CreateTemplate(ctx context.Context, in Input, createdBy *uuid.UUID) (Template, error) {
	return one(s.q.CreatePromptTemplate(ctx, sqlc.CreatePromptTemplateParams{
		Kind: string(in.Kind), Name: in.Name, Template: in.Template, NegativePrompt: in.NegativePrompt,
		CreatedBy: createdBy,
	}))
}

func (s *PostgresStore) Activate(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeactivatePromptTemplates(ctx, id); err != nil {
		return fmt.Errorf("deactivate templates: %w", err)
	}
	n, err := s.q.ActivatePromptTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("activate template: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeletePromptTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func one(r sqlc.PromptTemplate, err error) (Template, error) {
	if db.IsNoRows(err) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("template: %w", err)
	}
	return toTemplate(r), nil
}

func toTemplate(r sqlc.PromptTemplate) Template {
	return Template{
		ID: r.ID, Kind: Kind(r.Kind), Name: r.Name, Template: r.Template, NegativePrompt: r.NegativePrompt,
		Version: int(r.Version), IsActive: r.IsActive, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
}
