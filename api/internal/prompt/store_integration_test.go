//go:build integration

package prompt

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func TestPostgresTemplates(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "prompt_templates", "audit_logs")
	svc := NewService(NewPostgresStore(pool))

	if _, err := svc.Active(ctx, KindPortrait); !errors.Is(err, ErrNoActive) {
		t.Errorf("Active on empty table err = %v", err)
	}
	seeded, err := svc.SeedDefaults(ctx)
	if err != nil || len(seeded) != 1 || seeded[0] != KindPortrait {
		t.Fatalf("SeedDefaults = %v, %v", seeded, err)
	}
	if seeded, err := svc.SeedDefaults(ctx); err != nil || len(seeded) != 0 {
		t.Errorf("second SeedDefaults = %v, %v; want nothing", seeded, err)
	}
	v1, err := svc.Active(ctx, KindPortrait)
	if err != nil || v1.Name != "default" || v1.Version != 1 || v1.Kind != KindPortrait {
		t.Fatalf("seeded = %+v, %v", v1, err)
	}

	if _, err := svc.Create(ctx, uuid.Nil, Input{Kind: "poem", Name: "x", Template: "x"}); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("unknown kind err = %v", err)
	}
	if _, err := svc.Create(ctx, uuid.Nil, Input{Kind: KindPortrait, Name: "x", Template: "{weapon}"}); !errors.Is(err, ErrBadPlaceholder) {
		t.Errorf("bad placeholder err = %v", err)
	}

	v2, err := svc.Create(ctx, uuid.Nil, Input{Kind: KindPortrait, Name: "default", Template: "A {race} {class}."})
	if err != nil || v2.Version != 2 || v2.IsActive {
		t.Fatalf("second version = %+v, %v", v2, err)
	}
	other, err := svc.Create(ctx, uuid.Nil, Input{Kind: KindPortrait, Name: "dramatic", Template: "{race}"})
	if err != nil || other.Version != 1 {
		t.Errorf("other name starts at version 1: %+v, %v", other, err)
	}

	if err := svc.Activate(ctx, uuid.Nil, v2.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := svc.Active(ctx, KindPortrait); err != nil || got.ID != v2.ID {
		t.Errorf("Active after activate = %+v, %v", got, err)
	}
	if err := svc.Delete(ctx, uuid.Nil, v2.ID); !errors.Is(err, ErrActiveInUse) {
		t.Errorf("delete active err = %v", err)
	}
	if err := svc.Delete(ctx, uuid.Nil, v1.ID); err != nil {
		t.Errorf("delete old version: %v", err)
	}
	if err := svc.Activate(ctx, uuid.Nil, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("activate missing err = %v", err)
	}
}
