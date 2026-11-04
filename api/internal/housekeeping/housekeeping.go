// Package housekeeping removes data nothing points to any more, such as
// storage objects left behind by deleted rows or interrupted uploads.
package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/gewall/dnd-forge/api/internal/storage"
)

// Prefixes holds every storage prefix the API writes to.
var Prefixes = []string{"portraits/", "content/"}

// DefaultMinAge keeps objects younger than this: an admin may upload an
// image a while before saving the article that uses it.
const DefaultMinAge = 24 * time.Hour

// referenceBatch bounds the keys checked per query.
const referenceBatch = 500

// Objects lists and deletes stored objects.
type Objects interface {
	List(ctx context.Context, prefix string) ([]storage.Object, error)
	Delete(ctx context.Context, key string) error
}

// References reports which of keys a database row still points to.
type References interface {
	ReferencedObjectKeys(ctx context.Context, keys []string) ([]string, error)
}

// Service finds and removes orphaned objects.
type Service struct {
	objects Objects
	refs    References
	now     func() time.Time
}

// NewService returns a Service.
func NewService(objects Objects, refs References) *Service {
	return &Service{objects: objects, refs: refs, now: time.Now}
}

// Orphans lists objects under Prefixes that are older than minAge and that
// no row references.
func (s *Service) Orphans(ctx context.Context, minAge time.Duration) ([]storage.Object, error) {
	cutoff := s.now().Add(-minAge)
	var candidates []storage.Object
	for _, prefix := range Prefixes {
		objs, err := s.objects.List(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("housekeeping: %w", err)
		}
		for _, o := range objs {
			if o.LastModified.Before(cutoff) {
				candidates = append(candidates, o)
			}
		}
	}

	var orphans []storage.Object
	for start := 0; start < len(candidates); start += referenceBatch {
		batch := candidates[start:min(start+referenceBatch, len(candidates))]
		keys := make([]string, len(batch))
		for i, o := range batch {
			keys[i] = o.Key
		}
		used, err := s.refs.ReferencedObjectKeys(ctx, keys)
		if err != nil {
			return nil, fmt.Errorf("housekeeping: references: %w", err)
		}
		inUse := make(map[string]bool, len(used))
		for _, k := range used {
			inUse[k] = true
		}
		for _, o := range batch {
			if !inUse[o.Key] {
				orphans = append(orphans, o)
			}
		}
	}
	return orphans, nil
}

// DeleteOrphans removes what Orphans finds and returns the removed objects.
func (s *Service) DeleteOrphans(ctx context.Context, minAge time.Duration) ([]storage.Object, error) {
	orphans, err := s.Orphans(ctx, minAge)
	if err != nil {
		return nil, err
	}
	for i, o := range orphans {
		if err := s.objects.Delete(ctx, o.Key); err != nil {
			return orphans[:i], fmt.Errorf("housekeeping: %w", err)
		}
	}
	return orphans, nil
}
