// Package db owns the PostgreSQL connection pool, schema migrations and the
// sqlc-generated queries in the sqlc subpackage.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open connects to PostgreSQL and verifies the connection.
//
// The pool reconnects on its own, which matters on the VPS: when the primary
// fails over, the "postgres" host name moves to the promoted standby and new
// connections land there.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db: parse url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.HealthCheckPeriod = 15 * time.Second
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// Migrate applies the application migrations and then River's own tables.
// A Postgres advisory lock keeps concurrent starts from racing.
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("db: migrations fs: %w", err)
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("db: session locker: %w", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, sub, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("db: goose provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	for _, r := range results {
		log.Info("migration applied", "source", r.Source.Path, "duration", r.Duration)
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("db: river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("db: river migrate: %w", err)
	}
	return nil
}

// InTx runs fn inside a transaction, committing when fn returns nil and
// rolling back otherwise.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

// Int32 converts n for an int4 column, saturating at the column's range.
// Inputs are already bounded by request validation; this only guards
// against silent wrap-around.
func Int32(n int) int32 {
	switch {
	case n > math.MaxInt32:
		return math.MaxInt32
	case n < math.MinInt32:
		return math.MinInt32
	}
	return int32(n)
}

// IsNoRows reports whether err means a query matched no row.
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsUniqueViolation reports whether err is a unique constraint violation.
func IsUniqueViolation(err error) bool { return hasCode(err, "23505") }

// IsForeignKeyViolation reports whether err is a foreign key violation,
// e.g. deleting a row other rows still reference.
func IsForeignKeyViolation(err error) bool { return hasCode(err, "23503") }

// ViolatedConstraint returns the name of the constraint err violated, or "".
func ViolatedConstraint(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.ConstraintName
	}
	return ""
}

func hasCode(err error, code string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == code
}
