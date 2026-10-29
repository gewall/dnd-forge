package character

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	chars map[uuid.UUID]Character
}

func newFakeStore() *fakeStore { return &fakeStore{chars: map[uuid.UUID]Character{}} }

func (f *fakeStore) CharactersByUser(_ context.Context, userID uuid.UUID) ([]Character, error) {
	var out []Character
	for _, c := range f.chars {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeStore) CharacterByID(_ context.Context, id uuid.UUID) (Character, error) {
	c, ok := f.chars[id]
	if !ok {
		return Character{}, ErrNotFound
	}
	return c, nil
}

func (f *fakeStore) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	cs, _ := f.CharactersByUser(ctx, userID)
	return len(cs), nil
}

func (f *fakeStore) CreateCharacter(_ context.Context, userID uuid.UUID, d Draft) (uuid.UUID, error) {
	id := uuid.New()
	f.chars[id] = fromDraft(id, userID, d)
	return id, nil
}

func (f *fakeStore) UpdateCharacter(_ context.Context, id, userID uuid.UUID, d Draft) error {
	c, ok := f.chars[id]
	if !ok || c.UserID != userID {
		return ErrNotFound
	}
	f.chars[id] = fromDraft(id, userID, d)
	return nil
}

func (f *fakeStore) DeleteCharacter(_ context.Context, id, userID uuid.UUID) error {
	c, ok := f.chars[id]
	if !ok || c.UserID != userID {
		return ErrNotFound
	}
	delete(f.chars, id)
	return nil
}

func fromDraft(id, userID uuid.UUID, d Draft) Character {
	c := Character{
		ID: id, UserID: userID, Name: d.Name, Gender: d.Gender, Appearance: d.Appearance, IsPublic: d.IsPublic,
		Race: Ref{ID: d.RaceID}, Class: Ref{ID: d.ClassID}, Background: Ref{ID: d.BackgroundID},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if d.SubraceID != nil {
		c.Subrace = &Ref{ID: *d.SubraceID}
	}
	return c
}

// fakeCatalog: elf has an active (high) and an inactive (drow) subrace,
// human has none, orc is inactive.
type fakeCatalog struct {
	elf, highElf, drow, human, orc, wizard, oldClass, sage uuid.UUID
}

func newFakeCatalog() fakeCatalog {
	return fakeCatalog{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
}

func (c fakeCatalog) Race(_ context.Context, id uuid.UUID) (RaceOption, error) {
	switch id {
	case c.elf:
		return RaceOption{Option: Option{ID: id, Active: true}, Subraces: []Option{{ID: c.highElf, Active: true}, {ID: c.drow}}}, nil
	case c.human:
		return RaceOption{Option: Option{ID: id, Active: true}}, nil
	case c.orc:
		return RaceOption{Option: Option{ID: id}}, nil
	}
	return RaceOption{}, ErrInvalidRace
}

func (c fakeCatalog) Class(_ context.Context, id uuid.UUID) (Option, error) {
	switch id {
	case c.wizard:
		return Option{ID: id, Active: true}, nil
	case c.oldClass:
		return Option{ID: id}, nil
	}
	return Option{}, ErrInvalidClass
}

func (c fakeCatalog) Background(_ context.Context, id uuid.UUID) (Option, error) {
	if id == c.sage {
		return Option{ID: id, Active: true}, nil
	}
	return Option{}, ErrInvalidBkg
}

func ptr(id uuid.UUID) *uuid.UUID { return &id }

func TestCreateValidatesChoices(t *testing.T) {
	ctx := context.Background()
	cat := newFakeCatalog()
	svc := NewService(newFakeStore(), cat)
	user := uuid.New()

	valid := Draft{Name: " Lia ", RaceID: cat.elf, SubraceID: ptr(cat.highElf), ClassID: cat.wizard, BackgroundID: cat.sage}

	tests := []struct {
		name   string
		change func(*Draft)
		want   error
	}{
		{"valid", func(*Draft) {}, nil},
		{"race without subraces", func(d *Draft) { d.RaceID, d.SubraceID = cat.human, nil }, nil},
		{"unknown race", func(d *Draft) { d.RaceID = uuid.New() }, ErrInvalidRace},
		{"inactive race", func(d *Draft) { d.RaceID, d.SubraceID = cat.orc, nil }, ErrInvalidRace},
		{"missing subrace", func(d *Draft) { d.SubraceID = nil }, ErrSubraceRequired},
		{"inactive subrace", func(d *Draft) { d.SubraceID = ptr(cat.drow) }, ErrInvalidSubrace},
		{"subrace of another race", func(d *Draft) { d.SubraceID = ptr(uuid.New()) }, ErrInvalidSubrace},
		{"subrace on race without any", func(d *Draft) { d.RaceID = cat.human }, ErrSubraceForbidden},
		{"inactive class", func(d *Draft) { d.ClassID = cat.oldClass }, ErrInvalidClass},
		{"unknown background", func(d *Draft) { d.BackgroundID = uuid.New() }, ErrInvalidBkg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := valid
			tt.change(&d)
			c, err := svc.Create(ctx, user, d)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if err == nil && (c.Name != "Lia" || c.UserID != user) {
				t.Errorf("created %+v", c)
			}
		})
	}
}

func TestOwnership(t *testing.T) {
	ctx := context.Background()
	cat := newFakeCatalog()
	svc := NewService(newFakeStore(), cat)
	owner, stranger := uuid.New(), uuid.New()
	d := Draft{Name: "Bram", RaceID: cat.human, ClassID: cat.wizard, BackgroundID: cat.sage}

	c, err := svc.Create(ctx, owner, d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, stranger, c.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger Get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Update(ctx, stranger, c.ID, d); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger Update err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, stranger, c.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger Delete err = %v, want ErrNotFound", err)
	}
	if list, _ := svc.List(ctx, stranger); len(list) != 0 {
		t.Errorf("stranger sees %d characters", len(list))
	}

	d.Name = "Bram Baru"
	if got, err := svc.Update(ctx, owner, c.ID, d); err != nil || got.Name != "Bram Baru" {
		t.Errorf("owner Update = %+v, %v", got, err)
	}
	if err := svc.Delete(ctx, owner, c.ID); err != nil {
		t.Errorf("owner Delete: %v", err)
	}
}

func TestLimitPerUser(t *testing.T) {
	ctx := context.Background()
	cat := newFakeCatalog()
	svc := NewService(newFakeStore(), cat)
	user := uuid.New()
	d := Draft{Name: "x", RaceID: cat.human, ClassID: cat.wizard, BackgroundID: cat.sage}

	for i := range MaxPerUser {
		if _, err := svc.Create(ctx, user, d); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if _, err := svc.Create(ctx, user, d); !errors.Is(err, ErrLimitReached) {
		t.Errorf("over limit err = %v, want ErrLimitReached", err)
	}
	if _, err := svc.Create(ctx, uuid.New(), d); err != nil {
		t.Errorf("other user blocked by limit: %v", err)
	}
}
