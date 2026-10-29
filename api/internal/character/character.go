// Package character stores the characters users build with the wizard and
// re-validates every choice against the game data.
package character

import (
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

// MaxPerUser bounds how many characters one account may keep.
const MaxPerUser = 50

var (
	ErrNotFound         = apperr.New(apperr.NotFound, "karakter tidak ditemukan")
	ErrLimitReached     = apperr.New(apperr.Invalid, "maksimal 50 karakter per akun; hapus yang tidak dipakai")
	ErrInvalidRace      = apperr.New(apperr.Invalid, "ras tidak tersedia")
	ErrInvalidClass     = apperr.New(apperr.Invalid, "kelas tidak tersedia")
	ErrInvalidBkg       = apperr.New(apperr.Invalid, "latar belakang tidak tersedia")
	ErrInvalidSubrace   = apperr.New(apperr.Invalid, "subras tidak tersedia untuk ras ini")
	ErrSubraceRequired  = apperr.New(apperr.Invalid, "ras ini wajib memilih subras")
	ErrSubraceForbidden = apperr.New(apperr.Invalid, "ras ini tidak punya subras")
)

// Draft holds what the wizard submits.
type Draft struct {
	Name         string
	Gender       string
	RaceID       uuid.UUID
	SubraceID    *uuid.UUID
	ClassID      uuid.UUID
	BackgroundID uuid.UUID
	Appearance   string
	IsPublic     bool
}

// Ref names a game data choice for display.
type Ref struct {
	ID   uuid.UUID
	Slug string
	Name string
}

// Character is a saved character with its choices resolved for display.
type Character struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	Gender     string
	Appearance string
	IsPublic   bool
	Race       Ref
	Subrace    *Ref
	Class      Ref
	Background Ref
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
