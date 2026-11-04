//go:build integration

package quota

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/dbtest"
)

func TestPostgresSettings(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t, "audit_logs")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE app_settings SET value = '5' WHERE key = 'default_daily_quota';
			UPDATE app_settings SET value = '10' WHERE key = 'default_daily_lore_quota'`)
	})
	svc := NewSettings(NewPostgresSettings(pool))

	if d, err := svc.Defaults(ctx); err != nil || d != (Defaults{Portrait: 5, Lore: 10}) {
		t.Errorf("seeded defaults = %+v, %v", d, err)
	}
	if _, err := svc.SetDefaults(ctx, uuid.Nil, Defaults{Portrait: 8, Lore: 20}); err != nil {
		t.Fatal(err)
	}
	if d, err := svc.Defaults(ctx); err != nil || d != (Defaults{Portrait: 8, Lore: 20}) {
		t.Errorf("stored defaults = %+v, %v", d, err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE entity = 'app_setting'").Scan(&n); err != nil || n != 1 {
		t.Errorf("audit entries = %d, %v", n, err)
	}
}
