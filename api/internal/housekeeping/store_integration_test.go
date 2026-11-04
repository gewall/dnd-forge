//go:build integration

package housekeeping

import (
	"context"
	"slices"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

func TestReferencedObjectKeys(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "ai_providers", "guide_categories")
	_, err := pool.Exec(ctx, `
		WITH u AS (INSERT INTO users (email) VALUES ('keeper@example.com') RETURNING id),
		     p AS (INSERT INTO ai_providers (type, capability, label, encrypted_api_key, key_last4, default_model)
		           VALUES ('fake', 'image', 'Fake', '\x00', 'abcd', 'fake-1') RETURNING id)
		INSERT INTO generated_images (user_id, provider_id, model, final_prompt, storage_key, thumb_key)
		SELECT u.id, p.id, 'fake-1', 'x', 'portraits/u/a.webp', 'portraits/u/a_thumb.webp' FROM u, p;
		WITH c AS (INSERT INTO guide_categories (slug, title) VALUES ('lore', 'Lore') RETURNING id)
		INSERT INTO guide_articles (category_id, slug, title, content, cover_key)
		SELECT c.id, 'naga', 'Naga', '![peta](https://api.example/v1/media/content/map.webp)', 'content/cover.webp' FROM c;`)
	if err != nil {
		t.Fatal(err)
	}

	got, err := sqlc.New(pool).ReferencedObjectKeys(ctx, []string{
		"portraits/u/a.webp", "portraits/u/a_thumb.webp", "portraits/u/b.webp",
		"content/cover.webp", "content/map.webp", "content/unused.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{"content/cover.webp", "content/map.webp", "portraits/u/a.webp", "portraits/u/a_thumb.webp"}
	if !slices.Equal(got, want) {
		t.Errorf("referenced = %v, want %v", got, want)
	}
}
