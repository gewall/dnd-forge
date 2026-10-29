package media

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/imaging"
)

// Handler exposes uploads and media redirects over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the media operations to api.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID:   "uploadImage",
		Method:        http.MethodPost,
		Path:          "/v1/admin/uploads",
		Summary:       "Upload gambar konten (cover, ras, kelas, latar belakang)",
		Description:   "Gambar dikonversi ke WebP maks 1600px. Pakai `key` di field imageKey/coverKey.",
		Tags:          []string{"admin"},
		DefaultStatus: http.StatusCreated,
		MaxBodyBytes:  maxUploadBytes + 1<<20,
		Errors:        []int{http.StatusBadRequest},
	}), h.upload)

	huma.Register(api, huma.Operation{
		OperationID:   "contentImage",
		Method:        http.MethodGet,
		Path:          "/v1/media/content/{name}",
		Summary:       "Redirect ke URL bertanda tangan untuk gambar konten (tanpa internal key)",
		Tags:          []string{"media"},
		DefaultStatus: http.StatusFound,
		Errors:        []int{http.StatusNotFound},
	}, h.contentImage)
}

type UploadForm struct {
	File huma.FormFile `form:"file" contentType:"image/jpeg,image/png,image/webp" required:"true"`
}

type UploadDTO struct {
	Key    string `json:"key" example:"content/4f1c7a52-0d8e-4c6b-9a43-0f7d2b1e9c11.webp"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type uploadInput struct {
	RawBody huma.MultipartFormFiles[UploadForm]
}

type uploadOutput struct {
	Body UploadDTO
}

type contentInput struct {
	Name string `path:"name" maxLength:"64"`
}

type redirectOutput struct {
	Location     string `header:"Location"`
	CacheControl string `header:"Cache-Control"`
}

func (h *Handler) upload(ctx context.Context, in *uploadInput) (*uploadOutput, error) {
	f := in.RawBody.Data().File
	defer f.Close()
	data, err := imaging.ReadLimited(f, maxUploadBytes)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	up, err := h.svc.UploadContentImage(ctx, data)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &uploadOutput{Body: UploadDTO{Key: up.Key, URL: up.URL, Width: up.Width, Height: up.Height}}, nil
}

func (h *Handler) contentImage(ctx context.Context, in *contentInput) (*redirectOutput, error) {
	url, ttl, err := h.svc.ContentURL(ctx, in.Name)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	// Let browsers reuse the redirect for most of the signature's lifetime.
	maxAge := int((ttl - 10*time.Minute).Seconds())
	return &redirectOutput{
		Location:     url,
		CacheControl: "public, max-age=" + strconv.Itoa(max(maxAge, 0)),
	}, nil
}
