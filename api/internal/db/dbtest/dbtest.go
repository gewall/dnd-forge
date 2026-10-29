// Package dbtest provides a migrated PostgreSQL pool for integration tests.
//
// Tests using it share one database, so run them with -p 1:
//
//	TEST_DATABASE_URL=postgres://... go test -p 1 -tags integration ./...
package dbtest

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/db"
)

// Pool returns a pool to TEST_DATABASE_URL with migrations applied and the
// given tables truncated. It skips the test when the variable is unset.
func Pool(t *testing.T, truncate ...string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	if len(truncate) > 0 {
		if _, err := pool.Exec(ctx, "TRUNCATE "+strings.Join(truncate, ", ")+" CASCADE"); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}
