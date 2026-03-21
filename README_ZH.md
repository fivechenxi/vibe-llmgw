# LLM Gateway

[English](./README.md) | 中文

企业内部 AI 聊天平台。通过统一后端账号中转多家 LLM Provider 的 API，为员工提供可管控的 AI 对话服务。

## 功能

- 企业 SSO 登录（企业微信 / LDAP / SAML）
- 支持国内外多种模型：GPT-4o、Claude、DeepSeek、通义千问等
- 按用户 × 模型维度分配 token quota
- 完整的请求 / 响应日志，用于审计和成本核算
- 后端统一管理 API Key，用户不可见
- 流式响应（SSE），打字机效果

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + Gin |
| 数据库 | PostgreSQL |
| 前端 | React + Tailwind CSS |

## 快速开始

### 方式一：Docker Compose（推荐用于测试验证）

最快上手方式，无需手动配置数据库。

```bash
# 启动测试环境（PostgreSQL + LLM Gateway）
./test/docker/setup-test-env.sh up

# 运行 API 演示脚本（使用 mock 模型）
./test/scripts/demo-chat.sh

# 停止环境
./test/docker/setup-test-env.sh down
```

启动后服务：
- **PostgreSQL** 端口 5433
- **LLM Gateway** 端口 8080

测试用户：`alice`（有 quota）、`bob`（已耗尽）、`charlie`（无 quota）

### 方式二：手动搭建

**前置条件：** Go 1.22+、Node.js 18+、PostgreSQL 14+

```bash
# 1. 建库 & 建表
psql -U postgres -c "CREATE DATABASE llmgw;"
psql -U postgres -d llmgw -f db/migrations/001_create_users.sql
psql -U postgres -d llmgw -f db/migrations/002_create_models.sql
psql -U postgres -d llmgw -f db/migrations/003_create_user_quotas.sql
psql -U postgres -d llmgw -f db/migrations/004_create_chat_logs.sql

# 2. 配置
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入 DB DSN、JWT Secret、SSO 参数、各 Provider API Key

# 3. 启动后端
go mod tidy
go run ./cmd/server        # 监听 :8080

# 4. 启动前端
cd web && npm install && npm run dev   # 访问 http://localhost:5173
```

## 文档

- [docs/Architecture.md](./docs/Architecture.md) — 目录结构、请求处理流程
- [docs/api.md](./docs/api.md) — REST API 完整定义
- [docs/Design.md](./docs/Design.md) — 需求与系统设计
- [docs/TESTING.md](./docs/TESTING.md) — 测试方法与运行命令
- [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) — Docker 部署指南
