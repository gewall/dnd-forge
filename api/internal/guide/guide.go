// Package guide manages the how-to-play articles: categories, Markdown
// articles with draft/published status, and full-text search.
package guide

import (
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

var (
	ErrCategoryNotFound = apperr.New(apperr.NotFound, "kategori tidak ditemukan")
	ErrArticleNotFound  = apperr.New(apperr.NotFound, "artikel tidak ditemukan")
	ErrSlugTaken        = apperr.New(apperr.Conflict, "slug sudah dipakai")
	ErrCategoryInUse    = apperr.New(apperr.Conflict, "kategori masih berisi artikel")
	ErrInvalidSlug      = apperr.New(apperr.Invalid, "slug hanya boleh huruf kecil, angka, dan tanda hubung")
	ErrQueryTooShort    = apperr.New(apperr.Invalid, "kata kunci minimal 2 karakter")
)

const (
	entityCategory = "guide_category"
	entityArticle  = "guide_article"
)

// Status is the publication state of an article.
type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
)

// CategoryInput holds the editable fields of a category.
type CategoryInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// Category groups articles.
type Category struct {
	ID uuid.UUID
	CategoryInput
}

// CategoryWithArticles is a category and its published articles.
type CategoryWithArticles struct {
	Category
	Articles []Summary
}

// Summary is an article without its body.
type Summary struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Slug        string
	Title       string
	Excerpt     string
	CoverKey    string
	SortOrder   int
	Status      Status
	PublishedAt *time.Time
	UpdatedAt   time.Time
}

// ArticleInput holds the editable fields of an article.
type ArticleInput struct {
	CategoryID uuid.UUID `json:"categoryId"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	Content    string    `json:"content"` // Markdown
	CoverKey   string    `json:"coverKey"`
	SortOrder  int       `json:"sortOrder"`
	Status     Status    `json:"status"`
}

// Article is a full article.
type Article struct {
	ID            uuid.UUID
	AuthorID      *uuid.UUID
	CategorySlug  string // set on public reads
	CategoryTitle string
	ArticleInput
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ArticleFilter narrows the admin article list; zero fields match all.
type ArticleFilter struct {
	CategoryID *uuid.UUID
	Status     *Status
}

// SearchHit is one search result. Headline is plain text with matches
// wrapped in <mark></mark>; everything else must be escaped by the client.
type SearchHit struct {
	ID            uuid.UUID
	Slug          string
	Title         string
	Excerpt       string
	CategorySlug  string
	CategoryTitle string
	Headline      string
}
