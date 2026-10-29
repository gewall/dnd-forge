package gamedata

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// srd52 holds the starter content from the SRD 5.2 (CC BY 4.0). Descriptions
// are our own Indonesian summaries; visual prompts are English for the
// image model.
//
//go:embed seed/srd52.json
var srd52 []byte

type seedRace struct {
	RaceInput
	Subraces []SubraceInput `json:"subraces"`
}

type seedData struct {
	Attribution string            `json:"attribution"`
	Races       []seedRace        `json:"races"`
	Classes     []ClassInput      `json:"classes"`
	Backgrounds []BackgroundInput `json:"backgrounds"`
}

// SeedReport counts what Seed created; existing rows are left untouched.
type SeedReport struct {
	Races, Subraces, Classes, Backgrounds int
	Attribution                           string
}

// Seed inserts the embedded SRD 5.2 races, classes and backgrounds whose
// slugs do not exist yet, so it is safe to run on every deploy. Rows an
// admin already edited are never overwritten.
func (s *Service) Seed(ctx context.Context) (SeedReport, error) {
	var data seedData
	if err := json.Unmarshal(srd52, &data); err != nil {
		return SeedReport{}, fmt.Errorf("gamedata: seed: decode: %w", err)
	}
	rep := SeedReport{Attribution: data.Attribution}
	system := uuid.Nil

	for _, sr := range data.Races {
		in := sr.RaceInput
		in.IsActive = true
		race, err := s.store.RaceBySlug(ctx, in.Slug)
		if errors.Is(err, ErrRaceNotFound) {
			if race, err = s.CreateRace(ctx, system, in); err == nil {
				rep.Races++
			}
		}
		if err != nil {
			return rep, fmt.Errorf("gamedata: seed race %s: %w", in.Slug, err)
		}

		have := make(map[string]bool, len(race.Subraces))
		for _, existing := range race.Subraces {
			have[existing.Slug] = true
		}
		for _, sub := range sr.Subraces {
			if have[sub.Slug] {
				continue
			}
			sub.IsActive = true
			if _, err := s.CreateSubrace(ctx, system, race.ID, sub); err != nil {
				return rep, fmt.Errorf("gamedata: seed subrace %s/%s: %w", in.Slug, sub.Slug, err)
			}
			rep.Subraces++
		}
	}

	for _, in := range data.Classes {
		in.IsActive = true
		_, err := s.store.ClassBySlug(ctx, in.Slug)
		if errors.Is(err, ErrClassNotFound) {
			if _, err = s.CreateClass(ctx, system, in); err == nil {
				rep.Classes++
			}
		}
		if err != nil {
			return rep, fmt.Errorf("gamedata: seed class %s: %w", in.Slug, err)
		}
	}

	for _, in := range data.Backgrounds {
		in.IsActive = true
		_, err := s.store.BackgroundBySlug(ctx, in.Slug)
		if errors.Is(err, ErrBackgroundNotFound) {
			if _, err = s.CreateBackground(ctx, system, in); err == nil {
				rep.Backgrounds++
			}
		}
		if err != nil {
			return rep, fmt.Errorf("gamedata: seed background %s: %w", in.Slug, err)
		}
	}
	return rep, nil
}
