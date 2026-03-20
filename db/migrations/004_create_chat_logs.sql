CREATE EXTENSION IF NOT EXISTS "pgcrypto";

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
