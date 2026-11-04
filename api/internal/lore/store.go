package lore

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/gewall/dnd-forge/api/internal/ai"
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
	r, err := s.q.CharacterForLore(ctx, characterID)
	if db.IsNoRows(err) {
		return Subject{}, ErrCharacterNotFound
	}
	if err != nil {
		return Subject{}, fmt.Errorf("subject: %w", err)
	}
	return Subject{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Gender: r.Gender, Appearance: r.Appearance,
		Race: r.RaceName, Subrace: deref(r.SubraceName), Class: r.ClassName, Background: r.BackgroundName,
	}, nil
}

func (s *PostgresStore) DailyLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	own, err := s.q.UserDailyLoreQuota(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("user lore quota: %w", err)
	}
	if own != nil {
		return int(*own), nil
	}
	return quota.DefaultLimit(ctx, s.q, quota.Lore)
}

func (s *PostgresStore) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	return countSince(ctx, s.q, userID, since)
}

func (s *PostgresStore) LoreByID(ctx context.Context, id uuid.UUID) (Lore, error) {
	r, err := s.q.LoreByID(ctx, id)
	if db.IsNoRows(err) {
		return Lore{}, ErrNotFound
	}
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}
	return toLore(r), nil
}

func (s *PostgresStore) LoresOf(ctx context.Context, characterID, userID uuid.UUID) ([]Lore, error) {
	rows, err := s.q.LoresByCharacter(ctx, sqlc.LoresByCharacterParams{CharacterID: &characterID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("lores: %w", err)
	}
	out := make([]Lore, 0, len(rows))
	for _, r := range rows {
		out = append(out, toLore(r))
	}
	return out, nil
}

func (s *PostgresStore) Complete(ctx context.Context, id uuid.UUID, content string, usage ai.Usage) error {
	err := s.q.CompleteLore(ctx, sqlc.CompleteLoreParams{
		ID: id, Content: &content, PromptTokens: tokens(usage.PromptTokens), CompletionTokens: tokens(usage.CompletionTokens),
	})
	if err != nil {
		return fmt.Errorf("complete lore: %w", err)
	}
	return nil
}

func (s *PostgresStore) Fail(ctx context.Context, id uuid.UUID, message string) error {
	if err := s.q.FailLore(ctx, sqlc.FailLoreParams{ID: id, ErrorMessage: &message}); err != nil {
		return fmt.Errorf("fail lore: %w", err)
	}
	return nil
}

func (s *PostgresStore) SetBackstory(ctx context.Context, characterID, userID uuid.UUID, text string) error {
	n, err := s.q.SetCharacterBackstory(ctx, sqlc.SetCharacterBackstoryParams{ID: characterID, UserID: userID, Backstory: text})
	if err != nil {
		return fmt.Errorf("set backstory: %w", err)
	}
	if n == 0 {
		return ErrCharacterNotFound
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
	if err := t.q.LockUserLores(ctx, userID.String()); err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}

func (t *txStore) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	return countSince(ctx, t.q, userID, since)
}

func (t *txStore) CreateLore(ctx context.Context, l NewLore) (Lore, error) {
	r, err := t.q.CreateLore(ctx, sqlc.CreateLoreParams{
		CharacterID: &l.CharacterID, UserID: l.UserID, ProviderID: l.ProviderID, Model: l.Model,
		UserPrompt: l.UserPrompt, FinalPrompt: l.FinalPrompt, Tone: l.Tone, Length: l.Length, Language: l.Language,
	})
	if err != nil {
		return Lore{}, fmt.Errorf("create lore: %w", err)
	}
	return toLore(r), nil
}

func (t *txStore) EnqueueGenerate(ctx context.Context, args GenerateArgs) error {
	if _, err := t.jobs.InsertTx(ctx, t.tx, args, nil); err != nil {
		return fmt.Errorf("enqueue lore: %w", err)
	}
	return nil
}

func countSince(ctx context.Context, q *sqlc.Queries, userID uuid.UUID, since time.Time) (int, error) {
	n, err := q.CountLoresSince(ctx, sqlc.CountLoresSinceParams{UserID: userID, CreatedAt: since})
	if err != nil {
		return 0, fmt.Errorf("count lores: %w", err)
	}
	return int(n), nil
}

func toLore(r sqlc.CharacterLore) Lore {
	return Lore{
		ID: r.ID, CharacterID: r.CharacterID, UserID: r.UserID, ProviderID: r.ProviderID, Model: r.Model,
		UserPrompt: r.UserPrompt, FinalPrompt: r.FinalPrompt, Tone: r.Tone, Length: r.Length, Language: r.Language,
		Status: Status(r.Status), Content: deref(r.Content), ErrorMessage: deref(r.ErrorMessage),
		CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// tokens stores a reported token count; zero means the vendor did not
// report one.
func tokens(n int) *int32 {
	if n <= 0 {
		return nil
	}
	v := int32(min(n, math.MaxInt32)) //nolint:gosec // G115: clamped above
	return &v
}
