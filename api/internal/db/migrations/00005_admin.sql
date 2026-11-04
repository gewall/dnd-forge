-- +goose Up
-- Fase 7: kuota lore per user (null = default global) dan pemakaian token
-- lore untuk statistik admin.
ALTER TABLE users ADD COLUMN daily_lore_quota int CHECK (daily_lore_quota >= 0);
ALTER TABLE users ADD CONSTRAINT users_daily_quota_check CHECK (daily_quota >= 0);

ALTER TABLE character_lores
  ADD COLUMN prompt_tokens     int,
  ADD COLUMN completion_tokens int;

CREATE INDEX users_created ON users (created_at DESC);
CREATE INDEX generated_images_created ON generated_images (created_at DESC);
CREATE INDEX character_lores_created ON character_lores (created_at DESC);

-- +goose Down
DROP INDEX character_lores_created;
DROP INDEX generated_images_created;
DROP INDEX users_created;
ALTER TABLE character_lores DROP COLUMN completion_tokens, DROP COLUMN prompt_tokens;
ALTER TABLE users DROP CONSTRAINT users_daily_quota_check;
ALTER TABLE users DROP COLUMN daily_lore_quota;
