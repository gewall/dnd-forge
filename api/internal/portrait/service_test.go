package portrait

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
	"github.com/gewall/dnd-forge/api/internal/imaging"
	"github.com/gewall/dnd-forge/api/internal/prompt"
)

// memStore keeps portraits in memory and serves as both Store and TxStore.
type memStore struct {
	subjects  map[uuid.UUID]Subject
	portraits map[uuid.UUID]Portrait
	limit     int
	used      int // returned by CountSince
	since     time.Time
	locked    []uuid.UUID
	jobs      []GenerateArgs
	completed map[uuid.UUID][2]string
	failed    map[uuid.UUID]string
}

func newMemStore() *memStore {
	return &memStore{
		subjects: map[uuid.UUID]Subject{}, portraits: map[uuid.UUID]Portrait{}, limit: 5,
		completed: map[uuid.UUID][2]string{}, failed: map[uuid.UUID]string{},
	}
}

func (m *memStore) Subject(_ context.Context, id uuid.UUID) (Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return Subject{}, ErrCharacterNotFound
	}
	return s, nil
}

func (m *memStore) DailyLimit(context.Context, uuid.UUID) (int, error) { return m.limit, nil }

func (m *memStore) CountSince(_ context.Context, _ uuid.UUID, since time.Time) (int, error) {
	m.since = since
	return m.used, nil
}

func (m *memStore) PortraitByID(_ context.Context, id uuid.UUID) (Portrait, error) {
	p, ok := m.portraits[id]
	if !ok {
		return Portrait{}, ErrNotFound
	}
	return p, nil
}

func (m *memStore) PortraitsOf(context.Context, uuid.UUID, uuid.UUID) ([]Portrait, error) {
	return nil, nil
}

func (m *memStore) Gallery(context.Context, *time.Time, int) ([]GalleryItem, error) { return nil, nil }

func (m *memStore) Complete(_ context.Context, id uuid.UUID, full, thumb string) error {
	m.completed[id] = [2]string{full, thumb}
	return nil
}

func (m *memStore) Fail(_ context.Context, id uuid.UUID, msg string) error {
	m.failed[id] = msg
	return nil
}

func (m *memStore) InTx(_ context.Context, fn func(TxStore) error) error { return fn(m) }

func (m *memStore) LockUser(_ context.Context, id uuid.UUID) error {
	m.locked = append(m.locked, id)
	return nil
}

func (m *memStore) CreatePortrait(_ context.Context, n NewPortrait) (Portrait, error) {
	p := Portrait{
		ID: uuid.New(), CharacterID: &n.CharacterID, UserID: n.UserID, ProviderID: n.ProviderID, Model: n.Model,
		UserPrompt: n.UserPrompt, FinalPrompt: n.FinalPrompt, Style: n.Style, AspectRatio: n.AspectRatio,
		Status: StatusPending,
	}
	m.portraits[p.ID] = p
	return p, nil
}

func (m *memStore) EnqueueGenerate(_ context.Context, args GenerateArgs) error {
	m.jobs = append(m.jobs, args)
	return nil
}

type fakeProviders struct {
	activeErr error
	image     *fakeImage
}

func (f fakeProviders) Active(_ context.Context, c ai.Capability) (aiprovider.Selected, error) {
	if c != ai.Image {
		return aiprovider.Selected{}, errors.New("portrait asked for a non-image provider")
	}
	return aiprovider.Selected{ID: providerID, Model: "img-1"}, f.activeErr
}

func (f fakeProviders) ResolveImage(context.Context, uuid.UUID) (ai.ImageProvider, string, error) {
	return f.image, "sk-test", nil
}

type fakeImage struct {
	err   error
	calls int
	got   ai.GenerateInput
}

func (*fakeImage) Type() ai.ProviderType                        { return ai.Fake }
func (*fakeImage) TestConnection(context.Context, string) error { return nil }
func (f *fakeImage) Generate(_ context.Context, in ai.GenerateInput) (ai.GenerateOutput, error) {
	f.calls++
	f.got = in
	return ai.GenerateOutput{Data: []byte("png"), MIMEType: "image/png"}, f.err
}

type fakeTemplates struct{}

func (fakeTemplates) Active(_ context.Context, k prompt.Kind) (prompt.Template, error) {
	if k != prompt.KindPortrait {
		return prompt.Template{}, prompt.ErrNoActive
	}
	return prompt.Template{Template: "A {race} {class}. {userPrompt}. Style: {style}.", NegativePrompt: "text"}, nil
}

type fakeModerator struct{ err error }

func (f fakeModerator) Clean(_ context.Context, s string) (string, error) {
	return strings.TrimSpace(s), f.err
}

type fakeEncoder struct{}

func (fakeEncoder) Encode(_ context.Context, _ []byte, sizes ...imaging.Size) ([]imaging.Result, error) {
	out := make([]imaging.Result, len(sizes))
	for i := range sizes {
		out[i] = imaging.Result{Data: []byte("webp"), ContentType: "image/webp"}
	}
	return out, nil
}

type fakeObjects struct{ keys []string }

func (f *fakeObjects) Put(_ context.Context, key string, _ []byte, _ string) error {
	f.keys = append(f.keys, key)
	return nil
}

func (f *fakeObjects) PresignGet(_ context.Context, key string) (string, time.Duration, error) {
	return "https://s3.example.com/" + key, time.Hour, nil
}

var (
	providerID  = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	ownerID     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	characterID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
)

type fixture struct {
	svc     *Service
	store   *memStore
	image   *fakeImage
	objects *fakeObjects
}

func newFixture(t *testing.T, mod error) fixture {
	t.Helper()
	store := newMemStore()
	store.subjects[characterID] = Subject{ID: characterID, UserID: ownerID, Race: "Elf", Class: "Wizard"}
	image := &fakeImage{}
	objects := &fakeObjects{}
	svc := NewService(Deps{
		Store: store, Providers: fakeProviders{image: image}, Templates: fakeTemplates{},
		Moderator: fakeModerator{err: mod}, Encoder: fakeEncoder{}, Objects: objects,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	// 23:30 WIB on 29 Oct 2026.
	svc.now = func() time.Time { return time.Date(2026, 10, 29, 16, 30, 0, 0, time.UTC) }
	return fixture{svc: svc, store: store, image: image, objects: objects}
}

func TestRequest(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, nil)

	p, err := f.svc.Request(ctx, ownerID, characterID, Request{Style: "anime", UserPrompt: " holding a staff "})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusPending || p.AspectRatio != ai.Square || p.ProviderID != providerID || p.Model != "img-1" {
		t.Errorf("portrait = %+v", p)
	}
	if want := "A Elf Wizard. holding a staff. Style: anime illustration."; p.FinalPrompt != want {
		t.Errorf("final prompt = %q, want %q", p.FinalPrompt, want)
	}
	if len(f.store.locked) != 1 || len(f.store.jobs) != 1 || f.store.jobs[0] != (GenerateArgs{PortraitID: p.ID, NegativePrompt: "text"}) {
		t.Errorf("locked %v, jobs %+v", f.store.locked, f.store.jobs)
	}
	// The quota day starts at midnight WIB, which is 17:00 UTC the day before.
	if want := time.Date(2026, 10, 28, 17, 0, 0, 0, time.UTC); !f.store.since.Equal(want) {
		t.Errorf("counted since %v, want %v", f.store.since, want)
	}
}

func TestRequestRejections(t *testing.T) {
	ctx := context.Background()
	stranger := uuid.New()
	tests := []struct {
		name    string
		user    uuid.UUID
		req     Request
		mod     error
		used    int
		noImage bool
		want    error
	}{
		{"unknown style", ownerID, Request{Style: "cubism"}, nil, 0, false, ErrInvalidStyle},
		{"bad aspect", ownerID, Request{Style: "anime", AspectRatio: "16:9"}, nil, 0, false, ErrInvalidAspect},
		{"someone else's character", stranger, Request{Style: "anime"}, nil, 0, false, ErrCharacterNotFound},
		{"moderated prompt", ownerID, Request{Style: "anime"}, errBlocked, 0, false, errBlocked},
		{"no image provider", ownerID, Request{Style: "anime"}, nil, 0, true, aiprovider.ErrNoImageProvider},
		{"quota used up", ownerID, Request{Style: "anime"}, nil, 5, false, ErrQuotaExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.mod)
			f.store.used = tt.used
			if tt.noImage {
				f.svc.Providers = fakeProviders{activeErr: aiprovider.ErrNoImageProvider, image: f.image}
			}
			_, err := f.svc.Request(ctx, tt.user, characterID, tt.req)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(f.store.portraits) != 0 || len(f.store.jobs) != 0 {
				t.Errorf("rejected request left %d portraits and %d jobs", len(f.store.portraits), len(f.store.jobs))
			}
		})
	}
}

var errBlocked = errors.New("blocked word")

func pending(f fixture) Portrait {
	p := Portrait{ID: uuid.New(), UserID: ownerID, ProviderID: providerID, Model: "img-1",
		FinalPrompt: "an elf", AspectRatio: ai.Portrait, Status: StatusPending}
	f.store.portraits[p.ID] = p
	return p
}

func TestGenerate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, nil)
	p := pending(f)

	if err := f.svc.Generate(ctx, GenerateArgs{PortraitID: p.ID, NegativePrompt: "text"}, false); err != nil {
		t.Fatal(err)
	}
	base := "portraits/" + ownerID.String() + "/" + p.ID.String()
	if got := f.store.completed[p.ID]; got != [2]string{base + ".webp", base + "_thumb.webp"} {
		t.Errorf("completed with %v", got)
	}
	if len(f.objects.keys) != 2 {
		t.Errorf("uploaded %v", f.objects.keys)
	}
	in := f.image.got
	if in.APIKey != "sk-test" || in.NegativePrompt != "text" || in.AspectRatio != ai.Portrait || in.Model != "img-1" {
		t.Errorf("provider input = %+v", in)
	}
}

func TestGenerateFailures(t *testing.T) {
	ctx := context.Background()
	transient := errors.New("connection reset")
	tests := []struct {
		name      string
		err       error
		last      bool
		permanent bool
		message   string // "" means not marked FAILED
	}{
		{"rejected by provider", ai.ErrRejected, false, true, msgRejected},
		{"bad key", ai.ErrBadKey, false, true, msgConfig},
		{"transient, will retry", transient, false, false, ""},
		{"transient on last attempt", transient, true, true, msgGaveUp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, nil)
			f.image.err = tt.err
			p := pending(f)

			err := f.svc.Generate(ctx, GenerateArgs{PortraitID: p.ID}, tt.last)
			if err == nil || !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, want it to wrap %v", err, tt.err)
			}
			if errors.Is(err, errPermanent) != tt.permanent {
				t.Errorf("permanent = %v, want %v", errors.Is(err, errPermanent), tt.permanent)
			}
			if got := f.store.failed[p.ID]; got != tt.message {
				t.Errorf("failure message = %q, want %q", got, tt.message)
			}
			if len(f.objects.keys) != 0 {
				t.Errorf("uploaded %v after a failure", f.objects.keys)
			}
		})
	}
}

func TestGenerateSkipsFinishedAndMissing(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, nil)
	p := pending(f)
	p.Status = StatusSucceeded
	f.store.portraits[p.ID] = p

	if err := f.svc.Generate(ctx, GenerateArgs{PortraitID: p.ID}, false); err != nil || f.image.calls != 0 {
		t.Errorf("finished portrait: err %v, provider calls %d", err, f.image.calls)
	}
	if err := f.svc.Generate(ctx, GenerateArgs{PortraitID: uuid.New()}, false); !errors.Is(err, errPermanent) {
		t.Errorf("missing portrait err = %v, want permanent", err)
	}
}

type generatorFunc func() error

func (g generatorFunc) Generate(context.Context, GenerateArgs, bool) error { return g() }

func TestWorkerCancelsPermanentFailures(t *testing.T) {
	job := &river.Job[GenerateArgs]{JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 3}}

	err := NewWorker(generatorFunc(func() error { return errPermanent })).Work(context.Background(), job)
	if _, ok := errors.AsType[*rivertype.JobCancelError](err); !ok {
		t.Errorf("permanent failure: err = %v, want JobCancel", err)
	}
	transient := errors.New("timeout")
	err = NewWorker(generatorFunc(func() error { return transient })).Work(context.Background(), job)
	if !errors.Is(err, transient) {
		t.Errorf("transient failure: err = %v, want it returned for a retry", err)
	}
	if _, ok := errors.AsType[*rivertype.JobCancelError](err); ok {
		t.Error("transient failure was cancelled")
	}
}

func TestOwnershipAndVisibility(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, nil)
	p := pending(f)

	if _, err := f.svc.Get(ctx, uuid.New(), p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger Get err = %v", err)
	}
	if _, _, err := f.svc.ImageURL(ctx, p.ID, false); !errors.Is(err, ErrNotFound) {
		t.Errorf("pending ImageURL err = %v", err)
	}

	p.Status, p.StorageKey, p.ThumbKey = StatusSucceeded, "full.webp", "thumb.webp"
	f.store.portraits[p.ID] = p
	if url, _, err := f.svc.ImageURL(ctx, p.ID, true); err != nil || !strings.HasSuffix(url, "/thumb.webp") {
		t.Errorf("thumb ImageURL = %q, %v", url, err)
	}

	p.IsHidden = true
	f.store.portraits[p.ID] = p
	if _, _, err := f.svc.ImageURL(ctx, p.ID, false); !errors.Is(err, ErrNotFound) {
		t.Errorf("hidden ImageURL err = %v", err)
	}
}

func TestQuota(t *testing.T) {
	f := newFixture(t, nil)
	f.store.limit, f.store.used = 3, 7 // the limit was lowered after use

	q, err := f.svc.Quota(context.Background(), ownerID)
	if err != nil {
		t.Fatal(err)
	}
	want := Quota{Limit: 3, Used: 7, Remaining: 0, ResetsAt: time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)}
	if q.Limit != want.Limit || q.Used != want.Used || q.Remaining != 0 || !q.ResetsAt.Equal(want.ResetsAt) {
		t.Errorf("Quota = %+v, want %+v", q, want)
	}
}

func TestStartOfDay(t *testing.T) {
	// 17:00 UTC is midnight WIB: the new day has just begun.
	start, next := startOfDay(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)) || !next.Equal(start.Add(24*time.Hour)) {
		t.Errorf("startOfDay = %v, %v", start, next)
	}
}
