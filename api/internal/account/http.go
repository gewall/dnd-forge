package account

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/identity"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// Handler exposes account administration over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the admin user operations to api.
func (h *Handler) Register(api huma.API) {
	op := func(id, method, path, summary string, errs ...int) huma.Operation {
		return httpapi.AdminOnly(huma.Operation{
			OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"admin"}, Errors: errs,
		})
	}
	huma.Register(api, op("adminListUsers", http.MethodGet, "/v1/admin/users",
		"Daftar user, terbaru dulu (cari email/nama, filter role/blokir)", http.StatusBadRequest), h.list)
	huma.Register(api, op("adminGetUser", http.MethodGet, "/v1/admin/users/{id}",
		"Detail user: total generate dan pemakaian kuota hari ini", http.StatusNotFound), h.get)
	huma.Register(api, op("adminUpdateUser", http.MethodPut, "/v1/admin/users/{id}",
		"Ubah role, blokir (keluar dari semua sesi) dan kuota harian sendiri; null = default",
		http.StatusBadRequest, http.StatusNotFound), h.update)
}

type AccountDTO struct {
	ID            uuid.UUID     `json:"id"`
	Email         string        `json:"email"`
	Name          string        `json:"name"`
	Role          identity.Role `json:"role" enum:"USER,ADMIN"`
	IsBanned      bool          `json:"isBanned"`
	PortraitQuota *int          `json:"portraitQuota" nullable:"true" doc:"kuota portrait harian sendiri; null = default"`
	LoreQuota     *int          `json:"loreQuota" nullable:"true" doc:"kuota lore harian sendiri; null = default"`
	HasPassword   bool          `json:"hasPassword"`
	HasGoogle     bool          `json:"hasGoogle"`
	Characters    int           `json:"characters"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type AccountList struct {
	Items []AccountDTO `json:"items"`
}

type DetailDTO struct {
	AccountDTO
	Portraits     int            `json:"portraits" doc:"semua permintaan portrait, termasuk yang gagal"`
	Lores         int            `json:"lores"`
	PortraitUsage quota.UsageDTO `json:"portraitUsage"`
	LoreUsage     quota.UsageDTO `json:"loreUsage"`
}

type AccountRequest struct {
	Role          identity.Role `json:"role" enum:"USER,ADMIN"`
	IsBanned      bool          `json:"isBanned"`
	PortraitQuota *int          `json:"portraitQuota" required:"true" nullable:"true" minimum:"0" maximum:"1000"`
	LoreQuota     *int          `json:"loreQuota" required:"true" nullable:"true" minimum:"0" maximum:"1000"`
}

type ListInput struct {
	Query  string        `query:"q" maxLength:"100" doc:"bagian dari email atau nama"`
	Role   identity.Role `query:"role" enum:"USER,ADMIN"`
	Banned string        `query:"banned" enum:"true,false"`
	Before time.Time     `query:"before" doc:"createdAt item terakhir halaman sebelumnya"`
	Limit  int           `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type IDInput struct {
	ID uuid.UUID `path:"id"`
}

type UpdateInput struct {
	ID   uuid.UUID `path:"id"`
	Body AccountRequest
}

type listOutput struct{ Body AccountList }
type detailOutput struct{ Body DetailDTO }

func (h *Handler) list(ctx context.Context, in *ListInput) (*listOutput, error) {
	f := Filter{Query: in.Query, Role: in.Role, Limit: in.Limit}
	if in.Banned != "" {
		b := in.Banned == "true"
		f.Banned = &b
	}
	if !in.Before.IsZero() {
		f.Before = &in.Before
	}
	accounts, err := h.svc.List(ctx, f)
	return httpapi.Respond(ctx, accounts, err, func(as []Account) *listOutput {
		items := make([]AccountDTO, len(as))
		for i, a := range as {
			items[i] = toDTO(a)
		}
		return &listOutput{Body: AccountList{Items: items}}
	})
}

func (h *Handler) get(ctx context.Context, in *IDInput) (*detailOutput, error) {
	d, err := h.svc.Get(ctx, in.ID)
	return httpapi.Respond(ctx, d, err, toDetail)
}

func (h *Handler) update(ctx context.Context, in *UpdateInput) (*detailOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*detailOutput, error) {
		b := in.Body
		d, err := h.svc.Update(ctx, actor, in.ID, Changes(b))
		return httpapi.Respond(ctx, d, err, toDetail)
	})
}

func toDTO(a Account) AccountDTO { return AccountDTO(a) }

func toDetail(d Detail) *detailOutput {
	return &detailOutput{Body: DetailDTO{
		AccountDTO: toDTO(d.Account), Portraits: d.Portraits, Lores: d.Lores,
		PortraitUsage: quota.UsageDTO(d.PortraitUsage), LoreUsage: quota.UsageDTO(d.LoreUsage),
	}}
}
