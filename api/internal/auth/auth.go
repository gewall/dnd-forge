// Package auth registers and signs in users, issues access tokens (JWT,
// EdDSA) and rotates refresh tokens.
package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

var (
	ErrEmailTaken          = apperr.New(apperr.Conflict, "email sudah terdaftar")
	ErrInvalidCredentials  = apperr.New(apperr.Unauthorized, "email atau password salah")
	ErrInvalidRefreshToken = apperr.New(apperr.Unauthorized, "sesi tidak valid, silakan login ulang")
	ErrInvalidAccessToken  = apperr.New(apperr.Unauthorized, "token tidak valid atau kedaluwarsa")
	ErrBanned              = apperr.New(apperr.Forbidden, "akun diblokir")
	ErrUserNotFound        = apperr.New(apperr.NotFound, "user tidak ditemukan")
	ErrGoogleDisabled      = apperr.New(apperr.Unavailable, "login Google belum dikonfigurasi")
	ErrGoogleUnverified    = apperr.New(apperr.Forbidden, "email Google belum terverifikasi")
	ErrGoogleExchange      = apperr.New(apperr.Unauthorized, "login Google gagal, silakan coba lagi")

	// errRefreshTokenNotFound is returned by TokenStore implementations and
	// never leaves the package.
	errRefreshTokenNotFound = apperr.New(apperr.NotFound, "refresh token tidak ditemukan")
)

// User is an account as seen by the auth service.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	Role         identity.Role
	DailyQuota   *int
	IsBanned     bool
	PasswordHash string // empty for Google-only accounts
	GoogleSub    string
	CreatedAt    time.Time
}

// Principal returns the identity carried in this user's access tokens.
func (u User) Principal() identity.Principal {
	return identity.Principal{UserID: u.ID, Role: u.Role}
}

// NewUser holds the fields needed to create an account.
type NewUser struct {
	Email        string
	Name         string
	Role         identity.Role
	PasswordHash string
	GoogleSub    string
}

// RefreshToken is the stored form of a refresh token; the raw token is only
// ever known to the client.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Hash      string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Session is what a successful sign-in or refresh hands back to the client.
type Session struct {
	User             User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// GoogleProfile is the verified identity returned by Google.
type GoogleProfile struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}
