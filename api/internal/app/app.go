// Package app is the composition root shared by the API server and the
// tools command: it wires concrete adapters into services and registers
// every feature's operations on one router.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/gewall/dnd-forge/api/internal/account"
	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/ai/providers"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/auth"
	"github.com/gewall/dnd-forge/api/internal/character"
	"github.com/gewall/dnd-forge/api/internal/config"
	"github.com/gewall/dnd-forge/api/internal/crypto"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
	"github.com/gewall/dnd-forge/api/internal/gamedata"
	"github.com/gewall/dnd-forge/api/internal/guide"
	"github.com/gewall/dnd-forge/api/internal/health"
	"github.com/gewall/dnd-forge/api/internal/housekeeping"
	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/imaging"
	"github.com/gewall/dnd-forge/api/internal/lore"
	"github.com/gewall/dnd-forge/api/internal/media"
	"github.com/gewall/dnd-forge/api/internal/moderation"
	"github.com/gewall/dnd-forge/api/internal/portrait"
	"github.com/gewall/dnd-forge/api/internal/prompt"
	"github.com/gewall/dnd-forge/api/internal/quota"
	"github.com/gewall/dnd-forge/api/internal/stats"
	"github.com/gewall/dnd-forge/api/internal/storage"
)

const (
	Title   = "DnD Forge API"
	Version = "0.6.1"

	webpQuality = 82

	// fakeDelay makes the fake provider slow enough to see PENDING when
	// polling.
	fakeDelay = 2 * time.Second
)

// Registrar is implemented by every feature handler.
type Registrar interface {
	Register(api huma.API)
}

// App holds the long-lived dependencies of a running API.
type App struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	Auth     *auth.Service
	GameData *gamedata.Service
	Guide    *guide.Service
	// Housekeeping finds storage objects no row points to.
	Housekeeping *housekeeping.Service
	Server       *httpapi.Server

	// jobs runs background work; it only starts with StartWorkers, so the
	// tools command never executes jobs.
	jobs *river.Client[pgx.Tx]
}

// New connects to the database, runs migrations when configured and builds
// the HTTP server and the background job client.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	key, err := cfg.SigningKey()
	if err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}
	sealer, err := sealerFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}
	store, err := storage.New(storage.Config{
		Endpoint:        cfg.S3Endpoint,
		PublicEndpoint:  cfg.S3PublicEndpoint,
		Region:          cfg.S3Region,
		Bucket:          cfg.S3Bucket,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
		PresignTTL:      cfg.PresignTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}

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

	urls := media.NewURLs(cfg.PublicBaseURL)
	mediaSvc := media.NewService(imaging.New(webpQuality), store, urls)
	gameSvc := gamedata.NewService(gamedata.NewPostgresStore(pool))
	guideSvc := guide.NewService(guide.NewPostgresStore(pool))
	charSvc := character.NewService(character.NewPostgresStore(pool), character.NewGameDataCatalog(gameSvc))

	aiSvc := aiprovider.NewService(aiprovider.NewPostgresStore(pool), sealer, registryFor(cfg))
	promptSvc := prompt.NewService(prompt.NewPostgresStore(pool))
	seeded, err := promptSvc.SeedDefaults(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: %w", err)
	}
	if len(seeded) > 0 {
		log.Info("seeded default prompt templates", "kinds", seeded)
	}
	modSvc := moderation.NewService(moderation.NewPostgresStore(pool))

	// Requests enqueue through an insert-only client, which exists before
	// the workers that need the portrait and lore services.
	driver := riverpgxv5.New(pool)
	inserter, err := river.NewClient(driver, &river.Config{Logger: log})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: river: %w", err)
	}
	portraitSvc := portrait.NewService(portrait.Deps{
		Store:     portrait.NewPostgresStore(pool, inserter),
		Providers: aiSvc,
		Templates: promptSvc,
		Moderator: modSvc,
		Encoder:   imaging.New(webpQuality),
		Objects:   store,
		Logger:    log,
	})
	loreSvc := lore.NewService(lore.Deps{
		Store:     lore.NewPostgresStore(pool, inserter),
		Providers: aiSvc,
		Templates: promptSvc,
		Moderator: modSvc,
		Logger:    log,
	})
	housekeepingSvc := housekeeping.NewService(store, sqlc.New(pool))
	workers := river.NewWorkers()
	river.AddWorker(workers, portrait.NewWorker(portraitSvc))
	river.AddWorker(workers, lore.NewWorker(loreSvc))
	river.AddWorker(workers, housekeeping.NewPurgeWorker(housekeepingSvc, log))
	jobs, err := river.NewClient(driver, &river.Config{
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: cfg.PortraitWorkers}},
		Workers:      workers,
		PeriodicJobs: []*river.PeriodicJob{housekeeping.PeriodicPurge()},
		Logger:       log,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: river: %w", err)
	}

	accountSvc := account.NewService(account.Deps{
		Store:    account.NewPostgresStore(pool),
		Portrait: portraitSvc,
		Lore:     loreSvc,
	})

	srv := httpapi.New(httpapi.Config{
		Title:          Title,
		Version:        Version,
		InternalAPIKey: cfg.InternalAPIKey,
		Verifier:       tokens,
		Accounts:       accountSvc,
		Logger:         log,
		PublicDocs:     cfg.IsDev(),
	})
	register(srv.API,
		health.NewHandler(pool),
		auth.NewHandler(authSvc, tokens),
		gamedata.NewHandler(gameSvc, urls),
		guide.NewHandler(guideSvc, urls),
		media.NewHandler(mediaSvc),
		character.NewHandler(charSvc),
		aiprovider.NewHandler(aiSvc),
		prompt.NewHandler(promptSvc),
		moderation.NewHandler(modSvc),
		portrait.NewHandler(portraitSvc, urls),
		portrait.NewAdminHandler(portrait.NewAdmin(portrait.NewPostgresAdminStore(pool), store)),
		lore.NewHandler(loreSvc),
		quota.NewHandler(portraitSvc, loreSvc),
		quota.NewSettingsHandler(quota.NewSettings(quota.NewPostgresSettings(pool))),
		account.NewHandler(accountSvc),
		stats.NewHandler(stats.NewService(stats.NewPostgresStore(pool))),
		audit.NewHandler(audit.NewReader(pool)),
	)

	return &App{
		Config:       cfg,
		Pool:         pool,
		Auth:         authSvc,
		GameData:     gameSvc,
		Guide:        guideSvc,
		Housekeeping: housekeepingSvc,
		Server:       srv,
		jobs:         jobs,
	}, nil
}

// StartWorkers starts executing background jobs. Cancelling ctx does not
// stop them; call StopWorkers for a graceful stop.
func (a *App) StartWorkers(ctx context.Context) error {
	if err := a.jobs.Start(context.WithoutCancel(ctx)); err != nil {
		return fmt.Errorf("app: start workers: %w", err)
	}
	return nil
}

// StopWorkers waits for running jobs until ctx ends, then cancels them;
// River retries cancelled jobs on the next start.
func (a *App) StopWorkers(ctx context.Context) error {
	if err := a.jobs.Stop(ctx); err == nil {
		return nil
	}
	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.jobs.StopAndCancel(cancelCtx); err != nil {
		return fmt.Errorf("app: stop workers: %w", err)
	}
	return nil
}

// Close releases the database pool.
func (a *App) Close() { a.Pool.Close() }

func sealerFor(cfg config.Config) (*crypto.Sealer, error) {
	key, err := cfg.EncryptionKeyBytes()
	if err != nil {
		return nil, err
	}
	return crypto.NewSealer(key)
}

// registryFor lists the AI vendors this deployment can call.
func registryFor(cfg config.Config) *ai.Registry {
	vendors := []ai.Vendor{providers.NewOpenAI(cfg.OpenAIBaseURL), providers.NewDeepSeek(cfg.DeepSeekBaseURL)}
	if cfg.IsDev() || cfg.AIFakeProvider {
		vendors = append(vendors, providers.Fake{Delay: fakeDelay})
	}
	return ai.NewRegistry(vendors...)
}

// OpenAPI builds the router with inert handlers and returns the OpenAPI 3.1
// document, without touching the database or any secret.
func OpenAPI() ([]byte, error) {
	srv := httpapi.New(httpapi.Config{Title: Title, Version: Version})
	register(srv.API,
		health.NewHandler(nil),
		auth.NewHandler(nil, nil),
		gamedata.NewHandler(nil, nil),
		guide.NewHandler(nil, nil),
		media.NewHandler(nil),
		character.NewHandler(nil),
		aiprovider.NewHandler(nil),
		prompt.NewHandler(nil),
		moderation.NewHandler(nil),
		portrait.NewHandler(nil, nil),
		portrait.NewAdminHandler(nil),
		lore.NewHandler(nil),
		quota.NewHandler(nil, nil),
		quota.NewSettingsHandler(nil),
		account.NewHandler(nil),
		stats.NewHandler(nil),
		audit.NewHandler(nil),
	)
	return srv.API.OpenAPI().MarshalJSON()
}

func register(api huma.API, rs ...Registrar) {
	for _, r := range rs {
		r.Register(api)
	}
}
