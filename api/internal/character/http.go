package character

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// Handler exposes the character builder over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the character operations to api. All of them act on the
// signed-in user's own characters.
func (h *Handler) Register(api huma.API) {
	tags := []string{"characters"}
	op := func(id, method, path, summary string, errs ...int) huma.Operation {
		o := huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: tags, Errors: errs}
		switch method {
		case http.MethodPost:
			o.DefaultStatus = http.StatusCreated
		case http.MethodDelete:
			o.DefaultStatus = http.StatusNoContent
		}
		return httpapi.Authenticated(o)
	}
	const (
		bad     = http.StatusBadRequest
		missing = http.StatusNotFound
	)

	huma.Register(api, op("listCharacters", http.MethodGet, "/v1/characters", "Karakter milik user"), h.list)
	huma.Register(api, op("createCharacter", http.MethodPost, "/v1/characters", "Simpan karakter dari wizard", bad), h.create)
	huma.Register(api, op("getCharacter", http.MethodGet, "/v1/characters/{id}", "Detail karakter", missing), h.get)
	huma.Register(api, op("updateCharacter", http.MethodPut, "/v1/characters/{id}", "Ubah karakter", bad, missing), h.update)
	huma.Register(api, op("deleteCharacter", http.MethodDelete, "/v1/characters/{id}", "Hapus karakter", missing), h.delete)
}

// DTO ------------------------------------------------------------------------

type CharacterRequest struct {
	Name         string     `json:"name" minLength:"1" maxLength:"60"`
	Gender       string     `json:"gender,omitempty" maxLength:"30"`
	RaceID       uuid.UUID  `json:"raceId"`
	SubraceID    *uuid.UUID `json:"subraceId,omitempty" doc:"wajib bila ras punya subras aktif"`
	ClassID      uuid.UUID  `json:"classId"`
	BackgroundID uuid.UUID  `json:"backgroundId"`
	Appearance   string     `json:"appearance,omitempty" maxLength:"1000" doc:"detail penampilan, dipakai prompt portrait"`
	IsPublic     bool       `json:"isPublic,omitempty" doc:"tampil di galeri publik"`
}

type RefDTO struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name string    `json:"name"`
}

type CharacterDTO struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Gender     string    `json:"gender"`
	Appearance string    `json:"appearance"`
	IsPublic   bool      `json:"isPublic"`
	Race       RefDTO    `json:"race"`
	Subrace    *RefDTO   `json:"subrace"`
	Class      RefDTO    `json:"class"`
	Background RefDTO    `json:"background"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CharacterList struct {
	Items []CharacterDTO `json:"items"`
}

type idInput struct {
	ID uuid.UUID `path:"id"`
}

type createInput struct{ Body CharacterRequest }
type updateInput struct {
	ID   uuid.UUID `path:"id"`
	Body CharacterRequest
}
type characterOutput struct{ Body CharacterDTO }
type listOutput struct{ Body CharacterList }

// Handlers -------------------------------------------------------------------

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*listOutput, error) {
		cs, err := h.svc.List(ctx, userID)
		return httpapi.Respond(ctx, cs, err, func(cs []Character) *listOutput {
			items := make([]CharacterDTO, len(cs))
			for i, c := range cs {
				items[i] = toDTO(c)
			}
			return &listOutput{Body: CharacterList{Items: items}}
		})
	})
}

func (h *Handler) get(ctx context.Context, in *idInput) (*characterOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*characterOutput, error) {
		c, err := h.svc.Get(ctx, userID, in.ID)
		return httpapi.Respond(ctx, c, err, toOutput)
	})
}

func (h *Handler) create(ctx context.Context, in *createInput) (*characterOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*characterOutput, error) {
		c, err := h.svc.Create(ctx, userID, in.Body.draft())
		return httpapi.Respond(ctx, c, err, toOutput)
	})
}

func (h *Handler) update(ctx context.Context, in *updateInput) (*characterOutput, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*characterOutput, error) {
		c, err := h.svc.Update(ctx, userID, in.ID, in.Body.draft())
		return httpapi.Respond(ctx, c, err, toOutput)
	})
}

func (h *Handler) delete(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(userID uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.Delete(ctx, userID, in.ID))
	})
}

// Mapping --------------------------------------------------------------------

func (r CharacterRequest) draft() Draft { return Draft(r) }

func toOutput(c Character) *characterOutput { return &characterOutput{Body: toDTO(c)} }

func toDTO(c Character) CharacterDTO {
	d := CharacterDTO{
		ID: c.ID, Name: c.Name, Gender: c.Gender, Appearance: c.Appearance, IsPublic: c.IsPublic,
		Race: RefDTO(c.Race), Class: RefDTO(c.Class), Background: RefDTO(c.Background),
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.Subrace != nil {
		sr := RefDTO(*c.Subrace)
		d.Subrace = &sr
	}
	return d
}
