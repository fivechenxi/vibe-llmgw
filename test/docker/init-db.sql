-- LLM Gateway Database Initialization Script
-- Combined migrations for test environment

-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 001: Create users table
CREATE TABLE IF NOT EXISTS users (
  id          VARCHAR(64) PRIMARY KEY,
  email       VARCHAR(255) UNIQUE NOT NULL,
  name        VARCHAR(255),
  created_at  TIMESTAMP DEFAULT NOW()
);

-- 002: Create models table
CREATE TABLE IF NOT EXISTS models (
  id          VARCHAR(64) PRIMARY KEY,
  name        VARCHAR(255),
  provider    VARCHAR(64),
  is_active   BOOLEAN DEFAULT TRUE
);

INSERT INTO models (id, name, provider) VALUES
  ('mock',                      'Mock Provider',              'mock'),
  ('gpt-4o',                    'GPT-4o',                     'openai'),
  ('gpt-4o-mini',               'GPT-4o Mini',                'openai'),
  ('gpt-5',                     'GPT-5',                      'openai'),
  ('claude-3-5-sonnet',         'Claude 3.5 Sonnet',          'anthropic'),
  ('claude-3-haiku',            'Claude 3 Haiku',             'anthropic'),
  ('claude-haiku-4-5',          'Claude Haiku 4.5',           'anthropic'),
  ('claude-3-5-haiku-20241022', 'Claude 3.5 Haiku',           'anthropic'),
  ('deepseek-v3',               'DeepSeek V3',                'deepseek'),
  ('deepseek-r1',               'DeepSeek R1',                'deepseek'),
  ('qwen-max',                  'Qwen Max',                   'alibaba'),
  ('qwen-plus',                 'Qwen Plus',                  'alibaba'),
  ('qwen3-max-2026-01-23',      'Qwen3 Max',                  'alibaba'),
  ('qwen3.5-plus',              'Qwen3.5 Plus',               'alibaba')
ON CONFLICT (id) DO NOTHING;

-- 003: Create user_quotas table
CREATE TABLE IF NOT EXISTS user_quotas (
  id            SERIAL PRIMARY KEY,
  user_id       VARCHAR(64) REFERENCES users(id),
  model_id      VARCHAR(64) REFERENCES models(id),
  quota_tokens  BIGINT NOT NULL,
  used_tokens   BIGINT DEFAULT 0,
  reset_period  VARCHAR(16) DEFAULT 'monthly',
  reset_date    DATE,
  UNIQUE (user_id, model_id)
);

-- 004: Create chat_logs table
CREATE TABLE IF NOT EXISTS chat_logs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           VARCHAR(64) REFERENCES users(id),
  session_id        UUID NOT NULL,
  model_id          VARCHAR(64),
  request_at        TIMESTAMP NOT NULL,
  response_at       TIMESTAMP,
  request_messages  JSONB,
  response_content  TEXT,
  input_tokens      INT DEFAULT 0,
  output_tokens     INT DEFAULT 0,
  status            VARCHAR(32),
  error_message     TEXT
);

CREATE INDEX IF NOT EXISTS idx_chat_logs_user_id    ON chat_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_logs_session_id ON chat_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_logs_request_at ON chat_logs(request_at);

-- 005: Create model_credentials table
CREATE TABLE IF NOT EXISTS model_credentials (
  id          SERIAL PRIMARY KEY,
  model_id    VARCHAR(64) REFERENCES models(id),
  api_key     TEXT NOT NULL,
  label       VARCHAR(255),
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_credentials_model_id ON model_credentials(model_id);

-- Add credential_id to chat_logs
ALTER TABLE chat_logs ADD COLUMN IF NOT EXISTS credential_id INT REFERENCES model_credentials(id);