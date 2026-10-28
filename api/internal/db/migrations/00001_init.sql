-- +goose Up
CREATE TYPE user_role AS ENUM ('USER', 'ADMIN');
CREATE TYPE content_status AS ENUM ('DRAFT', 'PUBLISHED');
CREATE TYPE image_status AS ENUM ('PENDING', 'SUCCEEDED', 'FAILED');

-- Identitas ------------------------------------------------------------------

CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         text UNIQUE NOT NULL,        -- selalu lowercase
  password_hash text,                        -- null untuk akun Google saja
  google_sub    text UNIQUE,
  name          text NOT NULL DEFAULT '',
  role          user_role NOT NULL DEFAULT 'USER',
  daily_quota   int,                         -- null = pakai default global
  is_banned     boolean NOT NULL DEFAULT false,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text UNIQUE NOT NULL,           -- SHA-256 hex dari token mentah
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user ON refresh_tokens (user_id);

CREATE TABLE app_settings (
  key        text PRIMARY KEY,
  value      jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO app_settings (key, value) VALUES ('default_daily_quota', '5');

-- Konten panduan -------------------------------------------------------------

CREATE TABLE guide_categories (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        text UNIQUE NOT NULL,
  title       text NOT NULL,
  description text NOT NULL DEFAULT '',
  sort_order  int NOT NULL DEFAULT 0
);

CREATE TABLE guide_articles (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  category_id  uuid NOT NULL REFERENCES guide_categories(id) ON DELETE RESTRICT,
  author_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  slug         text NOT NULL,
  title        text NOT NULL,
  excerpt      text NOT NULL DEFAULT '',
  content      text NOT NULL DEFAULT '',     -- Markdown
  cover_key    text,                         -- objek di storage
  sort_order   int NOT NULL DEFAULT 0,
  status       content_status NOT NULL DEFAULT 'DRAFT',
  published_at timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  search       tsvector GENERATED ALWAYS AS (
                 to_tsvector('simple', title || ' ' || excerpt || ' ' || content)
               ) STORED,
  UNIQUE (category_id, slug)
);
CREATE INDEX guide_articles_search ON guide_articles USING gin (search);

-- Data game ------------------------------------------------------------------

CREATE TABLE races (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug          text UNIQUE NOT NULL,
  name          text NOT NULL,
  description   text NOT NULL DEFAULT '',
  traits        jsonb NOT NULL DEFAULT '[]',
  visual_prompt text NOT NULL DEFAULT '',    -- bahasa Inggris, mis. 'pointed ears, slender build'
  image_key     text,
  sort_order    int NOT NULL DEFAULT 0,
  is_active     boolean NOT NULL DEFAULT true
);

CREATE TABLE subraces (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  race_id       uuid NOT NULL REFERENCES races(id) ON DELETE CASCADE,
  slug          text NOT NULL,
  name          text NOT NULL,
  description   text NOT NULL DEFAULT '',
  traits        jsonb NOT NULL DEFAULT '[]',
  visual_prompt text NOT NULL DEFAULT '',
  is_active     boolean NOT NULL DEFAULT true,
  UNIQUE (race_id, slug)
);

CREATE TABLE classes (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            text UNIQUE NOT NULL,
  name            text NOT NULL,
  hit_die         int NOT NULL,
  primary_ability text NOT NULL,
  description     text NOT NULL DEFAULT '',
  features        jsonb NOT NULL DEFAULT '[]',
  visual_prompt   text NOT NULL DEFAULT '',
  image_key       text,
  sort_order      int NOT NULL DEFAULT 0,
  is_active       boolean NOT NULL DEFAULT true
);

CREATE TABLE backgrounds (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug                text UNIQUE NOT NULL,
  name                text NOT NULL,
  description         text NOT NULL DEFAULT '',
  skill_proficiencies jsonb NOT NULL DEFAULT '[]',
  visual_prompt       text NOT NULL DEFAULT '',
  image_key           text,
  sort_order          int NOT NULL DEFAULT 0,
  is_active           boolean NOT NULL DEFAULT true
);

-- Hasil user + AI ------------------------------------------------------------

CREATE TABLE characters (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name           text NOT NULL,
  gender         text NOT NULL DEFAULT '',
  race_id        uuid NOT NULL REFERENCES races(id),
  subrace_id     uuid REFERENCES subraces(id),
  class_id       uuid NOT NULL REFERENCES classes(id),
  background_id  uuid NOT NULL REFERENCES backgrounds(id),
  appearance     text NOT NULL DEFAULT '',
  ability_scores jsonb,
  is_public      boolean NOT NULL DEFAULT false,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX characters_user ON characters (user_id, created_at DESC);

CREATE TABLE ai_providers (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  type              text NOT NULL,           -- openai | google | stability | replicate
  label             text NOT NULL,
  encrypted_api_key bytea NOT NULL,          -- AES-256-GCM, kunci master di env Go
  key_last4         text NOT NULL,
  default_model     text NOT NULL,
  image_size        text NOT NULL DEFAULT '1024x1024',
  is_active         boolean NOT NULL DEFAULT false,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ai_providers_one_active ON ai_providers (is_active) WHERE is_active;

CREATE TABLE prompt_templates (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name            text NOT NULL,
  template        text NOT NULL,
  negative_prompt text NOT NULL DEFAULT '',
  version         int NOT NULL DEFAULT 1,
  is_active       boolean NOT NULL DEFAULT false,
  created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX prompt_templates_one_active ON prompt_templates (is_active) WHERE is_active;

CREATE TABLE banned_words (
  word       text PRIMARY KEY,               -- selalu lowercase
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE generated_images (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id  uuid REFERENCES characters(id) ON DELETE SET NULL,
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id   uuid NOT NULL REFERENCES ai_providers(id),
  model         text NOT NULL,
  user_prompt   text NOT NULL DEFAULT '',
  final_prompt  text NOT NULL,
  style         text NOT NULL DEFAULT '',
  aspect_ratio  text NOT NULL DEFAULT '1:1',
  storage_key   text,                        -- objek asli di Garage
  thumb_key     text,                        -- thumbnail WebP
  status        image_status NOT NULL DEFAULT 'PENDING',
  error_message text,
  is_hidden     boolean NOT NULL DEFAULT false,
  created_at    timestamptz NOT NULL DEFAULT now(),
  completed_at  timestamptz
);
CREATE INDEX generated_images_user_day ON generated_images (user_id, created_at);
CREATE INDEX generated_images_character ON generated_images (character_id, created_at DESC);

CREATE TABLE audit_logs (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  actor_id   uuid REFERENCES users(id) ON DELETE SET NULL,
  action     text NOT NULL,                  -- create | update | delete | activate | ...
  entity     text NOT NULL,                  -- race | guide_article | ai_provider | ...
  entity_id  text NOT NULL,
  diff       jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_created ON audit_logs (created_at DESC);

-- +goose Down
DROP TABLE audit_logs;
DROP TABLE generated_images;
DROP TABLE banned_words;
DROP TABLE prompt_templates;
DROP TABLE ai_providers;
DROP TABLE characters;
DROP TABLE backgrounds;
DROP TABLE classes;
DROP TABLE subraces;
DROP TABLE races;
DROP TABLE guide_articles;
DROP TABLE guide_categories;
DROP TABLE app_settings;
DROP TABLE refresh_tokens;
DROP TABLE users;
DROP TYPE image_status;
DROP TYPE content_status;
DROP TYPE user_role;
