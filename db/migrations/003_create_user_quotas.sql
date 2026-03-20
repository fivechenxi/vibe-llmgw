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