package guide

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

//go:embed seed/categories.json
var seedCategories []byte

// SeedCategories creates the starter guide categories whose slugs do not
// exist yet and reports how many it created.
func (s *Service) SeedCategories(ctx context.Context) (int, error) {
	var want []CategoryInput
	if err := json.Unmarshal(seedCategories, &want); err != nil {
		return 0, fmt.Errorf("guide: seed: decode: %w", err)
	}
	existing, err := s.store.Categories(ctx)
	if err != nil {
		return 0, fmt.Errorf("guide: seed: %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, c := range existing {
		have[c.Slug] = true
	}

	created := 0
	for _, in := range want {
		if have[in.Slug] {
			continue
		}
		if _, err := s.CreateCategory(ctx, uuid.Nil, in); err != nil {
			return created, fmt.Errorf("guide: seed category %s: %w", in.Slug, err)
		}
		created++
	}
	return created, nil
}
