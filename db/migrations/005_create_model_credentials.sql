CREATE TABLE IF NOT EXISTS model_credentials (
  id          SERIAL PRIMARY KEY,
  model_id    VARCHAR(64) REFERENCES models(id),
  api_key     TEXT NOT NULL,
  label       VARCHAR(255),
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_credentials_model_id ON model_credentials(model_id);

-- Add credential_id to chat_logs to bind each request to the backend account used
ALTER TABLE chat_logs ADD COLUMN IF NOT EXISTS credential_id INT REFERENCES model_credentials(id);
