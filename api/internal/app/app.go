// Package app is the composition root shared by the API server and the
// tools command: it wires concrete adapters into services and registers
// every feature's operations on one router.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/config"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/health"
	"github.com/gewall/dnd-forge/api/internal/httpapi"
)

const (
	Title   = "DnD Forge API"
	Version = "0.1.0"
)

// Registrar is implemented by every feature handler.
type Registrar interface {
	Register(api huma.API)
}

// App holds the long-lived dependencies of a running API.
type App struct {
	Config config.Config
	Pool   *pgxpool.Pool
	Auth   *auth.Service
	Server *httpapi.Server
}

// New connects to the database, runs migrations when configured and builds
// the HTTP server.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.MigrateOnStart {
		if err := db.Migrate(ctx, pool, log); err != nil {
			pool.Close()
			return nil, err
		}
	}

	key, err := cfg.SigningKey()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: %w", err)
	}
	tokens := auth.NewTokens(key, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)

	var google auth.GoogleIdentity
	if cfg.GoogleEnabled() {
		google = auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	}
	authSvc := auth.NewService(auth.NewPostgresStore(pool), tokens, auth.Config{
		RefreshTTL:  cfg.RefreshTTL,
		AdminEmails: cfg.AdminEmails,
		Google:      google,
		Logger:      log,
	})

	srv := httpapi.New(httpapi.Config{
		Title:          Title,
		Version:        Version,
		InternalAPIKey: cfg.InternalAPIKey,
		Verifier:       tokens,
		Logger:         log,
		PublicDocs:     cfg.IsDev(),
	})
	register(srv.API,
		health.NewHandler(pool),
		auth.NewHandler(authSvc, tokens),
	)

	return &App{Config: cfg, Pool: pool, Auth: authSvc, Server: srv}, nil
}

// Close releases the database pool.
func (a *App) Close() { a.Pool.Close() }

// OpenAPI builds the router with inert handlers and returns the OpenAPI 3.1
// document, without touching the database or any secret.
func OpenAPI() ([]byte, error) {
	srv := httpapi.New(httpapi.Config{Title: Title, Version: Version})
	register(srv.API,
		health.NewHandler(nil),
		auth.NewHandler(nil, nil),
	)
	return srv.API.OpenAPI().MarshalJSON()
}

func register(api huma.API, rs ...Registrar) {
	for _, r := range rs {
		r.Register(api)
	}
}
