-- name: LockUserPortraits :exec
-- Serialises quota checks of one user across concurrent requests.
SELECT pg_advisory_xact_lock(hashtextextended('portrait:' || sqlc.arg(user_id)::text, 0));

-- name: CountPortraitsSince :one
-- FAILED images do not count against the quota.
SELECT count(*) FROM generated_images
WHERE user_id = $1 AND created_at >= $2 AND status <> 'FAILED';

-- name: UserDailyQuota :one
SELECT daily_quota FROM users WHERE id = $1;

-- name: CharacterForPortrait :one
SELECT c.id, c.user_id, c.name, c.gender, c.appearance,
       r.name AS race_name, r.visual_prompt AS race_visual,
       s.name AS subrace_name, s.visual_prompt AS subrace_visual,
       k.name AS class_name, k.visual_prompt AS class_visual,
       b.name AS background_name, b.visual_prompt AS background_visual
FROM characters c
JOIN races r ON r.id = c.race_id
LEFT JOIN subraces s ON s.id = c.subrace_id
JOIN classes k ON k.id = c.class_id
JOIN backgrounds b ON b.id = c.background_id
WHERE c.id = $1;

-- name: CreatePortrait :one
INSERT INTO generated_images (character_id, user_id, provider_id, model, user_prompt, final_prompt, style, aspect_ratio)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: PortraitByID :one
SELECT * FROM generated_images WHERE id = $1;

-- name: PortraitsByCharacter :many
SELECT * FROM generated_images
WHERE character_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT 50;

-- name: CompletePortrait :exec
UPDATE generated_images
SET status = 'SUCCEEDED', storage_key = $2, thumb_key = $3, error_message = NULL, completed_at = now()
WHERE id = $1;

-- name: FailPortrait :exec
UPDATE generated_images
SET status = 'FAILED', error_message = $2, completed_at = now()
WHERE id = $1 AND status = 'PENDING';

-- name: Gallery :many
-- Public characters with their newest visible portrait, newest first.
-- Cursor pagination: pass the last item's created_at as before.
SELECT latest.* FROM (
  SELECT DISTINCT ON (g.character_id)
         g.id, g.character_id, g.created_at, g.style,
         c.name AS character_name, r.name AS race_name, k.name AS class_name
  FROM generated_images g
  JOIN characters c ON c.id = g.character_id
  JOIN races r ON r.id = c.race_id
  JOIN classes k ON k.id = c.class_id
  WHERE c.is_public AND g.status = 'SUCCEEDED' AND NOT g.is_hidden
  ORDER BY g.character_id, g.created_at DESC
) latest
WHERE sqlc.narg(before)::timestamptz IS NULL OR latest.created_at < sqlc.narg(before)
ORDER BY latest.created_at DESC
LIMIT sqlc.arg(max_results);

-- Admin moderation ------------------------------------------------------------

-- name: AdminImages :many
-- Newest first; pass the last item's created_at as before for the next page.
SELECT g.id, g.character_id, g.user_id, g.model, g.user_prompt, g.style, g.aspect_ratio,
       g.status, g.error_message, g.storage_key, g.thumb_key, g.is_hidden, g.created_at, g.completed_at,
       u.email AS user_email, c.name AS character_name, c.is_public AS character_public
FROM generated_images g
JOIN users u ON u.id = g.user_id
LEFT JOIN characters c ON c.id = g.character_id
WHERE (sqlc.narg(status)::image_status IS NULL OR g.status = sqlc.narg(status))
  AND (sqlc.narg(hidden)::boolean IS NULL OR g.is_hidden = sqlc.narg(hidden))
  AND (sqlc.narg(user_id)::uuid IS NULL OR g.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(before)::timestamptz IS NULL OR g.created_at < sqlc.narg(before))
ORDER BY g.created_at DESC
LIMIT sqlc.arg(max_results);

-- name: AdminImageByID :one
SELECT g.id, g.character_id, g.user_id, g.model, g.user_prompt, g.style, g.aspect_ratio,
       g.status, g.error_message, g.storage_key, g.thumb_key, g.is_hidden, g.created_at, g.completed_at,
       u.email AS user_email, c.name AS character_name, c.is_public AS character_public
FROM generated_images g
JOIN users u ON u.id = g.user_id
LEFT JOIN characters c ON c.id = g.character_id
WHERE g.id = $1;

-- name: LockPortrait :one
SELECT * FROM generated_images WHERE id = $1 FOR UPDATE;

-- name: SetPortraitHidden :exec
UPDATE generated_images SET is_hidden = $2 WHERE id = $1;

-- name: ClearPortraitFiles :exec
-- The row stays for quota and statistics; without files it can never be shown.
UPDATE generated_images SET storage_key = NULL, thumb_key = NULL, is_hidden = true WHERE id = $1;
