package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// Record is one stored entry.
type Record struct {
	ID         int64
	ActorID    *uuid.UUID // nil for the system or a deleted user
	ActorEmail string
	Action     string
	Entity     string
	EntityID   string
	Diff       any // decoded JSON; nil when none was recorded
	CreatedAt  time.Time
}

// Filter narrows the log. Zero values do not filter.
type Filter struct {
	Entity   string
	EntityID string
	ActorID  *uuid.UUID
	BeforeID int64 // id cursor
	Limit    int
}

const (
	defaultPage = 50
	maxPage     = 200
)

// Reader lists the audit log for admins.
type Reader struct {
	q *sqlc.Queries
}

// NewReader returns a Reader using db (a pool).
func NewReader(db sqlc.DBTX) *Reader { return &Reader{q: sqlc.New(db)} }

// List returns entries matching f, newest first.
func (r *Reader) List(ctx context.Context, f Filter) ([]Record, error) {
	if f.Limit <= 0 {
		f.Limit = defaultPage
	}
	arg := sqlc.ListAuditLogsParams{
		ActorID:    f.ActorID,
		MaxResults: int32(min(f.Limit, maxPage)), //nolint:gosec // G115: capped
	}
	if f.Entity != "" {
		arg.Entity = &f.Entity
	}
	if f.EntityID != "" {
		arg.EntityID = &f.EntityID
	}
	if f.BeforeID > 0 {
		arg.BeforeID = &f.BeforeID
	}
	rows, err := r.q.ListAuditLogs(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("audit: list: %w", err)
	}
	out := make([]Record, len(rows))
	for i, row := range rows {
		rec := Record{
			ID: row.ID, ActorID: row.ActorID, Action: row.Action, Entity: row.Entity,
			EntityID: row.EntityID, CreatedAt: row.CreatedAt,
		}
		if row.ActorEmail != nil {
			rec.ActorEmail = *row.ActorEmail
		}
		if len(row.Diff) > 0 {
			if err := json.Unmarshal(row.Diff, &rec.Diff); err != nil {
				return nil, fmt.Errorf("audit: decode diff %d: %w", row.ID, err)
			}
		}
		out[i] = rec
	}
	return out, nil
}
