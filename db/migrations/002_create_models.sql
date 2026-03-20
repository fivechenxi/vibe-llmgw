CREATE TABLE IF NOT EXISTS models (
  id          VARCHAR(64) PRIMARY KEY,
  name        VARCHAR(255),
  provider    VARCHAR(64),
  is_active   BOOLEAN DEFAULT TRUE
);

INSERT INTO models (id, name, provider) VALUES
  ('gpt-4o',            'GPT-4o',             'openai'),
  ('gpt-4o-mini',       'GPT-4o Mini',         'openai'),
  ('claude-3-5-sonnet', 'Claude 3.5 Sonnet',   'anthropic'),
  ('claude-3-haiku',    'Claude 3 Haiku',       'anthropic'),
  ('deepseek-v3',       'DeepSeek V3',          'deepseek'),
  ('deepseek-r1',       'DeepSeek R1',          'deepseek'),
  ('qwen-max',          'Qwen Max',             'alibaba'),
  ('qwen-plus',         'Qwen Plus',            'alibaba')
ON CONFLICT (id) DO NOTHING;