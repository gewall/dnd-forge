//go:build integration

package guide

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func TestPostgresGuide(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "guide_categories", "audit_logs")
	svc := NewService(NewPostgresStore(pool))

	cat, err := svc.CreateCategory(ctx, uuid.Nil, CategoryInput{Slug: "combat", Title: "Combat", SortOrder: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategory(ctx, uuid.Nil, CategoryInput{Slug: "combat", Title: "Lagi"}); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("duplicate category err = %v", err)
	}

	write := func(slug, content string, status Status) Article {
		t.Helper()
		a, err := svc.CreateArticle(ctx, uuid.Nil, ArticleInput{
			CategoryID: cat.ID, Slug: slug, Title: strings.ToUpper(slug), Excerpt: "ringkas", Content: content, Status: status,
		})
		if err != nil {
			t.Fatalf("CreateArticle %s: %v", slug, err)
		}
		return a
	}
	first := write("inisiatif", "Setiap petarung melempar d20 untuk inisiatif. Naga merah bergerak cepat.", StatusPublished)
	write("serangan", "Serangan memakai d20 ditambah bonus. Naga punya armor tebal.", StatusPublished)
	write("draft-naga", "Naga rahasia yang belum terbit.", StatusDraft)

	if _, err := svc.CreateArticle(ctx, uuid.Nil, ArticleInput{CategoryID: uuid.New(), Slug: "x", Title: "x", Status: StatusDraft}); !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("article in missing category err = %v", err)
	}

	contents, err := svc.Contents(ctx)
	if err != nil || len(contents) != 1 || len(contents[0].Articles) != 2 {
		t.Fatalf("Contents = %+v, %v", contents, err)
	}

	page, related, err := svc.Article(ctx, "combat", "inisiatif")
	if err != nil {
		t.Fatal(err)
	}
	if page.ID != first.ID || page.CategorySlug != "combat" || page.PublishedAt == nil {
		t.Errorf("Article = %+v", page)
	}
	if len(related) != 1 || related[0].Slug != "serangan" {
		t.Errorf("related = %+v", related)
	}

	hits, err := svc.Search(ctx, "naga", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("search hits = %d, want 2 (draft excluded): %+v", len(hits), hits)
	}
	if !strings.Contains(strings.ToLower(hits[0].Headline), "<mark>naga</mark>") {
		t.Errorf("headline without highlight: %q", hits[0].Headline)
	}
	if hits, _ := svc.Search(ctx, `"naga merah"`, 10); len(hits) != 1 || hits[0].Slug != "inisiatif" {
		t.Errorf("phrase search = %+v", hits)
	}

	drafts := StatusDraft
	list, err := svc.Articles(ctx, ArticleFilter{CategoryID: &cat.ID, Status: &drafts})
	if err != nil || len(list) != 1 || list[0].Slug != "draft-naga" {
		t.Errorf("filtered admin list = %+v, %v", list, err)
	}

	if err := svc.DeleteCategory(ctx, uuid.Nil, cat.ID); !errors.Is(err, ErrCategoryInUse) {
		t.Errorf("delete non-empty category err = %v", err)
	}
}
