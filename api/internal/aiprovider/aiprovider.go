// Package aiprovider lets admins configure which AI vendor and model
// generate portraits (image) and character lore (text). API keys are sealed at rest and are only opened in
// memory when a provider is called; the API never returns them.
package aiprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/audit"
)

var (
	ErrNotFound        = apperr.New(apperr.NotFound, "provider AI tidak ditemukan")
	ErrNoImageProvider = apperr.New(apperr.Unavailable, "belum ada provider gambar aktif; hubungi admin")
	ErrNoTextProvider  = apperr.New(apperr.Unavailable, "belum ada provider teks aktif; hubungi admin")
	ErrInUse           = apperr.New(apperr.Conflict, "provider sudah pernah dipakai; biarkan nonaktif saja")
	ErrActive          = apperr.New(apperr.Conflict, "provider aktif tidak bisa dihapus; aktifkan provider lain dulu")
	ErrConnection      = apperr.New(apperr.Invalid, "uji koneksi gagal")
	ErrKeyUnreadable   = errors.New("aiprovider: stored API key cannot be decrypted (ENCRYPTION_KEY changed?)")
)

const entityProvider = "ai_provider"

// Provider is a configured vendor account.
type Provider struct {
	ID           uuid.UUID
	Type         ai.ProviderType
	Capability   ai.Capability
	Label        string
	KeyLast4     string
	DefaultModel string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	sealedKey    []byte
}

// Input holds editable fields. Type and Capability are set on create only.
// APIKey is required on create and optional on update, where empty keeps
// the stored key.
type Input struct {
	Type         ai.ProviderType
	Capability   ai.Capability
	Label        string
	DefaultModel string
	APIKey       string
}

// Selected is what a generation request needs from the active provider.
type Selected struct {
	ID    uuid.UUID
	Model string
}

// Store persists providers. UpdateProvider keeps the stored key when
// sealedKey is nil.
type Store interface {
	Providers(ctx context.Context) ([]Provider, error)
	ProviderByID(ctx context.Context, id uuid.UUID) (Provider, error)      // ErrNotFound
	ActiveProvider(ctx context.Context, c ai.Capability) (Provider, error) // ErrNotFound
	CreateProvider(ctx context.Context, in Input, sealedKey []byte, last4 string) (Provider, error)
	UpdateProvider(ctx context.Context, id uuid.UUID, in Input, sealedKey []byte, last4 string) (Provider, error)
	Activate(ctx context.Context, id uuid.UUID) error       // deactivates the others of its capability
	DeleteProvider(ctx context.Context, id uuid.UUID) error // ErrInUse when images reference it
	RecordAudit(ctx context.Context, e audit.Entry) error
	InTx(ctx context.Context, fn func(Store) error) error
}

// Sealer encrypts API keys.
type Sealer interface {
	Seal(plaintext []byte) ([]byte, error)
	Open(sealed []byte) ([]byte, error)
}

// Registry resolves vendor implementations.
type Registry interface {
	Vendor(t ai.ProviderType) (ai.Vendor, error)
	Image(t ai.ProviderType) (ai.ImageProvider, error)
	Text(t ai.ProviderType) (ai.TextProvider, error)
	Supports(t ai.ProviderType, c ai.Capability) bool
	Types() []ai.TypeInfo
}

// Service manages providers and hands decrypted credentials to callers
// inside the API process.
type Service struct {
	store    Store
	sealer   Sealer
	registry Registry
}

// NewService returns a Service.
func NewService(store Store, sealer Sealer, registry Registry) *Service {
	return &Service{store: store, sealer: sealer, registry: registry}
}

// Types lists the vendor types this build supports.
func (s *Service) Types() []ai.TypeInfo { return s.registry.Types() }

// List returns every configured provider.
func (s *Service) List(ctx context.Context) ([]Provider, error) { return s.store.Providers(ctx) }

// Create stores a provider with its sealed key. It starts inactive.
func (s *Service) Create(ctx context.Context, actor uuid.UUID, in Input) (Provider, error) {
	in = normalize(in)
	if _, err := s.registry.Vendor(in.Type); err != nil {
		return Provider{}, err
	}
	if !s.registry.Supports(in.Type, in.Capability) {
		return Provider{}, fmt.Errorf("%w: %s/%s", ai.ErrUnsupported, in.Type, in.Capability)
	}
	sealed, last4, err := s.seal(in.APIKey)
	if err != nil {
		return Provider{}, err
	}
	var p Provider
	err = s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityProvider, Diff: auditDiff(in, last4)},
		func(tx Store) (_ uuid.UUID, err error) {
			p, err = tx.CreateProvider(ctx, in, sealed, last4)
			return p.ID, err
		})
	return p, err
}

// Update changes label and model, and the key when in.APIKey is set.
// Replacing a key means typing a new one; the old key is never shown.
func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in Input) (Provider, error) {
	in = normalize(in)
	var sealed []byte
	var last4 string
	if in.APIKey != "" {
		var err error
		if sealed, last4, err = s.seal(in.APIKey); err != nil {
			return Provider{}, err
		}
	}
	var p Provider
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityProvider, Diff: auditDiff(in, last4)},
		func(tx Store) (_ uuid.UUID, err error) {
			p, err = tx.UpdateProvider(ctx, id, in, sealed, last4)
			return id, err
		})
	return p, err
}

// Activate makes the provider the one used for new requests of its
// capability.
func (s *Service) Activate(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionActivate, Entity: entityProvider},
		func(tx Store) (uuid.UUID, error) { return id, tx.Activate(ctx, id) })
}

// Delete removes an inactive provider that no image references.
func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityProvider},
		func(tx Store) (uuid.UUID, error) {
			p, err := tx.ProviderByID(ctx, id)
			if err != nil {
				return id, err
			}
			if p.IsActive {
				return id, ErrActive
			}
			return id, tx.DeleteProvider(ctx, id)
		})
}

// Test calls the vendor with the stored key.
func (s *Service) Test(ctx context.Context, id uuid.UUID) error {
	p, key, err := s.credentials(ctx, id)
	if err != nil {
		return err
	}
	impl, err := s.registry.Vendor(p.Type)
	if err != nil {
		return err
	}
	if err := impl.TestConnection(ctx, key); err != nil {
		return fmt.Errorf("%w: %w", ErrConnection, err)
	}
	return nil
}

// Active returns the provider and model new requests of capability c
// should use. A provider whose type this build does not register (e.g. the
// fake one once AI_FAKE_PROVIDER is off) counts as none, so requests fail
// fast instead of queueing jobs that cannot run.
func (s *Service) Active(ctx context.Context, c ai.Capability) (Selected, error) {
	p, err := s.store.ActiveProvider(ctx, c)
	if errors.Is(err, ErrNotFound) || err == nil && !s.registry.Supports(p.Type, c) {
		if c == ai.Text {
			return Selected{}, ErrNoTextProvider
		}
		return Selected{}, ErrNoImageProvider
	}
	if err != nil {
		return Selected{}, err
	}
	return Selected{ID: p.ID, Model: p.DefaultModel}, nil
}

// ResolveImage returns the image implementation and decrypted key of a
// provider. The key must stay in memory and never be logged.
func (s *Service) ResolveImage(ctx context.Context, id uuid.UUID) (ai.ImageProvider, string, error) {
	p, key, err := s.credentials(ctx, id)
	if err != nil {
		return nil, "", err
	}
	impl, err := s.registry.Image(p.Type)
	if err != nil {
		return nil, "", err
	}
	return impl, key, nil
}

// ResolveText returns the text implementation and decrypted key of a
// provider. The key must stay in memory and never be logged.
func (s *Service) ResolveText(ctx context.Context, id uuid.UUID) (ai.TextProvider, string, error) {
	p, key, err := s.credentials(ctx, id)
	if err != nil {
		return nil, "", err
	}
	impl, err := s.registry.Text(p.Type)
	if err != nil {
		return nil, "", err
	}
	return impl, key, nil
}

// credentials loads a provider and opens its key.
func (s *Service) credentials(ctx context.Context, id uuid.UUID) (Provider, string, error) {
	p, err := s.store.ProviderByID(ctx, id)
	if err != nil {
		return Provider{}, "", err
	}
	key, err := s.sealer.Open(p.sealedKey)
	if err != nil {
		return Provider{}, "", ErrKeyUnreadable
	}
	return p, string(key), nil
}

func (s *Service) seal(apiKey string) ([]byte, string, error) {
	sealed, err := s.sealer.Seal([]byte(apiKey))
	if err != nil {
		return nil, "", err
	}
	last4 := apiKey
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return sealed, last4, nil
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
		return fmt.Errorf("aiprovider: %s: %w", e.Action, err)
	}
	return nil
}

// auditDiff records the change without the key.
func auditDiff(in Input, last4 string) map[string]string {
	d := map[string]string{"label": in.Label, "defaultModel": in.DefaultModel}
	if in.Type != "" {
		d["type"], d["capability"] = string(in.Type), string(in.Capability)
	}
	if last4 != "" {
		d["keyLast4"] = last4
	}
	return d
}

func normalize(in Input) Input {
	in.Label = strings.TrimSpace(in.Label)
	in.DefaultModel = strings.TrimSpace(in.DefaultModel)
	in.APIKey = strings.TrimSpace(in.APIKey)
	return in
}
