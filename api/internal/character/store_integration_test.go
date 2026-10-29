//go:build integration

package character

import (
	"context"
	"errors"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/gamedata"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

func TestPostgresCharacters(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "races", "classes", "backgrounds", "audit_logs")

	game := gamedata.NewService(gamedata.NewPostgresStore(pool))
	if _, err := game.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	owner, err := auth.NewPostgresStore(pool).CreateUser(ctx, auth.NewUser{Email: "builder@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewPostgresStore(pool), NewGameDataCatalog(game))

	elf, _ := game.Race(ctx, "elf")
	human, _ := game.Race(ctx, "human")
	wizard, _ := game.Class(ctx, "wizard")
	sage, _ := game.Background(ctx, "sage")
	wood := elf.Subraces[0].ID
	for _, sr := range elf.Subraces {
		if sr.Slug == "wood-elf" {
			wood = sr.ID
		}
	}

	draft := Draft{Name: "Lia", RaceID: elf.ID, ClassID: wizard.ID, BackgroundID: sage.ID, Appearance: "silver hair"}
	if _, err := svc.Create(ctx, owner.ID, draft); !errors.Is(err, ErrSubraceRequired) {
		t.Errorf("elf without lineage err = %v", err)
	}

	draft.SubraceID = &wood
	c, err := svc.Create(ctx, owner.ID, draft)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Race.Slug != "elf" || c.Subrace == nil || c.Subrace.Slug != "wood-elf" || c.Class.Name != "Wizard" || c.Background.Slug != "sage" {
		t.Errorf("resolved refs = %+v / %+v / %+v / %+v", c.Race, c.Subrace, c.Class, c.Background)
	}

	draft.RaceID, draft.SubraceID, draft.Name = human.ID, nil, "Lia Manusia"
	updated, err := svc.Update(ctx, owner.ID, c.ID, draft)
	if err != nil || updated.Race.Slug != "human" || updated.Subrace != nil || !updated.UpdatedAt.After(c.CreatedAt) {
		t.Errorf("Update = %+v, %v", updated, err)
	}

	list, err := svc.List(ctx, owner.ID)
	if err != nil || len(list) != 1 || list[0].Name != "Lia Manusia" {
		t.Errorf("List = %+v, %v", list, err)
	}

	// Game data in use cannot be deleted; admins deactivate it instead.
	if err := game.DeleteClass(ctx, owner.ID, wizard.ID); !errors.Is(err, gamedata.ErrInUse) {
		t.Errorf("delete used class err = %v, want ErrInUse", err)
	}

	if err := svc.Delete(ctx, owner.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, owner.ID, c.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete err = %v", err)
	}
}
