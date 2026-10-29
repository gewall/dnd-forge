-- Categories -----------------------------------------------------------------

-- name: ListGuideCategories :many
SELECT * FROM guide_categories ORDER BY sort_order, title;

-- name: GuideCategoryByID :one
SELECT * FROM guide_categories WHERE id = $1;

-- name: CreateGuideCategory :one
INSERT INTO guide_categories (slug, title, description, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateGuideCategory :one
UPDATE guide_categories SET slug = $2, title = $3, description = $4, sort_order = $5
WHERE id = $1
RETURNING *;

-- name: DeleteGuideCategory :execrows
DELETE FROM guide_categories WHERE id = $1;

-- Public articles ------------------------------------------------------------

-- name: ListPublishedGuideSummaries :many
SELECT a.id, a.category_id, a.slug, a.title, a.excerpt, a.cover_key, a.sort_order, a.published_at
FROM guide_articles a
WHERE a.status = 'PUBLISHED'
ORDER BY a.sort_order, a.published_at DESC;

-- name: PublishedGuideArticle :one
SELECT a.*, c.slug AS category_slug, c.title AS category_title
FROM guide_articles a
JOIN guide_categories c ON c.id = a.category_id
WHERE c.slug = sqlc.arg(category_slug) AND a.slug = sqlc.arg(slug) AND a.status = 'PUBLISHED';

-- name: RelatedGuideArticles :many
SELECT a.id, a.category_id, a.slug, a.title, a.excerpt, a.cover_key, a.sort_order, a.published_at
FROM guide_articles a
WHERE a.category_id = sqlc.arg(category_id) AND a.id <> sqlc.arg(exclude_id) AND a.status = 'PUBLISHED'
ORDER BY a.sort_order, a.published_at DESC
LIMIT sqlc.arg(max_results);

-- name: SearchGuideArticles :many
-- The tsvector expression must match the guide_articles_search index.
SELECT a.id, a.slug, a.title, a.excerpt, c.slug AS category_slug, c.title AS category_title,
       ts_headline('simple', a.content, q,
         'MaxFragments=2, MinWords=8, MaxWords=24, StartSel=<mark>, StopSel=</mark>')::text AS headline
FROM guide_articles a
JOIN guide_categories c ON c.id = a.category_id,
     websearch_to_tsquery('simple', sqlc.arg(query)) q
WHERE a.status = 'PUBLISHED'
  AND to_tsvector('simple', a.title || ' ' || a.excerpt || ' ' || a.content) @@ q
ORDER BY ts_rank(to_tsvector('simple', a.title || ' ' || a.excerpt || ' ' || a.content), q) DESC,
         a.published_at DESC
LIMIT sqlc.arg(max_results);

-- Admin articles -------------------------------------------------------------

-- name: ListGuideArticles :many
SELECT a.id, a.category_id, a.slug, a.title, a.excerpt, a.cover_key, a.sort_order, a.status,
       a.published_at, a.updated_at
FROM guide_articles a
WHERE (sqlc.narg(category_id)::uuid IS NULL OR a.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(status)::content_status IS NULL OR a.status = sqlc.narg(status))
ORDER BY a.updated_at DESC;

-- name: GuideArticleByID :one
SELECT * FROM guide_articles WHERE id = $1;

-- name: CreateGuideArticle :one
INSERT INTO guide_articles (category_id, author_id, slug, title, excerpt, content, cover_key, sort_order, status, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateGuideArticle :one
UPDATE guide_articles SET
  category_id = $2, slug = $3, title = $4, excerpt = $5, content = $6, cover_key = $7,
  sort_order = $8, status = $9, published_at = $10, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteGuideArticle :execrows
DELETE FROM guide_articles WHERE id = $1;
