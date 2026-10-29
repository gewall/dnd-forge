package gamedata

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

func (h *Handler) registerAdmin(api huma.API) {
	tags := []string{"admin"}
	op := func(id, method, path, summary string, errs ...int) huma.Operation {
		o := huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: tags, Errors: errs}
		switch method {
		case http.MethodPost:
			o.DefaultStatus = http.StatusCreated
		case http.MethodDelete:
			o.DefaultStatus = http.StatusNoContent
		}
		return httpapi.AdminOnly(o)
	}
	const (
		bad      = http.StatusBadRequest
		missing  = http.StatusNotFound
		conflict = http.StatusConflict
	)

	huma.Register(api, op("adminListRaces", http.MethodGet, "/v1/admin/races", "Semua ras termasuk nonaktif"), h.adminListRaces)
	huma.Register(api, op("adminCreateRace", http.MethodPost, "/v1/admin/races", "Tambah ras", bad, conflict), h.adminCreateRace)
	huma.Register(api, op("adminGetRace", http.MethodGet, "/v1/admin/races/{id}", "Detail ras", missing), h.adminGetRace)
	huma.Register(api, op("adminUpdateRace", http.MethodPut, "/v1/admin/races/{id}", "Ubah ras", bad, missing, conflict), h.adminUpdateRace)
	huma.Register(api, op("adminDeleteRace", http.MethodDelete, "/v1/admin/races/{id}", "Hapus ras", missing, conflict), h.adminDeleteRace)

	huma.Register(api, op("adminCreateSubrace", http.MethodPost, "/v1/admin/races/{id}/subraces", "Tambah subras", bad, missing, conflict), h.adminCreateSubrace)
	huma.Register(api, op("adminUpdateSubrace", http.MethodPut, "/v1/admin/subraces/{id}", "Ubah subras", bad, missing, conflict), h.adminUpdateSubrace)
	huma.Register(api, op("adminDeleteSubrace", http.MethodDelete, "/v1/admin/subraces/{id}", "Hapus subras", missing, conflict), h.adminDeleteSubrace)

	huma.Register(api, op("adminListClasses", http.MethodGet, "/v1/admin/classes", "Semua kelas termasuk nonaktif"), h.adminListClasses)
	huma.Register(api, op("adminCreateClass", http.MethodPost, "/v1/admin/classes", "Tambah kelas", bad, conflict), h.adminCreateClass)
	huma.Register(api, op("adminGetClass", http.MethodGet, "/v1/admin/classes/{id}", "Detail kelas", missing), h.adminGetClass)
	huma.Register(api, op("adminUpdateClass", http.MethodPut, "/v1/admin/classes/{id}", "Ubah kelas", bad, missing, conflict), h.adminUpdateClass)
	huma.Register(api, op("adminDeleteClass", http.MethodDelete, "/v1/admin/classes/{id}", "Hapus kelas", missing, conflict), h.adminDeleteClass)

	huma.Register(api, op("adminListBackgrounds", http.MethodGet, "/v1/admin/backgrounds", "Semua latar belakang termasuk nonaktif"), h.adminListBackgrounds)
	huma.Register(api, op("adminCreateBackground", http.MethodPost, "/v1/admin/backgrounds", "Tambah latar belakang", bad, conflict), h.adminCreateBackground)
	huma.Register(api, op("adminGetBackground", http.MethodGet, "/v1/admin/backgrounds/{id}", "Detail latar belakang", missing), h.adminGetBackground)
	huma.Register(api, op("adminUpdateBackground", http.MethodPut, "/v1/admin/backgrounds/{id}", "Ubah latar belakang", bad, missing, conflict), h.adminUpdateBackground)
	huma.Register(api, op("adminDeleteBackground", http.MethodDelete, "/v1/admin/backgrounds/{id}", "Hapus latar belakang", missing, conflict), h.adminDeleteBackground)
}

// Requests -------------------------------------------------------------------

type RaceRequest struct {
	Slug         string     `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Name         string     `json:"name" minLength:"1" maxLength:"80"`
	Description  string     `json:"description" maxLength:"8000"`
	Traits       []TraitDTO `json:"traits" maxItems:"30"`
	VisualPrompt string     `json:"visualPrompt" maxLength:"500" doc:"Bahasa Inggris, dipakai prompt gambar"`
	ImageKey     string     `json:"imageKey,omitempty" pattern:"^content/[0-9a-f-]{36}\\.webp$" doc:"key dari POST /v1/admin/uploads"`
	SortOrder    int        `json:"sortOrder" minimum:"0" maximum:"10000"`
	IsActive     bool       `json:"isActive"`
}

type SubraceRequest struct {
	Slug         string     `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Name         string     `json:"name" minLength:"1" maxLength:"80"`
	Description  string     `json:"description" maxLength:"8000"`
	Traits       []TraitDTO `json:"traits" maxItems:"30"`
	VisualPrompt string     `json:"visualPrompt" maxLength:"500"`
	IsActive     bool       `json:"isActive"`
}

type ClassRequest struct {
	Slug           string     `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Name           string     `json:"name" minLength:"1" maxLength:"80"`
	HitDie         int        `json:"hitDie" enum:"6,8,10,12"`
	PrimaryAbility string     `json:"primaryAbility" minLength:"1" maxLength:"60"`
	Description    string     `json:"description" maxLength:"8000"`
	Features       []TraitDTO `json:"features" maxItems:"40"`
	VisualPrompt   string     `json:"visualPrompt" maxLength:"500"`
	ImageKey       string     `json:"imageKey,omitempty" pattern:"^content/[0-9a-f-]{36}\\.webp$"`
	SortOrder      int        `json:"sortOrder" minimum:"0" maximum:"10000"`
	IsActive       bool       `json:"isActive"`
}

type BackgroundRequest struct {
	Slug               string   `json:"slug" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"64"`
	Name               string   `json:"name" minLength:"1" maxLength:"80"`
	Description        string   `json:"description" maxLength:"8000"`
	SkillProficiencies []string `json:"skillProficiencies" maxItems:"10"`
	VisualPrompt       string   `json:"visualPrompt" maxLength:"500"`
	ImageKey           string   `json:"imageKey,omitempty" pattern:"^content/[0-9a-f-]{36}\\.webp$"`
	SortOrder          int      `json:"sortOrder" minimum:"0" maximum:"10000"`
	IsActive           bool     `json:"isActive"`
}

type idInput struct {
	ID uuid.UUID `path:"id"`
}

type createRaceInput struct{ Body RaceRequest }
type updateRaceInput struct {
	idInput
	Body RaceRequest
}
type createSubraceInput struct {
	idInput
	Body SubraceRequest
}
type subraceOutput struct{ Body SubraceDTO }
type createClassInput struct{ Body ClassRequest }
type updateClassInput struct {
	idInput
	Body ClassRequest
}
type createBackgroundInput struct{ Body BackgroundRequest }
type updateBackgroundInput struct {
	idInput
	Body BackgroundRequest
}

func (r RaceRequest) input() RaceInput {
	return RaceInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, Traits: traitsFrom(r.Traits),
		VisualPrompt: r.VisualPrompt, ImageKey: r.ImageKey, SortOrder: r.SortOrder, IsActive: r.IsActive,
	}
}

func (r SubraceRequest) input() SubraceInput {
	return SubraceInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, Traits: traitsFrom(r.Traits),
		VisualPrompt: r.VisualPrompt, IsActive: r.IsActive,
	}
}

func (r ClassRequest) input() ClassInput {
	return ClassInput{
		Slug: r.Slug, Name: r.Name, HitDie: r.HitDie, PrimaryAbility: r.PrimaryAbility, Description: r.Description,
		Features: traitsFrom(r.Features), VisualPrompt: r.VisualPrompt, ImageKey: r.ImageKey,
		SortOrder: r.SortOrder, IsActive: r.IsActive,
	}
}

func (r BackgroundRequest) input() BackgroundInput {
	skills := r.SkillProficiencies
	if skills == nil {
		skills = []string{}
	}
	return BackgroundInput{
		Slug: r.Slug, Name: r.Name, Description: r.Description, SkillProficiencies: skills,
		VisualPrompt: r.VisualPrompt, ImageKey: r.ImageKey, SortOrder: r.SortOrder, IsActive: r.IsActive,
	}
}

// Handlers -------------------------------------------------------------------

func (h *Handler) adminListRaces(ctx context.Context, _ *struct{}) (*racesOutput, error) {
	rs, err := h.svc.AllRaces(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &racesOutput{Body: RaceList{Items: mapSlice(rs, h.raceDTO)}}, nil
}

func (h *Handler) adminGetRace(ctx context.Context, in *idInput) (*raceOutput, error) {
	r, err := h.svc.RaceByID(ctx, in.ID)
	return httpapi.Respond(ctx, r, err, func(r Race) *raceOutput { return &raceOutput{Body: h.raceDTO(r)} })
}

func (h *Handler) adminCreateRace(ctx context.Context, in *createRaceInput) (*raceOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*raceOutput, error) {
		r, err := h.svc.CreateRace(ctx, actor, in.Body.input())
		return httpapi.Respond(ctx, r, err, func(r Race) *raceOutput { return &raceOutput{Body: h.raceDTO(r)} })
	})
}

func (h *Handler) adminUpdateRace(ctx context.Context, in *updateRaceInput) (*raceOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*raceOutput, error) {
		r, err := h.svc.UpdateRace(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, r, err, func(r Race) *raceOutput { return &raceOutput{Body: h.raceDTO(r)} })
	})
}

func (h *Handler) adminDeleteRace(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteRace(ctx, actor, in.ID))
	})
}

func (h *Handler) adminCreateSubrace(ctx context.Context, in *createSubraceInput) (*subraceOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*subraceOutput, error) {
		sr, err := h.svc.CreateSubrace(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, sr, err, func(sr Subrace) *subraceOutput { return &subraceOutput{Body: subraceDTO(sr)} })
	})
}

func (h *Handler) adminUpdateSubrace(ctx context.Context, in *createSubraceInput) (*subraceOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*subraceOutput, error) {
		sr, err := h.svc.UpdateSubrace(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, sr, err, func(sr Subrace) *subraceOutput { return &subraceOutput{Body: subraceDTO(sr)} })
	})
}

func (h *Handler) adminDeleteSubrace(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteSubrace(ctx, actor, in.ID))
	})
}

func (h *Handler) adminListClasses(ctx context.Context, _ *struct{}) (*classesOutput, error) {
	cs, err := h.svc.AllClasses(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &classesOutput{Body: ClassList{Items: mapSlice(cs, h.classDTO)}}, nil
}

func (h *Handler) adminGetClass(ctx context.Context, in *idInput) (*classOutput, error) {
	c, err := h.svc.ClassByID(ctx, in.ID)
	return httpapi.Respond(ctx, c, err, func(c Class) *classOutput { return &classOutput{Body: h.classDTO(c)} })
}

func (h *Handler) adminCreateClass(ctx context.Context, in *createClassInput) (*classOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*classOutput, error) {
		c, err := h.svc.CreateClass(ctx, actor, in.Body.input())
		return httpapi.Respond(ctx, c, err, func(c Class) *classOutput { return &classOutput{Body: h.classDTO(c)} })
	})
}

func (h *Handler) adminUpdateClass(ctx context.Context, in *updateClassInput) (*classOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*classOutput, error) {
		c, err := h.svc.UpdateClass(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, c, err, func(c Class) *classOutput { return &classOutput{Body: h.classDTO(c)} })
	})
}

func (h *Handler) adminDeleteClass(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteClass(ctx, actor, in.ID))
	})
}

func (h *Handler) adminListBackgrounds(ctx context.Context, _ *struct{}) (*backgroundsOutput, error) {
	bs, err := h.svc.AllBackgrounds(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &backgroundsOutput{Body: BackgroundList{Items: mapSlice(bs, h.backgroundDTO)}}, nil
}

func (h *Handler) adminGetBackground(ctx context.Context, in *idInput) (*backgroundOutput, error) {
	b, err := h.svc.BackgroundByID(ctx, in.ID)
	return httpapi.Respond(ctx, b, err, func(b Background) *backgroundOutput { return &backgroundOutput{Body: h.backgroundDTO(b)} })
}

func (h *Handler) adminCreateBackground(ctx context.Context, in *createBackgroundInput) (*backgroundOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*backgroundOutput, error) {
		b, err := h.svc.CreateBackground(ctx, actor, in.Body.input())
		return httpapi.Respond(ctx, b, err, func(b Background) *backgroundOutput { return &backgroundOutput{Body: h.backgroundDTO(b)} })
	})
}

func (h *Handler) adminUpdateBackground(ctx context.Context, in *updateBackgroundInput) (*backgroundOutput, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*backgroundOutput, error) {
		b, err := h.svc.UpdateBackground(ctx, actor, in.ID, in.Body.input())
		return httpapi.Respond(ctx, b, err, func(b Background) *backgroundOutput { return &backgroundOutput{Body: h.backgroundDTO(b)} })
	})
}

func (h *Handler) adminDeleteBackground(ctx context.Context, in *idInput) (*struct{}, error) {
	return httpapi.AsUser(ctx, func(actor uuid.UUID) (*struct{}, error) {
		return nil, httpapi.NoContent(ctx, h.svc.DeleteBackground(ctx, actor, in.ID))
	})
}
