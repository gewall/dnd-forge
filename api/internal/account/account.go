// Package account lets admins look after user accounts (role, block, own
// daily limits) and re-checks every signed-in caller against the stored
// account, so a block or role change applies before the access token
// expires.
package account

import (
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/identity"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

var (
	ErrNotFound    = apperr.New(apperr.NotFound, "user tidak ditemukan")
	ErrBanned      = apperr.New(apperr.Forbidden, "akun diblokir")
	ErrGone        = apperr.New(apperr.Unauthorized, "akun tidak ditemukan; silakan login ulang")
	ErrSelfChange  = apperr.New(apperr.Invalid, "admin tidak bisa memblokir atau menurunkan role akunnya sendiri")
	ErrInvalidRole = apperr.New(apperr.Invalid, "role harus USER atau ADMIN")
)

const entityUser = "user"

// Account is a user as admins see it.
type Account struct {
	ID       uuid.UUID
	Email    string
	Name     string
	Role     identity.Role
	IsBanned bool
	// PortraitQuota and LoreQuota are the user's own daily limits; nil
	// means the default applies.
	PortraitQuota *int
	LoreQuota     *int
	HasPassword   bool
	HasGoogle     bool
	Characters    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Detail adds lifetime totals and today's usage to an account.
type Detail struct {
	Account
	Portraits     int // every portrait request, any status
	Lores         int
	PortraitUsage quota.Usage
	LoreUsage     quota.Usage
}

// Filter narrows the account list. Zero values do not filter.
type Filter struct {
	Query  string // part of the email or name
	Role   identity.Role
	Banned *bool
	Before *time.Time // created_at cursor
	Limit  int
}

// Changes is the full editable state of an account.
type Changes struct {
	Role          identity.Role
	IsBanned      bool
	PortraitQuota *int
	LoreQuota     *int
}

// Status is what every request is checked against.
type Status struct {
	Role   identity.Role
	Banned bool
}
