package lore

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/prompt"
)

// Handler exposes lore drafts over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the lore operations to api. All of them act on the
// signed-in user's own characters and drafts.
func (h *Handler) Register(api huma.API) {
	tags := []string{"lore"}
	const (
		bad         = http.StatusBadRequest
		missing     = http.StatusNotFound
		unavailable = http.StatusServiceUnavailable
	)

	huma.Register(api, httpapi.RateLimited(httpapi.Authenticated(huma.Operation{
		OperationID: "requestLore", Method: http.MethodPost, Path: "/v1/characters/{id}/lore",
		Summary: "Minta draf lore (kisah latar) AI untuk karakter",
		Description: "Draf ditulis di background. Poll `GET /v1/lore/{id}` sampai status bukan PENDING, lalu " +
			"`POST /v1/lore/{id}/apply` untuk menjadikannya backstory. 429 bila kuota harian habis; draf FAILED tidak memotong kuota.",
		Tags: tags, DefaultStatus: http.StatusAccepted, Errors: []int{bad, missing, unavailable},
	}), "lore"), h.request)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "listCharacterLore", Method: http.MethodGet, Path: "/v1/characters/{id}/lore",
		Summary: "Riwayat draf lore karakter (50 terbaru)", Tags: tags,
	}), h.listForCharacter)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "getLore", Method: http.MethodGet, Path: "/v1/lore/{id}",
		Summary: "Status dan isi draf lore", Tags: tags, Errors: []int{missing},
	}), h.get)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "applyLore", Method: http.MethodPost, Path: "/v1/lore/{id}/apply",
		Summary: "Jadikan draf lore sebagai backstory karakter", Tags: tags,
		DefaultStatus: http.StatusNoContent, Errors: []int{bad, missing},
	}), h.apply)
	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "loreOptions", Method: http.MethodGet, Path: "/v1/lore-options",
		Summary: "Nada, panjang, dan bahasa lore yang bisa dipilih, serta placeholder template", Tags: tags,
	}), h.options)
}

// DTO ------------------------------------------------------------------------

type LoreRequest struct {
	Tone     string `json:"tone" minLength:"1" doc:"id dari GET /v1/lore-options"`
	Length   string `json:"length,omitempty" default:"medium" doc:"short, medium, atau long"`
	Language string `json:"language,omitempty" default:"id" doc:"id atau en"`
	Prompt   string `json:"prompt,omitempty" maxLength:"500" doc:"keinginan user tentang kisahnya (dimoderasi)"`
}

type LoreDTO struct {
	ID           uuid.UUID  `json:"id"`
	CharacterID  *uuid.UUID `json:"characterId" doc:"null bila karakter sudah dihapus"`
	Status       Status     `json:"status" enum:"PENDING,SUCCEEDED,FAILED"`
	Tone         string     `json:"tone"`
	Length       string     `json:"length"`
	Language     string     `json:"language"`
	Prompt       string     `json:"prompt"`
	Content      string     `json:"content,omitempty" doc:"ada bila status SUCCEEDED"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

type LoreList struct {
	Items []LoreDTO `json:"items"`
}

type OptionDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type OptionsDTO struct {
	Tones        []OptionDTO `json:"tones"`
	Lengths      []OptionDTO `json:"lengths"`
	Languages    []OptionDTO `json:"languages"`
	Placeholders []string    `json:"placeholders" doc:"untuk template prompt jenis lore"`
}

type idInput struct {
	ID uuid.UUID `path:"id"`
}

type requestInput struct {
	ID   uuid.UUID `path:"id" doc:"id karakter"`
	Body LoreRequest
}

type loreOutput struct{ Body LoreDTO }
type listOutput struct{ Body LoreList }
type optionsOutput struct{ Body OptionsDTO }

// Handlers -------------------------------------------------------------------

func (h *Handler) request(ctx context.Context, in *requestInput) (*loreOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*loreOutput, error) {
		b := in.Body
		l, err := h.svc.Request(ctx, userID, in.ID, Request{Tone: b.Tone, Length: b.Length, Language: b.Language, UserPrompt: b.Prompt})
		return httpapi.Respond(ctx, l, err, toOutput)
	})
}

func (h *Handler) listForCharacter(ctx context.Context, in *idInput) (*listOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*listOutput, error) {
		ls, err := h.svc.ListForCharacter(ctx, userID, in.ID)
		return httpapi.Respond(ctx, ls, err, func(ls []Lore) *listOutput {
			items := make([]LoreDTO, len(ls))
			for i, l := range ls {
				items[i] = toDTO(l)
			}
			return &listOutput{Body: LoreList{Items: items}}
		})
	})
}

func (h *Handler) get(ctx context.Context, in *idInput) (*loreOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*loreOutput, error) {
		l, err := h.svc.Get(ctx, userID, in.ID)
		return httpapi.Respond(ctx, l, err, toOutput)
	})
}

func (h *Handler) apply(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Apply(ctx, userID, in.ID))
	})
}

func (h *Handler) options(context.Context, *struct{}) (*optionsOutput, error) {
	return &optionsOutput{Body: OptionsDTO{
		Tones:        toOptionDTOs(sortedOptions(Tones, func(o Option) Option { return o })),
		Lengths:      toOptionDTOs(sortedOptions(Lengths, func(l length) Option { return l.Option })),
		Languages:    toOptionDTOs(sortedOptions(Languages, func(o Option) Option { return o })),
		Placeholders: prompt.Placeholders(prompt.KindLore),
	}}, nil
}

// Mapping --------------------------------------------------------------------

func toOutput(l Lore) *loreOutput { return &loreOutput{Body: toDTO(l)} }

func toDTO(l Lore) LoreDTO {
	return LoreDTO{
		ID: l.ID, CharacterID: l.CharacterID, Status: l.Status, Tone: l.Tone, Length: l.Length, Language: l.Language,
		Prompt: l.UserPrompt, Content: l.Content, ErrorMessage: l.ErrorMessage, CreatedAt: l.CreatedAt, CompletedAt: l.CompletedAt,
	}
}

func toOptionDTOs(os []Option) []OptionDTO {
	out := make([]OptionDTO, len(os))
	for i, o := range os {
		out[i] = OptionDTO{ID: o.ID, Description: o.Description}
	}
	return out
}
