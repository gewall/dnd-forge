-- name: LockUserLores :exec
-- Serialises quota checks of one user across concurrent requests.
SELECT pg_advisory_xact_lock(hashtextextended('lore:' || sqlc.arg(user_id)::text, 0));

-- name: CountLoresSince :one
-- FAILED drafts do not count against the quota.
SELECT count(*) FROM character_lores
WHERE user_id = $1 AND created_at >= $2 AND status <> 'FAILED';

-- name: CharacterForLore :one
SELECT c.id, c.user_id, c.name, c.gender, c.appearance,
       r.name AS race_name, s.name AS subrace_name, k.name AS class_name, b.name AS background_name
FROM characters c
JOIN races r ON r.id = c.race_id
LEFT JOIN subraces s ON s.id = c.subrace_id
JOIN classes k ON k.id = c.class_id
JOIN backgrounds b ON b.id = c.background_id
WHERE c.id = $1;

-- name: CreateLore :one
INSERT INTO character_lores (character_id, user_id, provider_id, model, user_prompt, final_prompt, tone, length, language)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: LoreByID :one
SELECT * FROM character_lores WHERE id = $1;

-- name: LoresByCharacter :many
SELECT * FROM character_lores
WHERE character_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT 50;

-- name: CompleteLore :exec
UPDATE character_lores
SET status = 'SUCCEEDED', content = $2, error_message = NULL, completed_at = now()
WHERE id = $1 AND status = 'PENDING';

-- name: FailLore :exec
UPDATE character_lores
SET status = 'FAILED', error_message = $2, completed_at = now()
WHERE id = $1 AND status = 'PENDING';

-- name: SetCharacterBackstory :execrows
UPDATE characters SET backstory = $3, updated_at = now()
WHERE id = $1 AND user_id = $2;
