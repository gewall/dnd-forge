package portrait

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// galleryPageSize is the default page of GET /v1/gallery.
const galleryPageSize = 24

// ImageURLs builds the stable URL of a portrait image.
type ImageURLs interface {
	Portrait(id uuid.UUID, thumb bool) string
}

// Handler exposes portrait generation, the gallery and portrait images
// over HTTP.
type Handler struct {
	svc  *Service
	urls ImageURLs
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service, urls ImageURLs) *Handler { return &Handler{svc: svc, urls: urls} }

// Register adds the portrait operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"portraits"}
	const (
		bad         = http.StatusBadRequest
		missing     = http.StatusNotFound
		unavailable = http.StatusServiceUnavailable
	)

	huma.Register(api, httpapi.RateLimited(httpapi.Authenticated(huma.Operation{
		OperationID: "requestPortrait", Method: http.MethodPost, Path: "/v1/characters/{id}/portraits",
		Summary: "Minta portrait AI untuk karakter",
		Description: "Portrait dibuat di background. Poll `GET /v1/portraits/{id}` sampai status bukan PENDING. " +
			"429 bila kuota harian habis; portrait yang FAILED tidak memotong kuota.",
		Tags: tags, DefaultStatus: http.StatusAccepted, Errors: []int{bad, missing, unavailable},
	}), "generate"), h.request)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "listCharacterPortraits", Method: http.MethodGet, Path: "/v1/characters/{id}/portraits",
		Summary: "Riwayat portrait karakter (50 terbaru)", Tags: tags,
	}), h.listForCharacter)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "getPortrait", Method: http.MethodGet, Path: "/v1/portraits/{id}",
		Summary: "Status dan URL portrait", Tags: tags, Errors: []int{missing},
	}), h.get)
	huma.Register(api, huma.Operation{
		OperationID: "gallery", Method: http.MethodGet, Path: "/v1/gallery",
		Summary:     "Galeri karakter publik dengan portrait terbarunya",
		Description: "Paginasi cursor: kirim `nextBefore` dari halaman sebelumnya sebagai `before`.",
		Tags:        tags,
	}, h.gallery)

	for _, thumb := range []bool{false, true} {
		op := huma.Operation{
			OperationID: "portraitImage", Method: http.MethodGet, Path: "/v1/media/portraits/{id}",
			Summary: "Redirect ke URL bertanda tangan untuk gambar portrait (tanpa internal key)",
			Tags:    []string{"media"}, DefaultStatus: http.StatusFound, Errors: []int{missing},
		}
		if thumb {
			op.OperationID, op.Path = "portraitThumb", op.Path+"/thumb"
			op.Summary = "Redirect ke URL bertanda tangan untuk thumbnail portrait (tanpa internal key)"
		}
		huma.Register(api, op, func(ctx context.Context, in *idInput) (*httpapi.Redirect, error) {
			url, ttl, err := h.svc.ImageURL(ctx, in.ID, thumb)
			if err != nil {
				return nil, httpapi.Error(ctx, err)
			}
			return httpapi.NewRedirect(url, ttl), nil
		})
	}
}

// DTO ------------------------------------------------------------------------

type PortraitRequest struct {
	Style       string `json:"style" minLength:"1" doc:"id dari GET /v1/portrait-styles"`
	Prompt      string `json:"prompt,omitempty" maxLength:"300" doc:"detail tambahan dari user (dimoderasi)"`
	AspectRatio string `json:"aspectRatio,omitempty" enum:"1:1,3:4,4:3" default:"1:1"`
}

type PortraitDTO struct {
	ID           uuid.UUID  `json:"id"`
	CharacterID  *uuid.UUID `json:"characterId" doc:"null bila karakter sudah dihapus"`
	Status       Status     `json:"status" enum:"PENDING,SUCCEEDED,FAILED"`
	Style        string     `json:"style"`
	AspectRatio  string     `json:"aspectRatio"`
	Prompt       string     `json:"prompt"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	ImageURL     string     `json:"imageUrl,omitempty" doc:"ada bila status SUCCEEDED"`
	ThumbURL     string     `json:"thumbUrl,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

type PortraitList struct {
	Items []PortraitDTO `json:"items"`
}

type GalleryItemDTO struct {
	PortraitID    uuid.UUID `json:"portraitId"`
	CharacterID   uuid.UUID `json:"characterId"`
	CharacterName string    `json:"characterName"`
	Race          string    `json:"race"`
	Class         string    `json:"class"`
	Style         string    `json:"style"`
	ImageURL      string    `json:"imageUrl"`
	ThumbURL      string    `json:"thumbUrl"`
	CreatedAt     time.Time `json:"createdAt"`
}

type GalleryPage struct {
	Items      []GalleryItemDTO `json:"items"`
	NextBefore *time.Time       `json:"nextBefore" doc:"null bila sudah halaman terakhir"`
}

// idInput carries a portrait or character id from the path.
type idInput struct {
	ID uuid.UUID `path:"id"`
}

type requestInput struct {
	ID   uuid.UUID `path:"id" doc:"id karakter"`
	Body PortraitRequest
}

type galleryInput struct {
	Before time.Time `query:"before" doc:"createdAt item terakhir halaman sebelumnya"`
	Limit  int       `query:"limit" minimum:"1" maximum:"60" default:"24"`
}

type portraitOutput struct{ Body PortraitDTO }
type listOutput struct{ Body PortraitList }
type galleryOutput struct{ Body GalleryPage }

// Handlers -------------------------------------------------------------------

func (h *Handler) request(ctx context.Context, in *requestInput) (*portraitOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*portraitOutput, error) {
		b := in.Body
		p, err := h.svc.Request(ctx, userID, in.ID, Request{Style: b.Style, UserPrompt: b.Prompt, AspectRatio: b.AspectRatio})
		return httpapi.Respond(ctx, p, err, h.toOutput)
	})
}

func (h *Handler) listForCharacter(ctx context.Context, in *idInput) (*listOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*listOutput, error) {
		ps, err := h.svc.ListForCharacter(ctx, userID, in.ID)
		return httpapi.Respond(ctx, ps, err, func(ps []Portrait) *listOutput {
			items := make([]PortraitDTO, len(ps))
			for i, p := range ps {
				items[i] = h.toDTO(p)
			}
			return &listOutput{Body: PortraitList{Items: items}}
		})
	})
}

func (h *Handler) get(ctx context.Context, in *idInput) (*portraitOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*portraitOutput, error) {
		p, err := h.svc.Get(ctx, userID, in.ID)
		return httpapi.Respond(ctx, p, err, h.toOutput)
	})
}

func (h *Handler) gallery(ctx context.Context, in *galleryInput) (*galleryOutput, error) {
	var before *time.Time
	if !in.Before.IsZero() {
		before = &in.Before
	}
	limit := in.Limit
	if limit == 0 {
		limit = galleryPageSize
	}
	items, err := h.svc.Gallery(ctx, before, limit)
	return httpapi.Respond(ctx, items, err, func(items []GalleryItem) *galleryOutput {
		page := GalleryPage{Items: make([]GalleryItemDTO, len(items))}
		for i, g := range items {
			page.Items[i] = GalleryItemDTO{
				PortraitID: g.PortraitID, CharacterID: g.CharacterID, CharacterName: g.CharacterName,
				Race: g.Race, Class: g.Class, Style: g.Style,
				ImageURL: h.urls.Portrait(g.PortraitID, false), ThumbURL: h.urls.Portrait(g.PortraitID, true),
				CreatedAt: g.CreatedAt,
			}
		}
		if len(items) == limit {
			next := items[len(items)-1].CreatedAt
			page.NextBefore = &next
		}
		return &galleryOutput{Body: page}
	})
}

// Mapping --------------------------------------------------------------------

func (h *Handler) toOutput(p Portrait) *portraitOutput { return &portraitOutput{Body: h.toDTO(p)} }

func (h *Handler) toDTO(p Portrait) PortraitDTO {
	d := PortraitDTO{
		ID: p.ID, CharacterID: p.CharacterID, Status: p.Status, Style: p.Style, AspectRatio: p.AspectRatio,
		Prompt: p.UserPrompt, ErrorMessage: p.ErrorMessage, CreatedAt: p.CreatedAt, CompletedAt: p.CompletedAt,
	}
	if p.Status == StatusSucceeded && !p.IsHidden {
		d.ImageURL, d.ThumbURL = h.urls.Portrait(p.ID, false), h.urls.Portrait(p.ID, true)
	}
	return d
}
