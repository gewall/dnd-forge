// Package gamedata manages the playable options of the character builder:
// races (with subraces), classes and backgrounds.
package gamedata

import (
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

var (
	ErrRaceNotFound       = apperr.New(apperr.NotFound, "ras tidak ditemukan")
	ErrSubraceNotFound    = apperr.New(apperr.NotFound, "subras tidak ditemukan")
	ErrClassNotFound      = apperr.New(apperr.NotFound, "kelas tidak ditemukan")
	ErrBackgroundNotFound = apperr.New(apperr.NotFound, "latar belakang tidak ditemukan")
	ErrSlugTaken          = apperr.New(apperr.Conflict, "slug sudah dipakai")
	ErrInUse              = apperr.New(apperr.Conflict, "data masih dipakai karakter; nonaktifkan saja")
	ErrInvalidSlug        = apperr.New(apperr.Invalid, "slug hanya boleh huruf kecil, angka, dan tanda hubung")
	ErrInvalidHitDie      = apperr.New(apperr.Invalid, "hit die harus 6, 8, 10, atau 12")
)

// Audit entity names.
const (
	entityRace       = "race"
	entitySubrace    = "subrace"
	entityClass      = "class"
	entityBackground = "background"
)

// Trait is a named rule text, used for racial traits and class features.
type Trait struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RaceInput holds the editable fields of a race.
type RaceInput struct {
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Traits       []Trait `json:"traits"`
	VisualPrompt string  `json:"visualPrompt"` // English, fed to the image model
	ImageKey     string  `json:"imageKey"`
	SortOrder    int     `json:"sortOrder"`
	IsActive     bool    `json:"isActive"`
}

// Race is a playable race (a "species" in the 2024 rules).
type Race struct {
	ID uuid.UUID
	RaceInput
	Subraces []Subrace
}

// SubraceInput holds the editable fields of a subrace.
type SubraceInput struct {
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Traits       []Trait `json:"traits"`
	VisualPrompt string  `json:"visualPrompt"`
	IsActive     bool    `json:"isActive"`
}

// Subrace is a lineage or legacy of a race.
type Subrace struct {
	ID     uuid.UUID
	RaceID uuid.UUID
	SubraceInput
}

// ClassInput holds the editable fields of a class.
type ClassInput struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	HitDie         int     `json:"hitDie"`
	PrimaryAbility string  `json:"primaryAbility"`
	Description    string  `json:"description"`
	Features       []Trait `json:"features"`
	VisualPrompt   string  `json:"visualPrompt"`
	ImageKey       string  `json:"imageKey"`
	SortOrder      int     `json:"sortOrder"`
	IsActive       bool    `json:"isActive"`
}

// Class is a character class.
type Class struct {
	ID uuid.UUID
	ClassInput
}

// BackgroundInput holds the editable fields of a background.
type BackgroundInput struct {
	Slug               string   `json:"slug"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	SkillProficiencies []string `json:"skillProficiencies"`
	VisualPrompt       string   `json:"visualPrompt"`
	ImageKey           string   `json:"imageKey"`
	SortOrder          int      `json:"sortOrder"`
	IsActive           bool     `json:"isActive"`
}

// Background is a character background.
type Background struct {
	ID uuid.UUID
	BackgroundInput
}
