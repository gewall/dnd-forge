-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, entity, entity_id, diff)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAuditLogs :many
-- Newest first; pass the last item's id as before_id for the next page.
SELECT a.id, a.actor_id, u.email AS actor_email, a.action, a.entity, a.entity_id, a.diff, a.created_at
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE (sqlc.narg(entity)::text IS NULL OR a.entity = sqlc.narg(entity))
  AND (sqlc.narg(entity_id)::text IS NULL OR a.entity_id = sqlc.narg(entity_id))
  AND (sqlc.narg(actor_id)::uuid IS NULL OR a.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(before_id)::bigint IS NULL OR a.id < sqlc.narg(before_id))
ORDER BY a.id DESC
LIMIT sqlc.arg(max_results);
