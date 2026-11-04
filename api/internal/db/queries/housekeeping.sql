-- name: ReferencedObjectKeys :many
-- Returns the given storage keys that a row still points to. Guide articles
-- may embed content images in their Markdown, so their text is searched too.
SELECT k::text AS key FROM unnest(sqlc.arg(keys)::text[]) AS k
WHERE EXISTS (SELECT 1 FROM generated_images g WHERE g.storage_key = k OR g.thumb_key = k)
   OR EXISTS (SELECT 1 FROM guide_articles a WHERE a.cover_key = k OR strpos(a.content, k) > 0)
   OR EXISTS (SELECT 1 FROM races r WHERE r.image_key = k)
   OR EXISTS (SELECT 1 FROM classes c WHERE c.image_key = k)
   OR EXISTS (SELECT 1 FROM backgrounds b WHERE b.image_key = k);
