//go:build integration

package portrait

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/character"
	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/gamedata"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

type recordedJobs struct{ args []river.JobArgs }

func (r *recordedJobs) InsertTx(_ context.Context, _ pgx.Tx, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	r.args = append(r.args, args)
	return &rivertype.JobInsertResult{}, nil
}

func TestPostgresPortraits(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "races", "classes", "backgrounds", "ai_providers", "audit_logs")

	game := gamedata.NewService(gamedata.NewPostgresStore(pool))
	if _, err := game.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	owner, err := auth.NewPostgresStore(pool).CreateUser(ctx, auth.NewUser{Email: "painter@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	human, _ := game.Race(ctx, "human")
	wizard, _ := game.Class(ctx, "wizard")
	sage, _ := game.Background(ctx, "sage")
	charID, err := character.NewPostgresStore(pool).CreateCharacter(ctx, owner.ID, character.Draft{
		Name: "Lia", RaceID: human.ID, ClassID: wizard.ID, BackgroundID: sage.ID, Appearance: "silver hair", IsPublic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := aiprovider.NewPostgresStore(pool).CreateProvider(ctx,
		aiprovider.Input{Type: ai.Fake, Capability: ai.Image, Label: "Fake", DefaultModel: "fake-1"}, []byte{0}, "abcd")
	if err != nil {
		t.Fatal(err)
	}

	jobs := &recordedJobs{}
	store := NewPostgresStore(pool, jobs)

	subj, err := store.Subject(ctx, charID)
	if err != nil || subj.UserID != owner.ID || subj.Race != human.Name || subj.Class != wizard.Name || subj.Appearance != "silver hair" {
		t.Errorf("Subject = %+v, %v", subj, err)
	}
	if limit, err := store.DailyLimit(ctx, owner.ID); err != nil || limit != 5 {
		t.Errorf("default DailyLimit = %d, %v; want 5 from app_settings", limit, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET daily_quota = 2 WHERE id = $1", owner.ID); err != nil {
		t.Fatal(err)
	}
	if limit, err := store.DailyLimit(ctx, owner.ID); err != nil || limit != 2 {
		t.Errorf("per-user DailyLimit = %d, %v; want 2", limit, err)
	}

	create := func() Portrait {
		t.Helper()
		var p Portrait
		err := store.InTx(ctx, func(tx TxStore) error {
			if err := tx.LockUser(ctx, owner.ID); err != nil {
				return err
			}
			var err error
			p, err = tx.CreatePortrait(ctx, NewPortrait{
				CharacterID: charID, UserID: owner.ID, ProviderID: provider.ID, Model: "fake-1",
				FinalPrompt: "a human wizard", Style: "anime", AspectRatio: ai.Square,
			})
			if err != nil {
				return err
			}
			return tx.EnqueueGenerate(ctx, GenerateArgs{PortraitID: p.ID})
		})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	first, second := create(), create()
	if first.Status != StatusPending || len(jobs.args) != 2 {
		t.Errorf("created %+v with %d jobs", first, len(jobs.args))
	}

	// A failed transaction leaves no portrait behind.
	boom := errors.New("boom")
	err = store.InTx(ctx, func(tx TxStore) error {
		if _, err := tx.CreatePortrait(ctx, NewPortrait{CharacterID: charID, UserID: owner.ID, ProviderID: provider.ID,
			Model: "fake-1", FinalPrompt: "x", AspectRatio: ai.Square}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Errorf("InTx err = %v", err)
	}

	dayStart := time.Now().Add(-time.Hour)
	if n, err := store.CountSince(ctx, owner.ID, dayStart); err != nil || n != 2 {
		t.Errorf("CountSince = %d, %v; want 2 (rolled back insert excluded)", n, err)
	}
	if err := store.Fail(ctx, second.ID, "gagal"); err != nil {
		t.Fatal(err)
	}
	if n, _ := store.CountSince(ctx, owner.ID, dayStart); n != 1 {
		t.Errorf("CountSince after a failure = %d; FAILED must not count", n)
	}

	if items, _ := store.Gallery(ctx, nil, 10); len(items) != 0 {
		t.Errorf("gallery before any success = %+v", items)
	}
	if err := store.Complete(ctx, first.ID, "portraits/a.webp", "portraits/a_thumb.webp"); err != nil {
		t.Fatal(err)
	}
	// Fail only touches PENDING portraits, so a late failure cannot undo a success.
	if err := store.Fail(ctx, first.ID, "late"); err != nil {
		t.Fatal(err)
	}
	done, err := store.PortraitByID(ctx, first.ID)
	if err != nil || done.Status != StatusSucceeded || done.StorageKey != "portraits/a.webp" || done.CompletedAt == nil || done.ErrorMessage != "" {
		t.Errorf("completed portrait = %+v, %v", done, err)
	}

	items, err := store.Gallery(ctx, nil, 10)
	if err != nil || len(items) != 1 || items[0].PortraitID != first.ID || items[0].CharacterName != "Lia" || items[0].Class != wizard.Name {
		t.Errorf("Gallery = %+v, %v", items, err)
	}
	if items, _ := store.Gallery(ctx, &done.CreatedAt, 10); len(items) != 0 {
		t.Errorf("Gallery before the newest item = %+v", items)
	}
	if _, err := pool.Exec(ctx, "UPDATE characters SET is_public = false WHERE id = $1", charID); err != nil {
		t.Fatal(err)
	}
	if items, _ := store.Gallery(ctx, nil, 10); len(items) != 0 {
		t.Errorf("private character in gallery: %+v", items)
	}

	list, err := store.PortraitsOf(ctx, charID, owner.ID)
	if err != nil || len(list) != 2 || list[0].ID != second.ID {
		t.Errorf("PortraitsOf = %d items (first %v), %v; want newest first", len(list), list, err)
	}
	if _, err := store.PortraitByID(ctx, provider.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown portrait err = %v", err)
	}
}
