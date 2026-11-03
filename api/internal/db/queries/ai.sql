-- AI providers -----------------------------------------------------------------

-- name: ListAIProviders :many
SELECT * FROM ai_providers ORDER BY created_at;

-- name: AIProviderByID :one
SELECT * FROM ai_providers WHERE id = $1;

-- name: ActiveAIProvider :one
SELECT * FROM ai_providers WHERE is_active AND capability = $1;

-- name: CreateAIProvider :one
INSERT INTO ai_providers (type, capability, label, encrypted_api_key, key_last4, default_model, image_size)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAIProvider :one
-- A NULL key keeps the stored one.
UPDATE ai_providers SET
  label = $2, default_model = $3, image_size = $4,
  encrypted_api_key = coalesce(sqlc.narg(encrypted_api_key), encrypted_api_key),
  key_last4 = coalesce(sqlc.narg(key_last4), key_last4),
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeactivateAIProviders :exec
-- Deactivates the providers that share the capability of provider $1.
UPDATE ai_providers SET is_active = false, updated_at = now()
WHERE is_active AND capability = (SELECT p.capability FROM ai_providers p WHERE p.id = $1);

-- name: ActivateAIProvider :execrows
UPDATE ai_providers SET is_active = true, updated_at = now() WHERE id = $1;

-- name: DeleteAIProvider :execrows
DELETE FROM ai_providers WHERE id = $1;

-- Prompt templates ---------------------------------------------------------------

-- name: ListPromptTemplates :many
SELECT * FROM prompt_templates ORDER BY kind, name, version DESC;

-- name: PromptTemplateByID :one
SELECT * FROM prompt_templates WHERE id = $1;

-- name: ActivePromptTemplate :one
SELECT * FROM prompt_templates WHERE is_active AND kind = $1;

-- name: CreatePromptTemplate :one
-- Each save of a kind and name is a new immutable version.
INSERT INTO prompt_templates (kind, name, template, negative_prompt, version, created_by)
VALUES ($1, $2, $3, $4,
        (SELECT coalesce(max(t.version), 0) + 1 FROM prompt_templates t WHERE t.kind = $1 AND t.name = $2),
        $5)
RETURNING *;

-- name: DeactivatePromptTemplates :exec
-- Deactivates the templates of the same kind as template $1.
UPDATE prompt_templates SET is_active = false
WHERE is_active AND kind = (SELECT t.kind FROM prompt_templates t WHERE t.id = $1);

-- name: ActivatePromptTemplate :execrows
UPDATE prompt_templates SET is_active = true WHERE id = $1;

-- name: DeletePromptTemplate :execrows
DELETE FROM prompt_templates WHERE id = $1 AND NOT is_active;

-- Banned words -----------------------------------------------------------------

-- name: ListBannedWords :many
SELECT word FROM banned_words ORDER BY word;

-- name: AddBannedWord :exec
INSERT INTO banned_words (word) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: DeleteBannedWord :execrows
DELETE FROM banned_words WHERE word = $1;

-- Settings ---------------------------------------------------------------------

-- name: SettingValue :one
SELECT value FROM app_settings WHERE key = $1;

-- name: SetSettingValue :exec
INSERT INTO app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now();
