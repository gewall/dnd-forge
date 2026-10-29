package guide

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/slug"
)

const (
	relatedLimit       = 3
	defaultSearchLimit = 10
	maxSearchLimit     = 50
)

// Reader loads guide content.
type Reader interface {
	Categories(ctx context.Context) ([]Category, error)
	CategoryByID(ctx context.Context, id uuid.UUID) (Category, error)
	PublishedSummaries(ctx context.Context) ([]Summary, error)
	PublishedArticle(ctx context.Context, categorySlug, slug string) (Article, error)
	RelatedArticles(ctx context.Context, categoryID, excludeID uuid.UUID, limit int) ([]Summary, error)
	Search(ctx context.Context, query string, limit int) ([]SearchHit, error)
	Articles(ctx context.Context, f ArticleFilter) ([]Summary, error)
	ArticleByID(ctx context.Context, id uuid.UUID) (Article, error)
}

// Writer changes guide content. Implementations map unique violations to
// ErrSlugTaken and missing rows to ErrCategoryNotFound/ErrArticleNotFound.
type Writer interface {
	CreateCategory(ctx context.Context, in CategoryInput) (Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, in CategoryInput) (Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error // ErrCategoryInUse when it has articles
	CreateArticle(ctx context.Context, a Article) (Article, error)
	UpdateArticle(ctx context.Context, a Article) (Article, error)
	DeleteArticle(ctx context.Context, id uuid.UUID) error
	RecordAudit(ctx context.Context, e audit.Entry) error
}

// Store is the persistence the service needs.
type Store interface {
	Reader
	Writer
	InTx(ctx context.Context, fn func(Store) error) error
}

// Service serves the guide pages and lets admins edit them.
type Service struct {
	store Store
	now   func() time.Time
}

// NewService returns a Service backed by store.
func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }

// Public reads ----------------------------------------------------------------

// Contents lists every category with its published articles, for the guide
// index and sidebar.
func (s *Service) Contents(ctx context.Context) ([]CategoryWithArticles, error) {
	cats, err := s.store.Categories(ctx)
	if err != nil {
		return nil, fmt.Errorf("guide: contents: %w", err)
	}
	sums, err := s.store.PublishedSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("guide: contents: %w", err)
	}
	byCat := make(map[uuid.UUID][]Summary, len(cats))
	for _, a := range sums {
		byCat[a.CategoryID] = append(byCat[a.CategoryID], a)
	}
	out := make([]CategoryWithArticles, 0, len(cats))
	for _, c := range cats {
		arts := byCat[c.ID]
		if arts == nil {
			arts = []Summary{}
		}
		out = append(out, CategoryWithArticles{Category: c, Articles: arts})
	}
	return out, nil
}

// Article returns a published article and up to three related ones.
func (s *Service) Article(ctx context.Context, categorySlug, slug string) (Article, []Summary, error) {
	a, err := s.store.PublishedArticle(ctx, categorySlug, slug)
	if err != nil {
		return Article{}, nil, fmt.Errorf("guide: article: %w", err)
	}
	related, err := s.store.RelatedArticles(ctx, a.CategoryID, a.ID, relatedLimit)
	if err != nil {
		return Article{}, nil, fmt.Errorf("guide: related: %w", err)
	}
	return a, related, nil
}

// Search finds published articles matching query (web search syntax:
// quoted phrases, OR, -exclude).
func (s *Service) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) < 2 {
		return nil, ErrQueryTooShort
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	hits, err := s.store.Search(ctx, query, min(limit, maxSearchLimit))
	if err != nil {
		return nil, fmt.Errorf("guide: search: %w", err)
	}
	return hits, nil
}

// Admin -------------------------------------------------------------------------

// Categories lists all categories.
func (s *Service) Categories(ctx context.Context) ([]Category, error) { return s.store.Categories(ctx) }

// Articles lists articles of any status.
func (s *Service) Articles(ctx context.Context, f ArticleFilter) ([]Summary, error) {
	return s.store.Articles(ctx, f)
}

// ArticleByID returns an article of any status.
func (s *Service) ArticleByID(ctx context.Context, id uuid.UUID) (Article, error) {
	return s.store.ArticleByID(ctx, id)
}

func (s *Service) CreateCategory(ctx context.Context, actor uuid.UUID, in CategoryInput) (Category, error) {
	if !slug.Valid(in.Slug) {
		return Category{}, ErrInvalidSlug
	}
	var c Category
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityCategory, Diff: in},
		func(tx Store) (id uuid.UUID, err error) {
			c, err = tx.CreateCategory(ctx, in)
			return c.ID, err
		})
	return c, err
}

func (s *Service) UpdateCategory(ctx context.Context, actor, id uuid.UUID, in CategoryInput) (Category, error) {
	if !slug.Valid(in.Slug) {
		return Category{}, ErrInvalidSlug
	}
	var c Category
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityCategory, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			c, err = tx.UpdateCategory(ctx, id, in)
			return id, err
		})
	return c, err
}

func (s *Service) DeleteCategory(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityCategory},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteCategory(ctx, id) })
}

// CreateArticle saves a new article authored by actor. Publishing stamps
// the publication time.
func (s *Service) CreateArticle(ctx context.Context, actor uuid.UUID, in ArticleInput) (Article, error) {
	if !slug.Valid(in.Slug) {
		return Article{}, ErrInvalidSlug
	}
	a := Article{AuthorID: &actor, ArticleInput: in}
	if in.Status == StatusPublished {
		now := s.now()
		a.PublishedAt = &now
	}
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionCreate, Entity: entityArticle, Diff: in},
		func(tx Store) (_ uuid.UUID, err error) {
			a, err = tx.CreateArticle(ctx, a)
			return a.ID, err
		})
	return a, err
}

// UpdateArticle replaces an article's fields. The first publication time is
// kept across edits; moving back to draft clears it.
func (s *Service) UpdateArticle(ctx context.Context, actor, id uuid.UUID, in ArticleInput) (Article, error) {
	if !slug.Valid(in.Slug) {
		return Article{}, ErrInvalidSlug
	}
	var a Article
	err := s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionUpdate, Entity: entityArticle, Diff: in},
		func(tx Store) (uuid.UUID, error) {
			cur, err := tx.ArticleByID(ctx, id)
			if err != nil {
				return id, err
			}
			a = cur
			a.ArticleInput = in
			switch {
			case in.Status == StatusDraft:
				a.PublishedAt = nil
			case cur.PublishedAt == nil:
				now := s.now()
				a.PublishedAt = &now
			}
			a, err = tx.UpdateArticle(ctx, a)
			return id, err
		})
	return a, err
}

func (s *Service) DeleteArticle(ctx context.Context, actor, id uuid.UUID) error {
	return s.write(ctx, audit.Entry{ActorID: actor, Action: audit.ActionDelete, Entity: entityArticle},
		func(tx Store) (uuid.UUID, error) { return id, tx.DeleteArticle(ctx, id) })
}

// write runs change in a transaction and records e for the id it returns.
func (s *Service) write(ctx context.Context, e audit.Entry, change func(Store) (uuid.UUID, error)) error {
	err := s.store.InTx(ctx, func(tx Store) error {
		id, err := change(tx)
		if err != nil {
			return err
		}
		e.EntityID = id.String()
		return tx.RecordAudit(ctx, e)
	})
	if err != nil {
		return fmt.Errorf("guide: %s %s: %w", e.Action, e.Entity, err)
	}
	return nil
}
