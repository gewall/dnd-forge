package character

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	q *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{q: sqlc.New(pool)}
}

func (s *PostgresStore) CharactersByUser(ctx context.Context, userID uuid.UUID) ([]Character, error) {
	rows, err := s.q.ListCharactersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	out := make([]Character, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCharacter(sqlc.CharacterByIDRow(r)))
	}
	return out, nil
}

func (s *PostgresStore) CharacterByID(ctx context.Context, id uuid.UUID) (Character, error) {
	row, err := s.q.CharacterByID(ctx, id)
	if db.IsNoRows(err) {
		return Character{}, ErrNotFound
	}
	if err != nil {
		return Character{}, fmt.Errorf("character: %w", err)
	}
	return toCharacter(row), nil
}

func (s *PostgresStore) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := s.q.CountCharactersByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count characters: %w", err)
	}
	return int(n), nil
}

func (s *PostgresStore) CreateCharacter(ctx context.Context, userID uuid.UUID, d Draft) (uuid.UUID, error) {
	id, err := s.q.CreateCharacter(ctx, sqlc.CreateCharacterParams{
		UserID: userID, Name: d.Name, Gender: d.Gender, RaceID: d.RaceID, SubraceID: d.SubraceID,
		ClassID: d.ClassID, BackgroundID: d.BackgroundID, Appearance: d.Appearance, IsPublic: d.IsPublic,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create character: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) UpdateCharacter(ctx context.Context, id, userID uuid.UUID, d Draft) error {
	n, err := s.q.UpdateCharacter(ctx, sqlc.UpdateCharacterParams{
		ID: id, UserID: userID, Name: d.Name, Gender: d.Gender, RaceID: d.RaceID, SubraceID: d.SubraceID,
		ClassID: d.ClassID, BackgroundID: d.BackgroundID, Appearance: d.Appearance, IsPublic: d.IsPublic,
	})
	if err != nil {
		return fmt.Errorf("update character: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteCharacter(ctx context.Context, id, userID uuid.UUID) error {
	n, err := s.q.DeleteCharacter(ctx, sqlc.DeleteCharacterParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("delete character: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func toCharacter(r sqlc.CharacterByIDRow) Character {
	c := Character{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Gender: r.Gender, Appearance: r.Appearance, IsPublic: r.IsPublic,
		Race:       Ref{ID: r.RaceID, Slug: r.RaceSlug, Name: r.RaceName},
		Class:      Ref{ID: r.ClassID, Slug: r.ClassSlug, Name: r.ClassName},
		Background: Ref{ID: r.BackgroundID, Slug: r.BackgroundSlug, Name: r.BackgroundName},
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if r.SubraceID != nil {
		c.Subrace = &Ref{ID: *r.SubraceID, Slug: deref(r.SubraceSlug), Name: deref(r.SubraceName)}
	}
	return c
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
