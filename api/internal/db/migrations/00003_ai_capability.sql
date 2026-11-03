-- +goose Up
-- Satu provider aktif per kemampuan (image untuk portrait, text untuk lore),
-- dan template prompt dibedakan per jenis. DEFAULT hanya untuk mengisi baris
-- lama; setelah itu kode wajib mengisi kolomnya sendiri.
ALTER TABLE ai_providers ADD COLUMN capability text NOT NULL DEFAULT 'image'
  CHECK (capability IN ('image', 'text'));
ALTER TABLE ai_providers ALTER COLUMN capability DROP DEFAULT;
DROP INDEX ai_providers_one_active;
CREATE UNIQUE INDEX ai_providers_one_active ON ai_providers (capability) WHERE is_active;

ALTER TABLE prompt_templates ADD COLUMN kind text NOT NULL DEFAULT 'portrait'
  CHECK (kind IN ('portrait', 'lore'));
ALTER TABLE prompt_templates ALTER COLUMN kind DROP DEFAULT;
DROP INDEX prompt_templates_one_active;
CREATE UNIQUE INDEX prompt_templates_one_active ON prompt_templates (kind) WHERE is_active;
CREATE UNIQUE INDEX prompt_templates_version ON prompt_templates (kind, name, version);

-- +goose Down
DROP INDEX prompt_templates_version;
DROP INDEX prompt_templates_one_active;
UPDATE prompt_templates SET is_active = false WHERE kind <> 'portrait';
ALTER TABLE prompt_templates DROP COLUMN kind;
CREATE UNIQUE INDEX prompt_templates_one_active ON prompt_templates (is_active) WHERE is_active;

DROP INDEX ai_providers_one_active;
UPDATE ai_providers SET is_active = false WHERE capability <> 'image';
ALTER TABLE ai_providers DROP COLUMN capability;
CREATE UNIQUE INDEX ai_providers_one_active ON ai_providers (is_active) WHERE is_active;
