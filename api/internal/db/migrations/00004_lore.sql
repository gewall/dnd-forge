-- +goose Up
-- Lore karakter: setiap generate disimpan sebagai draf; user memilih satu
-- untuk disalin ke characters.backstory (yang juga bisa diedit manual).
ALTER TABLE characters ADD COLUMN backstory text NOT NULL DEFAULT '';

CREATE TABLE character_lores (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id  uuid REFERENCES characters(id) ON DELETE SET NULL, -- tetap dihitung kuota
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id   uuid NOT NULL REFERENCES ai_providers(id),
  model         text NOT NULL,
  user_prompt   text NOT NULL DEFAULT '',
  final_prompt  text NOT NULL,
  tone          text NOT NULL,
  length        text NOT NULL CHECK (length IN ('short', 'medium', 'long')),
  language      text NOT NULL CHECK (language IN ('id', 'en')),
  status        image_status NOT NULL DEFAULT 'PENDING',
  content       text,
  error_message text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  completed_at  timestamptz
);
CREATE INDEX character_lores_user_day ON character_lores (user_id, created_at);
CREATE INDEX character_lores_character ON character_lores (character_id, created_at DESC);

INSERT INTO app_settings (key, value) VALUES ('default_daily_lore_quota', '10')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM app_settings WHERE key = 'default_daily_lore_quota';
DROP TABLE character_lores;
DELETE FROM prompt_templates WHERE kind = 'lore';
ALTER TABLE characters DROP COLUMN backstory;
