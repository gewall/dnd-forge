package portrait

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
)

type memAdmin struct {
	images     map[uuid.UUID]*AdminImage
	audits     []audit.Entry
	lastFilter AdminFilter
}

func (m *memAdmin) AdminImages(_ context.Context, f AdminFilter) ([]AdminImage, error) {
	m.lastFilter = f
	var out []AdminImage
	for _, img := range m.images {
		out = append(out, *img)
	}
	return out, nil
}

func (m *memAdmin) AdminImage(_ context.Context, id uuid.UUID) (AdminImage, error) {
	img, ok := m.images[id]
	if !ok {
		return AdminImage{}, ErrNotFound
	}
	return *img, nil
}

func (m *memAdmin) InTx(_ context.Context, fn func(AdminTx) error) error { return fn(m) }

func (m *memAdmin) LockPortrait(_ context.Context, id uuid.UUID) (Portrait, error) {
	img, ok := m.images[id]
	if !ok {
		return Portrait{}, ErrNotFound
	}
	return img.Portrait, nil
}

func (m *memAdmin) SetHidden(_ context.Context, id uuid.UUID, hidden bool) error {
	m.images[id].IsHidden = hidden
	return nil
}

func (m *memAdmin) ClearFiles(_ context.Context, id uuid.UUID) error {
	img := m.images[id]
	img.StorageKey, img.ThumbKey, img.IsHidden = "", "", true
	return nil
}

func (m *memAdmin) RecordAudit(_ context.Context, e audit.Entry) error {
	m.audits = append(m.audits, e)
	return nil
}

type memObjects struct {
	deleted []string
	failOn  string
}

func (o *memObjects) Delete(_ context.Context, key string) error {
	if key == o.failOn {
		return errors.New("storage down")
	}
	o.deleted = append(o.deleted, key)
	return nil
}

func (o *memObjects) PresignGet(_ context.Context, key string) (string, time.Duration, error) {
	return "https://s3.example/" + key + "?sig", time.Hour, nil
}

func newAdminFixture() (*Admin, *memAdmin, *memObjects, uuid.UUID, uuid.UUID) {
	done, pending := uuid.New(), uuid.New()
	store := &memAdmin{images: map[uuid.UUID]*AdminImage{
		done:    {Portrait: Portrait{ID: done, Status: StatusSucceeded, StorageKey: "portraits/u/a.webp", ThumbKey: "portraits/u/a_thumb.webp"}},
		pending: {Portrait: Portrait{ID: pending, Status: StatusPending}},
	}}
	objects := &memObjects{}
	return NewAdmin(store, objects), store, objects, done, pending
}

func TestAdminList(t *testing.T) {
	admin, store, _, done, pending := newAdminFixture()
	store.images[done].IsHidden = true
	images, err := admin.List(context.Background(), AdminFilter{Limit: 1000})
	if err != nil || len(images) != 2 {
		t.Fatalf("List = %d images, %v", len(images), err)
	}
	if store.lastFilter.Limit != maxAdminPage {
		t.Errorf("limit = %d, want %d", store.lastFilter.Limit, maxAdminPage)
	}
	for _, img := range images {
		switch img.ID {
		case done: // hidden images stay viewable for admins
			if img.ImageURL == "" || img.ThumbURL == "" {
				t.Errorf("hidden image without links: %+v", img)
			}
		case pending:
			if img.ImageURL != "" || img.HasFiles() {
				t.Errorf("pending image has links: %+v", img)
			}
		}
	}
}

func TestAdminSetHidden(t *testing.T) {
	admin, store, _, done, _ := newAdminFixture()
	ctx, actor := context.Background(), uuid.New()
	img, err := admin.SetHidden(ctx, actor, done, true)
	if err != nil || !img.IsHidden {
		t.Fatalf("hide = %+v, %v", img, err)
	}
	if _, err := admin.SetHidden(ctx, actor, done, true); err != nil {
		t.Fatal(err)
	}
	if len(store.audits) != 1 || store.audits[0].ActorID != actor || store.audits[0].Entity != entityImage {
		t.Errorf("audits = %+v; a repeated hide must not log again", store.audits)
	}
	if _, err := admin.SetHidden(ctx, actor, uuid.New(), true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown image err = %v", err)
	}
}

func TestAdminDeleteFiles(t *testing.T) {
	ctx, actor := context.Background(), uuid.New()

	admin, store, objects, done, pending := newAdminFixture()
	if err := admin.DeleteFiles(ctx, actor, pending); !errors.Is(err, ErrStillPending) {
		t.Errorf("pending err = %v", err)
	}
	if err := admin.DeleteFiles(ctx, actor, done); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(objects.deleted, []string{"portraits/u/a.webp", "portraits/u/a_thumb.webp"}) {
		t.Errorf("deleted = %v", objects.deleted)
	}
	img := store.images[done]
	if img.StorageKey != "" || !img.IsHidden || len(store.audits) != 1 || store.audits[0].Action != audit.ActionDelete {
		t.Errorf("after delete: %+v, audits %+v", img, store.audits)
	}
	// Deleting again is harmless: there is nothing left to remove.
	if err := admin.DeleteFiles(ctx, actor, done); err != nil || len(objects.deleted) != 2 {
		t.Errorf("second delete = %v, deleted %v", err, objects.deleted)
	}

	// A storage failure leaves the row untouched.
	admin, store, objects, done, _ = newAdminFixture()
	objects.failOn = "portraits/u/a_thumb.webp"
	if err := admin.DeleteFiles(ctx, actor, done); err == nil {
		t.Fatal("storage failure ignored")
	}
	if store.images[done].StorageKey == "" || len(store.audits) != 0 {
		t.Errorf("row changed despite failure: %+v", store.images[done])
	}
}
