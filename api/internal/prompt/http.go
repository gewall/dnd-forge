package prompt

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Handler exposes template administration over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the admin template operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"admin"}
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminListPromptTemplates", Method: http.MethodGet, Path: "/v1/admin/prompt-templates",
		Summary: "Semua versi template prompt", Tags: tags,
	}), h.list)
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminCreatePromptTemplate", Method: http.MethodPost, Path: "/v1/admin/prompt-templates",
		Summary: "Simpan versi baru template per jenis (versi lama tetap tersimpan)", Tags: tags,
		DefaultStatus: http.StatusCreated, Errors: []int{http.StatusBadRequest},
	}), h.create)
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminActivatePromptTemplate", Method: http.MethodPost, Path: "/v1/admin/prompt-templates/{id}/activate",
		Summary: "Jadikan template aktif untuk jenisnya", Tags: tags, DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusNotFound},
	}), h.activate)
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminDeletePromptTemplate", Method: http.MethodDelete, Path: "/v1/admin/prompt-templates/{id}",
		Summary: "Hapus versi template yang tidak aktif", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound, http.StatusConflict},
	}), h.delete)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "portraitStyles", Method: http.MethodGet, Path: "/v1/portrait-styles",
		Summary: "Gaya gambar yang bisa dipilih user, dan placeholder template", Tags: []string{"portraits"},
	}), h.styles)
}

type TemplateRequest struct {
	Kind           Kind   `json:"kind" enum:"portrait,lore"`
	Name           string `json:"name" minLength:"1" maxLength:"60"`
	Template       string `json:"template" minLength:"1" maxLength:"4000" doc:"Placeholder per jenis: lihat GET /v1/portrait-styles (portrait) dan GET /v1/lore-options (lore)"`
	NegativePrompt string `json:"negativePrompt,omitempty" maxLength:"500"`
}

type TemplateDTO struct {
	ID             uuid.UUID  `json:"id"`
	Kind           Kind       `json:"kind"`
	Name           string     `json:"name"`
	Template       string     `json:"template"`
	NegativePrompt string     `json:"negativePrompt"`
	Version        int        `json:"version"`
	IsActive       bool       `json:"isActive"`
	CreatedBy      *uuid.UUID `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type TemplateList struct {
	Items []TemplateDTO `json:"items"`
}

type StyleDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type StylesDTO struct {
	Styles       []StyleDTO `json:"styles"`
	Placeholders []string   `json:"placeholders"`
}

type createInput struct{ Body TemplateRequest }
type idInput struct {
	ID uuid.UUID `path:"id"`
}
type templateOutput struct{ Body TemplateDTO }
type listOutput struct{ Body TemplateList }
type stylesOutput struct{ Body StylesDTO }

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	ts, err := h.svc.List(ctx)
	return httpapi.Respond(ctx, ts, err, func(ts []Template) *listOutput {
		items := make([]TemplateDTO, len(ts))
		for i, t := range ts {
			items[i] = TemplateDTO(t)
		}
		return &listOutput{Body: TemplateList{Items: items}}
	})
}

func (h *Handler) create(ctx context.Context, in *createInput) (*templateOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*templateOutput, error) {
		t, err := h.svc.Create(ctx, actor, Input(in.Body))
		return httpapi.Respond(ctx, t, err, func(t Template) *templateOutput { return &templateOutput{Body: TemplateDTO(t)} })
	})
}

func (h *Handler) activate(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Activate(ctx, actor, in.ID))
	})
}

func (h *Handler) delete(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Delete(ctx, actor, in.ID))
	})
}

func (h *Handler) styles(context.Context, *struct{}) (*stylesOutput, error) {
	out := StylesDTO{Placeholders: Placeholders(KindPortrait)}
	for id, desc := range Styles {
		out.Styles = append(out.Styles, StyleDTO{ID: id, Description: desc})
	}
	sort.Slice(out.Styles, func(i, j int) bool { return out.Styles[i].ID < out.Styles[j].ID })
	return &stylesOutput{Body: out}, nil
}
