// Package config loads the API configuration from environment variables.
package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is the full runtime configuration. Secrets are only ever read from
// the environment, never from files committed to the repository.
type Config struct {
	Env      string `env:"APP_ENV" envDefault:"production"`
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// PublicBaseURL is the externally visible origin of this API, used to
	// build stable media URLs.
	PublicBaseURL string `env:"PUBLIC_BASE_URL" envDefault:"http://localhost:8080"`

	DatabaseURL    string `env:"DATABASE_URL,required,notEmpty"`
	MigrateOnStart bool   `env:"MIGRATE_ON_START" envDefault:"true"`

	// InternalAPIKey is shared with the Next.js server; every request except
	// health checks and media redirects must present it.
	InternalAPIKey string `env:"INTERNAL_API_KEY,required,notEmpty"`

	JWTPrivateKey string        `env:"JWT_PRIVATE_KEY,required,notEmpty"` // base64 Ed25519 seed (32 byte)
	JWTIssuer     string        `env:"JWT_ISSUER" envDefault:"dnd-forge-api"`
	JWTAudience   string        `env:"JWT_AUDIENCE" envDefault:"dnd-forge-web"`
	AccessTTL     time.Duration `env:"ACCESS_TTL" envDefault:"15m"`
	RefreshTTL    time.Duration `env:"REFRESH_TTL" envDefault:"720h"`

	// AdminEmails are promoted to ADMIN when they register or sign in.
	AdminEmails []string `env:"ADMIN_EMAILS" envSeparator:","`

	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL"` // callback di Next.js, sama persis dengan yang terdaftar di Google

	// Object storage (Garage). S3Endpoint is used for uploads; presigned
	// download URLs are signed for S3PublicEndpoint.
	S3Endpoint        string        `env:"S3_ENDPOINT,required,notEmpty"`
	S3PublicEndpoint  string        `env:"S3_PUBLIC_ENDPOINT,required,notEmpty"`
	S3Region          string        `env:"S3_REGION" envDefault:"garage"`
	S3Bucket          string        `env:"S3_BUCKET,required,notEmpty"`
	S3AccessKeyID     string        `env:"S3_ACCESS_KEY_ID,required,notEmpty"`
	S3SecretAccessKey string        `env:"S3_SECRET_ACCESS_KEY,required,notEmpty"`
	PresignTTL        time.Duration `env:"PRESIGN_TTL" envDefault:"6h"`
}

// Load reads and validates the configuration from the environment.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	for i, e := range cfg.AdminEmails {
		cfg.AdminEmails[i] = strings.ToLower(strings.TrimSpace(e))
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	var errs []error
	if len(c.InternalAPIKey) < 32 {
		errs = append(errs, errors.New("INTERNAL_API_KEY must be at least 32 characters"))
	}
	if _, err := c.SigningKey(); err != nil {
		errs = append(errs, err)
	}
	if c.GoogleClientID != "" && (c.GoogleClientSecret == "" || c.GoogleRedirectURL == "") {
		errs = append(errs, errors.New("GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL are required when GOOGLE_CLIENT_ID is set"))
	}
	return errors.Join(errs...)
}

// IsDev reports whether the API runs on a developer machine.
func (c Config) IsDev() bool { return c.Env == "development" }

// GoogleEnabled reports whether Google sign-in is configured.
func (c Config) GoogleEnabled() bool { return c.GoogleClientID != "" }

// SigningKey decodes JWT_PRIVATE_KEY into an Ed25519 private key.
func (c Config) SigningKey() (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(c.JWTPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY is not valid base64: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY must decode to %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
