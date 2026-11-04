package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

const testKey = "0123456789abcdef0123456789abcdef"

var (
	userID    = uuid.New()
	errDomain = apperr.New(apperr.Conflict, "sudah ada")
)

// fakeVerifier accepts "user" and "admin" as tokens.
type fakeVerifier struct{}

func (fakeVerifier) Verify(token string) (identity.Principal, error) {
	switch token {
	case "user":
		return identity.Principal{UserID: userID, Role: identity.RoleUser}, nil
	case "admin":
		return identity.Principal{UserID: userID, Role: identity.RoleAdmin}, nil
	}
	return identity.Principal{}, errors.New("bad token")
}

type whoami struct {
	Body struct {
		UserID string `json:"userId"`
		IP     string `json:"ip"`
	}
}

func newTestServer() *Server { return newGuardedServer(nil) }

func newGuardedServer(g AccountGuard) *Server {
	s := New(Config{Title: "test", Version: "0", InternalAPIKey: testKey, Verifier: fakeVerifier{}, Accounts: g})
	h := func(ctx context.Context, _ *struct{}) (*whoami, error) {
		out := &whoami{}
		if p, ok := identity.FromContext(ctx); ok {
			out.Body.UserID = p.UserID.String()
		}
		out.Body.IP = ClientIP(ctx)
		return out, nil
	}
	huma.Register(s.API, huma.Operation{OperationID: "public", Method: http.MethodGet, Path: "/public"}, h)
	huma.Register(s.API, huma.Operation{OperationID: "health", Method: http.MethodGet, Path: "/healthz"}, h)
	huma.Register(s.API, Authenticated(huma.Operation{OperationID: "user", Method: http.MethodGet, Path: "/user"}), h)
	huma.Register(s.API, AdminOnly(huma.Operation{OperationID: "admin", Method: http.MethodGet, Path: "/admin"}), h)
	huma.Register(s.API, OptionalAuth(huma.Operation{OperationID: "optional", Method: http.MethodGet, Path: "/optional"}), h)
	huma.Register(s.API, RateLimited(huma.Operation{OperationID: "limited", Method: http.MethodPost, Path: "/limited"}, "auth"), h)
	huma.Register(s.API, huma.Operation{OperationID: "fails", Method: http.MethodGet, Path: "/fails"},
		func(ctx context.Context, _ *struct{}) (*whoami, error) { return nil, Error(ctx, errDomain) })
	huma.Register(s.API, huma.Operation{OperationID: "crashes", Method: http.MethodGet, Path: "/crashes"},
		func(ctx context.Context, _ *struct{}) (*whoami, error) { return nil, Error(ctx, errors.New("db down")) })
	return s
}

func do(s *Server, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	return rec
}

func TestAccessRules(t *testing.T) {
	s := newTestServer()
	key := map[string]string{HeaderInternalKey: testKey}
	with := func(token string) map[string]string {
		return map[string]string{HeaderInternalKey: testKey, "Authorization": "Bearer " + token}
	}

	tests := []struct {
		name    string
		path    string
		headers map[string]string
		want    int
	}{
		{"no internal key", "/public", nil, http.StatusUnauthorized},
		{"wrong internal key", "/public", map[string]string{HeaderInternalKey: "x"}, http.StatusUnauthorized},
		{"healthz needs no key", "/healthz", nil, http.StatusOK},
		{"public with key", "/public", key, http.StatusOK},
		{"user without token", "/user", key, http.StatusUnauthorized},
		{"user with bad token", "/user", with("bogus"), http.StatusUnauthorized},
		{"user with token", "/user", with("user"), http.StatusOK},
		{"admin as user", "/admin", with("user"), http.StatusForbidden},
		{"admin as admin", "/admin", with("admin"), http.StatusOK},
		{"optional without token", "/optional", key, http.StatusOK},
		{"optional with bad token", "/optional", with("bogus"), http.StatusUnauthorized},
		{"domain error mapped", "/fails", key, http.StatusConflict},
		{"internal error hidden", "/crashes", key, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(s, http.MethodGet, tt.path, tt.headers)
			if rec.Code != tt.want {
				t.Errorf("GET %s = %d, want %d; body %s", tt.path, rec.Code, tt.want, rec.Body)
			}
		})
	}
}

// guard answers with a fixed result, as the account service would.
type guard struct {
	role identity.Role
	err  error
}

func (g guard) Current(_ context.Context, p identity.Principal) (identity.Principal, error) {
	if g.err != nil {
		return identity.Principal{}, g.err
	}
	p.Role = g.role
	return p, nil
}

func TestAccountGuard(t *testing.T) {
	with := func(token string) map[string]string {
		return map[string]string{HeaderInternalKey: testKey, "Authorization": "Bearer " + token}
	}
	banned := apperr.New(apperr.Forbidden, "akun diblokir")
	gone := apperr.New(apperr.Unauthorized, "akun tidak ditemukan")
	tests := []struct {
		name  string
		g     guard
		path  string
		token string
		want  int
		body  string
	}{
		{"blocked user", guard{err: banned}, "/user", "user", http.StatusForbidden, "akun diblokir"},
		{"deleted user", guard{err: gone}, "/user", "user", http.StatusUnauthorized, "akun tidak ditemukan"},
		{"database down", guard{err: errors.New("db down")}, "/user", "user", http.StatusInternalServerError, msgInternal},
		{"promoted since token was issued", guard{role: identity.RoleAdmin}, "/admin", "user", http.StatusOK, ""},
		{"demoted since token was issued", guard{role: identity.RoleUser}, "/admin", "admin", http.StatusForbidden, "khusus admin"},
		{"blocked user browses anonymously", guard{err: banned}, "/optional", "user", http.StatusOK, `"userId":""`},
		{"public skips the check", guard{err: banned}, "/public", "user", http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newGuardedServer(tt.g), http.MethodGet, tt.path, with(tt.token))
			if rec.Code != tt.want || !strings.Contains(rec.Body.String(), tt.body) {
				t.Errorf("GET %s = %d %s; want %d containing %q", tt.path, rec.Code, rec.Body, tt.want, tt.body)
			}
		})
	}
}

func TestSecureHeaders(t *testing.T) {
	s := newTestServer()
	huma.Register(s.API, huma.Operation{OperationID: "cached", Method: http.MethodGet, Path: "/cached"},
		func(context.Context, *struct{}) (*Redirect, error) {
			return NewRedirect("https://s3.example/x", time.Hour), nil
		})
	key := map[string]string{HeaderInternalKey: testKey}

	rec := do(s, http.MethodGet, "/public", key)
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("default headers = %v", rec.Header())
	}
	if rec := do(s, http.MethodGet, "/nope", nil); rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("rejected request headers = %v", rec.Header())
	}
	if rec := do(s, http.MethodGet, "/cached", key); !strings.HasPrefix(rec.Header().Get("Cache-Control"), "public, max-age=") {
		t.Errorf("redirect Cache-Control = %q, want the operation's own", rec.Header().Get("Cache-Control"))
	}
}

func TestInternalErrorDoesNotLeak(t *testing.T) {
	rec := do(newTestServer(), http.MethodGet, "/crashes", map[string]string{HeaderInternalKey: testKey})
	if body := rec.Body.String(); strings.Contains(body, "db down") {
		t.Errorf("internal error leaked: %s", body)
	}
}

func TestClientIPTrustsHeaderOnlyWithKey(t *testing.T) {
	s := newTestServer()
	trusted := do(s, http.MethodGet, "/public", map[string]string{
		HeaderInternalKey: testKey, HeaderClientIP: "203.0.113.7", "X-Real-IP": "76.76.21.21",
	})
	if !strings.Contains(trusted.Body.String(), `"ip":"203.0.113.7"`) {
		t.Errorf("trusted request ip: %s", trusted.Body)
	}

	// /healthz is public: a spoofed X-Client-IP without the key is ignored.
	spoofed := do(s, http.MethodGet, "/healthz", map[string]string{
		HeaderClientIP: "203.0.113.7", "X-Real-IP": "76.76.21.21",
	})
	if !strings.Contains(spoofed.Body.String(), `"ip":"76.76.21.21"`) {
		t.Errorf("untrusted request ip: %s", spoofed.Body)
	}
}

func TestRateLimitPolicy(t *testing.T) {
	s := newTestServer()
	headers := map[string]string{HeaderInternalKey: testKey, HeaderClientIP: "198.51.100.1"}
	burst := policies["auth"].Burst
	for i := range burst {
		if rec := do(s, http.MethodPost, "/limited", headers); rec.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, rec.Code)
		}
	}
	rec := do(s, http.MethodPost, "/limited", headers)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request over burst = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}

	// Another visitor has its own bucket.
	other := map[string]string{HeaderInternalKey: testKey, HeaderClientIP: "198.51.100.2"}
	if rec := do(s, http.MethodPost, "/limited", other); rec.Code != http.StatusOK {
		t.Errorf("other client = %d, want 200", rec.Code)
	}
}
