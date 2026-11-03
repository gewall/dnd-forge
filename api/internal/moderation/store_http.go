package moderation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	q *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{q: sqlc.New(pool)} }

func (s *PostgresStore) BannedWords(ctx context.Context) ([]string, error) {
	words, err := s.q.ListBannedWords(ctx)
	if err != nil {
		return nil, fmt.Errorf("banned words: %w", err)
	}
	if words == nil {
		words = []string{}
	}
	return words, nil
}

func (s *PostgresStore) AddBannedWord(ctx context.Context, word string) error {
	if err := s.q.AddBannedWord(ctx, word); err != nil {
		return fmt.Errorf("add banned word: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteBannedWord(ctx context.Context, word string) error {
	n, err := s.q.DeleteBannedWord(ctx, word)
	if err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}
	if n == 0 {
		return ErrWordNotFound
	}
	return nil
}

// Handler exposes the banned word list to admins.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler; svc may be nil when only generating OpenAPI.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds the banned word operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"admin"}
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminListBannedWords", Method: http.MethodGet, Path: "/v1/admin/banned-words",
		Summary: "Daftar kata terlarang untuk prompt user", Tags: tags,
	}), h.list)
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminAddBannedWord", Method: http.MethodPost, Path: "/v1/admin/banned-words",
		Summary: "Tambah kata atau frasa terlarang", Tags: tags, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest},
	}), h.add)
	huma.Register(api, httpapi.AdminOnly(huma.Operation{
		OperationID: "adminDeleteBannedWord", Method: http.MethodDelete, Path: "/v1/admin/banned-words/{word}",
		Summary: "Hapus kata terlarang", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}), h.remove)
}

type WordList struct {
	Items []string `json:"items"`
}

type WordRequest struct {
	Word string `json:"word" minLength:"2" maxLength:"60"`
}

type listOutput struct{ Body WordList }
type addInput struct{ Body WordRequest }
type addOutput struct{ Body WordRequest }
type removeInput struct {
	Word string `path:"word" maxLength:"60"`
}

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	ws, err := h.svc.Words(ctx)
	return httpapi.Respond(ctx, ws, err, func(ws []string) *listOutput { return &listOutput{Body: WordList{Items: ws}} })
}

func (h *Handler) add(ctx context.Context, in *addInput) (*addOutput, error) {
	w, err := h.svc.Add(ctx, in.Body.Word)
	return httpapi.Respond(ctx, w, err, func(w string) *addOutput { return &addOutput{Body: WordRequest{Word: w}} })
}

func (h *Handler) remove(ctx context.Context, in *removeInput) (*struct{}, error) {
	return nil, httpapi.NoContent(ctx, h.svc.Remove(ctx, in.Word))
}
