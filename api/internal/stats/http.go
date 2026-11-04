package stats

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Handler exposes the admin dashboard over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the stats operation to api.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminStats", Method: http.MethodGet, Path: "/v1/admin/stats",
		Summary: "Ringkasan dashboard: total, aktivitas harian (WIB) dan pemakaian AI per model",
		Description: "Permintaan FAILED tidak memotong kuota tapi tetap memanggil provider, jadi dihitung terpisah. " +
			"Token hanya ada untuk provider teks yang melaporkannya; tidak ada estimasi biaya karena harga vendor berubah.",
		Tags: []string{"admin"}, Errors: []int{http.StatusBadRequest},
	}), h.report)
}

type TotalsDTO struct {
	Users              int `json:"users"`
	BannedUsers        int `json:"bannedUsers"`
	Admins             int `json:"admins"`
	Characters         int `json:"characters"`
	PublicCharacters   int `json:"publicCharacters"`
	Portraits          int `json:"portraits"`
	PortraitsSucceeded int `json:"portraitsSucceeded"`
	PortraitsFailed    int `json:"portraitsFailed"`
	PortraitsPending   int `json:"portraitsPending"`
	PortraitsHidden    int `json:"portraitsHidden"`
	Lores              int `json:"lores"`
	LoresSucceeded     int `json:"loresSucceeded"`
	LoresFailed        int `json:"loresFailed"`
	LoresPending       int `json:"loresPending"`
}

type DayDTO struct {
	Date               string `json:"date" format:"date" example:"2026-11-04"`
	NewUsers           int    `json:"newUsers"`
	NewCharacters      int    `json:"newCharacters"`
	PortraitsSucceeded int    `json:"portraitsSucceeded"`
	PortraitsFailed    int    `json:"portraitsFailed"`
	LoresSucceeded     int    `json:"loresSucceeded"`
	LoresFailed        int    `json:"loresFailed"`
}

type ModelUsageDTO struct {
	Capability       string    `json:"capability" enum:"image,text"`
	ProviderID       uuid.UUID `json:"providerId"`
	ProviderLabel    string    `json:"providerLabel"`
	Model            string    `json:"model"`
	Requests         int       `json:"requests"`
	Succeeded        int       `json:"succeeded"`
	Failed           int       `json:"failed"`
	PromptTokens     int64     `json:"promptTokens"`
	CompletionTokens int64     `json:"completionTokens"`
}

type ReportDTO struct {
	From   string          `json:"from" format:"date"`
	To     string          `json:"to" format:"date"`
	Totals TotalsDTO       `json:"totals" doc:"sepanjang waktu"`
	Days   []DayDTO        `json:"days" doc:"satu baris per hari WIB dalam periode, termasuk hari kosong"`
	Models []ModelUsageDTO `json:"models" doc:"dalam periode, tersibuk dulu"`
}

type ReportInput struct {
	Days int `query:"days" minimum:"1" maximum:"90" default:"30"`
}

type reportOutput struct{ Body ReportDTO }

const dateLayout = "2006-01-02"

func (h *Handler) report(ctx context.Context, in *ReportInput) (*reportOutput, error) {
	r, err := h.svc.Report(ctx, in.Days)
	return httpapi.Respond(ctx, r, err, func(r Report) *reportOutput {
		out := ReportDTO{
			From: r.From.Format(dateLayout), To: r.To.Format(dateLayout), Totals: TotalsDTO(r.Totals),
			Days: make([]DayDTO, len(r.Days)), Models: make([]ModelUsageDTO, len(r.Models)),
		}
		for i, d := range r.Days {
			out.Days[i] = DayDTO{
				Date: d.Date.Format(dateLayout), NewUsers: d.NewUsers, NewCharacters: d.NewCharacters,
				PortraitsSucceeded: d.PortraitsSucceeded, PortraitsFailed: d.PortraitsFailed,
				LoresSucceeded: d.LoresSucceeded, LoresFailed: d.LoresFailed,
			}
		}
		for i, m := range r.Models {
			out.Models[i] = ModelUsageDTO(m)
		}
		return &reportOutput{Body: out}
	})
}
