package guide

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
)

type fakeStore struct {
	cats     map[uuid.UUID]Category
	articles map[uuid.UUID]Article
	audits   []audit.Entry
	lastQ    string
	lastN    int
}

func newFakeStore() *fakeStore {
	return &fakeStore{cats: map[uuid.UUID]Category{}, articles: map[uuid.UUID]Article{}}
}

func (f *fakeStore) InTx(_ context.Context, fn func(Store) error) error { return fn(f) }
func (f *fakeStore) RecordAudit(_ context.Context, e audit.Entry) error {
	f.audits = append(f.audits, e)
	return nil
}

func (f *fakeStore) Categories(context.Context) ([]Category, error) {
	var out []Category
	for _, c := range f.cats {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeStore) CategoryByID(_ context.Context, id uuid.UUID) (Category, error) {
	c, ok := f.cats[id]
	if !ok {
		return Category{}, ErrCategoryNotFound
	}
	return c, nil
}

func (f *fakeStore) PublishedSummaries(context.Context) ([]Summary, error) {
	var out []Summary
	for _, a := range f.articles {
		if a.Status == StatusPublished {
			out = append(out, Summary{ID: a.ID, CategoryID: a.CategoryID, Slug: a.Slug, Title: a.Title, Status: a.Status})
		}
	}
	return out, nil
}

func (f *fakeStore) PublishedArticle(_ context.Context, categorySlug, slug string) (Article, error) {
	for _, a := range f.articles {
		if a.Slug == slug && a.Status == StatusPublished && f.cats[a.CategoryID].Slug == categorySlug {
			return a, nil
		}
	}
	return Article{}, ErrArticleNotFound
}

func (f *fakeStore) RelatedArticles(_ context.Context, categoryID, excludeID uuid.UUID, limit int) ([]Summary, error) {
	out := []Summary{}
	for _, a := range f.articles {
		if a.CategoryID == categoryID && a.ID != excludeID && a.Status == StatusPublished && len(out) < limit {
			out = append(out, Summary{ID: a.ID, Slug: a.Slug})
		}
	}
	return out, nil
}

func (f *fakeStore) Search(_ context.Context, q string, limit int) ([]SearchHit, error) {
	f.lastQ, f.lastN = q, limit
	return []SearchHit{}, nil
}

func (f *fakeStore) Articles(context.Context, ArticleFilter) ([]Summary, error) { return nil, nil }

func (f *fakeStore) ArticleByID(_ context.Context, id uuid.UUID) (Article, error) {
	a, ok := f.articles[id]
	if !ok {
		return Article{}, ErrArticleNotFound
	}
	return a, nil
}

func (f *fakeStore) CreateCategory(_ context.Context, in CategoryInput) (Category, error) {
	c := Category{ID: uuid.New(), CategoryInput: in}
	f.cats[c.ID] = c
	return c, nil
}

func (f *fakeStore) UpdateCategory(context.Context, uuid.UUID, CategoryInput) (Category, error) {
	return Category{}, ErrCategoryNotFound
}

func (f *fakeStore) DeleteCategory(_ context.Context, id uuid.UUID) error {
	for _, a := range f.articles {
		if a.CategoryID == id {
			return ErrCategoryInUse
		}
	}
	delete(f.cats, id)
	return nil
}

func (f *fakeStore) CreateArticle(_ context.Context, a Article) (Article, error) {
	if _, ok := f.cats[a.CategoryID]; !ok {
		return Article{}, ErrCategoryNotFound
	}
	a.ID = uuid.New()
	f.articles[a.ID] = a
	return a, nil
}

func (f *fakeStore) UpdateArticle(_ context.Context, a Article) (Article, error) {
	f.articles[a.ID] = a
	return a, nil
}

func (f *fakeStore) DeleteArticle(_ context.Context, id uuid.UUID) error {
	delete(f.articles, id)
	return nil
}

var admin = uuid.New()

type fixture struct {
	svc   *Service
	store *fakeStore
	cat   Category
	clock time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{store: newFakeStore(), clock: time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)}
	f.svc = NewService(f.store)
	f.svc.now = func() time.Time { return f.clock }
	cat, err := f.svc.CreateCategory(context.Background(), admin, CategoryInput{Slug: "dasar-aturan", Title: "Dasar Aturan"})
	if err != nil {
		t.Fatal(err)
	}
	f.cat = cat
	return f
}

func (f *fixture) article(slug string, status Status) ArticleInput {
	return ArticleInput{CategoryID: f.cat.ID, Slug: slug, Title: slug, Content: "# " + slug, Status: status}
}

func TestPublicationTime(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	draft, err := f.svc.CreateArticle(ctx, admin, f.article("lempar-dadu", StatusDraft))
	if err != nil {
		t.Fatal(err)
	}
	if draft.PublishedAt != nil || draft.AuthorID == nil || *draft.AuthorID != admin {
		t.Fatalf("draft = %+v", draft)
	}

	firstPublish := f.clock
	pub, _ := f.svc.UpdateArticle(ctx, admin, draft.ID, f.article("lempar-dadu", StatusPublished))
	if pub.PublishedAt == nil || !pub.PublishedAt.Equal(firstPublish) {
		t.Fatalf("published at = %v, want %v", pub.PublishedAt, firstPublish)
	}

	f.clock = f.clock.Add(24 * time.Hour)
	edited, _ := f.svc.UpdateArticle(ctx, admin, draft.ID, f.article("lempar-dadu", StatusPublished))
	if !edited.PublishedAt.Equal(firstPublish) {
		t.Errorf("edit moved publish time to %v", edited.PublishedAt)
	}

	unpublished, _ := f.svc.UpdateArticle(ctx, admin, draft.ID, f.article("lempar-dadu", StatusDraft))
	if unpublished.PublishedAt != nil {
		t.Errorf("draft kept publish time %v", unpublished.PublishedAt)
	}
}

func TestContentsAndArticle(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	empty, _ := f.svc.CreateCategory(ctx, admin, CategoryInput{Slug: "combat", Title: "Combat"})

	a, _ := f.svc.CreateArticle(ctx, admin, f.article("inisiatif", StatusPublished))
	_, _ = f.svc.CreateArticle(ctx, admin, f.article("aksi", StatusPublished))
	_, _ = f.svc.CreateArticle(ctx, admin, f.article("rahasia", StatusDraft))

	contents, err := f.svc.Contents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[uuid.UUID]int{}
	for _, c := range contents {
		counts[c.ID] = len(c.Articles)
		if c.Articles == nil {
			t.Errorf("category %s has nil articles, want empty slice", c.Slug)
		}
	}
	if counts[f.cat.ID] != 2 || counts[empty.ID] != 0 {
		t.Errorf("article counts = %v", counts)
	}

	got, related, err := f.svc.Article(ctx, "dasar-aturan", "inisiatif")
	if err != nil || got.ID != a.ID {
		t.Fatalf("Article = %+v, %v", got, err)
	}
	if len(related) != 1 || related[0].Slug != "aksi" {
		t.Errorf("related = %+v, want [aksi]", related)
	}
	if _, _, err := f.svc.Article(ctx, "dasar-aturan", "rahasia"); !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("draft visible publicly: %v", err)
	}
}

func TestSearchValidation(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	if _, err := f.svc.Search(ctx, "  a ", 10); !errors.Is(err, ErrQueryTooShort) {
		t.Errorf("short query err = %v", err)
	}
	tests := []struct{ in, want int }{{0, defaultSearchLimit}, {5, 5}, {500, maxSearchLimit}}
	for _, tt := range tests {
		if _, err := f.svc.Search(ctx, " naga merah ", tt.in); err != nil {
			t.Fatal(err)
		}
		if f.store.lastQ != "naga merah" || f.store.lastN != tt.want {
			t.Errorf("limit %d -> store got %q/%d, want %d", tt.in, f.store.lastQ, f.store.lastN, tt.want)
		}
	}
}

func TestSystemActorIsNotAuthor(t *testing.T) {
	f := newFixture(t)
	a, err := f.svc.CreateArticle(context.Background(), uuid.Nil, f.article("sistem", StatusDraft))
	if err != nil {
		t.Fatal(err)
	}
	if a.AuthorID != nil {
		t.Errorf("AuthorID = %v, want nil for the system actor", a.AuthorID)
	}
}

func TestDeleteCategoryInUse(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	_, _ = f.svc.CreateArticle(ctx, admin, f.article("x", StatusDraft))
	if err := f.svc.DeleteCategory(ctx, admin, f.cat.ID); !errors.Is(err, ErrCategoryInUse) {
		t.Errorf("err = %v, want ErrCategoryInUse", err)
	}
}
