//go:build integration

package lore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestPostgresLore(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "races", "classes", "backgrounds", "ai_providers", "audit_logs")

	game := gamedata.NewService(gamedata.NewPostgresStore(pool))
	if _, err := game.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	users := auth.NewPostgresStore(pool)
	owner, err := users.CreateUser(ctx, auth.NewUser{Email: "bard@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := users.CreateUser(ctx, auth.NewUser{Email: "other@example.com", Role: identity.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	human, _ := game.Race(ctx, "human")
	bard, _ := game.Class(ctx, "bard")
	sage, _ := game.Background(ctx, "sage")
	chars := character.NewPostgresStore(pool)
	charID, err := chars.CreateCharacter(ctx, owner.ID, character.Draft{Name: "Ravi", Gender: "pria", RaceID: human.ID, ClassID: bard.ID, BackgroundID: sage.ID})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := aiprovider.NewPostgresStore(pool).CreateProvider(ctx,
		aiprovider.Input{Type: ai.DeepSeek, Capability: ai.Text, Label: "DeepSeek", DefaultModel: "deepseek-v4-flash"}, []byte{0}, "abcd")
	if err != nil {
		t.Fatal(err)
	}

	jobs := &recordedJobs{}
	store := NewPostgresStore(pool, jobs)

	subj, err := store.Subject(ctx, charID)
	if err != nil || subj.UserID != owner.ID || subj.Name != "Ravi" || subj.Class != bard.Name || subj.Subrace != "" || subj.Gender != "pria" {
		t.Errorf("Subject = %+v, %v", subj, err)
	}
	if _, err := store.Subject(ctx, uuid.New()); !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("unknown character err = %v", err)
	}
	if limit, err := store.DailyLimit(ctx, owner.ID); err != nil || limit != 10 {
		t.Errorf("DailyLimit = %d, %v; want 10 from app_settings", limit, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET daily_lore_quota = 3 WHERE id = $1", stranger.ID); err != nil {
		t.Fatal(err)
	}
	if limit, err := store.DailyLimit(ctx, stranger.ID); err != nil || limit != 3 {
		t.Errorf("per-user DailyLimit = %d, %v; want 3", limit, err)
	}

	create := func() Lore {
		t.Helper()
		var l Lore
		err := store.InTx(ctx, func(tx TxStore) error {
			if err := tx.LockUser(ctx, owner.ID); err != nil {
				return err
			}
			var err error
			l, err = tx.CreateLore(ctx, NewLore{
				CharacterID: charID, UserID: owner.ID, ProviderID: provider.ID, Model: "deepseek-v4-flash",
				UserPrompt: "yatim", FinalPrompt: "write", Tone: "dark", Length: "short", Language: "id",
			})
			if err != nil {
				return err
			}
			return tx.EnqueueGenerate(ctx, GenerateArgs{LoreID: l.ID})
		})
		if err != nil {
			t.Fatal(err)
		}
		return l
	}
	first, second := create(), create()
	if first.Status != StatusPending || first.Language != "id" || len(jobs.args) != 2 {
		t.Errorf("created %+v with %d jobs", first, len(jobs.args))
	}

	dayStart := time.Now().Add(-time.Hour)
	if err := store.Fail(ctx, second.ID, "gagal"); err != nil {
		t.Fatal(err)
	}
	if n, err := store.CountSince(ctx, owner.ID, dayStart); err != nil || n != 1 {
		t.Errorf("CountSince = %d, %v; FAILED must not count", n, err)
	}

	if err := store.Complete(ctx, first.ID, "Ravi bernyanyi."); err != nil {
		t.Fatal(err)
	}
	// Complete and Fail only touch PENDING drafts.
	if err := store.Complete(ctx, second.ID, "late"); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(ctx, first.ID, "late"); err != nil {
		t.Fatal(err)
	}
	done, err := store.LoreByID(ctx, first.ID)
	if err != nil || done.Status != StatusSucceeded || done.Content != "Ravi bernyanyi." || done.ErrorMessage != "" || done.CompletedAt == nil {
		t.Errorf("completed draft = %+v, %v", done, err)
	}
	if failed, _ := store.LoreByID(ctx, second.ID); failed.Status != StatusFailed || failed.Content != "" {
		t.Errorf("failed draft = %+v", failed)
	}

	list, err := store.LoresOf(ctx, charID, owner.ID)
	if err != nil || len(list) != 2 || list[0].ID != second.ID {
		t.Errorf("LoresOf = %d items, %v; want newest first", len(list), err)
	}
	if list, _ := store.LoresOf(ctx, charID, stranger.ID); len(list) != 0 {
		t.Errorf("stranger sees %d drafts", len(list))
	}

	if err := store.SetBackstory(ctx, charID, stranger.ID, "x"); !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("stranger SetBackstory err = %v", err)
	}
	if err := store.SetBackstory(ctx, charID, owner.ID, done.Content); err != nil {
		t.Fatal(err)
	}
	c, err := chars.CharacterByID(ctx, charID)
	if err != nil || c.Backstory != "Ravi bernyanyi." {
		t.Errorf("backstory = %q, %v", c.Backstory, err)
	}

	// Deleting the character keeps its drafts, so they still count.
	if err := chars.DeleteCharacter(ctx, charID, owner.ID); err != nil {
		t.Fatal(err)
	}
	orphan, err := store.LoreByID(ctx, first.ID)
	if err != nil || orphan.CharacterID != nil {
		t.Errorf("draft after character delete = %+v, %v", orphan, err)
	}
	if n, _ := store.CountSince(ctx, owner.ID, dayStart); n != 1 {
		t.Errorf("CountSince after character delete = %d, want 1", n)
	}
}
