package portrait

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// PostgresAdminStore implements AdminStore with sqlc.
type PostgresAdminStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresAdminStore returns an AdminStore backed by pool.
func NewPostgresAdminStore(pool *pgxpool.Pool) *PostgresAdminStore {
	return &PostgresAdminStore{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresAdminStore) AdminImages(ctx context.Context, f AdminFilter) ([]AdminImage, error) {
	arg := sqlc.AdminImagesParams{
		Hidden: f.Hidden, UserID: f.UserID, Before: f.Before,
		MaxResults: int32(f.Limit), //nolint:gosec // G115: capped by the service
	}
	if f.Status != "" {
		st := sqlc.ImageStatus(f.Status)
		arg.Status = &st
	}
	rows, err := s.q.AdminImages(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("admin images: %w", err)
	}
	out := make([]AdminImage, len(rows))
	for i, r := range rows {
		out[i] = toAdminImage(r)
	}
	return out, nil
}

func (s *PostgresAdminStore) AdminImage(ctx context.Context, id uuid.UUID) (AdminImage, error) {
	r, err := s.q.AdminImageByID(ctx, id)
	if db.IsNoRows(err) {
		return AdminImage{}, ErrNotFound
	}
	if err != nil {
		return AdminImage{}, fmt.Errorf("admin image: %w", err)
	}
	return toAdminImage(sqlc.AdminImagesRow(r)), nil
}

func (s *PostgresAdminStore) InTx(ctx context.Context, fn func(AdminTx) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(adminTx{q: s.q.WithTx(tx)})
	})
}

type adminTx struct{ q *sqlc.Queries }

func (t adminTx) LockPortrait(ctx context.Context, id uuid.UUID) (Portrait, error) {
	r, err := t.q.LockPortrait(ctx, id)
	if db.IsNoRows(err) {
		return Portrait{}, ErrNotFound
	}
	if err != nil {
		return Portrait{}, fmt.Errorf("lock portrait: %w", err)
	}
	return toPortrait(r), nil
}

func (t adminTx) SetHidden(ctx context.Context, id uuid.UUID, hidden bool) error {
	if err := t.q.SetPortraitHidden(ctx, sqlc.SetPortraitHiddenParams{ID: id, IsHidden: hidden}); err != nil {
		return fmt.Errorf("set hidden: %w", err)
	}
	return nil
}

func (t adminTx) ClearFiles(ctx context.Context, id uuid.UUID) error {
	if err := t.q.ClearPortraitFiles(ctx, id); err != nil {
		return fmt.Errorf("clear files: %w", err)
	}
	return nil
}

func (t adminTx) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, t.q, e)
}

func toAdminImage(r sqlc.AdminImagesRow) AdminImage {
	img := AdminImage{
		Portrait: Portrait{
			ID: r.ID, CharacterID: r.CharacterID, UserID: r.UserID, Model: r.Model, UserPrompt: r.UserPrompt,
			Style: r.Style, AspectRatio: r.AspectRatio, Status: Status(r.Status), ErrorMessage: deref(r.ErrorMessage),
			StorageKey: deref(r.StorageKey), ThumbKey: deref(r.ThumbKey), IsHidden: r.IsHidden,
			CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
		},
		UserEmail:     r.UserEmail,
		CharacterName: deref(r.CharacterName),
	}
	if r.CharacterPublic != nil {
		img.CharacterPublic = *r.CharacterPublic
	}
	return img
}
