-- name: StatsTotals :one
SELECT
  (SELECT count(*) FROM users) AS users,
  (SELECT count(*) FROM users WHERE is_banned) AS banned_users,
  (SELECT count(*) FROM users WHERE role = 'ADMIN') AS admins,
  (SELECT count(*) FROM characters) AS characters,
  (SELECT count(*) FROM characters WHERE is_public) AS public_characters,
  (SELECT count(*) FROM generated_images) AS portraits,
  (SELECT count(*) FROM generated_images WHERE status = 'SUCCEEDED') AS portraits_succeeded,
  (SELECT count(*) FROM generated_images WHERE status = 'FAILED') AS portraits_failed,
  (SELECT count(*) FROM generated_images WHERE status = 'PENDING') AS portraits_pending,
  (SELECT count(*) FROM generated_images WHERE is_hidden) AS portraits_hidden,
  (SELECT count(*) FROM character_lores) AS lores,
  (SELECT count(*) FROM character_lores WHERE status = 'SUCCEEDED') AS lores_succeeded,
  (SELECT count(*) FROM character_lores WHERE status = 'FAILED') AS lores_failed,
  (SELECT count(*) FROM character_lores WHERE status = 'PENDING') AS lores_pending;

-- name: DailyActivity :many
-- One row per WIB day from from_day to to_day, including empty days.
WITH days AS (
  SELECT generate_series(sqlc.arg(from_day)::date, sqlc.arg(to_day)::date, interval '1 day')::date AS day
), u AS (
  SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS day, count(*) AS n
  FROM users WHERE created_at >= sqlc.arg(since)::timestamptz GROUP BY 1
), c AS (
  SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS day, count(*) AS n
  FROM characters WHERE created_at >= sqlc.arg(since)::timestamptz GROUP BY 1
), g AS (
  SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS day,
         count(*) FILTER (WHERE status = 'SUCCEEDED') AS ok, count(*) FILTER (WHERE status = 'FAILED') AS failed
  FROM generated_images WHERE created_at >= sqlc.arg(since)::timestamptz GROUP BY 1
), l AS (
  SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS day,
         count(*) FILTER (WHERE status = 'SUCCEEDED') AS ok, count(*) FILTER (WHERE status = 'FAILED') AS failed
  FROM character_lores WHERE created_at >= sqlc.arg(since)::timestamptz GROUP BY 1
)
SELECT days.day,
       coalesce(u.n, 0)::int AS new_users,
       coalesce(c.n, 0)::int AS new_characters,
       coalesce(g.ok, 0)::int AS portraits_succeeded,
       coalesce(g.failed, 0)::int AS portraits_failed,
       coalesce(l.ok, 0)::int AS lores_succeeded,
       coalesce(l.failed, 0)::int AS lores_failed
FROM days
LEFT JOIN u ON u.day = days.day
LEFT JOIN c ON c.day = days.day
LEFT JOIN g ON g.day = days.day
LEFT JOIN l ON l.day = days.day
ORDER BY days.day;

-- name: ModelUsage :many
-- Requests per provider and model since a moment, busiest first.
SELECT u.capability, u.provider_id, u.provider_label, u.model,
       u.requests, u.succeeded, u.failed, u.prompt_tokens, u.completion_tokens
FROM (
  SELECT 'image'::text AS capability, g.provider_id, p.label AS provider_label, g.model,
         count(*)::int AS requests,
         (count(*) FILTER (WHERE g.status = 'SUCCEEDED'))::int AS succeeded,
         (count(*) FILTER (WHERE g.status = 'FAILED'))::int AS failed,
         0::bigint AS prompt_tokens, 0::bigint AS completion_tokens
  FROM generated_images g JOIN ai_providers p ON p.id = g.provider_id
  WHERE g.created_at >= sqlc.arg(since)::timestamptz
  GROUP BY g.provider_id, p.label, g.model
  UNION ALL
  SELECT 'text'::text, l.provider_id, p.label, l.model,
         count(*)::int,
         (count(*) FILTER (WHERE l.status = 'SUCCEEDED'))::int,
         (count(*) FILTER (WHERE l.status = 'FAILED'))::int,
         coalesce(sum(l.prompt_tokens), 0)::bigint, coalesce(sum(l.completion_tokens), 0)::bigint
  FROM character_lores l JOIN ai_providers p ON p.id = l.provider_id
  WHERE l.created_at >= sqlc.arg(since)::timestamptz
  GROUP BY l.provider_id, p.label, l.model
) u
ORDER BY u.requests DESC, u.model;
