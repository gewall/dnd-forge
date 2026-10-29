-- +goose Up
-- Ganti kolom tsvector dengan index ekspresi supaya SELECT * tidak membawa
-- tsvector ke kode Go. Ekspresi harus sama persis dengan yang dipakai query.
DROP INDEX guide_articles_search;
ALTER TABLE guide_articles DROP COLUMN search;
CREATE INDEX guide_articles_search ON guide_articles
  USING gin (to_tsvector('simple', title || ' ' || excerpt || ' ' || content));

-- +goose Down
DROP INDEX guide_articles_search;
ALTER TABLE guide_articles ADD COLUMN search tsvector GENERATED ALWAYS AS (
  to_tsvector('simple', title || ' ' || excerpt || ' ' || content)
) STORED;
CREATE INDEX guide_articles_search ON guide_articles USING gin (search);
