package aiprovider

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Handler exposes provider administration over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the admin AI provider operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"admin"}
	op := func(id, method, path, summary string, status int, errs ...int) huma.Operation {
		return httpapi.AdminOnly(huma.Operation{
			OperationID: id, Method: method, Path: path, Summary: summary, Tags: tags,
			DefaultStatus: status, Errors: errs,
		})
	}
	const (
		bad      = http.StatusBadRequest
		missing  = http.StatusNotFound
		conflict = http.StatusConflict
	)
	huma.Register(api, op("adminListAIProviders", http.MethodGet, "/v1/admin/ai/providers", "Provider AI yang dikonfigurasi", 0), h.list)
	huma.Register(api, op("adminCreateAIProvider", http.MethodPost, "/v1/admin/ai/providers", "Tambah provider (API key disimpan terenkripsi)", http.StatusCreated, bad), h.create)
	huma.Register(api, op("adminUpdateAIProvider", http.MethodPut, "/v1/admin/ai/providers/{id}", "Ubah label/model; isi apiKey hanya untuk mengganti key", 0, bad, missing), h.update)
	huma.Register(api, op("adminDeleteAIProvider", http.MethodDelete, "/v1/admin/ai/providers/{id}", "Hapus provider yang tidak aktif dan belum dipakai", http.StatusNoContent, missing, conflict), h.delete)
	huma.Register(api, op("adminTestAIProvider", http.MethodPost, "/v1/admin/ai/providers/{id}/test", "Uji koneksi dengan key tersimpan", http.StatusNoContent, bad, missing), h.test)
	huma.Register(api, op("adminActivateAIProvider", http.MethodPost, "/v1/admin/ai/providers/{id}/activate", "Jadikan provider aktif untuk kemampuannya (image/text)", http.StatusNoContent, missing), h.activate)
	huma.Register(api, op("adminAIProviderTypes", http.MethodGet, "/v1/admin/ai/provider-types", "Jenis provider yang didukung build ini dan kemampuannya", 0), h.types)
}

type ProviderRequest struct {
	Type         ai.ProviderType `json:"type" enum:"openai,google,stability,replicate,fake"`
	Capability   ai.Capability   `json:"capability" enum:"image,text" doc:"image untuk portrait, text untuk lore"`
	Label        string          `json:"label" minLength:"1" maxLength:"60"`
	DefaultModel string          `json:"defaultModel" minLength:"1" maxLength:"80" example:"gpt-image-1"`
	APIKey       string          `json:"apiKey" minLength:"8" maxLength:"300" writeOnly:"true"`
}

type ProviderUpdateRequest struct {
	Label        string `json:"label" minLength:"1" maxLength:"60"`
	DefaultModel string `json:"defaultModel" minLength:"1" maxLength:"80"`
	APIKey       string `json:"apiKey,omitempty" maxLength:"300" writeOnly:"true" doc:"kosongkan untuk mempertahankan key lama"`
}

type ProviderDTO struct {
	ID           uuid.UUID       `json:"id"`
	Type         ai.ProviderType `json:"type"`
	Capability   ai.Capability   `json:"capability"`
	Label        string          `json:"label"`
	KeyHint      string          `json:"keyHint" example:"••••abcd"`
	DefaultModel string          `json:"defaultModel"`
	IsActive     bool            `json:"isActive"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type ProviderList struct {
	Items []ProviderDTO `json:"items"`
}

type TypeDTO struct {
	Type         ai.ProviderType `json:"type"`
	Capabilities []ai.Capability `json:"capabilities"`
}

type TypeList struct {
	Items []TypeDTO `json:"items"`
}

type createInput struct{ Body ProviderRequest }
type updateInput struct {
	ID   uuid.UUID `path:"id"`
	Body ProviderUpdateRequest
}
type idInput struct {
	ID uuid.UUID `path:"id"`
}
type providerOutput struct{ Body ProviderDTO }
type listOutput struct{ Body ProviderList }
type typesOutput struct{ Body TypeList }

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	ps, err := h.svc.List(ctx)
	return httpapi.Respond(ctx, ps, err, func(ps []Provider) *listOutput {
		items := make([]ProviderDTO, len(ps))
		for i, p := range ps {
			items[i] = toDTO(p)
		}
		return &listOutput{Body: ProviderList{Items: items}}
	})
}

func (h *Handler) create(ctx context.Context, in *createInput) (*providerOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*providerOutput, error) {
		p, err := h.svc.Create(ctx, actor, Input(in.Body))
		return httpapi.Respond(ctx, p, err, toOutput)
	})
}

func (h *Handler) update(ctx context.Context, in *updateInput) (*providerOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*providerOutput, error) {
		b := in.Body
		p, err := h.svc.Update(ctx, actor, in.ID, Input{Label: b.Label, DefaultModel: b.DefaultModel, APIKey: b.APIKey})
		return httpapi.Respond(ctx, p, err, toOutput)
	})
}

func (h *Handler) delete(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Delete(ctx, actor, in.ID))
	})
}

func (h *Handler) activate(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Activate(ctx, actor, in.ID))
	})
}

func (h *Handler) test(ctx context.Context, in *idInput) (*struct{}, error) {
	return nil, httpapi.NoContent(ctx, h.svc.Test(ctx, in.ID))
}

func (h *Handler) types(context.Context, *struct{}) (*typesOutput, error) {
	ts := h.svc.Types()
	items := make([]TypeDTO, len(ts))
	for i, t := range ts {
		items[i] = TypeDTO(t)
	}
	return &typesOutput{Body: TypeList{Items: items}}, nil
}

func toOutput(p Provider) *providerOutput { return &providerOutput{Body: toDTO(p)} }

func toDTO(p Provider) ProviderDTO {
	return ProviderDTO{
		ID: p.ID, Type: p.Type, Capability: p.Capability, Label: p.Label, KeyHint: "••••" + p.KeyLast4,
		DefaultModel: p.DefaultModel, IsActive: p.IsActive, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}
