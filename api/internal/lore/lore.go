// Package lore writes character backstories with the active AI text
// provider: it checks quota and prompt safety, queues a River job, and the
// worker stores the result as a draft the user may apply to the character.
package lore

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

var (
	ErrCharacterNotFound = apperr.New(apperr.NotFound, "karakter tidak ditemukan")
	ErrNotFound          = apperr.New(apperr.NotFound, "lore tidak ditemukan")
	ErrQuotaExceeded     = apperr.New(apperr.TooManyRequests, "kuota lore hari ini sudah habis; coba lagi besok")
	ErrInvalidTone       = apperr.New(apperr.Invalid, "nada lore tidak dikenal")
	ErrInvalidLength     = apperr.New(apperr.Invalid, "panjang lore harus short, medium, atau long")
	ErrInvalidLanguage   = apperr.New(apperr.Invalid, "bahasa lore harus id atau en")
	ErrNotReady          = apperr.New(apperr.Invalid, "lore ini belum selesai atau gagal, jadi belum bisa dipakai")
)

// MaxPromptRunes bounds the wishes a user may write.
const MaxPromptRunes = 500

// maxContentRunes caps a stored draft, well above the longest length.
const maxContentRunes = 20000

// Status of a lore draft.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
)

// Messages shown to users for failed drafts. Vendor errors are logged,
// never returned, because they may contain prompt or account details.
const (
	msgRejected = "Permintaan lore ditolak oleh moderasi provider AI. Coba ubah keinginanmu."
	msgConfig   = "Provider AI sedang bermasalah. Admin sudah bisa melihat log-nya."
	msgGaveUp   = "Gagal menulis lore setelah beberapa percobaan. Kuota tidak terpotong; silakan coba lagi."
)

// systemPrompt frames every request; the admin-edited template carries the
// actual instructions.
const systemPrompt = "You are a creative writer of fantasy lore for tabletop role-playing games. " +
	"Reply with the story text only."

// Option is a choice the user can pick, with what it means to the model.
type Option struct {
	ID          string
	Description string // shown to users (Indonesian)
	Prompt      string // sent to the model (English)
}

// Tones the user can pick.
var Tones = map[string]Option{
	"epic":   {Description: "Epik dan megah", Prompt: "epic and grand"},
	"heroic": {Description: "Heroik dan penuh harapan", Prompt: "heroic and hopeful"},
	"dark":   {Description: "Kelam dan suram", Prompt: "dark and grim"},
	"tragic": {Description: "Tragis dan melankolis", Prompt: "tragic and melancholic"},
	"humor":  {Description: "Ringan dan lucu", Prompt: "light-hearted and humorous"},
}

// length is a target size and the token budget that fits it.
type length struct {
	Option
	maxTokens int
}

// Lengths the user can pick. Token budgets leave room for Indonesian, which
// takes more tokens per word than English.
var Lengths = map[string]length{
	"short":  {Option{Description: "Pendek (±150 kata)", Prompt: "about 120-180 words"}, 700},
	"medium": {Option{Description: "Sedang (±350 kata)", Prompt: "about 300-400 words"}, 1400},
	"long":   {Option{Description: "Panjang (±700 kata)", Prompt: "about 600-800 words"}, 2800},
}

// Languages the draft can be written in.
var Languages = map[string]Option{
	"id": {Description: "Bahasa Indonesia", Prompt: "Indonesian (Bahasa Indonesia)"},
	"en": {Description: "English", Prompt: "English"},
}

// Defaults for omitted request fields.
const (
	DefaultLength   = "medium"
	DefaultLanguage = "id"
)

// sortedOptions lists m by id.
func sortedOptions[V any](m map[string]V, opt func(V) Option) []Option {
	out := make([]Option, 0, len(m))
	for id, v := range m {
		o := opt(v)
		o.ID = id
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Request is what the user submits.
type Request struct {
	Tone       string
	Length     string
	Language   string
	UserPrompt string
}

// Lore is one generation attempt and its draft.
type Lore struct {
	ID           uuid.UUID
	CharacterID  *uuid.UUID
	UserID       uuid.UUID
	ProviderID   uuid.UUID
	Model        string
	UserPrompt   string
	FinalPrompt  string
	Tone         string
	Length       string
	Language     string
	Status       Status
	Content      string
	ErrorMessage string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// NewLore holds the fields of a queued draft.
type NewLore struct {
	CharacterID uuid.UUID
	UserID      uuid.UUID
	ProviderID  uuid.UUID
	Model       string
	UserPrompt  string
	FinalPrompt string
	Tone        string
	Length      string
	Language    string
}

// Subject is a character with everything the prompt needs.
type Subject struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	Gender     string
	Appearance string
	Race       string
	Subrace    string
	Class      string
	Background string
}
