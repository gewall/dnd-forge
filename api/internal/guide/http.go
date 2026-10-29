package guide

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// ImageURLs turns a stored image key into a public URL.
type ImageURLs interface {
	Content(key string) string
}

// Handler exposes the guide over HTTP.
type Handler struct {
	svc  *Service
	urls ImageURLs
}

// NewHandler returns a Handler; arguments may be nil when only generating
// OpenAPI.
func NewHandler(svc *Service, urls ImageURLs) *Handler { return &Handler{svc: svc, urls: urls} }

// Register adds the public and admin guide operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"guide"}
	huma.Register(api, huma.Operation{
		OperationID: "guideContents", Method: http.MethodGet, Path: "/v1/guides/categories",
		Summary: "Kategori panduan beserta artikel yang sudah terbit", Tags: tags,
	}, h.contents)
	huma.Register(api, huma.Operation{
		OperationID: "guideSearch", Method: http.MethodGet, Path: "/v1/guides/search",
		Summary: "Cari artikel panduan", Tags: tags, Errors: []int{http.StatusBadRequest},
	}, h.search)
	huma.Register(api, huma.Operation{
		OperationID: "guideArticle", Method: http.MethodGet, Path: "/v1/guides/{category}/{slug}",
		Summary: "Artikel panduan beserta artikel terkait", Tags: tags, Errors: []int{http.StatusNotFound},
	}, h.article)

	h.registerAdmin(api)
}

// DTO ------------------------------------------------------------------------

type CategoryDTO struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sortOrder"`
}

type ArticleSummaryDTO struct {
	ID          uuid.UUID  `json:"id"`
	CategoryID  uuid.UUID  `json:"categoryId"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	CoverKey    string     `json:"coverKey,omitempty"`
	CoverURL    string     `json:"coverUrl,omitempty"`
	SortOrder   int        `json:"sortOrder"`
	Status      Status     `json:"status" enum:"DRAFT,PUBLISHED"`
	PublishedAt *time.Time `json:"publishedAt"`
}

type CategoryContentsDTO struct {
	CategoryDTO
	Articles []ArticleSummaryDTO `json:"articles"`
}

type ArticleDTO struct {
	ID            uuid.UUID  `json:"id"`
	CategoryID    uuid.UUID  `json:"categoryId"`
	CategorySlug  string     `json:"categorySlug,omitempty"`
	CategoryTitle string     `json:"categoryTitle,omitempty"`
	AuthorID      *uuid.UUID `json:"authorId"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Excerpt       string     `json:"excerpt"`
	Content       string     `json:"content" doc:"Markdown"`
	CoverKey      string     `json:"coverKey,omitempty"`
	CoverURL      string     `json:"coverUrl,omitempty"`
	SortOrder     int        `json:"sortOrder"`
	Status        Status     `json:"status" enum:"DRAFT,PUBLISHED"`
	PublishedAt   *time.Time `json:"publishedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type ArticlePageDTO struct {
	Article ArticleDTO          `json:"article"`
	Related []ArticleSummaryDTO `json:"related"`
}

type SearchHitDTO struct {
	ID            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Excerpt       string    `json:"excerpt"`
	CategorySlug  string    `json:"categorySlug"`
	CategoryTitle string    `json:"categoryTitle"`
	Headline      string    `json:"headline" doc:"Teks polos; kata yang cocok dibungkus <mark></mark>. Escape sisanya sebelum dirender."`
}

type ContentsDTO struct {
	Items []CategoryContentsDTO `json:"items"`
}

type SearchResultDTO struct {
	Items []SearchHitDTO `json:"items"`
}

type contentsOutput struct{ Body ContentsDTO }
type articlePageOutput struct{ Body ArticlePageDTO }
type searchOutput struct{ Body SearchResultDTO }

type articleInput struct {
	Category string `path:"category" maxLength:"64"`
	Slug     string `path:"slug" maxLength:"64"`
}

type searchInput struct {
	Q     string `query:"q" minLength:"2" maxLength:"100" required:"true"`
	Limit int    `query:"limit" minimum:"1" maximum:"50" default:"10"`
}

// Handlers -------------------------------------------------------------------

func (h *Handler) contents(ctx context.Context, _ *struct{}) (*contentsOutput, error) {
	cs, err := h.svc.Contents(ctx)
	return httpapi.Respond(ctx, cs, err, func(cs []CategoryWithArticles) *contentsOutput {
		items := make([]CategoryContentsDTO, len(cs))
		for i, c := range cs {
			items[i] = CategoryContentsDTO{CategoryDTO: categoryDTO(c.Category), Articles: h.summaryDTOs(c.Articles)}
		}
		return &contentsOutput{Body: ContentsDTO{Items: items}}
	})
}

func (h *Handler) article(ctx context.Context, in *articleInput) (*articlePageOutput, error) {
	a, related, err := h.svc.Article(ctx, in.Category, in.Slug)
	return httpapi.Respond(ctx, a, err, func(a Article) *articlePageOutput {
		return &articlePageOutput{Body: ArticlePageDTO{Article: h.articleDTO(a), Related: h.summaryDTOs(related)}}
	})
}

func (h *Handler) search(ctx context.Context, in *searchInput) (*searchOutput, error) {
	hits, err := h.svc.Search(ctx, in.Q, in.Limit)
	return httpapi.Respond(ctx, hits, err, func(hits []SearchHit) *searchOutput {
		items := make([]SearchHitDTO, len(hits))
		for i, hit := range hits {
			items[i] = SearchHitDTO(hit)
		}
		return &searchOutput{Body: SearchResultDTO{Items: items}}
	})
}

// Mapping --------------------------------------------------------------------

func categoryDTO(c Category) CategoryDTO {
	return CategoryDTO{ID: c.ID, Slug: c.Slug, Title: c.Title, Description: c.Description, SortOrder: c.SortOrder}
}

func (h *Handler) summaryDTOs(ss []Summary) []ArticleSummaryDTO {
	out := make([]ArticleSummaryDTO, len(ss))
	for i, s := range ss {
		out[i] = ArticleSummaryDTO{
			ID: s.ID, CategoryID: s.CategoryID, Slug: s.Slug, Title: s.Title, Excerpt: s.Excerpt,
			CoverKey: s.CoverKey, CoverURL: h.urls.Content(s.CoverKey), SortOrder: s.SortOrder,
			Status: s.Status, PublishedAt: s.PublishedAt,
		}
	}
	return out
}

func (h *Handler) articleDTO(a Article) ArticleDTO {
	return ArticleDTO{
		ID: a.ID, CategoryID: a.CategoryID, CategorySlug: a.CategorySlug, CategoryTitle: a.CategoryTitle,
		AuthorID: a.AuthorID, Slug: a.Slug, Title: a.Title, Excerpt: a.Excerpt, Content: a.Content,
		CoverKey: a.CoverKey, CoverURL: h.urls.Content(a.CoverKey), SortOrder: a.SortOrder, Status: a.Status,
		PublishedAt: a.PublishedAt, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}
