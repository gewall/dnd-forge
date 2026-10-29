package guide

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

func (h *Handler) registerAdmin(api huma.API) {
	tags := []string{"admin"}
	op := func(id, method, path, summary string, errs ...int) huma.Operation {
		o := huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: tags, Errors: errs}
		switch method {
		case http.MethodPost:
			o.DefaultStatus = http.StatusCreated
		case http.MethodDelete:
			o.DefaultStatus = http.StatusNoContent
		}
		return httpapi.AdminOnly(o)
	}
	const (
		bad      = http.StatusBadRequest
		missing  = http.StatusNotFound
		conflict = http.StatusConflict
	)

	huma.Register(api, op("adminListGuideCategories", http.MethodGet, "/v1/admin/guide-categories", "Semua kategori panduan"), h.adminListCategories)
	huma.Register(api, op("adminCreateGuideCategory", http.MethodPost, "/v1/admin/guide-categories", "Tambah kategori", bad, conflict), h.adminCreateCategory)
	huma.Register(api, op("adminUpdateGuideCategory", http.MethodPut, "/v1/admin/guide-categories/{id}", "Ubah kategori", bad, missing, conflict), h.adminUpdateCategory)
	huma.Register(api, op("adminDeleteGuideCategory", http.MethodDelete, "/v1/admin/guide-categories/{id}", "Hapus kategori kosong", missing, conflict), h.adminDeleteCategory)

	huma.Register(api, op("adminListGuides", http.MethodGet, "/v1/admin/guides", "Semua artikel (draft & terbit)"), h.adminListArticles)
	huma.Register(api, op("adminCreateGuide", http.MethodPost, "/v1/admin/guides", "Tulis artikel", bad, missing, conflict), h.adminCreateArticle)
	huma.Register(api, op("adminGetGuide", http.MethodGet, "/v1/admin/guides/{id}", "Detail artikel", missing), h.adminGetArticle)
	huma.Register(api, op("adminUpdateGuide", http.MethodPut, "/v1/admin/guides/{id}", "Ubah artikel", bad, missing, conflict), h.adminUpdateArticle)
	huma.Register(api, op("adminDeleteGuide", http.MethodDelete, "/v1/admin/guides/{id}", "Hapus artikel", missing), h.adminDeleteArticle)
}

// Requests -------------------------------------------------------------------

type CategoryRequest struct {
	Slug        string `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Title       string `json:"title" minLength:"1" maxLength:"120"`
	Description string `json:"description,omitempty" maxLength:"1000"`
	SortOrder   int    `json:"sortOrder,omitempty" minimum:"0" maximum:"10000"`
}

type ArticleRequest struct {
	CategoryID uuid.UUID `json:"categoryId"`
	Slug       string    `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Title      string    `json:"title" minLength:"1" maxLength:"160"`
	Excerpt    string    `json:"excerpt,omitempty" maxLength:"500"`
	Content    string    `json:"content,omitempty" maxLength:"200000" doc:"Markdown"`
	CoverKey   string    `json:"coverKey,omitempty" pattern:"^(content/[0-9a-f-]{36}\\.webp)?$" doc:"key dari POST /v1/admin/uploads; kosong = tanpa gambar"`
	SortOrder  int       `json:"sortOrder,omitempty" minimum:"0" maximum:"10000"`
	Status     Status    `json:"status" enum:"DRAFT,PUBLISHED"`
}

func (r CategoryRequest) input() CategoryInput { return CategoryInput(r) }

func (r ArticleRequest) input() ArticleInput { return ArticleInput(r) }

type idInput struct {
	ID uuid.UUID `path:"id"`
}

type createCategoryInput struct{ Body CategoryRequest }
type updateCategoryInput struct {
	idInput
	Body CategoryRequest
}
type createArticleInput struct{ Body ArticleRequest }
type updateArticleInput struct {
	idInput
	Body ArticleRequest
}

type listArticlesInput struct {
	CategoryID string `query:"categoryId" format:"uuid" doc:"filter kategori"`
	Status     string `query:"status" enum:"DRAFT,PUBLISHED" doc:"filter status"`
}

type CategoryList struct {
	Items []CategoryDTO `json:"items"`
}

type ArticleList struct {
	Items []ArticleSummaryDTO `json:"items"`
}

type categoryOutput struct{ Body CategoryDTO }
type categoriesOutput struct{ Body CategoryList }
type articleOutput struct{ Body ArticleDTO }
type articlesOutput struct{ Body ArticleList }

// Handlers -------------------------------------------------------------------

func (h *Handler) adminListCategories(ctx context.Context, _ *struct{}) (*categoriesOutput, error) {
	cs, err := h.svc.Categories(ctx)
	return httpapi.Respond(ctx, cs, err, func(cs []Category) *categoriesOutput {
		items := make([]CategoryDTO, len(cs))
		for i, c := range cs {
			items[i] = categoryDTO(c)
		}
		return &categoriesOutput{Body: CategoryList{Items: items}}
	})
}

func (h *Handler) adminCreateCategory(ctx context.Context, in *createCategoryInput) (*categoryOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*categoryOutput, error) {
		c, err := h.svc.CreateCategory(ctx, actor, in.Body.input())
		return httpapi.Respond(ctx, c, err, toCategoryOutput)
	})
}

func (h *Handler) adminUpdateCategory(ctx context.Context, in *updateCategoryInput) (*categoryOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*categoryOutput, error) {
		c, err := h.svc.UpdateCategory(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, c, err, toCategoryOutput)
	})
}

func (h *Handler) adminDeleteCategory(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteCategory(ctx, actor, in.ID))
	})
}

func (h *Handler) adminListArticles(ctx context.Context, in *listArticlesInput) (*articlesOutput, error) {
	var f ArticleFilter
	if in.CategoryID != "" {
		id, err := uuid.Parse(in.CategoryID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("categoryId bukan UUID")
		}
		f.CategoryID = &id
	}
	if in.Status != "" {
		st := Status(in.Status)
		f.Status = &st
	}
	ss, err := h.svc.Articles(ctx, f)
	return httpapi.Respond(ctx, ss, err, func(ss []Summary) *articlesOutput {
		return &articlesOutput{Body: ArticleList{Items: h.summaryDTOs(ss)}}
	})
}

func (h *Handler) adminGetArticle(ctx context.Context, in *idInput) (*articleOutput, error) {
	a, err := h.svc.ArticleByID(ctx, in.ID)
	return httpapi.Respond(ctx, a, err, h.toArticleOutput)
}

func (h *Handler) adminCreateArticle(ctx context.Context, in *createArticleInput) (*articleOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*articleOutput, error) {
		a, err := h.svc.CreateArticle(ctx, actor, in.Body.input())
		return httpapi.Respond(ctx, a, err, h.toArticleOutput)
	})
}

func (h *Handler) adminUpdateArticle(ctx context.Context, in *updateArticleInput) (*articleOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*articleOutput, error) {
		a, err := h.svc.UpdateArticle(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, a, err, h.toArticleOutput)
	})
}

func (h *Handler) adminDeleteArticle(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteArticle(ctx, actor, in.ID))
	})
}

func toCategoryOutput(c Category) *categoryOutput { return &categoryOutput{Body: categoryDTO(c)} }

func (h *Handler) toArticleOutput(a Article) *articleOutput {
	return &articleOutput{Body: h.articleDTO(a)}
}
