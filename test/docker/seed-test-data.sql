-- LLM Gateway Test Data Seed Script
-- This script inserts test data for black-box testing

-- Insert test users
INSERT INTO users (id, email, name) VALUES
  ('alice', 'alice@test.com', 'Alice Test'),
  ('bob', 'bob@test.com', 'Bob Test'),
  ('charlie', 'charlie@test.com', 'Charlie NoQuota')
ON CONFLICT (id) DO NOTHING;

-- Insert user quotas
-- Alice has quota for mock and gpt-4o
INSERT INTO user_quotas (user_id, model_id, quota_tokens, used_tokens) VALUES
  ('alice', 'mock', 1000000, 0),
  ('alice', 'gpt-4o', 500000, 10000)
ON CONFLICT (user_id, model_id) DO NOTHING;

-- Bob has exhausted quota for mock
INSERT INTO user_quotas (user_id, model_id, quota_tokens, used_tokens) VALUES
  ('bob', 'mock', 100000, 100000)
ON CONFLICT (user_id, model_id) DO NOTHING;

-- Charlie has no quota (test empty quota scenario)

-- Insert mock credentials for testing
-- Multiple credentials for mock model to test session-sticky and round-robin
INSERT INTO model_credentials (model_id, api_key, label, is_active) VALUES
  ('mock', 'mock-api-key-1', 'mock-credential-1', true),
  ('mock', 'mock-api-key-2', 'mock-credential-2', true),
  ('mock', 'mock-api-key-3', 'mock-credential-3', true)
ON CONFLICT DO NOTHING;

-- Single credential for gpt-4o (for provider tests, will use mock in test env)
INSERT INTO model_credentials (model_id, api_key, label, is_active) VALUES
  ('gpt-4o', 'test-openai-key', 'openai-test', true)
ON CONFLICT DO NOTHING;

-- Insert some chat logs for session history tests
INSERT INTO chat_logs (id, user_id, session_id, model_id, request_at, response_at, request_messages, response_content, input_tokens, output_tokens, status, credential_id) VALUES
  (gen_random_uuid(), 'alice', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'mock', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour' + INTERVAL '2 seconds',
   '[{"role":"user","content":"Hello"}]'::jsonb, 'Hi there! How can I help you?', 5, 10, 'success', 1),
  (gen_random_uuid(), 'alice', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'mock', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes' + INTERVAL '3 seconds',
   '[{"role":"user","content":"Hello"},{"role":"assistant","content":"Hi there!"},{"role":"user","content":"Tell me a joke"}]'::jsonb,
   'Why did the chicken cross the road? To get to the other side!', 15, 20, 'success', 1),
  (gen_random_uuid(), 'alice', 'b1ffcd00-0d1c-5fe9-cc7e-7cca0e491b22', 'mock', NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '10 minutes' + INTERVAL '1 second',
   '[{"role":"user","content":"Quick test"}]'::jsonb, 'Test response', 3, 5, 'success', 2)
ON CONFLICT DO NOTHING;

-- Grant permissions (for PostgreSQL in Docker)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO llmgw;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO llmgw;