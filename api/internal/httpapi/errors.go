package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

const msgInternal = "terjadi kesalahan pada server"

// Error translates a service error into the HTTP error a handler returns.
// Domain errors (apperr) keep their message; anything else is logged and
// hidden behind a generic 500.
func Error(ctx context.Context, err error) error {
	e, ok := errors.AsType[*apperr.Error](err)
	if !ok {
		slog.ErrorContext(ctx, "unhandled error", "err", err)
		return huma.Error500InternalServerError(msgInternal)
	}
	switch e.Kind {
	case apperr.Invalid:
		return huma.Error400BadRequest(e.Msg)
	case apperr.Unauthorized:
		return huma.Error401Unauthorized(e.Msg)
	case apperr.Forbidden:
		return huma.Error403Forbidden(e.Msg)
	case apperr.NotFound:
		return huma.Error404NotFound(e.Msg)
	case apperr.Conflict:
		return huma.Error409Conflict(e.Msg)
	case apperr.TooManyRequests:
		return huma.Error429TooManyRequests(e.Msg)
	case apperr.Unavailable:
		return huma.Error503ServiceUnavailable(e.Msg)
	default:
		slog.ErrorContext(ctx, "internal error", "err", err)
		return huma.Error500InternalServerError(msgInternal)
	}
}

// writeProblem writes an RFC 9457 problem from plain net/http middleware,
// matching the shape Huma uses for operation errors.
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(huma.ErrorModel{
		Status: status,
		Title:  http.StatusText(status),
		Detail: detail,
	})
}
