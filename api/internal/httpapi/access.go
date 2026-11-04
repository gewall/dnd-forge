package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

const (
	schemeInternal = "internalKey"
	schemeBearer   = "bearer"

	metaAccess    = "access"
	metaRateLimit = "rateLimit"
)

type access int

const (
	accessPublic access = iota
	accessOptional
	accessUser
	accessAdmin
)

// Authenticated marks op as requiring a signed-in user.
func Authenticated(op huma.Operation) huma.Operation {
	return withAccess(op, accessUser)
}

// AdminOnly marks op as requiring an ADMIN user.
func AdminOnly(op huma.Operation) huma.Operation {
	return withAccess(op, accessAdmin)
}

// OptionalAuth makes the principal available to op when a valid token is
// sent, without requiring one.
func OptionalAuth(op huma.Operation) huma.Operation {
	return withAccess(op, accessOptional)
}

// RateLimited applies the named limit policy (see policies) to op, keyed by
// user when signed in and by client IP otherwise.
func RateLimited(op huma.Operation, policy string) huma.Operation {
	if _, ok := policies[policy]; !ok {
		panic("httpapi: unknown rate limit policy " + policy)
	}
	setMeta(&op, metaRateLimit, policy)
	op.Errors = appendUnique(op.Errors, http.StatusTooManyRequests)
	return op
}

func withAccess(op huma.Operation, a access) huma.Operation {
	setMeta(&op, metaAccess, a)
	if a != accessOptional {
		op.Security = []map[string][]string{{schemeInternal: {}, schemeBearer: {}}}
		op.Errors = appendUnique(op.Errors, http.StatusUnauthorized, http.StatusForbidden) // 403: blocked or not admin
	}
	return op
}

func setMeta(op *huma.Operation, key string, v any) {
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata[key] = v
}

func appendUnique(codes []int, add ...int) []int {
	for _, code := range add {
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}
	return codes
}

// authenticate resolves the bearer token and enforces the access level the
// operation declared.
func (s *Server) authenticate(ctx huma.Context, next func(huma.Context)) {
	need, _ := ctx.Operation().Metadata[metaAccess].(access)
	if need == accessPublic {
		next(ctx)
		return
	}

	token, hasToken := bearerToken(ctx.Header("Authorization"))
	if !hasToken {
		if need == accessOptional {
			next(ctx)
			return
		}
		s.writeErr(ctx, http.StatusUnauthorized, "butuh login")
		return
	}
	if s.verifier == nil {
		s.writeErr(ctx, http.StatusServiceUnavailable, "autentikasi belum dikonfigurasi")
		return
	}
	p, err := s.verifier.Verify(token)
	if err != nil {
		s.writeErr(ctx, http.StatusUnauthorized, "token tidak valid atau kedaluwarsa")
		return
	}
	if s.accounts != nil {
		current, err := s.accounts.Current(ctx.Context(), p)
		if err != nil {
			if need == accessOptional && apperr.KindOf(err) != apperr.Internal {
				next(ctx) // a blocked or deleted caller browses anonymously
				return
			}
			s.writeError(ctx, err)
			return
		}
		p = current
	}
	if need == accessAdmin && !p.IsAdmin() {
		s.writeErr(ctx, http.StatusForbidden, "khusus admin")
		return
	}
	next(huma.WithContext(ctx, identity.WithPrincipal(ctx.Context(), p)))
}

// Principal returns the caller of an Authenticated or AdminOnly operation.
// The error only occurs when an operation forgot to declare its access.
func Principal(ctx context.Context) (identity.Principal, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return identity.Principal{}, errors.New("httpapi: no principal; operation is missing Authenticated/AdminOnly")
	}
	return p, nil
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}

// writeError writes a domain error with the status Error maps it to.
func (s *Server) writeError(ctx huma.Context, err error) {
	status, msg := http.StatusInternalServerError, msgInternal
	if se, ok := errors.AsType[huma.StatusError](Error(ctx.Context(), err)); ok {
		status, msg = se.GetStatus(), se.Error()
	}
	s.writeErr(ctx, status, msg)
}

func (s *Server) writeErr(ctx huma.Context, status int, msg string) {
	if err := huma.WriteErr(s.API, ctx, status, msg); err != nil {
		s.log.ErrorContext(ctx.Context(), "write error response", "err", err)
	}
}
