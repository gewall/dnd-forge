//go:build integration

package stats

import (
	"context"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func TestPostgresStats(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "ai_providers")
	// Two users today, one banned admin; an image provider with one success
	// and one failure; a text provider with one lore and its tokens. One
	// portrait from long ago falls outside the period.
	_, err := pool.Exec(ctx, `
		INSERT INTO users (email, role, is_banned) VALUES ('a@example.com', 'ADMIN', true), ('b@example.com', 'USER', false);
		INSERT INTO ai_providers (type, capability, label, encrypted_api_key, key_last4, default_model)
		VALUES ('fake', 'image', 'Img', '\x00', 'abcd', 'img-1'), ('deepseek', 'text', 'Txt', '\x00', 'abcd', 'txt-1');
		INSERT INTO generated_images (user_id, provider_id, model, final_prompt, status, is_hidden, created_at)
		SELECT u.id, p.id, 'img-1', 'x', s.status::image_status, s.hidden, s.at
		FROM users u, ai_providers p,
		     (VALUES ('SUCCEEDED', true, now()), ('FAILED', false, now()), ('SUCCEEDED', false, now() - interval '200 days')) AS s(status, hidden, at)
		WHERE u.email = 'b@example.com' AND p.label = 'Img';
		INSERT INTO character_lores (user_id, provider_id, model, final_prompt, tone, length, language, status, prompt_tokens, completion_tokens)
		SELECT u.id, p.id, 'txt-1', 'x', 'epic', 'short', 'id', 'SUCCEEDED', 300, 500
		FROM users u, ai_providers p WHERE u.email = 'b@example.com' AND p.label = 'Txt';`)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(NewPostgresStore(pool))
	r, err := svc.Report(ctx, 30)
	if err != nil {
		t.Fatal(err)
	}
	tot := r.Totals
	if tot.Users != 2 || tot.BannedUsers != 1 || tot.Admins != 1 || tot.Portraits != 3 || tot.PortraitsSucceeded != 2 ||
		tot.PortraitsFailed != 1 || tot.PortraitsHidden != 1 || tot.Lores != 1 || tot.LoresSucceeded != 1 {
		t.Errorf("totals = %+v", tot)
	}
	if len(r.Days) != 30 || !r.Days[29].Date.Equal(r.To) || !r.Days[0].Date.Equal(r.From) {
		t.Fatalf("days = %d, last %v, want 30 ending %v", len(r.Days), r.Days[len(r.Days)-1].Date, r.To)
	}
	today := r.Days[29]
	if today.NewUsers != 2 || today.PortraitsSucceeded != 1 || today.PortraitsFailed != 1 || today.LoresSucceeded != 1 {
		t.Errorf("today = %+v", today)
	}
	var older int
	for _, d := range r.Days[:29] {
		older += d.NewUsers + d.PortraitsSucceeded + d.PortraitsFailed
	}
	if older != 0 {
		t.Errorf("activity before today = %d", older)
	}

	if len(r.Models) != 2 {
		t.Fatalf("models = %+v", r.Models)
	}
	byCap := map[string]ModelUsage{}
	for _, m := range r.Models {
		byCap[m.Capability] = m
	}
	if img := byCap["image"]; img.Model != "img-1" || img.Requests != 2 || img.Succeeded != 1 || img.Failed != 1 || img.ProviderLabel != "Img" {
		t.Errorf("image usage = %+v (the 200-day-old portrait must be excluded)", img)
	}
	if txt := byCap["text"]; txt.Requests != 1 || txt.PromptTokens != 300 || txt.CompletionTokens != 500 {
		t.Errorf("text usage = %+v", txt)
	}
}
