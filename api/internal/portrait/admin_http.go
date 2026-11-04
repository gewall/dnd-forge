package portrait

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// AdminHandler exposes portrait moderation over HTTP.
type AdminHandler struct {
	svc *Admin
}

// NewAdminHandler returns an AdminHandler; svc may be nil when only
// generating OpenAPI.
func NewAdminHandler(svc *Admin) *AdminHandler { return &AdminHandler{svc: svc} }

// Register adds the admin image operations to api.
func (h *AdminHandler) Register(api huma.API) {
	op := func(id, method, path, summary string, status int, errs ...int) huma.Operation {
		return httpapi.AdminOnly(huma.Operation{
			OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"admin"},
			DefaultStatus: status, Errors: errs,
		})
	}
	huma.Register(api, op("adminListImages", http.MethodGet, "/v1/admin/images",
		"Portrait untuk moderasi, terbaru dulu (termasuk yang disembunyikan)", 0), h.list)
	huma.Register(api, op("adminGetImage", http.MethodGet, "/v1/admin/images/{id}",
		"Detail portrait untuk moderasi", 0, http.StatusNotFound), h.get)
	huma.Register(api, op("adminUpdateImage", http.MethodPatch, "/v1/admin/images/{id}",
		"Sembunyikan atau tampilkan lagi portrait (galeri dan link media)", 0, http.StatusNotFound), h.update)
	huma.Register(api, op("adminDeleteImageFiles", http.MethodDelete, "/v1/admin/images/{id}",
		"Hapus file gambar dari storage; portrait tetap tercatat dan tersembunyi permanen",
		http.StatusNoContent, http.StatusNotFound, http.StatusConflict), h.deleteFiles)
}

type AdminImageDTO struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"userId"`
	UserEmail       string     `json:"userEmail"`
	CharacterID     *uuid.UUID `json:"characterId" doc:"null bila karakter sudah dihapus"`
	CharacterName   string     `json:"characterName"`
	CharacterPublic bool       `json:"characterPublic"`
	Status          Status     `json:"status" enum:"PENDING,SUCCEEDED,FAILED"`
	Style           string     `json:"style"`
	AspectRatio     string     `json:"aspectRatio"`
	Model           string     `json:"model"`
	Prompt          string     `json:"prompt"`
	ErrorMessage    string     `json:"errorMessage,omitempty"`
	IsHidden        bool       `json:"isHidden"`
	HasFiles        bool       `json:"hasFiles" doc:"false setelah file dihapus admin"`
	ImageURL        string     `json:"imageUrl,omitempty" doc:"URL presigned langsung ke storage (berlaku PRESIGN_TTL)"`
	ThumbURL        string     `json:"thumbUrl,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt"`
}

type AdminImageList struct {
	Items []AdminImageDTO `json:"items"`
}

type AdminImagesInput struct {
	Status Status    `query:"status" enum:"PENDING,SUCCEEDED,FAILED"`
	Hidden string    `query:"hidden" enum:"true,false"`
	UserID string    `query:"userId" format:"uuid"`
	Before time.Time `query:"before" doc:"createdAt item terakhir halaman sebelumnya"`
	Limit  int       `query:"limit" minimum:"1" maximum:"100" default:"30"`
}

type AdminImageInput struct {
	ID uuid.UUID `path:"id"`
}

type AdminImageUpdate struct {
	IsHidden bool `json:"isHidden"`
}

type AdminImageUpdateInput struct {
	ID   uuid.UUID `path:"id"`
	Body AdminImageUpdate
}

type adminListOutput struct{ Body AdminImageList }
type adminImageOutput struct{ Body AdminImageDTO }

func (h *AdminHandler) list(ctx context.Context, in *AdminImagesInput) (*adminListOutput, error) {
	f := AdminFilter{Status: in.Status, Limit: in.Limit}
	if in.Hidden != "" {
		b := in.Hidden == "true"
		f.Hidden = &b
	}
	if in.UserID != "" {
		id, err := uuid.Parse(in.UserID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("userId bukan UUID")
		}
		f.UserID = &id
	}
	if !in.Before.IsZero() {
		f.Before = &in.Before
	}
	images, err := h.svc.List(ctx, f)
	return httpapi.Respond(ctx, images, err, func(imgs []AdminImage) *adminListOutput {
		items := make([]AdminImageDTO, len(imgs))
		for i, img := range imgs {
			items[i] = toAdminImageDTO(img)
		}
		return &adminListOutput{Body: AdminImageList{Items: items}}
	})
}

func (h *AdminHandler) get(ctx context.Context, in *AdminImageInput) (*adminImageOutput, error) {
	img, err := h.svc.Get(ctx, in.ID)
	return httpapi.Respond(ctx, img, err, toAdminImageOutput)
}

func (h *AdminHandler) update(ctx context.Context, in *AdminImageUpdateInput) (*adminImageOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*adminImageOutput, error) {
		img, err := h.svc.SetHidden(ctx, actor, in.ID, in.Body.IsHidden)
		return httpapi.Respond(ctx, img, err, toAdminImageOutput)
	})
}

func (h *AdminHandler) deleteFiles(ctx context.Context, in *AdminImageInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteFiles(ctx, actor, in.ID))
	})
}

func toAdminImageOutput(img AdminImage) *adminImageOutput {
	return &adminImageOutput{Body: toAdminImageDTO(img)}
}

func toAdminImageDTO(img AdminImage) AdminImageDTO {
	return AdminImageDTO{
		ID: img.ID, UserID: img.UserID, UserEmail: img.UserEmail, CharacterID: img.CharacterID,
		CharacterName: img.CharacterName, CharacterPublic: img.CharacterPublic,
		Status: img.Status, Style: img.Style, AspectRatio: img.AspectRatio, Model: img.Model,
		Prompt: img.UserPrompt, ErrorMessage: img.ErrorMessage, IsHidden: img.IsHidden, HasFiles: img.HasFiles(),
		ImageURL: img.ImageURL, ThumbURL: img.ThumbURL, CreatedAt: img.CreatedAt, CompletedAt: img.CompletedAt,
	}
}
