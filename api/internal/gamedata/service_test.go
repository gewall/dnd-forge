package gamedata

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
)

// fakeStore keeps game data in memory and records audit entries. Only races
// and subraces are fully modelled; classes and backgrounds embed the same
// pattern and are covered by the integration tests.
type fakeStore struct {
	races    map[uuid.UUID]Race
	classes  map[uuid.UUID]Class
	bgs      map[uuid.UUID]Background
	audits   []audit.Entry
	failNext error // returned by the next write, to test rollback
}

func newFakeStore() *fakeStore {
	return &fakeStore{races: map[uuid.UUID]Race{}, classes: map[uuid.UUID]Class{}, bgs: map[uuid.UUID]Background{}}
}

func (f *fakeStore) InTx(_ context.Context, fn func(Store) error) error {
	snapshot := len(f.audits)
	if err := fn(f); err != nil {
		f.audits = f.audits[:snapshot]
		return err
	}
	return nil
}

func (f *fakeStore) RecordAudit(_ context.Context, e audit.Entry) error {
	f.audits = append(f.audits, e)
	return nil
}

func (f *fakeStore) Races(_ context.Context, includeInactive bool) ([]Race, error) {
	var out []Race
	for _, r := range f.races {
		if includeInactive || r.IsActive {
			if !includeInactive {
				r.Subraces = activeSubraces(r.Subraces)
			}
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) RaceBySlug(_ context.Context, slug string) (Race, error) {
	for _, r := range f.races {
		if r.Slug == slug {
			return r, nil
		}
	}
	return Race{}, ErrRaceNotFound
}

func (f *fakeStore) RaceByID(_ context.Context, id uuid.UUID) (Race, error) {
	r, ok := f.races[id]
	if !ok {
		return Race{}, ErrRaceNotFound
	}
	return r, nil
}

func (f *fakeStore) CreateRace(_ context.Context, in RaceInput) (Race, error) {
	if err := f.takeFailure(); err != nil {
		return Race{}, err
	}
	for _, r := range f.races {
		if r.Slug == in.Slug {
			return Race{}, ErrSlugTaken
		}
	}
	r := Race{ID: uuid.New(), RaceInput: in, Subraces: []Subrace{}}
	f.races[r.ID] = r
	return r, nil
}

func (f *fakeStore) UpdateRace(_ context.Context, id uuid.UUID, in RaceInput) (Race, error) {
	r, ok := f.races[id]
	if !ok {
		return Race{}, ErrRaceNotFound
	}
	r.RaceInput = in
	f.races[id] = r
	return r, nil
}

func (f *fakeStore) DeleteRace(_ context.Context, id uuid.UUID) error {
	if _, ok := f.races[id]; !ok {
		return ErrRaceNotFound
	}
	delete(f.races, id)
	return nil
}

func (f *fakeStore) CreateSubrace(_ context.Context, raceID uuid.UUID, in SubraceInput) (Subrace, error) {
	r, ok := f.races[raceID]
	if !ok {
		return Subrace{}, ErrRaceNotFound
	}
	sr := Subrace{ID: uuid.New(), RaceID: raceID, SubraceInput: in}
	r.Subraces = append(r.Subraces, sr)
	f.races[raceID] = r
	return sr, nil
}

func (f *fakeStore) UpdateSubrace(context.Context, uuid.UUID, SubraceInput) (Subrace, error) {
	return Subrace{}, ErrSubraceNotFound
}
func (f *fakeStore) DeleteSubrace(context.Context, uuid.UUID) error { return ErrSubraceNotFound }

func (f *fakeStore) Classes(context.Context, bool) ([]Class, error) { return nil, nil }
func (f *fakeStore) ClassBySlug(_ context.Context, slug string) (Class, error) {
	for _, c := range f.classes {
		if c.Slug == slug {
			return c, nil
		}
	}
	return Class{}, ErrClassNotFound
}
func (f *fakeStore) ClassByID(context.Context, uuid.UUID) (Class, error) {
	return Class{}, ErrClassNotFound
}
func (f *fakeStore) CreateClass(_ context.Context, in ClassInput) (Class, error) {
	c := Class{ID: uuid.New(), ClassInput: in}
	f.classes[c.ID] = c
	return c, nil
}
func (f *fakeStore) UpdateClass(context.Context, uuid.UUID, ClassInput) (Class, error) {
	return Class{}, ErrClassNotFound
}
func (f *fakeStore) DeleteClass(context.Context, uuid.UUID) error { return ErrClassNotFound }
func (f *fakeStore) Backgrounds(context.Context, bool) ([]Background, error) {
	return nil, nil
}
func (f *fakeStore) BackgroundBySlug(_ context.Context, slug string) (Background, error) {
	for _, b := range f.bgs {
		if b.Slug == slug {
			return b, nil
		}
	}
	return Background{}, ErrBackgroundNotFound
}
func (f *fakeStore) BackgroundByID(context.Context, uuid.UUID) (Background, error) {
	return Background{}, ErrBackgroundNotFound
}
func (f *fakeStore) CreateBackground(_ context.Context, in BackgroundInput) (Background, error) {
	b := Background{ID: uuid.New(), BackgroundInput: in}
	f.bgs[b.ID] = b
	return b, nil
}
func (f *fakeStore) UpdateBackground(context.Context, uuid.UUID, BackgroundInput) (Background, error) {
	return Background{}, ErrBackgroundNotFound
}
func (f *fakeStore) DeleteBackground(context.Context, uuid.UUID) error { return ErrBackgroundNotFound }

func (f *fakeStore) takeFailure() error {
	err := f.failNext
	f.failNext = nil
	return err
}

var admin = uuid.New()

func elf() RaceInput {
	return RaceInput{Slug: "elf", Name: "Elf", Traits: []Trait{{Name: "Darkvision"}}, IsActive: true}
}

func TestCreateRaceWritesAudit(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store)

	r, err := svc.CreateRace(ctx, admin, elf())
	if err != nil {
		t.Fatal(err)
	}
	if len(store.audits) != 1 {
		t.Fatalf("audits = %d, want 1", len(store.audits))
	}
	got := store.audits[0]
	if got.ActorID != admin || got.Action != audit.ActionCreate || got.Entity != entityRace || got.EntityID != r.ID.String() {
		t.Errorf("audit entry = %+v", got)
	}

	if _, err := svc.CreateRace(ctx, admin, elf()); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("duplicate slug err = %v, want ErrSlugTaken", err)
	}
	if len(store.audits) != 1 {
		t.Errorf("failed write left an audit entry")
	}
}

func TestCreateRollsBackAuditOnFailure(t *testing.T) {
	store := newFakeStore()
	store.failNext = errors.New("db down")
	if _, err := NewService(store).CreateRace(context.Background(), admin, elf()); err == nil {
		t.Fatal("expected error")
	}
	if len(store.audits) != 0 {
		t.Errorf("audits = %d, want 0", len(store.audits))
	}
}

func TestValidation(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeStore())

	bad := elf()
	bad.Slug = "High Elf"
	if _, err := svc.CreateRace(ctx, admin, bad); !errors.Is(err, ErrInvalidSlug) {
		t.Errorf("race slug err = %v, want ErrInvalidSlug", err)
	}
	for _, die := range []int{0, 4, 20} {
		_, err := svc.CreateClass(ctx, admin, ClassInput{Slug: "wizard", Name: "Wizard", HitDie: die})
		if !errors.Is(err, ErrInvalidHitDie) {
			t.Errorf("hit die %d err = %v, want ErrInvalidHitDie", die, err)
		}
	}
	if _, err := svc.CreateClass(ctx, admin, ClassInput{Slug: "wizard", Name: "Wizard", HitDie: 6}); err != nil {
		t.Errorf("valid class: %v", err)
	}
}

func TestPublicReadsHideInactive(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRace(ctx, admin, elf())
	if _, err := svc.CreateSubrace(ctx, admin, r.ID, SubraceInput{Slug: "high-elf", Name: "High Elf", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSubrace(ctx, admin, r.ID, SubraceInput{Slug: "drow", Name: "Drow", IsActive: false}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Race(ctx, "elf")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Subraces) != 1 || got.Subraces[0].Slug != "high-elf" {
		t.Errorf("public subraces = %+v, want only high-elf", got.Subraces)
	}

	hidden := elf()
	hidden.IsActive = false
	if _, err := svc.UpdateRace(ctx, admin, r.ID, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Race(ctx, "elf"); !errors.Is(err, ErrRaceNotFound) {
		t.Errorf("inactive race err = %v, want ErrRaceNotFound", err)
	}
	if all, _ := svc.AllRaces(ctx); len(all) != 1 {
		t.Errorf("admin list = %d races, want 1", len(all))
	}
}

func TestUpdateRaceReturnsSubraces(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeStore())
	r, _ := svc.CreateRace(ctx, admin, elf())
	_, _ = svc.CreateSubrace(ctx, admin, r.ID, SubraceInput{Slug: "wood-elf", Name: "Wood Elf", IsActive: true})

	in := elf()
	in.Name = "Elf (2024)"
	got, err := svc.UpdateRace(ctx, admin, r.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Elf (2024)" || len(got.Subraces) != 1 {
		t.Errorf("UpdateRace = %+v", got)
	}
}

func TestSeedIsValidAndIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store)

	first, err := svc.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if first.Races != 9 || first.Classes != 12 || first.Backgrounds != 4 || first.Subraces == 0 {
		t.Errorf("first seed = %+v", first)
	}
	if first.Attribution == "" {
		t.Error("missing SRD attribution")
	}
	for _, r := range store.races {
		if r.VisualPrompt == "" || r.Description == "" || len(r.Traits) == 0 {
			t.Errorf("race %s is missing content", r.Slug)
		}
	}

	second, err := svc.Seed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.Races+second.Subraces+second.Classes+second.Backgrounds != 0 {
		t.Errorf("second seed created rows: %+v", second)
	}
}
