# LLM Gateway

English | [中文](./README_ZH.md)

An internal enterprise AI chat platform that proxies multiple LLM provider APIs through a unified backend account, giving employees controlled access to AI models.

## Features

- Enterprise SSO login (WeCom / LDAP / SAML)
- Multiple models: GPT-4o, Claude, DeepSeek, Qwen, and more
- Per-user, per-model token quota management
- Full request / response audit logging
- API keys managed server-side — never exposed to users
- Streaming responses (SSE) with typewriter effect

## Tech Stack

| Layer    | Technology          |
|----------|---------------------|
| Backend  | Go + Gin            |
| Database | PostgreSQL          |
| Frontend | React + Tailwind CSS |

## Quick Start

**Prerequisites:** Go 1.22+, Node.js 18+, PostgreSQL 14+

```bash
# 1. Create database and run migrations
psql -U postgres -c "CREATE DATABASE llmgw;"
psql -U postgres -d llmgw -f db/migrations/001_create_users.sql
psql -U postgres -d llmgw -f db/migrations/002_create_models.sql
psql -U postgres -d llmgw -f db/migrations/003_create_user_quotas.sql
psql -U postgres -d llmgw -f db/migrations/004_create_chat_logs.sql

# 2. Configure
cp config.yaml.example config.yaml
# Edit config.yaml: DB DSN, JWT secret, SSO settings, provider API keys

# 3. Start backend
go mod tidy
go run ./cmd/server        # listens on :8080

# 4. Start frontend
cd web && npm install && npm run dev   # http://localhost:5173
```

## Documentation

- [docs/Architecture.md](./docs/Architecture.md) — Project structure and request flow
- [docs/api.md](./docs/api.md) — REST API reference
- [docs/Design.md](./docs/Design.md) — Requirements and system design (Chinese)
- [docs/TESTING.md](./docs/TESTING.md) — How to run tests
