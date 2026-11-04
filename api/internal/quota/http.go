package quota

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Meter reports a user's usage of one quota today.
type Meter interface {
	Quota(ctx context.Context, userID uuid.UUID) (Usage, error)
}

// Handler exposes the signed-in user's quotas over HTTP.
type Handler struct {
	portrait Meter
	lore     Meter
}

// NewHandler returns a Handler; the meters may be nil when only generating
// OpenAPI.
func NewHandler(portrait, lore Meter) *Handler { return &Handler{portrait: portrait, lore: lore} }

// Register adds the quota operation to api.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "myQuota", Method: http.MethodGet, Path: "/v1/me/quota",
		Summary: "Sisa kuota generate hari ini (reset tengah malam WIB)", Tags: []string{"me"},
	}), h.quota)
}

type UsageDTO struct {
	Limit     int       `json:"limit"`
	Used      int       `json:"used"`
	Remaining int       `json:"remaining"`
	ResetsAt  time.Time `json:"resetsAt"`
}

type QuotasDTO struct {
	Portrait UsageDTO `json:"portrait"`
	Lore     UsageDTO `json:"lore"`
}

type quotaOutput struct{ Body QuotasDTO }

func (h *Handler) quota(ctx context.Context, _ *struct{}) (*quotaOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*quotaOutput, error) {
		portrait, err := h.portrait.Quota(ctx, userID)
		if err != nil {
			return nil, httpapi.Error(ctx, err)
		}
		lore, err := h.lore.Quota(ctx, userID)
		if err != nil {
			return nil, httpapi.Error(ctx, err)
		}
		return &quotaOutput{Body: QuotasDTO{Portrait: UsageDTO(portrait), Lore: UsageDTO(lore)}}, nil
	})
}

// SettingsHandler exposes the default daily limits to admins.
type SettingsHandler struct {
	svc *Settings
}

// NewSettingsHandler returns a SettingsHandler; svc may be nil when only
// generating OpenAPI.
func NewSettingsHandler(svc *Settings) *SettingsHandler { return &SettingsHandler{svc: svc} }

// Register adds the quota settings operations to api.
func (h *SettingsHandler) Register(api huma.API) {
	op := func(id, method, summary string, errs ...int) huma.Operation {
		return httpapi.AdminOnly(huma.Operation{
			OperationID: id, Method: method, Path: "/v1/admin/settings/quota", Summary: summary,
			Tags: []string{"admin"}, Errors: errs,
		})
	}
	huma.Register(api, op("adminQuotaDefaults", http.MethodGet, "Kuota harian default (user tanpa kuota sendiri)"), h.get)
	huma.Register(api, op("adminSetQuotaDefaults", http.MethodPut, "Ubah kuota harian default", http.StatusBadRequest), h.set)
}

type DefaultsDTO struct {
	Portrait int `json:"portrait" minimum:"0" maximum:"1000"`
	Lore     int `json:"lore" minimum:"0" maximum:"1000"`
}

type DefaultsInput struct {
	Body DefaultsDTO
}

type defaultsOutput struct{ Body DefaultsDTO }

func toDefaultsOutput(d Defaults) *defaultsOutput { return &defaultsOutput{Body: DefaultsDTO(d)} }

func (h *SettingsHandler) get(ctx context.Context, _ *struct{}) (*defaultsOutput, error) {
	d, err := h.svc.Defaults(ctx)
	return httpapi.Respond(ctx, d, err, toDefaultsOutput)
}

func (h *SettingsHandler) set(ctx context.Context, in *DefaultsInput) (*defaultsOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*defaultsOutput, error) {
		d, err := h.svc.SetDefaults(ctx, actor, Defaults(in.Body))
		return httpapi.Respond(ctx, d, err, toDefaultsOutput)
	})
}
