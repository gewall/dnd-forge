-- Races ----------------------------------------------------------------------

-- name: ListRaces :many
SELECT * FROM races
WHERE sqlc.arg(include_inactive)::bool OR is_active
ORDER BY sort_order, name;

-- name: RaceBySlug :one
SELECT * FROM races WHERE slug = $1;

-- name: RaceByID :one
SELECT * FROM races WHERE id = $1;

-- name: CreateRace :one
INSERT INTO races (slug, name, description, traits, visual_prompt, image_key, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateRace :one
UPDATE races SET
  slug = $2, name = $3, description = $4, traits = $5, visual_prompt = $6,
  image_key = $7, sort_order = $8, is_active = $9
WHERE id = $1
RETURNING *;

-- name: DeleteRace :execrows
DELETE FROM races WHERE id = $1;

-- Subraces -------------------------------------------------------------------

-- name: ListSubraces :many
SELECT * FROM subraces
WHERE (sqlc.arg(include_inactive)::bool OR is_active)
  AND (sqlc.narg(race_id)::uuid IS NULL OR race_id = sqlc.narg(race_id))
ORDER BY name;

-- name: SubraceByID :one
SELECT * FROM subraces WHERE id = $1;

-- name: CreateSubrace :one
INSERT INTO subraces (race_id, slug, name, description, traits, visual_prompt, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateSubrace :one
UPDATE subraces SET
  slug = $2, name = $3, description = $4, traits = $5, visual_prompt = $6, is_active = $7
WHERE id = $1
RETURNING *;

-- name: DeleteSubrace :execrows
DELETE FROM subraces WHERE id = $1;

-- Classes --------------------------------------------------------------------

-- name: ListClasses :many
SELECT * FROM classes
WHERE sqlc.arg(include_inactive)::bool OR is_active
ORDER BY sort_order, name;

-- name: ClassBySlug :one
SELECT * FROM classes WHERE slug = $1;

-- name: ClassByID :one
SELECT * FROM classes WHERE id = $1;

-- name: CreateClass :one
INSERT INTO classes (slug, name, hit_die, primary_ability, description, features, visual_prompt, image_key, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateClass :one
UPDATE classes SET
  slug = $2, name = $3, hit_die = $4, primary_ability = $5, description = $6,
  features = $7, visual_prompt = $8, image_key = $9, sort_order = $10, is_active = $11
WHERE id = $1
RETURNING *;

-- name: DeleteClass :execrows
DELETE FROM classes WHERE id = $1;

-- Backgrounds ----------------------------------------------------------------

-- name: ListBackgrounds :many
SELECT * FROM backgrounds
WHERE sqlc.arg(include_inactive)::bool OR is_active
ORDER BY sort_order, name;

-- name: BackgroundBySlug :one
SELECT * FROM backgrounds WHERE slug = $1;

-- name: BackgroundByID :one
SELECT * FROM backgrounds WHERE id = $1;

-- name: CreateBackground :one
INSERT INTO backgrounds (slug, name, description, skill_proficiencies, visual_prompt, image_key, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateBackground :one
UPDATE backgrounds SET
  slug = $2, name = $3, description = $4, skill_proficiencies = $5, visual_prompt = $6,
  image_key = $7, sort_order = $8, is_active = $9
WHERE id = $1
RETURNING *;

-- name: DeleteBackground :execrows
DELETE FROM backgrounds WHERE id = $1;
