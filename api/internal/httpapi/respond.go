package httpapi

import (
	"context"

	"github.com/google/uuid"
)

// AsUser runs fn with the id of the signed-in caller. Use it in handlers of
// Authenticated or AdminOnly operations.
func AsUser[O any](ctx context.Context, fn func(userID uuid.UUID) (*O, error)) (*O, error) {
	p, err := Principal(ctx)
	if err != nil {
		return nil, Error(ctx, err)
	}
	return fn(p.UserID)
}

// Respond maps a service result to an operation output, translating err.
func Respond[T, O any](ctx context.Context, v T, err error, out func(T) *O) (*O, error) {
	if err != nil {
		return nil, Error(ctx, err)
	}
	return out(v), nil
}

// NoContent translates err for operations without a response body.
func NoContent(ctx context.Context, err error) error {
	if err != nil {
		return Error(ctx, err)
	}
	return nil
}
