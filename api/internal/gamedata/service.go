package gamedata

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/slug"
)

// Reader loads game data. includeInactive=false hides disabled rows.
type Reader interface {
	Races(ctx context.Context, includeInactive bool) ([]Race, error)
	RaceBySlug(ctx context.Context, slug string) (Race, error) // with all subraces
	RaceByID(ctx context.Context, id uuid.UUID) (Race, error)
	Classes(ctx context.Context, includeInactive bool) ([]Class, error)
	ClassBySlug(ctx context.Context, slug string) (Class, error)
	ClassByID(ctx context.Context, id uuid.UUID) (Class, error)
	Backgrounds(ctx context.Context, includeInactive bool) ([]Background, error)
	BackgroundBySlug(ctx context.Context, slug string) (Background, error)
	BackgroundByID(ctx context.Context, id uuid.UUID) (Background, error)
}

// Writer changes game data. Implementations map unique violations to
// ErrSlugTaken, foreign key violations to ErrInUse and missing rows to the
// matching ErrXNotFound.
type Writer interface {
	CreateRace(ctx context.Context, in RaceInput) (Race, error)
	UpdateRace(ctx context.Context, id uuid.UUID, in RaceInput) (Race, error)
	DeleteRace(ctx context.Context, id uuid.UUID) error
	CreateSubrace(ctx context.Context, raceID uuid.UUID, in SubraceInput) (Subrace, error)
	UpdateSubrace(ctx context.Context, id uuid.UUID, in SubraceInput) (Subrace, error)
	DeleteSubrace(ctx context.Context, id uuid.UUID) error
	CreateClass(ctx context.Context, in ClassInput) (Class, error)
	UpdateClass(ctx context.Context, id uuid.UUID, in ClassInput) (Class, error)
	DeleteClass(ctx context.Context, id uuid.UUID) error
	CreateBackground(ctx context.Context, in BackgroundInput) (Background, error)
	UpdateBackground(ctx context.Context, id uuid.UUID, in BackgroundInput) (Background, error)
	DeleteBackground(ctx context.Context, id uuid.UUID) error
	RecordAudit(ctx context.Context, e audit.Entry) error
}

// Store is the persistence the service needs. InTx binds a Store to one
// transaction so a change and its audit entry commit together.
type Store interface {
	Reader
	Writer
	InTx(ctx context.Context, fn func(Store) error) error
}

// Service serves game data to the builder and lets admins edit it.
type Service struct {
	store Store
}

// NewService returns a Service backed by store.
func NewService(store Store) *Service { return &Service{store: store} }

// Public reads ----------------------------------------------------------------

// Races lists active races with their active subraces.
func (s *Service) Races(ctx context.Context) ([]Race, error) {
	races, err := s.store.Races(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("gamedata: races: %w", err)
	}
	return races, nil
}

// Race returns an active race by slug with its active subraces.
func (s *Service) Race(ctx context.Context, slug string) (Race, error) {
	r, err := s.store.RaceBySlug(ctx, slug)
	if err != nil {
		return Race{}, fmt.Errorf("gamedata: race: %w", err)
	}
	if !r.IsActive {
		return Race{}, ErrRaceNotFound
	}
	r.Subraces = activeSubraces(r.Subraces)
	return r, nil
}

// Classes lists active classes.
func (s *Service) Classes(ctx context.Context) ([]Class, error) {
	cs, err := s.store.Classes(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("gamedata: classes: %w", err)
	}
	return cs, nil
}

// Class returns an active class by slug.
func (s *Service) Class(ctx context.Context, slug string) (Class, error) {
	c, err := s.store.ClassBySlug(ctx, slug)
	if err != nil {
		return Class{}, fmt.Errorf("gamedata: class: %w", err)
	}
	if !c.IsActive {
		return Class{}, ErrClassNotFound
	}
	return c, nil
}

// Backgrounds lists active backgrounds.
func (s *Service) Backgrounds(ctx context.Context) ([]Background, error) {
	bs, err := s.store.Backgrounds(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("gamedata: backgrounds: %w", err)
	}
	return bs, nil
}

// Background returns an active background by slug.
func (s *Service) Background(ctx context.Context, slug string) (Background, error) {
	b, err := s.store.BackgroundBySlug(ctx, slug)
	if err != nil {
		return Background{}, fmt.Errorf("gamedata: background: %w", err)
	}
	if !b.IsActive {
		return Background{}, ErrBackgroundNotFound
	}
	return b, nil
}

// Admin reads -----------------------------------------------------------------

// AllRaces lists every race and subrace, active or not.
func (s *Service) AllRaces(ctx context.Context) ([]Race, error) { return s.store.Races(ctx, true) }

// RaceByID returns any race with all its subraces.
func (s *Service) RaceByID(ctx context.Context, id uuid.UUID) (Race, error) {
	return s.store.RaceByID(ctx, id)
}

// AllClasses lists every class.
func (s *Service) AllClasses(ctx context.Context) ([]Class, error) { return s.store.Classes(ctx, true) }

// ClassByID returns any class.
func (s *Service) ClassByID(ctx context.Context, id uuid.UUID) (Class, error) {
	return s.store.ClassByID(ctx, id)
}

// AllBackgrounds lists every background.
func (s *Service) AllBackgrounds(ctx context.Context) ([]Background, error) {
	return s.store.Backgrounds(ctx, true)
}

// BackgroundByID returns any background.
func (s *Service) BackgroundByID(ctx context.Context, id uuid.UUID) (Background, error) {
	return s.store.BackgroundByID(ctx, id)
}

// Admin writes ----------------------------------------------------------------

func (s *Service) CreateRace(ctx context.Context, actor uuid.UUID, in RaceInput) (Race, error) {
	if !slug.Valid(in.Slug) {
		return Race{}, ErrInvalidSlug
	}
	var r Race
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityRace, Diff: in},
		func(tx Store) (id uuid.UUID, err error) {
			r, err = tx.CreateRace(ctx, in)
			return r.ID, err
		})
	return r, err
}

func (s *Service) UpdateRace(ctx context.Context, actor, id uuid.UUID, in RaceInput) (Race, error) {
	if !slug.Valid(in.Slug) {
		return Race{}, ErrInvalidSlug
	}
	var r Race
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityRace, Diff: in},
		func(tx Store) (uuid.UUID, error) {
			var err error
			if r, err = tx.UpdateRace(ctx, id, in); err != nil {
				return id, err
			}
			r.Subraces, err = subracesOf(ctx, tx, id)
			return id, err
		})
	return r, err
}

func (s *Service) DeleteRace(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityRace},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteRace(ctx, id) })
}

func (s *Service) CreateSubrace(ctx context.Context, actor, raceID uuid.UUID, in SubraceInput) (Subrace, error) {
	if !slug.Valid(in.Slug) {
		return Subrace{}, ErrInvalidSlug
	}
	var sr Subrace
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entitySubrace, Diff: in},
		func(tx Store) (id uuid.UUID, err error) {
			sr, err = tx.CreateSubrace(ctx, raceID, in)
			return sr.ID, err
		})
	return sr, err
}

func (s *Service) UpdateSubrace(ctx context.Context, actor, id uuid.UUID, in SubraceInput) (Subrace, error) {
	if !slug.Valid(in.Slug) {
		return Subrace{}, ErrInvalidSlug
	}
	var sr Subrace
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entitySubrace, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			sr, err = tx.UpdateSubrace(ctx, id, in)
			return id, err
		})
	return sr, err
}

func (s *Service) DeleteSubrace(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entitySubrace},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteSubrace(ctx, id) })
}

func (s *Service) CreateClass(ctx context.Context, actor uuid.UUID, in ClassInput) (Class, error) {
	if err := validateClass(in); err != nil {
		return Class{}, err
	}
	var c Class
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityClass, Diff: in},
		func(tx Store) (id uuid.UUID, err error) {
			c, err = tx.CreateClass(ctx, in)
			return c.ID, err
		})
	return c, err
}

func (s *Service) UpdateClass(ctx context.Context, actor, id uuid.UUID, in ClassInput) (Class, error) {
	if err := validateClass(in); err != nil {
		return Class{}, err
	}
	var c Class
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityClass, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			c, err = tx.UpdateClass(ctx, id, in)
			return id, err
		})
	return c, err
}

func (s *Service) DeleteClass(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityClass},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteClass(ctx, id) })
}

func (s *Service) CreateBackground(ctx context.Context, actor uuid.UUID, in BackgroundInput) (Background, error) {
	if !slug.Valid(in.Slug) {
		return Background{}, ErrInvalidSlug
	}
	var b Background
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityBackground, Diff: in},
		func(tx Store) (id uuid.UUID, err error) {
			b, err = tx.CreateBackground(ctx, in)
			return b.ID, err
		})
	return b, err
}

func (s *Service) UpdateBackground(ctx context.Context, actor, id uuid.UUID, in BackgroundInput) (Background, error) {
	if !slug.Valid(in.Slug) {
		return Background{}, ErrInvalidSlug
	}
	var b Background
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityBackground, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			b, err = tx.UpdateBackground(ctx, id, in)
			return id, err
		})
	return b, err
}

func (s *Service) DeleteBackground(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityBackground},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteBackground(ctx, id) })
}

// write runs change in a transaction and records e for the id it returns.
func (s *Service) write(ctx context.Context, e audit.Entry, change func(Store) (uuid.UUID, error)) error {
	err := s.store.InTx(ctx, func(tx Store) error {
		id, err := change(tx)
		if err != nil {
			return err
		}
		e.EntityID = id.String()
		return tx.RecordAudit(ctx, e)
	})
	if err != nil {
		return fmt.Errorf("gamedata: %s %s: %w", e.Action, e.Entity, err)
	}
	return nil
}

func validateClass(in ClassInput) error {
	if !slug.Valid(in.Slug) {
		return ErrInvalidSlug
	}
	switch in.HitDie {
	case 6, 8, 10, 12:
		return nil
	}
	return ErrInvalidHitDie
}

func subracesOf(ctx context.Context, r Reader, raceID uuid.UUID) ([]Subrace, error) {
	race, err := r.RaceByID(ctx, raceID)
	return race.Subraces, err
}

func activeSubraces(all []Subrace) []Subrace {
	out := make([]Subrace, 0, len(all))
	for _, sr := range all {
		if sr.IsActive {
			out = append(out, sr)
		}
	}
	return out
}
