// Package portrait turns a saved character into an AI-generated portrait:
// it checks quota and prompt safety, queues a River job, and the worker
// calls the active provider and stores the image in object storage.
package portrait

import (
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

var (
	ErrCharacterNotFound = apperr.New(apperr.NotFound, "karakter tidak ditemukan")
	ErrNotFound          = apperr.New(apperr.NotFound, "portrait tidak ditemukan")
	ErrQuotaExceeded     = apperr.New(apperr.TooManyRequests, "kuota generate hari ini sudah habis; coba lagi besok")
	ErrInvalidStyle      = apperr.New(apperr.Invalid, "gaya gambar tidak dikenal")
	ErrInvalidAspect     = apperr.New(apperr.Invalid, "rasio gambar harus 1:1, 3:4, atau 4:3")
)

// Status of a portrait.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
)

// Messages shown to users for failed portraits. Vendor errors are logged,
// never returned, because they may contain prompt or account details.
const (
	msgRejected = "Prompt atau gambar ditolak oleh moderasi provider AI. Coba ubah prompt tambahan."
	msgConfig   = "Provider AI sedang bermasalah. Admin sudah bisa melihat log-nya."
	msgGaveUp   = "Gagal membuat gambar setelah beberapa percobaan. Kuota tidak terpotong; silakan coba lagi."
)

// wib is Indonesia's western time zone; quotas reset at local midnight.
// Indonesia has no daylight saving, so a fixed zone is exact and avoids
// shipping tzdata in the static image.
var wib = time.FixedZone("WIB", 7*60*60)

// Request is what the user submits.
type Request struct {
	Style       string // key of prompt.Styles
	UserPrompt  string
	AspectRatio string
}

// Portrait is one generation attempt and its result.
type Portrait struct {
	ID           uuid.UUID
	CharacterID  *uuid.UUID
	UserID       uuid.UUID
	ProviderID   uuid.UUID
	Model        string
	UserPrompt   string
	FinalPrompt  string
	Style        string
	AspectRatio  string
	Status       Status
	ErrorMessage string
	StorageKey   string
	ThumbKey     string
	IsHidden     bool
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// NewPortrait holds the fields of a queued portrait.
type NewPortrait struct {
	CharacterID uuid.UUID
	UserID      uuid.UUID
	ProviderID  uuid.UUID
	Model       string
	UserPrompt  string
	FinalPrompt string
	Style       string
	AspectRatio string
}

// Subject is a character with everything the prompt needs.
type Subject struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Name             string
	Gender           string
	Appearance       string
	Race             string
	RaceVisual       string
	Subrace          string
	SubraceVisual    string
	Class            string
	ClassVisual      string
	Background       string
	BackgroundVisual string
}

// Quota is a user's daily generation allowance.
type Quota struct {
	Limit     int
	Used      int
	Remaining int
	ResetsAt  time.Time
}

// GalleryItem is a public character with its newest portrait.
type GalleryItem struct {
	PortraitID    uuid.UUID
	CharacterID   uuid.UUID
	CharacterName string
	Race          string
	Class         string
	Style         string
	CreatedAt     time.Time
}

// startOfDay returns local midnight (WIB) of t and the next midnight.
func startOfDay(t time.Time) (time.Time, time.Time) {
	y, m, d := t.In(wib).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, wib)
	return start, start.AddDate(0, 0, 1)
}
