// Package identity carries the authenticated caller through a request context.
package identity

import (
	"context"

	"github.com/google/uuid"
)

// Role is a user's authorization level.
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Principal is the caller proven by a valid access token.
type Principal struct {
	UserID uuid.UUID
	Role   Role
}

// IsAdmin reports whether the principal may use admin endpoints.
func (p Principal) IsAdmin() bool { return p.Role == RoleAdmin }

type ctxKey struct{}

// WithPrincipal returns a copy of ctx that carries p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the principal stored in ctx, if any.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
