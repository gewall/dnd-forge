// Package health exposes the liveness probe used by Docker and deploys.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Pinger checks a dependency, typically the database pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler serves GET /healthz.
type Handler struct {
	db Pinger
}

// NewHandler returns a Handler that reports unhealthy when db fails a ping.
func NewHandler(db Pinger) *Handler { return &Handler{db: db} }

type HealthStatus struct {
	Status string `json:"status" example:"ok"`
}

type output struct{ Body HealthStatus }

// Register adds the health operation to api.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Health check (tanpa internal key)",
		Tags:        []string{"internal"},
		Errors:      []int{http.StatusServiceUnavailable},
	}, h.check)
}

func (h *Handler) check(ctx context.Context, _ *struct{}) (*output, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		return nil, huma.Error503ServiceUnavailable("database tidak tersedia")
	}
	return &output{Body: HealthStatus{Status: "ok"}}, nil
}
