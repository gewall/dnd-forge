package portrait

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/audit"
)

// ErrStillPending rejects deleting a portrait the worker may still write.
var ErrStillPending = apperr.New(apperr.Conflict, "portrait masih diproses; tunggu sampai selesai atau gagal")

const entityImage = "generated_image"

// AdminImage is a portrait with what a moderator needs to judge it.
type AdminImage struct {
	Portrait
	UserEmail       string
	CharacterName   string // empty when the character was deleted
	CharacterPublic bool
	// ImageURL and ThumbURL are presigned and empty when the files are gone.
	ImageURL string
	ThumbURL string
}

// HasFiles reports whether the image is still in object storage.
func (a AdminImage) HasFiles() bool { return a.StorageKey != "" }

// AdminFilter narrows the moderation list. Zero values do not filter.
type AdminFilter struct {
	Status Status
	Hidden *bool
	UserID *uuid.UUID
	Before *time.Time
	Limit  int
}

// AdminStore reads and moderates portraits.
type AdminStore interface {
	AdminImages(ctx context.Context, f AdminFilter) ([]AdminImage, error)
	AdminImage(ctx context.Context, id uuid.UUID) (AdminImage, error) // ErrNotFound
	InTx(ctx context.Context, fn func(AdminTx) error) error
}

// AdminTx changes one portrait inside a transaction.
type AdminTx interface {
	LockPortrait(ctx context.Context, id uuid.UUID) (Portrait, error) // ErrNotFound
	SetHidden(ctx context.Context, id uuid.UUID, hidden bool) error
	ClearFiles(ctx context.Context, id uuid.UUID) error
	RecordAudit(ctx context.Context, e audit.Entry) error
}

// AdminObjects deletes and links stored images.
type AdminObjects interface {
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string) (string, time.Duration, error)
}

// Admin moderates generated portraits.
type Admin struct {
	store   AdminStore
	objects AdminObjects
}

// NewAdmin returns an Admin.
func NewAdmin(store AdminStore, objects AdminObjects) *Admin {
	return &Admin{store: store, objects: objects}
}

const (
	defaultAdminPage = 30
	maxAdminPage     = 100
)

// List returns portraits matching f, newest first, with presigned links so
// admins can see hidden images too.
func (a *Admin) List(ctx context.Context, f AdminFilter) ([]AdminImage, error) {
	if f.Limit <= 0 {
		f.Limit = defaultAdminPage
	}
	f.Limit = min(f.Limit, maxAdminPage)
	images, err := a.store.AdminImages(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("portrait: admin list: %w", err)
	}
	for i := range images {
		if err := a.link(ctx, &images[i]); err != nil {
			return nil, err
		}
	}
	return images, nil
}

// Get returns one portrait for moderation.
func (a *Admin) Get(ctx context.Context, id uuid.UUID) (AdminImage, error) {
	img, err := a.store.AdminImage(ctx, id)
	if err != nil {
		return AdminImage{}, fmt.Errorf("portrait: admin: %w", err)
	}
	return img, a.link(ctx, &img)
}

// SetHidden hides a portrait from the gallery and media links, or shows it
// again.
func (a *Admin) SetHidden(ctx context.Context, actor, id uuid.UUID, hidden bool) (AdminImage, error) {
	err := a.store.InTx(ctx, func(tx AdminTx) error {
		p, err := tx.LockPortrait(ctx, id)
		if err != nil {
			return err
		}
		if p.IsHidden == hidden {
			return nil
		}
		if err := tx.SetHidden(ctx, id, hidden); err != nil {
			return err
		}
		return tx.RecordAudit(ctx, audit.Entry{
			ActorID: actor, Action: audit.ActionUpdate, Entity: entityImage, EntityID: id.String(),
			Diff: map[string][2]bool{"isHidden": {p.IsHidden, hidden}},
		})
	})
	if err != nil {
		return AdminImage{}, fmt.Errorf("portrait: set hidden: %w", err)
	}
	return a.Get(ctx, id)
}

// DeleteFiles removes a portrait's image and thumbnail from storage and
// hides it for good. The row stays, so quota and statistics are unchanged.
// Objects are deleted before the row is updated: if that update fails the
// portrait merely points at missing files, never the other way round.
func (a *Admin) DeleteFiles(ctx context.Context, actor, id uuid.UUID) error {
	err := a.store.InTx(ctx, func(tx AdminTx) error {
		p, err := tx.LockPortrait(ctx, id)
		if err != nil {
			return err
		}
		if p.Status == StatusPending {
			return ErrStillPending
		}
		for _, key := range []string{p.StorageKey, p.ThumbKey} {
			if key == "" {
				continue
			}
			if err := a.objects.Delete(ctx, key); err != nil {
				return err
			}
		}
		if err := tx.ClearFiles(ctx, id); err != nil {
			return err
		}
		return tx.RecordAudit(ctx, audit.Entry{
			ActorID: actor, Action: audit.ActionDelete, Entity: entityImage, EntityID: id.String(),
			Diff: map[string]any{"storageKey": p.StorageKey, "thumbKey": p.ThumbKey},
		})
	})
	if err != nil {
		return fmt.Errorf("portrait: delete files: %w", err)
	}
	return nil
}

func (a *Admin) link(ctx context.Context, img *AdminImage) error {
	if !img.HasFiles() {
		return nil
	}
	var err error
	if img.ImageURL, _, err = a.objects.PresignGet(ctx, img.StorageKey); err != nil {
		return fmt.Errorf("portrait: presign: %w", err)
	}
	if img.ThumbKey != "" {
		if img.ThumbURL, _, err = a.objects.PresignGet(ctx, img.ThumbKey); err != nil {
			return fmt.Errorf("portrait: presign: %w", err)
		}
	}
	return nil
}
