-- name: ListAccounts :many
-- Newest first; pass the last item's created_at as before for the next page.
SELECT u.id, u.email, u.name, u.role, u.is_banned, u.daily_quota, u.daily_lore_quota,
       (u.password_hash IS NOT NULL)::boolean AS has_password, (u.google_sub IS NOT NULL)::boolean AS has_google,
       u.created_at, u.updated_at,
       (SELECT count(*) FROM characters c WHERE c.user_id = u.id) AS characters
FROM users u
WHERE (sqlc.narg(pattern)::text IS NULL OR u.email ILIKE sqlc.narg(pattern) OR u.name ILIKE sqlc.narg(pattern))
  AND (sqlc.narg(role)::user_role IS NULL OR u.role = sqlc.narg(role))
  AND (sqlc.narg(banned)::boolean IS NULL OR u.is_banned = sqlc.narg(banned))
  AND (sqlc.narg(before)::timestamptz IS NULL OR u.created_at < sqlc.narg(before))
ORDER BY u.created_at DESC
LIMIT sqlc.arg(max_results);

-- name: AccountByID :one
SELECT u.id, u.email, u.name, u.role, u.is_banned, u.daily_quota, u.daily_lore_quota,
       (u.password_hash IS NOT NULL)::boolean AS has_password, (u.google_sub IS NOT NULL)::boolean AS has_google,
       u.created_at, u.updated_at,
       (SELECT count(*) FROM characters c WHERE c.user_id = u.id) AS characters,
       (SELECT count(*) FROM generated_images g WHERE g.user_id = u.id) AS portraits,
       (SELECT count(*) FROM character_lores l WHERE l.user_id = u.id) AS lores
FROM users u
WHERE u.id = $1;

-- name: LockAccount :one
SELECT role, is_banned, daily_quota, daily_lore_quota FROM users WHERE id = $1 FOR UPDATE;

-- name: UpdateAccount :exec
UPDATE users
SET role = $2, is_banned = $3, daily_quota = $4, daily_lore_quota = $5, updated_at = now()
WHERE id = $1;

-- name: AccountStatus :one
SELECT role, is_banned FROM users WHERE id = $1;
