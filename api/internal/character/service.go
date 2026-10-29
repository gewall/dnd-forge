package character

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Store persists characters. Writes are scoped to the owner: UpdateCharacter
// and DeleteCharacter return ErrNotFound when id does not belong to userID.
type Store interface {
	CharactersByUser(ctx context.Context, userID uuid.UUID) ([]Character, error)
	CharacterByID(ctx context.Context, id uuid.UUID) (Character, error) // ErrNotFound
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CreateCharacter(ctx context.Context, userID uuid.UUID, d Draft) (uuid.UUID, error)
	UpdateCharacter(ctx context.Context, id, userID uuid.UUID, d Draft) error
	DeleteCharacter(ctx context.Context, id, userID uuid.UUID) error
}

// Option is a selectable game data entry.
type Option struct {
	ID     uuid.UUID
	Active bool
}

// RaceOption is a race and its subraces.
type RaceOption struct {
	Option
	Subraces []Option
}

// Catalog looks up game data choices. Lookups of unknown ids return
// ErrInvalidRace, ErrInvalidClass or ErrInvalidBkg.
type Catalog interface {
	Race(ctx context.Context, id uuid.UUID) (RaceOption, error)
	Class(ctx context.Context, id uuid.UUID) (Option, error)
	Background(ctx context.Context, id uuid.UUID) (Option, error)
}

// Service manages a user's characters.
type Service struct {
	store   Store
	catalog Catalog
}

// NewService returns a Service.
func NewService(store Store, catalog Catalog) *Service {
	return &Service{store: store, catalog: catalog}
}

// List returns the user's characters, most recently edited first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Character, error) {
	cs, err := s.store.CharactersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("character: list: %w", err)
	}
	return cs, nil
}

// Get returns a character owned by userID. Characters of other users are
// reported as not found so their existence does not leak.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Character, error) {
	c, err := s.store.CharacterByID(ctx, id)
	if err != nil {
		return Character{}, fmt.Errorf("character: get: %w", err)
	}
	if c.UserID != userID {
		return Character{}, ErrNotFound
	}
	return c, nil
}

// Create validates d and saves it as a new character of userID.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, d Draft) (Character, error) {
	d = normalize(d)
	if err := s.validate(ctx, d); err != nil {
		return Character{}, err
	}
	n, err := s.store.CountByUser(ctx, userID)
	if err != nil {
		return Character{}, fmt.Errorf("character: create: %w", err)
	}
	if n >= MaxPerUser {
		return Character{}, ErrLimitReached
	}
	id, err := s.store.CreateCharacter(ctx, userID, d)
	if err != nil {
		return Character{}, fmt.Errorf("character: create: %w", err)
	}
	return s.store.CharacterByID(ctx, id)
}

// Update replaces a character owned by userID.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, d Draft) (Character, error) {
	d = normalize(d)
	if err := s.validate(ctx, d); err != nil {
		return Character{}, err
	}
	if err := s.store.UpdateCharacter(ctx, id, userID, d); err != nil {
		return Character{}, fmt.Errorf("character: update: %w", err)
	}
	return s.store.CharacterByID(ctx, id)
}

// Delete removes a character owned by userID.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.store.DeleteCharacter(ctx, id, userID); err != nil {
		return fmt.Errorf("character: delete: %w", err)
	}
	return nil
}

// validate checks every choice is active and the subrace fits the race:
// races with active subraces (elf lineages, dragon ancestries, ...) require
// one, races without any must not have one.
func (s *Service) validate(ctx context.Context, d Draft) error {
	race, err := s.catalog.Race(ctx, d.RaceID)
	if err != nil {
		return err
	}
	if !race.Active {
		return ErrInvalidRace
	}
	var active []Option
	for _, sr := range race.Subraces {
		if sr.Active {
			active = append(active, sr)
		}
	}
	switch {
	case d.SubraceID == nil && len(active) > 0:
		return ErrSubraceRequired
	case d.SubraceID != nil && len(active) == 0:
		return ErrSubraceForbidden
	case d.SubraceID != nil && !containsID(active, *d.SubraceID):
		return ErrInvalidSubrace
	}

	class, err := s.catalog.Class(ctx, d.ClassID)
	if err != nil {
		return err
	}
	if !class.Active {
		return ErrInvalidClass
	}
	bkg, err := s.catalog.Background(ctx, d.BackgroundID)
	if err != nil {
		return err
	}
	if !bkg.Active {
		return ErrInvalidBkg
	}
	return nil
}

func normalize(d Draft) Draft {
	d.Name = strings.TrimSpace(d.Name)
	d.Gender = strings.TrimSpace(d.Gender)
	d.Appearance = strings.TrimSpace(d.Appearance)
	return d
}

func containsID(opts []Option, id uuid.UUID) bool {
	for _, o := range opts {
		if o.ID == id {
			return true
		}
	}
	return false
}
