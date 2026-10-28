// Package apperr defines the error kinds shared by every feature package.
//
// Domain packages declare their sentinel errors with New, so the transport
// layer can translate any of them to an HTTP status without importing the
// domain package that produced it.
package apperr

import "errors"

// Kind classifies an error by how a client should react to it.
type Kind int

const (
	Internal Kind = iota
	Invalid
	Unauthorized
	Forbidden
	NotFound
	Conflict
	TooManyRequests
	Unavailable
)

// Error is a domain error with a kind and a message that is safe to show to
// end users.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// New returns an error of the given kind. Use it for package-level sentinels:
//
//	var ErrEmailTaken = apperr.New(apperr.Conflict, "email sudah terdaftar")
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Msg: msg}
}

// KindOf reports the kind of the first *Error in err's chain, or Internal.
func KindOf(err error) Kind {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Kind
	}
	return Internal
}
