package guide

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresStore) InTx(ctx context.Context, fn func(Store) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&PostgresStore{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func (s *PostgresStore) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, s.q, e)
}

// Categories ------------------------------------------------------------------

func (s *PostgresStore) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.q.ListGuideCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	out := make([]Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCategory(r))
	}
	return out, nil
}

func (s *PostgresStore) CategoryByID(ctx context.Context, id uuid.UUID) (Category, error) {
	row, err := s.q.GuideCategoryByID(ctx, id)
	if err != nil {
		return Category{}, mapErr(err, ErrCategoryNotFound)
	}
	return toCategory(row), nil
}

func (s *PostgresStore) CreateCategory(ctx context.Context, in CategoryInput) (Category, error) {
	row, err := s.q.CreateGuideCategory(ctx, sqlc.CreateGuideCategoryParams{
		Slug: in.Slug, Title: in.Title, Description: in.Description, SortOrder: db.Int32(in.SortOrder),
	})
	if err != nil {
		return Category{}, mapErr(err, ErrCategoryNotFound)
	}
	return toCategory(row), nil
}

func (s *PostgresStore) UpdateCategory(ctx context.Context, id uuid.UUID, in CategoryInput) (Category, error) {
	row, err := s.q.UpdateGuideCategory(ctx, sqlc.UpdateGuideCategoryParams{
		ID: id, Slug: in.Slug, Title: in.Title, Description: in.Description, SortOrder: db.Int32(in.SortOrder),
	})
	if err != nil {
		return Category{}, mapErr(err, ErrCategoryNotFound)
	}
	return toCategory(row), nil
}

func (s *PostgresStore) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteGuideCategory(ctx, id)
	switch {
	case db.IsForeignKeyViolation(err):
		return ErrCategoryInUse
	case err != nil:
		return fmt.Errorf("delete category: %w", err)
	case n == 0:
		return ErrCategoryNotFound
	}
	return nil
}

// Public articles ---------------------------------------------------------------

func (s *PostgresStore) PublishedSummaries(ctx context.Context) ([]Summary, error) {
	rows, err := s.q.ListPublishedGuideSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, Summary{
			ID: r.ID, CategoryID: r.CategoryID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
			CoverKey: deref(r.CoverKey), SortOrder: int(r.SortOrder), Status: StatusPublished, PublishedAt: r.PublishedAt,
		})
	}
	return out, nil
}

func (s *PostgresStore) PublishedArticle(ctx context.Context, categorySlug, slug string) (Article, error) {
	r, err := s.q.PublishedGuideArticle(ctx, sqlc.PublishedGuideArticleParams{CategorySlug: categorySlug, Slug: slug})
	if err != nil {
		return Article{}, mapErr(err, ErrArticleNotFound)
	}
	a := toArticle(sqlc.GuideArticle{
		ID: r.ID, CategoryID: r.CategoryID, AuthorID: r.AuthorID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
		Content: r.Content, CoverKey: r.CoverKey, SortOrder: r.SortOrder, Status: r.Status,
		PublishedAt: r.PublishedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
	a.CategorySlug, a.CategoryTitle = r.CategorySlug, r.CategoryTitle
	return a, nil
}

func (s *PostgresStore) RelatedArticles(ctx context.Context, categoryID, excludeID uuid.UUID, limit int) ([]Summary, error) {
	rows, err := s.q.RelatedGuideArticles(ctx, sqlc.RelatedGuideArticlesParams{
		CategoryID: categoryID, ExcludeID: excludeID, MaxResults: db.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("related articles: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, Summary{
			ID: r.ID, CategoryID: r.CategoryID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
			CoverKey: deref(r.CoverKey), SortOrder: int(r.SortOrder), Status: StatusPublished, PublishedAt: r.PublishedAt,
		})
	}
	return out, nil
}

func (s *PostgresStore) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	rows, err := s.q.SearchGuideArticles(ctx, sqlc.SearchGuideArticlesParams{Query: query, MaxResults: db.Int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit(r))
	}
	return out, nil
}

// Admin articles ----------------------------------------------------------------

func (s *PostgresStore) Articles(ctx context.Context, f ArticleFilter) ([]Summary, error) {
	var status *sqlc.ContentStatus
	if f.Status != nil {
		st := sqlc.ContentStatus(*f.Status)
		status = &st
	}
	rows, err := s.q.ListGuideArticles(ctx, sqlc.ListGuideArticlesParams{CategoryID: f.CategoryID, Status: status})
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, Summary{
			ID: r.ID, CategoryID: r.CategoryID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
			CoverKey: deref(r.CoverKey), SortOrder: int(r.SortOrder), Status: Status(r.Status),
			PublishedAt: r.PublishedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (s *PostgresStore) ArticleByID(ctx context.Context, id uuid.UUID) (Article, error) {
	row, err := s.q.GuideArticleByID(ctx, id)
	if err != nil {
		return Article{}, mapErr(err, ErrArticleNotFound)
	}
	return toArticle(row), nil
}

func (s *PostgresStore) CreateArticle(ctx context.Context, a Article) (Article, error) {
	row, err := s.q.CreateGuideArticle(ctx, sqlc.CreateGuideArticleParams{
		CategoryID: a.CategoryID, AuthorID: a.AuthorID, Slug: a.Slug, Title: a.Title, Excerpt: a.Excerpt,
		Content: a.Content, CoverKey: optional(a.CoverKey), SortOrder: db.Int32(a.SortOrder),
		Status: sqlc.ContentStatus(a.Status), PublishedAt: a.PublishedAt,
	})
	if err != nil {
		return Article{}, mapArticleErr(err)
	}
	return toArticle(row), nil
}

func (s *PostgresStore) UpdateArticle(ctx context.Context, a Article) (Article, error) {
	row, err := s.q.UpdateGuideArticle(ctx, sqlc.UpdateGuideArticleParams{
		ID: a.ID, CategoryID: a.CategoryID, Slug: a.Slug, Title: a.Title, Excerpt: a.Excerpt,
		Content: a.Content, CoverKey: optional(a.CoverKey), SortOrder: db.Int32(a.SortOrder),
		Status: sqlc.ContentStatus(a.Status), PublishedAt: a.PublishedAt,
	})
	if err != nil {
		return Article{}, mapArticleErr(err)
	}
	return toArticle(row), nil
}

func (s *PostgresStore) DeleteArticle(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteGuideArticle(ctx, id)
	if err != nil {
		return fmt.Errorf("delete article: %w", err)
	}
	if n == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// Mapping -----------------------------------------------------------------------

func mapErr(err, notFound error) error {
	switch {
	case db.IsNoRows(err):
		return notFound
	case db.IsUniqueViolation(err):
		return ErrSlugTaken
	}
	return err
}

// mapArticleErr also covers the category foreign key: writing an article
// into a missing category.
func mapArticleErr(err error) error {
	if db.IsForeignKeyViolation(err) {
		return ErrCategoryNotFound
	}
	return mapErr(err, ErrArticleNotFound)
}

func toCategory(r sqlc.GuideCategory) Category {
	return Category{ID: r.ID, CategoryInput: CategoryInput{
		Slug: r.Slug, Title: r.Title, Description: r.Description, SortOrder: int(r.SortOrder),
	}}
}

func toArticle(r sqlc.GuideArticle) Article {
	return Article{
		ID:       r.ID,
		AuthorID: r.AuthorID,
		ArticleInput: ArticleInput{
			CategoryID: r.CategoryID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt, Content: r.Content,
			CoverKey: deref(r.CoverKey), SortOrder: int(r.SortOrder), Status: Status(r.Status),
		},
		PublishedAt: r.PublishedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
