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
