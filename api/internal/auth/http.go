package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

// PublicKeySource exposes the access token verification key.
type PublicKeySource interface {
	PublicKeyPEM() (string, error)
}

// Handler exposes the auth service over HTTP.
type Handler struct {
	svc  *Service
	keys PublicKeySource
}

// NewHandler returns a Handler. Both arguments may be nil when the handler
// is only used to generate the OpenAPI document.
func NewHandler(svc *Service, keys PublicKeySource) *Handler {
	return &Handler{svc: svc, keys: keys}
}

// Register adds the auth operations to api.
func (h *Handler) Register(api huma.API) {
	tags := []string{"auth"}

	huma.Register(api, httpapi.RateLimited(huma.Operation{
		OperationID:   "register",
		Method:        http.MethodPost,
		Path:          "/v1/auth/register",
		Summary:       "Daftar dengan email dan password",
		Tags:          tags,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusConflict},
	}, "auth"), h.register)

	huma.Register(api, httpapi.RateLimited(huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/v1/auth/login",
		Summary:     "Login dengan email dan password",
		Tags:        tags,
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, "auth"), h.login)

	huma.Register(api, httpapi.RateLimited(huma.Operation{
		OperationID: "loginGoogle",
		Method:      http.MethodPost,
		Path:        "/v1/auth/google",
		Summary:     "Login dengan authorization code Google",
		Description: "Next.js menjalankan redirect OAuth + PKCE, lalu mengirim code dan code_verifier ke sini.",
		Tags:        tags,
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable},
	}, "auth"), h.loginGoogle)

	huma.Register(api, httpapi.RateLimited(huma.Operation{
		OperationID: "refreshSession",
		Method:      http.MethodPost,
		Path:        "/v1/auth/refresh",
		Summary:     "Tukar refresh token dengan sesi baru",
		Tags:        tags,
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, "auth"), h.refresh)

	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/v1/auth/logout",
		Summary:       "Cabut refresh token",
		Tags:          tags,
		DefaultStatus: http.StatusNoContent,
	}, h.logout)

	huma.Register(api, huma.Operation{
		OperationID: "authPublicKey",
		Method:      http.MethodGet,
		Path:        "/v1/auth/public-key",
		Summary:     "Public key Ed25519 untuk memverifikasi access token",
		Tags:        tags,
	}, h.publicKey)

	huma.Register(api, httpapi.Authenticated(huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/v1/me",
		Summary:     "Profil user yang sedang login",
		Tags:        []string{"me"},
	}), h.me)
}

// DTO ------------------------------------------------------------------------

type UserDTO struct {
	ID         uuid.UUID     `json:"id"`
	Email      string        `json:"email" format:"email"`
	Name       string        `json:"name"`
	Role       identity.Role `json:"role" enum:"USER,ADMIN"`
	DailyQuota *int          `json:"dailyQuota" doc:"null = memakai kuota default"`
	CreatedAt  time.Time     `json:"createdAt"`
}

type SessionDTO struct {
	User                  UserDTO   `json:"user"`
	AccessToken           string    `json:"accessToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
}

type sessionOutput struct {
	Body SessionDTO
}

type RegisterRequest struct {
	Email    string `json:"email" format:"email" maxLength:"254"`
	Password string `json:"password" minLength:"8" maxLength:"128"`
	Name     string `json:"name,omitempty" maxLength:"80"`
}

type LoginRequest struct {
	Email    string `json:"email" format:"email" maxLength:"254"`
	Password string `json:"password" minLength:"1" maxLength:"128"`
}

type GoogleLoginRequest struct {
	Code         string `json:"code" minLength:"1" maxLength:"2048"`
	CodeVerifier string `json:"codeVerifier,omitempty" maxLength:"128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" minLength:"1" maxLength:"128"`
}

type PublicKeyDTO struct {
	Algorithm    string `json:"algorithm" example:"EdDSA"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

type registerInput struct{ Body RegisterRequest }
type loginInput struct{ Body LoginRequest }
type googleInput struct{ Body GoogleLoginRequest }
type refreshInput struct{ Body RefreshRequest }
type publicKeyOutput struct{ Body PublicKeyDTO }

type userOutput struct {
	Body UserDTO
}

// Handlers -------------------------------------------------------------------

func (h *Handler) register(ctx context.Context, in *registerInput) (*sessionOutput, error) {
	s, err := h.svc.Register(ctx, in.Body.Email, in.Body.Password, in.Body.Name)
	return sessionResult(ctx, s, err)
}

func (h *Handler) login(ctx context.Context, in *loginInput) (*sessionOutput, error) {
	s, err := h.svc.Login(ctx, in.Body.Email, in.Body.Password)
	return sessionResult(ctx, s, err)
}

func (h *Handler) loginGoogle(ctx context.Context, in *googleInput) (*sessionOutput, error) {
	s, err := h.svc.LoginWithGoogle(ctx, in.Body.Code, in.Body.CodeVerifier)
	return sessionResult(ctx, s, err)
}

func (h *Handler) refresh(ctx context.Context, in *refreshInput) (*sessionOutput, error) {
	s, err := h.svc.Refresh(ctx, in.Body.RefreshToken)
	return sessionResult(ctx, s, err)
}

func (h *Handler) logout(ctx context.Context, in *refreshInput) (*struct{}, error) {
	if err := h.svc.Logout(ctx, in.Body.RefreshToken); err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return nil, nil
}

func (h *Handler) publicKey(ctx context.Context, _ *struct{}) (*publicKeyOutput, error) {
	pem, err := h.keys.PublicKeyPEM()
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &publicKeyOutput{Body: PublicKeyDTO{Algorithm: "EdDSA", PublicKeyPEM: pem}}, nil
}

func (h *Handler) me(ctx context.Context, _ *struct{}) (*userOutput, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, httpapi.Error(ctx, errors.New("auth: principal missing on authenticated route"))
	}
	u, err := h.svc.User(ctx, p.UserID)
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &userOutput{Body: toUserDTO(u)}, nil
}

func sessionResult(ctx context.Context, s Session, err error) (*sessionOutput, error) {
	if err != nil {
		return nil, httpapi.Error(ctx, err)
	}
	return &sessionOutput{Body: SessionDTO{
		User:                  toUserDTO(s.User),
		AccessToken:           s.AccessToken,
		AccessTokenExpiresAt:  s.AccessExpiresAt,
		RefreshToken:          s.RefreshToken,
		RefreshTokenExpiresAt: s.RefreshExpiresAt,
	}}, nil
}

func toUserDTO(u User) UserDTO {
	return UserDTO{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		Role:       u.Role,
		DailyQuota: u.DailyQuota,
		CreatedAt:  u.CreatedAt,
	}
}
