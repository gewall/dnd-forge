-- name: ListCharactersByUser :many
SELECT c.id, c.user_id, c.name, c.gender, c.appearance, c.is_public, c.created_at, c.updated_at,
       r.id AS race_id, r.slug AS race_slug, r.name AS race_name,
       s.id AS subrace_id, s.slug AS subrace_slug, s.name AS subrace_name,
       k.id AS class_id, k.slug AS class_slug, k.name AS class_name,
       b.id AS background_id, b.slug AS background_slug, b.name AS background_name
FROM characters c
JOIN races r ON r.id = c.race_id
LEFT JOIN subraces s ON s.id = c.subrace_id
JOIN classes k ON k.id = c.class_id
JOIN backgrounds b ON b.id = c.background_id
WHERE c.user_id = $1
ORDER BY c.updated_at DESC;

-- name: CharacterByID :one
SELECT c.id, c.user_id, c.name, c.gender, c.appearance, c.is_public, c.created_at, c.updated_at,
       r.id AS race_id, r.slug AS race_slug, r.name AS race_name,
       s.id AS subrace_id, s.slug AS subrace_slug, s.name AS subrace_name,
       k.id AS class_id, k.slug AS class_slug, k.name AS class_name,
       b.id AS background_id, b.slug AS background_slug, b.name AS background_name
FROM characters c
JOIN races r ON r.id = c.race_id
LEFT JOIN subraces s ON s.id = c.subrace_id
JOIN classes k ON k.id = c.class_id
JOIN backgrounds b ON b.id = c.background_id
WHERE c.id = $1;

-- name: CountCharactersByUser :one
SELECT count(*) FROM characters WHERE user_id = $1;

-- name: CreateCharacter :one
INSERT INTO characters (user_id, name, gender, race_id, subrace_id, class_id, background_id, appearance, is_public)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: UpdateCharacter :execrows
UPDATE characters SET
  name = $3, gender = $4, race_id = $5, subrace_id = $6, class_id = $7, background_id = $8,
  appearance = $9, is_public = $10, updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: DeleteCharacter :execrows
DELETE FROM characters WHERE id = $1 AND user_id = $2;
