// Package audit records who changed what. Stores call Write inside the same
// transaction as the change, so a change is never saved without its log.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
)

// Actions.
const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionActivate = "activate"
)

// Entry describes one change. A zero ActorID means the system (e.g. seeding).
type Entry struct {
	ActorID  uuid.UUID
	Action   string
	Entity   string
	EntityID string
	Diff     any // marshalled to JSON; nil for deletes
}

// Write inserts e using q, which should be bound to the change's transaction.
func Write(ctx context.Context, q *sqlc.Queries, e Entry) error {
	var diff []byte
	if e.Diff != nil {
		b, err := json.Marshal(e.Diff)
		if err != nil {
			return fmt.Errorf("audit: marshal diff: %w", err)
		}
		diff = b
	}
	var actor *uuid.UUID
	if e.ActorID != uuid.Nil {
		actor = &e.ActorID
	}
	err := q.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorID:  actor,
		Action:   e.Action,
		Entity:   e.Entity,
		EntityID: e.EntityID,
		Diff:     diff,
	})
	if err != nil {
		return fmt.Errorf("audit: insert: %w", err)
	}
	return nil
}
