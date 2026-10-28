// Package httpapi builds the HTTP router: chi for transport middleware and
// Huma for typed operations and the generated OpenAPI document.
//
// Feature packages register their own operations on Server.API and describe
// their access rules with Authenticated, AdminOnly, OptionalAuth and
// RateLimited; this package enforces them.
package httpapi

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

// HeaderInternalKey carries the secret shared with the Next.js server.
const HeaderInternalKey = "X-Internal-Key"

// TokenVerifier validates access tokens.
type TokenVerifier interface {
	Verify(token string) (identity.Principal, error)
}

// Config configures the router.
type Config struct {
	Title          string
	Version        string
	InternalAPIKey string
	Verifier       TokenVerifier // may be nil when only generating OpenAPI
	Logger         *slog.Logger
	// PublicDocs serves /docs and /openapi.json without the internal key;
	// enable it only in development.
	PublicDocs bool
}

// Server holds the router and the Huma API registered on it.
type Server struct {
	Router chi.Router
	API    huma.API

	internalKey []byte
	verifier    TokenVerifier
	log         *slog.Logger
	publicDocs  bool
	limits      *limiters
}

// New returns a Server with every middleware installed and no operations.
func New(cfg Config) *Server {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		Router:      chi.NewRouter(),
		internalKey: []byte(cfg.InternalAPIKey),
		verifier:    cfg.Verifier,
		log:         log,
		publicDocs:  cfg.PublicDocs,
		limits:      newLimiters(),
	}

	s.Router.Use(
		middleware.RequestID,
		s.identifyClient,
		s.accessLog,
		s.recoverer,
		s.requireInternalKey,
		s.limitByIP,
	)

	hc := huma.DefaultConfig(cfg.Title, cfg.Version)
	hc.CreateHooks = nil // no $schema links in response bodies
	hc.Info.Description = "Backend DnD Guide & Character Forge. Semua request (kecuali /healthz dan /v1/media) " +
		"wajib membawa header " + HeaderInternalKey + "."
	hc.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		schemeInternal: {Type: "apiKey", In: "header", Name: HeaderInternalKey},
		schemeBearer:   {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
	}
	hc.Security = []map[string][]string{{schemeInternal: {}}}

	s.API = humachi.New(s.Router, hc)
	s.API.UseMiddleware(s.authenticate, s.rateLimit)
	return s
}

// isPublicPath reports whether path may be called without the internal key.
func (s *Server) isPublicPath(path string) bool {
	switch {
	case path == "/healthz", strings.HasPrefix(path, "/v1/media/"):
		return true
	case s.publicDocs && (path == "/docs" || strings.HasPrefix(path, "/openapi") || strings.HasPrefix(path, "/schemas/")):
		return true
	}
	return false
}

// identifyClient checks the internal key and records the client IP.
// Requests from Next.js arrive from Vercel, so the real visitor IP is taken
// from X-Client-IP, but only when the internal key proves the header was set
// by our own frontend.
func (s *Server) identifyClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(HeaderInternalKey)
		trusted := len(s.internalKey) > 0 && subtle.ConstantTimeCompare([]byte(key), s.internalKey) == 1
		ctx := withClient(r.Context(), client{ip: clientIP(r, trusted), trusted: trusted})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireInternalKey rejects requests without the internal key, except on
// public paths.
func (s *Server) requireInternalKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !clientFrom(r.Context()).trusted && !s.isPublicPath(r.URL.Path) {
			writeProblem(w, http.StatusUnauthorized, "header "+HeaderInternalKey+" tidak valid")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := nowFunc()
		next.ServeHTTP(ww, r)
		if r.URL.Path == "/healthz" {
			return
		}
		s.log.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", nowFunc().Sub(start).Milliseconds(),
			"ip", ClientIP(r.Context()),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				s.log.ErrorContext(r.Context(), "panic", "value", v, "path", r.URL.Path)
				writeProblem(w, http.StatusInternalServerError, msgInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
