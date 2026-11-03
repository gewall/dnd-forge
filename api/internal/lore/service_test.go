package lore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/prompt"
)

// memStore keeps drafts in memory and serves as both Store and TxStore.
type memStore struct {
	subjects  map[uuid.UUID]Subject
	lores     map[uuid.UUID]Lore
	limit     int
	used      int
	since     time.Time
	locked    int
	jobs      []GenerateArgs
	backstory map[uuid.UUID]string
}

func newMemStore() *memStore {
	return &memStore{subjects: map[uuid.UUID]Subject{}, lores: map[uuid.UUID]Lore{}, limit: 10, backstory: map[uuid.UUID]string{}}
}

func (m *memStore) Subject(_ context.Context, id uuid.UUID) (Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return Subject{}, ErrCharacterNotFound
	}
	return s, nil
}

func (m *memStore) DailyLimit(context.Context) (int, error) { return m.limit, nil }

func (m *memStore) CountSince(_ context.Context, _ uuid.UUID, since time.Time) (int, error) {
	m.since = since
	return m.used, nil
}

func (m *memStore) LoreByID(_ context.Context, id uuid.UUID) (Lore, error) {
	l, ok := m.lores[id]
	if !ok {
		return Lore{}, ErrNotFound
	}
	return l, nil
}

func (m *memStore) LoresOf(context.Context, uuid.UUID, uuid.UUID) ([]Lore, error) { return nil, nil }

func (m *memStore) Complete(_ context.Context, id uuid.UUID, content string) error {
	l := m.lores[id]
	l.Status, l.Content = StatusSucceeded, content
	m.lores[id] = l
	return nil
}

func (m *memStore) Fail(_ context.Context, id uuid.UUID, msg string) error {
	l := m.lores[id]
	l.Status, l.ErrorMessage = StatusFailed, msg
	m.lores[id] = l
	return nil
}

func (m *memStore) SetBackstory(_ context.Context, characterID, userID uuid.UUID, text string) error {
	s, ok := m.subjects[characterID]
	if !ok || s.UserID != userID {
		return ErrCharacterNotFound
	}
	m.backstory[characterID] = text
	return nil
}

func (m *memStore) InTx(_ context.Context, fn func(TxStore) error) error { return fn(m) }

func (m *memStore) LockUser(context.Context, uuid.UUID) error {
	m.locked++
	return nil
}

func (m *memStore) CreateLore(_ context.Context, n NewLore) (Lore, error) {
	l := Lore{
		ID: uuid.New(), CharacterID: &n.CharacterID, UserID: n.UserID, ProviderID: n.ProviderID, Model: n.Model,
		UserPrompt: n.UserPrompt, FinalPrompt: n.FinalPrompt, Tone: n.Tone, Length: n.Length, Language: n.Language,
		Status: StatusPending,
	}
	m.lores[l.ID] = l
	return l, nil
}

func (m *memStore) EnqueueGenerate(_ context.Context, args GenerateArgs) error {
	m.jobs = append(m.jobs, args)
	return nil
}

type fakeProviders struct {
	activeErr error
	text      *fakeText
}

func (f fakeProviders) Active(_ context.Context, c ai.Capability) (aiprovider.Selected, error) {
	if c != ai.Text {
		return aiprovider.Selected{}, errors.New("lore asked for a non-text provider")
	}
	return aiprovider.Selected{ID: providerID, Model: "ds-1"}, f.activeErr
}

func (f fakeProviders) ResolveText(context.Context, uuid.UUID) (ai.TextProvider, string, error) {
	return f.text, "sk-ds", nil
}

type fakeText struct {
	reply string
	err   error
	calls int
	got   ai.TextInput
}

func (*fakeText) Type() ai.ProviderType                        { return ai.Fake }
func (*fakeText) TestConnection(context.Context, string) error { return nil }
func (f *fakeText) Complete(_ context.Context, in ai.TextInput) (string, error) {
	f.calls++
	f.got = in
	return f.reply, f.err
}

type fakeTemplates struct{}

func (fakeTemplates) Active(_ context.Context, k prompt.Kind) (prompt.Template, error) {
	if k != prompt.KindLore {
		return prompt.Template{}, prompt.ErrNoActive
	}
	return prompt.Template{
		Template:       "Story of {name}, a {race} {class}.\nSubrace: {subrace}.\nWishes: {userPrompt}.\nTone: {tone}. Length: {length}. Write in {language}.",
		NegativePrompt: "gore",
	}, nil
}

type fakeModerator struct {
	err     error
	maxSeen int
}

func (f *fakeModerator) CleanUpTo(_ context.Context, s string, maxRunes int) (string, error) {
	f.maxSeen = maxRunes
	return strings.TrimSpace(s), f.err
}

var (
	providerID  = uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	ownerID     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	characterID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
)

type fixture struct {
	svc   *Service
	store *memStore
	text  *fakeText
	mod   *fakeModerator
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store := newMemStore()
	store.subjects[characterID] = Subject{ID: characterID, UserID: ownerID, Name: "Lia", Race: "Elf", Class: "Wizard", Background: "Sage"}
	text := &fakeText{reply: "  Lia lahir di hutan.  "}
	mod := &fakeModerator{}
	svc := NewService(Deps{
		Store: store, Providers: fakeProviders{text: text}, Templates: fakeTemplates{}, Moderator: mod,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	// 23:30 WIB on 29 Oct 2026.
	svc.now = func() time.Time { return time.Date(2026, 10, 29, 16, 30, 0, 0, time.UTC) }
	return fixture{svc: svc, store: store, text: text, mod: mod}
}

func TestRequest(t *testing.T) {
	f := newFixture(t)
	l, err := f.svc.Request(context.Background(), ownerID, characterID, Request{Tone: "dark", UserPrompt: " yatim piatu "})
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != StatusPending || l.Length != DefaultLength || l.Language != DefaultLanguage || l.Model != "ds-1" || l.UserPrompt != "yatim piatu" {
		t.Errorf("lore = %+v", l)
	}
	want := "Story of Lia, a Elf Wizard.\nWishes: yatim piatu.\nTone: dark and grim. Length: about 300-400 words. Write in Indonesian (Bahasa Indonesia).\nAvoid: gore."
	if l.FinalPrompt != want {
		t.Errorf("final prompt =\n%s\nwant\n%s", l.FinalPrompt, want)
	}
	if f.mod.maxSeen != MaxPromptRunes || f.store.locked != 1 || len(f.store.jobs) != 1 || f.store.jobs[0].LoreID != l.ID {
		t.Errorf("moderation max %d, locks %d, jobs %+v", f.mod.maxSeen, f.store.locked, f.store.jobs)
	}
	if want := time.Date(2026, 10, 28, 17, 0, 0, 0, time.UTC); !f.store.since.Equal(want) {
		t.Errorf("counted since %v, want %v (midnight WIB)", f.store.since, want)
	}
}

var errBlocked = errors.New("blocked word")

func TestRequestRejections(t *testing.T) {
	tests := []struct {
		name   string
		user   uuid.UUID
		req    Request
		mod    error
		used   int
		noText bool
		want   error
	}{
		{"unknown tone", ownerID, Request{Tone: "cheerful"}, nil, 0, false, ErrInvalidTone},
		{"bad length", ownerID, Request{Tone: "epic", Length: "epic"}, nil, 0, false, ErrInvalidLength},
		{"bad language", ownerID, Request{Tone: "epic", Language: "fr"}, nil, 0, false, ErrInvalidLanguage},
		{"someone else's character", uuid.New(), Request{Tone: "epic"}, nil, 0, false, ErrCharacterNotFound},
		{"moderated wishes", ownerID, Request{Tone: "epic"}, errBlocked, 0, false, errBlocked},
		{"no text provider", ownerID, Request{Tone: "epic"}, nil, 0, true, aiprovider.ErrNoTextProvider},
		{"quota used up", ownerID, Request{Tone: "epic"}, nil, 10, false, ErrQuotaExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.mod.err = tt.mod
			f.store.used = tt.used
			if tt.noText {
				f.svc.Providers = fakeProviders{activeErr: aiprovider.ErrNoTextProvider, text: f.text}
			}
			if _, err := f.svc.Request(context.Background(), tt.user, characterID, tt.req); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(f.store.lores) != 0 || len(f.store.jobs) != 0 {
				t.Errorf("rejected request left %d drafts and %d jobs", len(f.store.lores), len(f.store.jobs))
			}
		})
	}
}

func pending(f fixture, length string) Lore {
	l := Lore{ID: uuid.New(), CharacterID: ptr(characterID), UserID: ownerID, ProviderID: providerID, Model: "ds-1",
		FinalPrompt: "write", Length: length, Status: StatusPending}
	f.store.lores[l.ID] = l
	return l
}

func ptr[T any](v T) *T { return &v }

func TestGenerate(t *testing.T) {
	f := newFixture(t)
	l := pending(f, "long")
	if err := f.svc.Generate(context.Background(), GenerateArgs{LoreID: l.ID}, false); err != nil {
		t.Fatal(err)
	}
	done := f.store.lores[l.ID]
	if done.Status != StatusSucceeded || done.Content != "Lia lahir di hutan." {
		t.Errorf("draft = %+v", done)
	}
	in := f.text.got
	if in.APIKey != "sk-ds" || in.Model != "ds-1" || in.MaxTokens != Lengths["long"].maxTokens || in.System == "" || in.Prompt != "write" {
		t.Errorf("provider input = %+v", in)
	}
}

func TestGenerateTruncatesHugeReplies(t *testing.T) {
	f := newFixture(t)
	f.text.reply = strings.Repeat("é", maxContentRunes+10)
	l := pending(f, "short")
	if err := f.svc.Generate(context.Background(), GenerateArgs{LoreID: l.ID}, false); err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(f.store.lores[l.ID].Content)); n != maxContentRunes {
		t.Errorf("stored %d runes, want %d", n, maxContentRunes)
	}
}

func TestGenerateFailures(t *testing.T) {
	transient := errors.New("connection reset")
	tests := []struct {
		name      string
		err       error
		last      bool
		permanent bool
		message   string
	}{
		{"rejected by provider", ai.ErrRejected, false, true, msgRejected},
		{"no balance", ai.ErrBilling, false, true, msgConfig},
		{"bad key", ai.ErrBadKey, false, true, msgConfig},
		{"transient, will retry", transient, false, false, ""},
		{"transient on last attempt", transient, true, true, msgGaveUp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.text.err = tt.err
			l := pending(f, "medium")
			err := f.svc.Generate(context.Background(), GenerateArgs{LoreID: l.ID}, tt.last)
			if !errors.Is(err, tt.err) || errors.Is(err, errPermanent) != tt.permanent {
				t.Errorf("err = %v; want it to wrap %v, permanent %v", err, tt.err, tt.permanent)
			}
			if got := f.store.lores[l.ID].ErrorMessage; got != tt.message {
				t.Errorf("failure message = %q, want %q", got, tt.message)
			}
		})
	}
}

func TestGenerateSkipsFinishedAndMissing(t *testing.T) {
	f := newFixture(t)
	l := pending(f, "short")
	l.Status = StatusFailed
	f.store.lores[l.ID] = l
	if err := f.svc.Generate(context.Background(), GenerateArgs{LoreID: l.ID}, false); err != nil || f.text.calls != 0 {
		t.Errorf("finished draft: err %v, provider calls %d", err, f.text.calls)
	}
	if err := f.svc.Generate(context.Background(), GenerateArgs{LoreID: uuid.New()}, false); !errors.Is(err, errPermanent) {
		t.Errorf("missing draft err = %v, want permanent", err)
	}
}

func TestApply(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	l := pending(f, "short")

	if err := f.svc.Apply(ctx, ownerID, l.ID); !errors.Is(err, ErrNotReady) {
		t.Errorf("apply pending err = %v, want ErrNotReady", err)
	}
	if err := f.svc.Generate(ctx, GenerateArgs{LoreID: l.ID}, false); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Apply(ctx, uuid.New(), l.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger apply err = %v, want ErrNotFound", err)
	}
	if err := f.svc.Apply(ctx, ownerID, l.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.store.backstory[characterID]; got != "Lia lahir di hutan." {
		t.Errorf("backstory = %q", got)
	}

	orphan := f.store.lores[l.ID]
	orphan.CharacterID = nil // the character was deleted
	f.store.lores[l.ID] = orphan
	if err := f.svc.Apply(ctx, ownerID, l.ID); !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("apply orphan err = %v, want ErrCharacterNotFound", err)
	}
}

func TestQuota(t *testing.T) {
	f := newFixture(t)
	f.store.used = 4
	q, err := f.svc.Quota(context.Background(), ownerID)
	if err != nil || q.Limit != 10 || q.Used != 4 || q.Remaining != 6 || !q.ResetsAt.Equal(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)) {
		t.Errorf("Quota = %+v, %v", q, err)
	}
}

type generatorFunc func() error

func (g generatorFunc) Generate(context.Context, GenerateArgs, bool) error { return g() }

func TestWorkerCancelsPermanentFailures(t *testing.T) {
	job := &river.Job[GenerateArgs]{JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 3}}
	err := NewWorker(generatorFunc(func() error { return errPermanent })).Work(context.Background(), job)
	if _, ok := errors.AsType[*rivertype.JobCancelError](err); !ok {
		t.Errorf("permanent failure: err = %v, want JobCancel", err)
	}
	transient := errors.New("timeout")
	err = NewWorker(generatorFunc(func() error { return transient })).Work(context.Background(), job)
	if _, ok := errors.AsType[*rivertype.JobCancelError](err); ok || !errors.Is(err, transient) {
		t.Errorf("transient failure: err = %v", err)
	}
}

func TestOptionsAreSorted(t *testing.T) {
	tones := sortedOptions(Tones, func(o Option) Option { return o })
	if len(tones) != len(Tones) || tones[0].ID != "dark" || tones[0].Description == "" {
		t.Errorf("tones = %+v", tones)
	}
}
