package stats

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// PostgresStore implements Store with sqlc.
type PostgresStore struct {
	q *sqlc.Queries
}

// NewPostgresStore returns a Store using db (a pool).
func NewPostgresStore(db sqlc.DBTX) *PostgresStore { return &PostgresStore{q: sqlc.New(db)} }

func (s *PostgresStore) Totals(ctx context.Context) (Totals, error) {
	r, err := s.q.StatsTotals(ctx)
	if err != nil {
		return Totals{}, err
	}
	return Totals{
		Users: int(r.Users), BannedUsers: int(r.BannedUsers), Admins: int(r.Admins),
		Characters: int(r.Characters), PublicCharacters: int(r.PublicCharacters),
		Portraits: int(r.Portraits), PortraitsSucceeded: int(r.PortraitsSucceeded), PortraitsFailed: int(r.PortraitsFailed),
		PortraitsPending: int(r.PortraitsPending), PortraitsHidden: int(r.PortraitsHidden),
		Lores: int(r.Lores), LoresSucceeded: int(r.LoresSucceeded), LoresFailed: int(r.LoresFailed), LoresPending: int(r.LoresPending),
	}, nil
}

func (s *PostgresStore) Days(ctx context.Context, from, to, since time.Time) ([]Day, error) {
	rows, err := s.q.DailyActivity(ctx, sqlc.DailyActivityParams{
		FromDay: pgtype.Date{Time: from, Valid: true}, ToDay: pgtype.Date{Time: to, Valid: true}, Since: since,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Day, len(rows))
	for i, r := range rows {
		out[i] = Day{
			Date: r.Day.Time, NewUsers: int(r.NewUsers), NewCharacters: int(r.NewCharacters),
			PortraitsSucceeded: int(r.PortraitsSucceeded), PortraitsFailed: int(r.PortraitsFailed),
			LoresSucceeded: int(r.LoresSucceeded), LoresFailed: int(r.LoresFailed),
		}
	}
	return out, nil
}

func (s *PostgresStore) Models(ctx context.Context, since time.Time) ([]ModelUsage, error) {
	rows, err := s.q.ModelUsage(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]ModelUsage, len(rows))
	for i, r := range rows {
		out[i] = ModelUsage{
			Capability: r.Capability, ProviderID: r.ProviderID, ProviderLabel: r.ProviderLabel, Model: r.Model,
			Requests: int(r.Requests), Succeeded: int(r.Succeeded), Failed: int(r.Failed),
			PromptTokens: r.PromptTokens, CompletionTokens: r.CompletionTokens,
		}
	}
	return out, nil
}
