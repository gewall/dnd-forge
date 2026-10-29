-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, entity, entity_id, diff)
VALUES ($1, $2, $3, $4, $5);
