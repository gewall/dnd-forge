// Package prompt manages the admin-edited prompt templates, one active per
// kind, and fills them with a character's details.
package prompt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/audit"
)

var (
	ErrNotFound       = apperr.New(apperr.NotFound, "template tidak ditemukan")
	ErrNoActive       = apperr.New(apperr.Unavailable, "belum ada template prompt aktif")
	ErrActiveInUse    = apperr.New(apperr.Conflict, "template aktif tidak bisa dihapus; aktifkan template lain dulu")
	ErrBadPlaceholder = apperr.New(apperr.Invalid, "template memakai placeholder yang tidak dikenal")
	ErrUnknownKind    = apperr.New(apperr.Invalid, "jenis template tidak dikenal")
)

const entityTemplate = "prompt_template"

// Kind is what a template generates. One template of each kind is active.
type Kind string

const (
	KindPortrait Kind = "portrait"
	KindLore     Kind = "lore"
)

// DefaultTemplate is the starter template from the architecture document.
const DefaultTemplate = `Fantasy character portrait of a {race} {class} with a {background} background.
{raceVisual}. {classVisual}. {appearance}.
Additional details: {userPrompt}.
Style: {style}, detailed, dramatic lighting, upper body, no text, no watermark.`

// DefaultNegative lists what the default template asks the model to avoid.
const DefaultNegative = "text, watermark, signature, extra limbs, deformed hands, blurry"

// DefaultLoreTemplate asks for a character backstory. Each optional detail
// sits on its own line so Build drops it cleanly when empty.
const DefaultLoreTemplate = `Write the backstory of a Dungeons & Dragons character for the player who created it.
Name: {name}. Race: {race}. Class: {class}. Background: {background}.
Subrace: {subrace}.
Gender: {gender}.
Appearance: {appearance}.
The player's wishes: {userPrompt}.
Tone: {tone}. Length: {length}.
Write in {language}. Use flowing prose in two to five paragraphs, without a title, headings or lists.
Keep it consistent with the D&D 5e setting, leave room for future adventures, and do not invent game rules or statistics.`

// DefaultLoreNegative lists what the default lore template asks to avoid.
const DefaultLoreNegative = "explicit sexual content, real people, existing copyrighted characters"

// Template is one immutable version of a named template.
type Template struct {
	ID             uuid.UUID
	Kind           Kind
	Name           string
	Template       string
	NegativePrompt string
	Version        int
	IsActive       bool
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
}

// Input holds a new version's content.
type Input struct {
	Kind           Kind   `json:"kind"`
	Name           string `json:"name"`
	Template       string `json:"template"`
	NegativePrompt string `json:"negativePrompt"`
}

// Store persists templates. CreateTemplate assigns the next version number
// for the kind and name.
type Store interface {
	Templates(ctx context.Context) ([]Template, error)
	TemplateByID(ctx context.Context, id uuid.UUID) (Template, error) // ErrNotFound
	ActiveTemplate(ctx context.Context, k Kind) (Template, error)     // ErrNoActive
	CreateTemplate(ctx context.Context, in Input, createdBy *uuid.UUID) (Template, error)
	Activate(ctx context.Context, id uuid.UUID) error // ErrNotFound; deactivates the others of its kind
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	RecordAudit(ctx context.Context, e audit.Entry) error
	InTx(ctx context.Context, fn func(Store) error) error
}

// Service manages templates.
type Service struct {
	store Store
}

// NewService returns a Service.
func NewService(store Store) *Service { return &Service{store: store} }

// List returns every version, grouped by kind and name, newest version
// first.
func (s *Service) List(ctx context.Context) ([]Template, error) { return s.store.Templates(ctx) }

// Active returns the template new requests of kind k use.
func (s *Service) Active(ctx context.Context, k Kind) (Template, error) {
	return s.store.ActiveTemplate(ctx, k)
}

// Create saves a new version of the template of in.Kind named in.Name.
func (s *Service) Create(ctx context.Context, actor uuid.UUID, in Input) (Template, error) {
	in.Name = strings.TrimSpace(in.Name)
	if _, ok := fields(in.Kind); !ok {
		return Template{}, fmt.Errorf("%w: %q", ErrUnknownKind, in.Kind)
	}
	if bad := unknownPlaceholders(in.Kind, in.Template); len(bad) > 0 {
		return Template{}, fmt.Errorf("%w: %s (tersedia: {%s})", ErrBadPlaceholder,
			strings.Join(bad, ", "), strings.Join(Placeholders(in.Kind), "}, {"))
	}
	var by *uuid.UUID
	if actor != uuid.Nil {
		by = &actor
	}
	var t Template
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityTemplate, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			t, err = tx.CreateTemplate(ctx, in, by)
			return t.ID, err
		})
	return t, err
}

// Activate makes the template the one used for new requests of its kind.
func (s *Service) Activate(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionActivate, Entity: entityTemplate},
		func(tx Store) (uuid.UUID, error) { return id, tx.Activate(ctx, id) })
}

// Delete removes an inactive template version.
func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityTemplate},
		func(tx Store) (uuid.UUID, error) {
			t, err := tx.TemplateByID(ctx, id)
			if err != nil {
				return id, err
			}
			if t.IsActive {
				return id, ErrActiveInUse
			}
			return id, tx.DeleteTemplate(ctx, id)
		})
}

// defaults are the starter templates, one per kind.
var defaults = []Input{
	{Kind: KindPortrait, Name: "default", Template: DefaultTemplate, NegativePrompt: DefaultNegative},
	{Kind: KindLore, Name: "default", Template: DefaultLoreTemplate, NegativePrompt: DefaultLoreNegative},
}

// SeedDefaults creates and activates the starter template of every kind
// that has no active template, and returns the kinds it seeded.
func (s *Service) SeedDefaults(ctx context.Context) ([]Kind, error) {
	var seeded []Kind
	for _, in := range defaults {
		_, err := s.store.ActiveTemplate(ctx, in.Kind)
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrNoActive) {
			return seeded, fmt.Errorf("prompt: seed %s: %w", in.Kind, err)
		}
		t, err := s.Create(ctx, uuid.Nil, in)
		if err != nil {
			return seeded, err
		}
		if err := s.Activate(ctx, uuid.Nil, t.ID); err != nil {
			return seeded, err
		}
		seeded = append(seeded, in.Kind)
	}
	return seeded, nil
}

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
		return fmt.Errorf("prompt: %s: %w", e.Action, err)
	}
	return nil
}
