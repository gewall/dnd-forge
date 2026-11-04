//go:build integration

package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

func TestReader(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "users", "audit_logs")
	var admin uuid.UUID
	if err := pool.QueryRow(ctx, "INSERT INTO users (email, role) VALUES ('boss@example.com', 'ADMIN') RETURNING id").Scan(&admin); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	for _, e := range []Entry{
		{Action: ActionCreate, Entity: "race", EntityID: "r1"},
		{ActorID: admin, Action: ActionUpdate, Entity: "user", EntityID: "u1", Diff: map[string][2]bool{"isBanned": {false, true}}},
		{ActorID: admin, Action: ActionDelete, Entity: "generated_image", EntityID: "g1"},
	} {
		if err := Write(ctx, q, e); err != nil {
			t.Fatal(err)
		}
	}

	r := NewReader(pool)
	all, err := r.List(ctx, Filter{})
	if err != nil || len(all) != 3 || all[0].Entity != "generated_image" {
		t.Fatalf("List = %+v, %v; want 3 newest first", all, err)
	}
	if all[2].ActorID != nil || all[2].ActorEmail != "" {
		t.Errorf("system entry = %+v", all[2])
	}
	user := all[1]
	diff, ok := user.Diff.(map[string]any)
	if user.ActorEmail != "boss@example.com" || !ok || diff["isBanned"] == nil {
		t.Errorf("user entry = %+v", user)
	}
	if got, _ := r.List(ctx, Filter{ActorID: &admin}); len(got) != 2 {
		t.Errorf("by actor = %d entries", len(got))
	}
	if got, _ := r.List(ctx, Filter{Entity: "user", EntityID: "u1"}); len(got) != 1 {
		t.Errorf("by entity = %d entries", len(got))
	}
	if got, _ := r.List(ctx, Filter{BeforeID: all[0].ID, Limit: 1}); len(got) != 1 || got[0].ID != all[1].ID {
		t.Errorf("next page = %+v", got)
	}
}
