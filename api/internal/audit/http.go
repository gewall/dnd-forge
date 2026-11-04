package audit

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Handler exposes the audit log to admins.
type Handler struct {
	r *Reader
}

// NewHandler returns a Handler; r may be nil when only generating OpenAPI.
func NewHandler(r *Reader) *Handler { return &Handler{r: r} }

// Register adds the audit log operation to api.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminAuditLogs", Method: http.MethodGet, Path: "/v1/admin/audit-logs",
		Summary:     "Riwayat perubahan oleh admin, terbaru dulu",
		Description: "Paginasi cursor: kirim `id` item terakhir sebagai `beforeId`. `diff` berisi nilai [lama, baru] per kolom.",
		Tags:        []string{"admin"},
	}), h.list)
}

type RecordDTO struct {
	ID         int64      `json:"id"`
	ActorID    *uuid.UUID `json:"actorId" doc:"null untuk sistem atau user yang sudah dihapus"`
	ActorEmail string     `json:"actorEmail,omitempty"`
	Action     string     `json:"action" example:"update"`
	Entity     string     `json:"entity" example:"user"`
	EntityID   string     `json:"entityId"`
	Diff       any        `json:"diff,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type RecordList struct {
	Items []RecordDTO `json:"items"`
}

type ListInput struct {
	Entity   string `query:"entity" maxLength:"40" example:"ai_provider"`
	EntityID string `query:"entityId" maxLength:"100"`
	ActorID  string `query:"actorId" format:"uuid"`
	BeforeID int64  `query:"beforeId" minimum:"1"`
	Limit    int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

type listOutput struct{ Body RecordList }

func (h *Handler) list(ctx context.Context, in *ListInput) (*listOutput, error) {
	f := Filter{Entity: in.Entity, EntityID: in.EntityID, BeforeID: in.BeforeID, Limit: in.Limit}
	if in.ActorID != "" {
		id, err := uuid.Parse(in.ActorID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("actorId bukan UUID")
		}
		f.ActorID = &id
	}
	recs, err := h.r.List(ctx, f)
	return httpapi.Respond(ctx, recs, err, func(rs []Record) *listOutput {
		items := make([]RecordDTO, len(rs))
		for i, r := range rs {
			items[i] = RecordDTO(r)
		}
		return &listOutput{Body: RecordList{Items: items}}
	})
}
