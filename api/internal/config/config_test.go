package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("INTERNAL_API_KEY", strings.Repeat("k", 32))
	t.Setenv("JWT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(make([]byte, ed25519.SeedSize)))
	t.Setenv("ADMIN_EMAILS", " DM@Example.com ,owner@example.com")
}

func TestLoad(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" || !cfg.MigrateOnStart || cfg.GoogleEnabled() {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if got := strings.Join(cfg.AdminEmails, ","); got != "dm@example.com,owner@example.com" {
		t.Errorf("AdminEmails = %q", got)
	}
}

func TestLoadRejects(t *testing.T) {
	tests := map[string]map[string]string{
		"short internal key":   {"INTERNAL_API_KEY": "short"},
		"jwt key not base64":   {"JWT_PRIVATE_KEY": "%%%"},
		"jwt key wrong length": {"JWT_PRIVATE_KEY": base64.StdEncoding.EncodeToString([]byte("abc"))},
		"google half set":      {"GOOGLE_CLIENT_ID": "id"},
		"missing database url": {"DATABASE_URL": ""},
	}
	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			setValidEnv(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("Load succeeded, want error")
			}
		})
	}
}
