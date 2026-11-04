//go:build integration

package aiprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/ai/providers"
	"github.com/gewall/dnd-forge/api/internal/crypto"
	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func TestPostgresProviders(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "ai_providers", "audit_logs")
	sealer, err := crypto.NewSealer(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewPostgresStore(pool), sealer, ai.NewRegistry(providers.Fake{}, providers.NewOpenAI("http://openai.invalid")))

	if _, err := svc.Active(ctx, ai.Image); !errors.Is(err, ErrNoImageProvider) {
		t.Errorf("Active(image) on empty table err = %v", err)
	}
	if _, err := svc.Create(ctx, uuid.Nil, Input{Type: ai.OpenAI, Capability: ai.Text, Label: "x", DefaultModel: "m", APIKey: "sk-12345678"}); !errors.Is(err, ai.ErrUnsupported) {
		t.Errorf("openai as text err = %v, want ErrUnsupported", err)
	}
	if _, err := svc.Create(ctx, uuid.Nil, Input{Type: ai.Stability, Capability: ai.Image, Label: "x", DefaultModel: "m", APIKey: "sk-12345678"}); !errors.Is(err, ai.ErrUnknownProvider) {
		t.Errorf("unregistered type err = %v, want ErrUnknownProvider", err)
	}

	first, err := svc.Create(ctx, uuid.Nil, Input{Type: ai.Fake, Capability: ai.Image, Label: " Satu ", DefaultModel: "fake-1", APIKey: "sk-first-abcd"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Capability != ai.Image || first.Label != "Satu" || first.KeyLast4 != "abcd" || first.IsActive {
		t.Errorf("created = %+v", first)
	}
	second, err := svc.Create(ctx, uuid.Nil, Input{Type: ai.Fake, Capability: ai.Image, Label: "Dua", DefaultModel: "fake-2", APIKey: "sk-second-wxyz"})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Activate(ctx, uuid.Nil, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Activate(ctx, uuid.Nil, second.ID); err != nil {
		t.Fatalf("activating a second image provider: %v", err)
	}
	sel, err := svc.Active(ctx, ai.Image)
	if err != nil || sel.ID != second.ID || sel.Model != "fake-2" {
		t.Errorf("Active(image) = %+v, %v; want the second provider", sel, err)
	}
	if _, err := svc.Active(ctx, ai.Text); !errors.Is(err, ErrNoTextProvider) {
		t.Errorf("Active(text) err = %v", err)
	}
	if err := svc.Delete(ctx, uuid.Nil, second.ID); !errors.Is(err, ErrActive) {
		t.Errorf("delete active err = %v, want ErrActive", err)
	}

	// The capability index allows one active provider per capability; insert
	// a text row directly since no text vendor is registered yet.
	if _, err := pool.Exec(ctx, `INSERT INTO ai_providers (type, capability, label, encrypted_api_key, key_last4, default_model, is_active)
		VALUES ('deepseek', 'text', 'Teks', '\x00', 'abcd', 'deepseek-v4-flash', true)`); err != nil {
		t.Errorf("active text provider beside an active image provider: %v", err)
	}
	if err := svc.Activate(ctx, uuid.Nil, first.ID); err != nil {
		t.Fatal(err)
	}
	var textActive bool
	if err := pool.QueryRow(ctx, "SELECT is_active FROM ai_providers WHERE capability = 'text'").Scan(&textActive); err != nil || !textActive {
		t.Errorf("activating an image provider deactivated the text one: active=%v, %v", textActive, err)
	}
	// This build registers no deepseek vendor, so its active row counts as none.
	if _, err := svc.Active(ctx, ai.Text); !errors.Is(err, ErrNoTextProvider) {
		t.Errorf("Active(text) with an unregistered vendor err = %v, want ErrNoTextProvider", err)
	}

	impl, key, err := svc.ResolveImage(ctx, first.ID)
	if err != nil || impl.Type() != ai.Fake || key != "sk-first-abcd" {
		t.Errorf("ResolveImage = %v, %q, %v", impl, key, err)
	}
	if err := svc.Test(ctx, first.ID); err != nil {
		t.Errorf("Test: %v", err)
	}
	if ids, err := svc.Models(ctx, first.ID); err != nil || len(ids) == 0 {
		t.Errorf("Models = %v, %v", ids, err)
	}
	if _, err := svc.Models(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Models of unknown provider err = %v", err)
	}

	updated, err := svc.Update(ctx, uuid.Nil, second.ID, Input{Label: "Dua v2", DefaultModel: "fake-3"})
	if err != nil || updated.KeyLast4 != "wxyz" || updated.Capability != ai.Image {
		t.Errorf("Update without key = %+v, %v", updated, err)
	}
	if err := svc.Delete(ctx, uuid.Nil, second.ID); err != nil {
		t.Errorf("delete inactive: %v", err)
	}
}
