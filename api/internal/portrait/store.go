package portrait

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// JobInserter inserts River jobs inside a caller's transaction.
type JobInserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// PostgresStore implements Store with sqlc and River.
type PostgresStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
	jobs JobInserter
}

// NewPostgresStore returns a Store backed by pool; jobs may be an
// insert-only River client.
func NewPostgresStore(pool *pgxpool.Pool, jobs JobInserter) *PostgresStore {
	return &PostgresStore{pool: pool, q: sqlc.New(pool), jobs: jobs}
}

func (s *PostgresStore) InTx(ctx context.Context, fn func(TxStore) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&txStore{q: s.q.WithTx(tx), tx: tx, jobs: s.jobs})
	})
}

func (s *PostgresStore) Subject(ctx context.Context, characterID uuid.UUID) (Subject, error) {
	r, err := s.q.CharacterForPortrait(ctx, characterID)
	if db.IsNoRows(err) {
		return Subject{}, ErrCharacterNotFound
	}
	if err != nil {
		return Subject{}, fmt.Errorf("subject: %w", err)
	}
	return Subject{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Gender: r.Gender, Appearance: r.Appearance,
		Race: r.RaceName, RaceVisual: r.RaceVisual, Subrace: deref(r.SubraceName), SubraceVisual: deref(r.SubraceVisual),
		Class: r.ClassName, ClassVisual: r.ClassVisual, Background: r.BackgroundName, BackgroundVisual: r.BackgroundVisual,
	}, nil
}

func (s *PostgresStore) DailyLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	own, err := s.q.UserDailyQuota(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("user quota: %w", err)
	}
	if own != nil {
		return int(*own), nil
	}
	return quota.DefaultLimit(ctx, s.q, quota.Portrait)
}

func (s *PostgresStore) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	return countSince(ctx, s.q, userID, since)
}

func (s *PostgresStore) PortraitByID(ctx context.Context, id uuid.UUID) (Portrait, error) {
	r, err := s.q.PortraitByID(ctx, id)
	if db.IsNoRows(err) {
		return Portrait{}, ErrNotFound
	}
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}
	return toPortrait(r), nil
}

func (s *PostgresStore) PortraitsOf(ctx context.Context, characterID, userID uuid.UUID) ([]Portrait, error) {
	rows, err := s.q.PortraitsByCharacter(ctx, sqlc.PortraitsByCharacterParams{CharacterID: &characterID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("portraits: %w", err)
	}
	out := make([]Portrait, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPortrait(r))
	}
	return out, nil
}

func (s *PostgresStore) Gallery(ctx context.Context, before *time.Time, limit int) ([]GalleryItem, error) {
	rows, err := s.q.Gallery(ctx, sqlc.GalleryParams{Before: before, MaxResults: db.Int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("gallery: %w", err)
	}
	out := make([]GalleryItem, 0, len(rows))
	for _, r := range rows {
		item := GalleryItem{
			PortraitID: r.ID, CharacterName: r.CharacterName, Race: r.RaceName, Class: r.ClassName,
			Style: r.Style, CreatedAt: r.CreatedAt,
		}
		if r.CharacterID != nil {
			item.CharacterID = *r.CharacterID
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *PostgresStore) Complete(ctx context.Context, id uuid.UUID, storageKey, thumbKey string) error {
	if err := s.q.CompletePortrait(ctx, sqlc.CompletePortraitParams{ID: id, StorageKey: &storageKey, ThumbKey: &thumbKey}); err != nil {
		return fmt.Errorf("complete portrait: %w", err)
	}
	return nil
}

func (s *PostgresStore) Fail(ctx context.Context, id uuid.UUID, message string) error {
	if err := s.q.FailPortrait(ctx, sqlc.FailPortraitParams{ID: id, ErrorMessage: &message}); err != nil {
		return fmt.Errorf("fail portrait: %w", err)
	}
	return nil
}

// txStore is bound to the request transaction.
type txStore struct {
	q    *sqlc.Queries
	tx   pgx.Tx
	jobs JobInserter
}

func (t *txStore) LockUser(ctx context.Context, userID uuid.UUID) error {
	if err := t.q.LockUserPortraits(ctx, userID.String()); err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}

func (t *txStore) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	return countSince(ctx, t.q, userID, since)
}

func (t *txStore) CreatePortrait(ctx context.Context, p NewPortrait) (Portrait, error) {
	r, err := t.q.CreatePortrait(ctx, sqlc.CreatePortraitParams{
		CharacterID: &p.CharacterID, UserID: p.UserID, ProviderID: p.ProviderID, Model: p.Model,
		UserPrompt: p.UserPrompt, FinalPrompt: p.FinalPrompt, Style: p.Style, AspectRatio: p.AspectRatio,
	})
	if err != nil {
		return Portrait{}, fmt.Errorf("create portrait: %w", err)
	}
	return toPortrait(r), nil
}

func (t *txStore) EnqueueGenerate(ctx context.Context, args GenerateArgs) error {
	if _, err := t.jobs.InsertTx(ctx, t.tx, args, nil); err != nil {
		return fmt.Errorf("enqueue portrait: %w", err)
	}
	return nil
}

func countSince(ctx context.Context, q *sqlc.Queries, userID uuid.UUID, since time.Time) (int, error) {
	n, err := q.CountPortraitsSince(ctx, sqlc.CountPortraitsSinceParams{UserID: userID, CreatedAt: since})
	if err != nil {
		return 0, fmt.Errorf("count portraits: %w", err)
	}
	return int(n), nil
}

func toPortrait(r sqlc.GeneratedImage) Portrait {
	return Portrait{
		ID: r.ID, CharacterID: r.CharacterID, UserID: r.UserID, ProviderID: r.ProviderID, Model: r.Model,
		UserPrompt: r.UserPrompt, FinalPrompt: r.FinalPrompt, Style: r.Style, AspectRatio: r.AspectRatio,
		Status: Status(r.Status), ErrorMessage: deref(r.ErrorMessage), StorageKey: deref(r.StorageKey),
		ThumbKey: deref(r.ThumbKey), IsHidden: r.IsHidden, CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
