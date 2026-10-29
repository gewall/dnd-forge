package gamedata

import (
	"context"
	"encoding/json"
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

// Races -----------------------------------------------------------------------

func (s *PostgresStore) Races(ctx context.Context, includeInactive bool) ([]Race, error) {
	rows, err := s.q.ListRaces(ctx, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list races: %w", err)
	}
	subs, err := s.q.ListSubraces(ctx, sqlc.ListSubracesParams{IncludeInactive: includeInactive})
	if err != nil {
		return nil, fmt.Errorf("list subraces: %w", err)
	}
	byRace := make(map[uuid.UUID][]Subrace)
	for _, sr := range subs {
		byRace[sr.RaceID] = append(byRace[sr.RaceID], toSubrace(sr))
	}
	races := make([]Race, 0, len(rows))
	for _, r := range rows {
		race := toRace(r)
		race.Subraces = nonNil(byRace[r.ID])
		races = append(races, race)
	}
	return races, nil
}

func (s *PostgresStore) RaceBySlug(ctx context.Context, slug string) (Race, error) {
	row, err := s.q.RaceBySlug(ctx, slug)
	return s.withSubraces(ctx, row, err)
}

func (s *PostgresStore) RaceByID(ctx context.Context, id uuid.UUID) (Race, error) {
	row, err := s.q.RaceByID(ctx, id)
	return s.withSubraces(ctx, row, err)
}

func (s *PostgresStore) withSubraces(ctx context.Context, row sqlc.Race, err error) (Race, error) {
	if err != nil {
		return Race{}, mapErr(err, ErrRaceNotFound)
	}
	subs, err := s.q.ListSubraces(ctx, sqlc.ListSubracesParams{IncludeInactive: true, RaceID: &row.ID})
	if err != nil {
		return Race{}, fmt.Errorf("list subraces: %w", err)
	}
	r := toRace(row)
	r.Subraces = make([]Subrace, 0, len(subs))
	for _, sr := range subs {
		r.Subraces = append(r.Subraces, toSubrace(sr))
	}
	return r, nil
}

func (s *PostgresStore) CreateRace(ctx context.Context, in RaceInput) (Race, error) {
	row, err := s.q.CreateRace(ctx, sqlc.CreateRaceParams{
		Slug: in.Slug, Name: in.Name, Description: in.Description, Traits: mustJSON(in.Traits),
		VisualPrompt: in.VisualPrompt, ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Race{}, mapErr(err, ErrRaceNotFound)
	}
	r := toRace(row)
	r.Subraces = []Subrace{}
	return r, nil
}

func (s *PostgresStore) UpdateRace(ctx context.Context, id uuid.UUID, in RaceInput) (Race, error) {
	row, err := s.q.UpdateRace(ctx, sqlc.UpdateRaceParams{
		ID: id, Slug: in.Slug, Name: in.Name, Description: in.Description, Traits: mustJSON(in.Traits),
		VisualPrompt: in.VisualPrompt, ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Race{}, mapErr(err, ErrRaceNotFound)
	}
	return toRace(row), nil
}

func (s *PostgresStore) DeleteRace(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteRace(ctx, id)
	return deleted(n, err, ErrRaceNotFound)
}

func (s *PostgresStore) CreateSubrace(ctx context.Context, raceID uuid.UUID, in SubraceInput) (Subrace, error) {
	row, err := s.q.CreateSubrace(ctx, sqlc.CreateSubraceParams{
		RaceID: raceID, Slug: in.Slug, Name: in.Name, Description: in.Description,
		Traits: mustJSON(in.Traits), VisualPrompt: in.VisualPrompt, IsActive: in.IsActive,
	})
	if err != nil {
		// The only foreign key here is race_id.
		if db.IsForeignKeyViolation(err) {
			return Subrace{}, ErrRaceNotFound
		}
		return Subrace{}, mapErr(err, ErrSubraceNotFound)
	}
	return toSubrace(row), nil
}

func (s *PostgresStore) UpdateSubrace(ctx context.Context, id uuid.UUID, in SubraceInput) (Subrace, error) {
	row, err := s.q.UpdateSubrace(ctx, sqlc.UpdateSubraceParams{
		ID: id, Slug: in.Slug, Name: in.Name, Description: in.Description,
		Traits: mustJSON(in.Traits), VisualPrompt: in.VisualPrompt, IsActive: in.IsActive,
	})
	if err != nil {
		return Subrace{}, mapErr(err, ErrSubraceNotFound)
	}
	return toSubrace(row), nil
}

func (s *PostgresStore) DeleteSubrace(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteSubrace(ctx, id)
	return deleted(n, err, ErrSubraceNotFound)
}

// Classes ---------------------------------------------------------------------

func (s *PostgresStore) Classes(ctx context.Context, includeInactive bool) ([]Class, error) {
	rows, err := s.q.ListClasses(ctx, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	out := make([]Class, 0, len(rows))
	for _, r := range rows {
		out = append(out, toClass(r))
	}
	return out, nil
}

func (s *PostgresStore) ClassBySlug(ctx context.Context, slug string) (Class, error) {
	row, err := s.q.ClassBySlug(ctx, slug)
	if err != nil {
		return Class{}, mapErr(err, ErrClassNotFound)
	}
	return toClass(row), nil
}

func (s *PostgresStore) ClassByID(ctx context.Context, id uuid.UUID) (Class, error) {
	row, err := s.q.ClassByID(ctx, id)
	if err != nil {
		return Class{}, mapErr(err, ErrClassNotFound)
	}
	return toClass(row), nil
}

func (s *PostgresStore) CreateClass(ctx context.Context, in ClassInput) (Class, error) {
	row, err := s.q.CreateClass(ctx, sqlc.CreateClassParams{
		Slug: in.Slug, Name: in.Name, HitDie: db.Int32(in.HitDie), PrimaryAbility: in.PrimaryAbility,
		Description: in.Description, Features: mustJSON(in.Features), VisualPrompt: in.VisualPrompt,
		ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Class{}, mapErr(err, ErrClassNotFound)
	}
	return toClass(row), nil
}

func (s *PostgresStore) UpdateClass(ctx context.Context, id uuid.UUID, in ClassInput) (Class, error) {
	row, err := s.q.UpdateClass(ctx, sqlc.UpdateClassParams{
		ID: id, Slug: in.Slug, Name: in.Name, HitDie: db.Int32(in.HitDie), PrimaryAbility: in.PrimaryAbility,
		Description: in.Description, Features: mustJSON(in.Features), VisualPrompt: in.VisualPrompt,
		ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Class{}, mapErr(err, ErrClassNotFound)
	}
	return toClass(row), nil
}

func (s *PostgresStore) DeleteClass(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteClass(ctx, id)
	return deleted(n, err, ErrClassNotFound)
}

// Backgrounds -----------------------------------------------------------------

func (s *PostgresStore) Backgrounds(ctx context.Context, includeInactive bool) ([]Background, error) {
	rows, err := s.q.ListBackgrounds(ctx, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list backgrounds: %w", err)
	}
	out := make([]Background, 0, len(rows))
	for _, r := range rows {
		out = append(out, toBackground(r))
	}
	return out, nil
}

func (s *PostgresStore) BackgroundBySlug(ctx context.Context, slug string) (Background, error) {
	row, err := s.q.BackgroundBySlug(ctx, slug)
	if err != nil {
		return Background{}, mapErr(err, ErrBackgroundNotFound)
	}
	return toBackground(row), nil
}

func (s *PostgresStore) BackgroundByID(ctx context.Context, id uuid.UUID) (Background, error) {
	row, err := s.q.BackgroundByID(ctx, id)
	if err != nil {
		return Background{}, mapErr(err, ErrBackgroundNotFound)
	}
	return toBackground(row), nil
}

func (s *PostgresStore) CreateBackground(ctx context.Context, in BackgroundInput) (Background, error) {
	row, err := s.q.CreateBackground(ctx, sqlc.CreateBackgroundParams{
		Slug: in.Slug, Name: in.Name, Description: in.Description, SkillProficiencies: mustJSON(in.SkillProficiencies),
		VisualPrompt: in.VisualPrompt, ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Background{}, mapErr(err, ErrBackgroundNotFound)
	}
	return toBackground(row), nil
}

func (s *PostgresStore) UpdateBackground(ctx context.Context, id uuid.UUID, in BackgroundInput) (Background, error) {
	row, err := s.q.UpdateBackground(ctx, sqlc.UpdateBackgroundParams{
		ID: id, Slug: in.Slug, Name: in.Name, Description: in.Description, SkillProficiencies: mustJSON(in.SkillProficiencies),
		VisualPrompt: in.VisualPrompt, ImageKey: optional(in.ImageKey), SortOrder: db.Int32(in.SortOrder), IsActive: in.IsActive,
	})
	if err != nil {
		return Background{}, mapErr(err, ErrBackgroundNotFound)
	}
	return toBackground(row), nil
}

func (s *PostgresStore) DeleteBackground(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteBackground(ctx, id)
	return deleted(n, err, ErrBackgroundNotFound)
}

// Mapping ---------------------------------------------------------------------

func mapErr(err, notFound error) error {
	switch {
	case db.IsNoRows(err):
		return notFound
	case db.IsUniqueViolation(err):
		return ErrSlugTaken
	case db.IsForeignKeyViolation(err):
		return ErrInUse
	}
	return err
}

// deleted turns the result of a DELETE ... :execrows into an error.
func deleted(n int64, err, notFound error) error {
	switch {
	case err != nil:
		return mapErr(err, notFound)
	case n == 0:
		return notFound
	}
	return nil
}

func toRace(r sqlc.Race) Race {
	return Race{ID: r.ID, RaceInput: RaceInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, Traits: traits(r.Traits),
		VisualPrompt: r.VisualPrompt, ImageKey: deref(r.ImageKey), SortOrder: int(r.SortOrder), IsActive: r.IsActive,
	}}
}

func toSubrace(r sqlc.Subrace) Subrace {
	return Subrace{ID: r.ID, RaceID: r.RaceID, SubraceInput: SubraceInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, Traits: traits(r.Traits),
		VisualPrompt: r.VisualPrompt, IsActive: r.IsActive,
	}}
}

func toClass(r sqlc.Class) Class {
	return Class{ID: r.ID, ClassInput: ClassInput{
		Slug: r.Slug, Name: r.Name, HitDie: int(r.HitDie), PrimaryAbility: r.PrimaryAbility,
		Description: r.Description, Features: traits(r.Features), VisualPrompt: r.VisualPrompt,
		ImageKey: deref(r.ImageKey), SortOrder: int(r.SortOrder), IsActive: r.IsActive,
	}}
}

func toBackground(r sqlc.Background) Background {
	skills := []string{}
	_ = json.Unmarshal(r.SkillProficiencies, &skills)
	return Background{ID: r.ID, BackgroundInput: BackgroundInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, SkillProficiencies: skills,
		VisualPrompt: r.VisualPrompt, ImageKey: deref(r.ImageKey), SortOrder: int(r.SortOrder), IsActive: r.IsActive,
	}}
}

// traits decodes a jsonb trait list; the column defaults to '[]', so a
// decoding failure only happens on hand-edited rows and yields no traits.
func traits(b []byte) []Trait {
	ts := []Trait{}
	_ = json.Unmarshal(b, &ts)
	return ts
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte("[]")
	}
	return b
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
