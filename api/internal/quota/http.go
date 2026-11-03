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
