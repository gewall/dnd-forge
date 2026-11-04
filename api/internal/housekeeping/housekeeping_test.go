package housekeeping

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gewall/dnd-forge/api/internal/storage"
)

var now = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

type bucket struct {
	objects []storage.Object
	deleted []string
}

func (b *bucket) List(_ context.Context, prefix string) ([]storage.Object, error) {
	var out []storage.Object
	for _, o := range b.objects {
		if strings.HasPrefix(o.Key, prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (b *bucket) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return nil
}

type refs struct {
	used    map[string]bool
	batches int
}

func (r *refs) ReferencedObjectKeys(_ context.Context, keys []string) ([]string, error) {
	r.batches++
	var out []string
	for _, k := range keys {
		if r.used[k] {
			out = append(out, k)
		}
	}
	return out, nil
}

func keys(objs []storage.Object) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.Key
	}
	slices.Sort(out)
	return out
}

func TestOrphans(t *testing.T) {
	old, fresh := now.Add(-48*time.Hour), now.Add(-time.Hour)
	b := &bucket{objects: []storage.Object{
		{Key: "portraits/u/used.webp", LastModified: old},
		{Key: "portraits/u/gone.webp", LastModified: old},
		{Key: "content/used.webp", LastModified: old},
		{Key: "content/gone.webp", LastModified: old},
		{Key: "content/just-uploaded.webp", LastModified: fresh},
		{Key: "elsewhere/gone.webp", LastModified: old},
	}}
	r := &refs{used: map[string]bool{"portraits/u/used.webp": true, "content/used.webp": true}}
	svc := NewService(b, r)
	svc.now = func() time.Time { return now }

	orphans, err := svc.Orphans(context.Background(), DefaultMinAge)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"content/gone.webp", "portraits/u/gone.webp"}; !slices.Equal(keys(orphans), want) {
		t.Errorf("orphans = %v, want %v", keys(orphans), want)
	}
	if len(b.deleted) != 0 {
		t.Errorf("Orphans deleted %v", b.deleted)
	}

	deleted, err := svc.DeleteOrphans(context.Background(), DefaultMinAge)
	if err != nil || len(deleted) != 2 || len(b.deleted) != 2 {
		t.Errorf("DeleteOrphans = %v, %v; bucket deleted %v", keys(deleted), err, b.deleted)
	}
}

func TestOrphansBatchesReferenceChecks(t *testing.T) {
	b := &bucket{}
	for i := range referenceBatch + 1 {
		b.objects = append(b.objects, storage.Object{Key: fmt.Sprintf("content/%d.webp", i), LastModified: now.Add(-48 * time.Hour)})
	}
	r := &refs{used: map[string]bool{}}
	svc := NewService(b, r)
	svc.now = func() time.Time { return now }
	orphans, err := svc.Orphans(context.Background(), DefaultMinAge)
	if err != nil || len(orphans) != referenceBatch+1 || r.batches != 2 {
		t.Errorf("Orphans = %d, %v in %d batches", len(orphans), err, r.batches)
	}
}
