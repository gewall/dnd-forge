package character

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/gamedata"
)

// GameData is the part of gamedata.Service the catalog reads.
type GameData interface {
	RaceByID(ctx context.Context, id uuid.UUID) (gamedata.Race, error)
	ClassByID(ctx context.Context, id uuid.UUID) (gamedata.Class, error)
	BackgroundByID(ctx context.Context, id uuid.UUID) (gamedata.Background, error)
}

// GameDataCatalog adapts the game data service to Catalog.
type GameDataCatalog struct {
	data GameData
}

// NewGameDataCatalog returns a Catalog backed by data.
func NewGameDataCatalog(data GameData) *GameDataCatalog { return &GameDataCatalog{data: data} }

func (c *GameDataCatalog) Race(ctx context.Context, id uuid.UUID) (RaceOption, error) {
	r, err := c.data.RaceByID(ctx, id)
	if errors.Is(err, gamedata.ErrRaceNotFound) {
		return RaceOption{}, ErrInvalidRace
	}
	if err != nil {
		return RaceOption{}, fmt.Errorf("catalog race: %w", err)
	}
	opt := RaceOption{Option: Option{ID: r.ID, Active: r.IsActive}}
	for _, sr := range r.Subraces {
		opt.Subraces = append(opt.Subraces, Option{ID: sr.ID, Active: sr.IsActive})
	}
	return opt, nil
}

func (c *GameDataCatalog) Class(ctx context.Context, id uuid.UUID) (Option, error) {
	k, err := c.data.ClassByID(ctx, id)
	if errors.Is(err, gamedata.ErrClassNotFound) {
		return Option{}, ErrInvalidClass
	}
	if err != nil {
		return Option{}, fmt.Errorf("catalog class: %w", err)
	}
	return Option{ID: k.ID, Active: k.IsActive}, nil
}

func (c *GameDataCatalog) Background(ctx context.Context, id uuid.UUID) (Option, error) {
	b, err := c.data.BackgroundByID(ctx, id)
	if errors.Is(err, gamedata.ErrBackgroundNotFound) {
		return Option{}, ErrInvalidBkg
	}
	if err != nil {
		return Option{}, fmt.Errorf("catalog background: %w", err)
	}
	return Option{ID: b.ID, Active: b.IsActive}, nil
}
