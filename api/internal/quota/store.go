package quota

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/audit"
	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// PostgresSettings implements SettingsStore with sqlc.
type PostgresSettings struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresSettings returns a SettingsStore backed by pool.
func NewPostgresSettings(pool *pgxpool.Pool) *PostgresSettings {
	return &PostgresSettings{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresSettings) InTx(ctx context.Context, fn func(SettingsStore) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&PostgresSettings{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func (s *PostgresSettings) SettingValue(ctx context.Context, key string) ([]byte, error) {
	return s.q.SettingValue(ctx, key)
}

func (s *PostgresSettings) SetSettingValue(ctx context.Context, key string, value []byte) error {
	return s.q.SetSettingValue(ctx, sqlc.SetSettingValueParams{Key: key, Value: value})
}

func (s *PostgresSettings) RecordAudit(ctx context.Context, e audit.Entry) error {
	return audit.Write(ctx, s.q, e)
}
