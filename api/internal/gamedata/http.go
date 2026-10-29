package gamedata

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// ImageURLs turns a stored image key into a public URL.
type ImageURLs interface {
	Content(key string) string
}

// Handler exposes game data over HTTP.
type Handler struct {
	svc  *Service
	urls ImageURLs
}

// NewHandler returns a Handler; arguments may be nil when only generating
// OpenAPI.
func NewHandler(svc *Service, urls ImageURLs) *Handler {
	return &Handler{svc: svc, urls: urls}
}

// Register adds the public and admin game data operations to api.
func (h *Handler) Register(api huma.API) {
	h.registerPublic(api)
	h.registerAdmin(api)
}

func (h *Handler) registerPublic(api huma.API) {
	tags := []string{"game data"}
	get := func(id, path, summary string) huma.Operation {
		return huma.Operation{OperationID: id, Method: http.MethodGet, Path: path, Summary: summary, Tags: tags}
	}
	notFound := func(op huma.Operation) huma.Operation { op.Errors = []int{http.StatusNotFound}; return op }

	huma.Register(api, get("listRaces", "/v1/races", "Daftar ras aktif beserta subras"), h.listRaces)
	huma.Register(api, notFound(get("getRace", "/v1/races/{slug}", "Detail ras")), h.getRace)
	huma.Register(api, get("listClasses", "/v1/classes", "Daftar kelas aktif"), h.listClasses)
	huma.Register(api, notFound(get("getClass", "/v1/classes/{slug}", "Detail kelas")), h.getClass)
	huma.Register(api, get("listBackgrounds", "/v1/backgrounds", "Daftar latar belakang aktif"), h.listBackgrounds)
	huma.Register(api, notFound(get("getBackground", "/v1/backgrounds/{slug}", "Detail latar belakang")), h.getBackground)
}

// DTO ------------------------------------------------------------------------

type TraitDTO struct {
	Name        string `json:"name" minLength:"1" maxLength:"120"`
	Description string `json:"description" maxLength:"4000"`
}

type SubraceDTO struct {
	ID           uuid.UUID  `json:"id"`
	RaceID       uuid.UUID  `json:"raceId"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Traits       []TraitDTO `json:"traits"`
	VisualPrompt string     `json:"visualPrompt"`
	IsActive     bool       `json:"isActive"`
}

type RaceDTO struct {
	ID           uuid.UUID    `json:"id"`
	Slug         string       `json:"slug"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Traits       []TraitDTO   `json:"traits"`
	VisualPrompt string       `json:"visualPrompt"`
	ImageKey     string       `json:"imageKey,omitempty"`
	ImageURL     string       `json:"imageUrl,omitempty"`
	SortOrder    int          `json:"sortOrder"`
	IsActive     bool         `json:"isActive"`
	Subraces     []SubraceDTO `json:"subraces"`
}

type ClassDTO struct {
	ID             uuid.UUID  `json:"id"`
	Slug           string     `json:"slug"`
	Name           string     `json:"name"`
	HitDie         int        `json:"hitDie"`
	PrimaryAbility string     `json:"primaryAbility"`
	Description    string     `json:"description"`
	Features       []TraitDTO `json:"features"`
	VisualPrompt   string     `json:"visualPrompt"`
	ImageKey       string     `json:"imageKey,omitempty"`
	ImageURL       string     `json:"imageUrl,omitempty"`
	SortOrder      int        `json:"sortOrder"`
	IsActive       bool       `json:"isActive"`
}

type BackgroundDTO struct {
	ID                 uuid.UUID `json:"id"`
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	SkillProficiencies []string  `json:"skillProficiencies"`
	VisualPrompt       string    `json:"visualPrompt"`
	ImageKey           string    `json:"imageKey,omitempty"`
	ImageURL           string    `json:"imageUrl,omitempty"`
	SortOrder          int       `json:"sortOrder"`
	IsActive           bool      `json:"isActive"`
}

type RaceList struct {
	Items []RaceDTO `json:"items"`
}

type ClassList struct {
	Items []ClassDTO `json:"items"`
}

type BackgroundList struct {
	Items []BackgroundDTO `json:"items"`
}

type slugInput struct {
	Slug string `path:"slug" maxLength:"64"`
}

type raceOutput struct{ Body RaceDTO }
type racesOutput struct{ Body RaceList }
type classOutput struct{ Body ClassDTO }
type classesOutput struct{ Body ClassList }
type backgroundOutput struct{ Body BackgroundDTO }
type backgroundsOutput struct{ Body BackgroundList }

// Public handlers ------------------------------------------------------------

func (h *Handler) listRaces(ctx context.Context, _ *struct{}) (*racesOutput, error) {
	rs, err := h.svc.Races(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &racesOutput{Body: RaceList{Items: mapSlice(rs, h.raceDTO)}}, nil
}

func (h *Handler) getRace(ctx context.Context, in *slugInput) (*raceOutput, error) {
	r, err := h.svc.Race(ctx, in.Slug)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &raceOutput{Body: h.raceDTO(r)}, nil
}

func (h *Handler) listClasses(ctx context.Context, _ *struct{}) (*classesOutput, error) {
	cs, err := h.svc.Classes(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &classesOutput{Body: ClassList{Items: mapSlice(cs, h.classDTO)}}, nil
}

func (h *Handler) getClass(ctx context.Context, in *slugInput) (*classOutput, error) {
	c, err := h.svc.Class(ctx, in.Slug)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &classOutput{Body: h.classDTO(c)}, nil
}

func (h *Handler) listBackgrounds(ctx context.Context, _ *struct{}) (*backgroundsOutput, error) {
	bs, err := h.svc.Backgrounds(ctx)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &backgroundsOutput{Body: BackgroundList{Items: mapSlice(bs, h.backgroundDTO)}}, nil
}

func (h *Handler) getBackground(ctx context.Context, in *slugInput) (*backgroundOutput, error) {
	b, err := h.svc.Background(ctx, in.Slug)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &backgroundOutput{Body: h.backgroundDTO(b)}, nil
}

// Mapping --------------------------------------------------------------------

func (h *Handler) raceDTO(r Race) RaceDTO {
	return RaceDTO{
		ID: r.ID, Slug: r.Slug, Name: r.Name, Description: r.Description, Traits: traitDTOs(r.Traits),
		VisualPrompt: r.VisualPrompt, ImageKey: r.ImageKey, ImageURL: h.urls.Content(r.ImageKey),
		SortOrder: r.SortOrder, IsActive: r.IsActive, Subraces: mapSlice(r.Subraces, subraceDTO),
	}
}

func subraceDTO(s Subrace) SubraceDTO {
	return SubraceDTO{
		ID: s.ID, RaceID: s.RaceID, Slug: s.Slug, Name: s.Name, Description: s.Description,
		Traits: traitDTOs(s.Traits), VisualPrompt: s.VisualPrompt, IsActive: s.IsActive,
	}
}

func (h *Handler) classDTO(c Class) ClassDTO {
	return ClassDTO{
		ID: c.ID, Slug: c.Slug, Name: c.Name, HitDie: c.HitDie, PrimaryAbility: c.PrimaryAbility,
		Description: c.Description, Features: traitDTOs(c.Features), VisualPrompt: c.VisualPrompt,
		ImageKey: c.ImageKey, ImageURL: h.urls.Content(c.ImageKey), SortOrder: c.SortOrder, IsActive: c.IsActive,
	}
}

func (h *Handler) backgroundDTO(b Background) BackgroundDTO {
	return BackgroundDTO{
		ID: b.ID, Slug: b.Slug, Name: b.Name, Description: b.Description, SkillProficiencies: b.SkillProficiencies,
		VisualPrompt: b.VisualPrompt, ImageKey: b.ImageKey, ImageURL: h.urls.Content(b.ImageKey),
		SortOrder: b.SortOrder, IsActive: b.IsActive,
	}
}

func traitDTOs(ts []Trait) []TraitDTO {
	return mapSlice(ts, func(t Trait) TraitDTO { return TraitDTO(t) })
}

func traitsFrom(ds []TraitDTO) []Trait {
	return mapSlice(ds, func(d TraitDTO) Trait { return Trait(d) })
}

func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}
