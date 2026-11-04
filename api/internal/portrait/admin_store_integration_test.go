//go:build integration

package portrait

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/character"
	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/gamedata"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

func TestPostgresAdminImages(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "races", "classes", "backgrounds", "ai_providers", "audit_logs")

	game := gamedata.NewService(gamedata.NewPostgresStore(pool))
	if _, err := game.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	owner, err := auth.NewPostgresStore(pool).CreateUser(ctx, auth.NewUser{Email: "mod@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	human, _ := game.Race(ctx, "human")
	wizard, _ := game.Class(ctx, "wizard")
	sage, _ := game.Background(ctx, "sage")
	charID, err := character.NewPostgresStore(pool).CreateCharacter(ctx, owner.ID, character.Draft{
		Name: "Lia", RaceID: human.ID, ClassID: wizard.ID, BackgroundID: sage.ID, IsPublic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := aiprovider.NewPostgresStore(pool).CreateProvider(ctx,
		aiprovider.Input{Type: ai.Fake, Capability: ai.Image, Label: "Fake", DefaultModel: "fake-1"}, []byte{0}, "abcd")
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresStore(pool, &recordedJobs{})
	create := func() Portrait {
		t.Helper()
		var p Portrait
		err := store.InTx(ctx, func(tx TxStore) error {
			var err error
			p, err = tx.CreatePortrait(ctx, NewPortrait{
				CharacterID: charID, UserID: owner.ID, ProviderID: provider.ID, Model: "fake-1",
				UserPrompt: "naga", FinalPrompt: "x", Style: "anime", AspectRatio: "1:1",
			})
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	done, failed, pending := create(), create(), create()
	if err := store.Complete(ctx, done.ID, "portraits/u/a.webp", "portraits/u/a_thumb.webp"); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(ctx, failed.ID, "gagal"); err != nil {
		t.Fatal(err)
	}

	objects := &memObjects{}
	admin := NewAdmin(NewPostgresAdminStore(pool), objects)

	all, err := admin.List(ctx, AdminFilter{})
	if err != nil || len(all) != 3 || all[0].ID != pending.ID {
		t.Fatalf("List = %d items, %v; want 3 newest first", len(all), err)
	}
	succeeded, _ := admin.List(ctx, AdminFilter{Status: StatusSucceeded})
	if len(succeeded) != 1 || succeeded[0].UserEmail != owner.Email || succeeded[0].CharacterName != "Lia" ||
		!succeeded[0].CharacterPublic || succeeded[0].ImageURL == "" {
		t.Errorf("succeeded = %+v", succeeded)
	}
	stranger := uuid.New()
	if none, _ := admin.List(ctx, AdminFilter{UserID: &stranger}); len(none) != 0 {
		t.Errorf("other user's filter = %d items", len(none))
	}
	if page, _ := admin.List(ctx, AdminFilter{Before: &all[0].CreatedAt, Limit: 1}); len(page) != 1 || page[0].ID != all[1].ID {
		t.Errorf("second page = %+v", page)
	}

	img, err := admin.SetHidden(ctx, uuid.Nil, done.ID, true)
	if err != nil || !img.IsHidden {
		t.Fatalf("hide = %+v, %v", img, err)
	}
	hidden := true
	if list, _ := admin.List(ctx, AdminFilter{Hidden: &hidden}); len(list) != 1 || list[0].ID != done.ID {
		t.Errorf("hidden filter = %+v", list)
	}

	if err := admin.DeleteFiles(ctx, uuid.Nil, pending.ID); !errors.Is(err, ErrStillPending) {
		t.Errorf("delete pending err = %v", err)
	}
	if err := admin.DeleteFiles(ctx, uuid.Nil, done.ID); err != nil {
		t.Fatal(err)
	}
	gone, err := admin.Get(ctx, done.ID)
	if err != nil || gone.HasFiles() || !gone.IsHidden || gone.ImageURL != "" || len(objects.deleted) != 2 {
		t.Errorf("after delete = %+v, %v; deleted %v", gone, err, objects.deleted)
	}
	if _, err := admin.Get(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown image err = %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE entity = 'generated_image' AND entity_id = $1",
		done.ID.String()).Scan(&n); err != nil || n != 2 {
		t.Errorf("audit entries = %d, %v; want hide + delete", n, err)
	}
}
