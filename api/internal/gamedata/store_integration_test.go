//go:build integration

package gamedata

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func newIntegrationService(t *testing.T) (*Service, *PostgresStore) {
	t.Helper()
	pool := dbtest.Pool(t, "races", "classes", "backgrounds", "audit_logs")
	store := NewPostgresStore(pool)
	return NewService(store), store
}

func TestPostgresRaces(t *testing.T) {
	ctx := context.Background()
	svc, _ := newIntegrationService(t)

	in := RaceInput{Slug: "dwarf", Name: "Dwarf", Traits: []Trait{{Name: "Darkvision", Description: "60 ft"}}, IsActive: true}
	r, err := svc.CreateRace(ctx, uuid.Nil, in)
	if err != nil {
		t.Fatalf("CreateRace: %v", err)
	}
	if _, err := svc.CreateRace(ctx, uuid.Nil, in); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("duplicate slug err = %v, want ErrSlugTaken", err)
	}

	for _, sr := range []SubraceInput{
		{Slug: "hill", Name: "Hill Dwarf", IsActive: true},
		{Slug: "mountain", Name: "Mountain Dwarf", IsActive: false},
	} {
		if _, err := svc.CreateSubrace(ctx, uuid.Nil, r.ID, sr); err != nil {
			t.Fatalf("CreateSubrace: %v", err)
		}
	}
	if _, err := svc.CreateSubrace(ctx, uuid.Nil, uuid.New(), SubraceInput{Slug: "x", Name: "x"}); !errors.Is(err, ErrRaceNotFound) {
		t.Errorf("subrace of missing race err = %v, want ErrRaceNotFound", err)
	}

	pub, err := svc.Race(ctx, "dwarf")
	if err != nil {
		t.Fatal(err)
	}
	if len(pub.Traits) != 1 || pub.Traits[0].Description != "60 ft" {
		t.Errorf("traits round trip = %+v", pub.Traits)
	}
	if len(pub.Subraces) != 1 || pub.Subraces[0].Slug != "hill" {
		t.Errorf("public subraces = %+v", pub.Subraces)
	}

	list, err := svc.Races(ctx)
	if err != nil || len(list) != 1 || len(list[0].Subraces) != 1 {
		t.Errorf("Races = %+v, %v", list, err)
	}
	all, _ := svc.AllRaces(ctx)
	if len(all[0].Subraces) != 2 {
		t.Errorf("admin subraces = %d, want 2", len(all[0].Subraces))
	}

	if err := svc.DeleteRace(ctx, uuid.Nil, r.ID); err != nil {
		t.Fatalf("DeleteRace: %v", err)
	}
	if err := svc.DeleteRace(ctx, uuid.Nil, r.ID); !errors.Is(err, ErrRaceNotFound) {
		t.Errorf("second delete err = %v, want ErrRaceNotFound", err)
	}
}

func TestPostgresClassesAndBackgrounds(t *testing.T) {
	ctx := context.Background()
	svc, _ := newIntegrationService(t)

	c, err := svc.CreateClass(ctx, uuid.Nil, ClassInput{
		Slug: "wizard", Name: "Wizard", HitDie: 6, PrimaryAbility: "Intelligence",
		Features: []Trait{{Name: "Spellcasting"}}, ImageKey: "content/x.webp", IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Class(ctx, "wizard")
	if err != nil || got.ID != c.ID || got.ImageKey != "content/x.webp" || len(got.Features) != 1 {
		t.Errorf("Class = %+v, %v", got, err)
	}

	b, err := svc.CreateBackground(ctx, uuid.Nil, BackgroundInput{
		Slug: "sage", Name: "Sage", SkillProficiencies: []string{"Arcana", "History"}, IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	in := b.BackgroundInput
	in.IsActive = false
	if _, err := svc.UpdateBackground(ctx, uuid.Nil, b.ID, in); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Background(ctx, "sage"); !errors.Is(err, ErrBackgroundNotFound) {
		t.Errorf("inactive background err = %v", err)
	}
	all, _ := svc.AllBackgrounds(ctx)
	if len(all) != 1 || len(all[0].SkillProficiencies) != 2 {
		t.Errorf("AllBackgrounds = %+v", all)
	}
}

func TestPostgresAuditLog(t *testing.T) {
	ctx := context.Background()
	svc, store := newIntegrationService(t)

	r, err := svc.CreateRace(ctx, uuid.Nil, RaceInput{Slug: "orc", Name: "Orc", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	var action, entity, entityID string
	err = store.pool.QueryRow(ctx, "SELECT action, entity, entity_id FROM audit_logs ORDER BY id DESC LIMIT 1").
		Scan(&action, &entity, &entityID)
	if err != nil {
		t.Fatal(err)
	}
	if action != "create" || entity != entityRace || entityID != r.ID.String() {
		t.Errorf("audit row = %s %s %s", action, entity, entityID)
	}
}

func TestPostgresSeed(t *testing.T) {
	ctx := context.Background()
	svc, _ := newIntegrationService(t)

	rep, err := svc.Seed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Races != 9 || rep.Classes != 12 || rep.Backgrounds != 4 {
		t.Errorf("seed = %+v", rep)
	}
	elf, err := svc.Race(ctx, "elf")
	if err != nil || len(elf.Subraces) != 3 {
		t.Errorf("elf = %+v, %v", elf.Subraces, err)
	}
	again, err := svc.Seed(ctx)
	if err != nil || again.Races+again.Subraces+again.Classes+again.Backgrounds != 0 {
		t.Errorf("second seed = %+v, %v", again, err)
	}
}
