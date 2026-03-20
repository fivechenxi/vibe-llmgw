# Architecture

## 目录结构

```
llmgw/
├── cmd/server/main.go               # 程序入口，初始化依赖、注册路由、启动 HTTP Server
├── internal/
│   ├── config/config.go             # 读取 config.yaml，映射为 Config 结构体
│   ├── db/postgres.go               # 创建 pgxpool 连接池
│   ├── domain/types.go              # 全局共享数据类型（User / Model / UserQuota / ChatLog / ChatRequest / ChatResponse）
│   ├── middleware/auth.go           # Gin 中间件：解析并验证 Authorization: Bearer <JWT>，将 userID 写入 Context
│   ├── auth/
│   │   ├── handler.go               # GET /auth/login → 重定向 SSO；GET /auth/callback → 换 token；POST /auth/logout
│   │   ├── jwt.go                   # 使用 HS256 签发 JWT，Payload 包含 sub / email / name / exp
│   │   └── sso.go                   # SSOProvider 接口定义，待实现：WechatWork / LDAP / SAML
│   ├── quota/
│   │   ├── repository.go            # Get(userID, modelID) / Deduct(tokens) / ListByUser
│   │   └── service.go               # Check：余量 ≤ 0 返回 ErrQuotaExceeded；Deduct：扣减已用 token
│   ├── proxy/
│   │   ├── handler.go               # POST /api/chat 主流程（见下方流程描述）
│   │   ├── router.go                # 维护 modelID → Provider 的映射表，NewRouter 时按配置初始化
│   │   └── providers/
│   │       ├── provider.go          # Provider 接口：Complete（同步）/ Stream（SSE）；QuotaDeductor / ChatLogger 窄接口
│   │       ├── openai.go            # 兼容 OpenAI Chat Completions 格式，复用于 DeepSeek / Alibaba Qwen
│   │       └── anthropic.go         # 将 OpenAI messages 格式转换为 Anthropic Messages API 格式
│   ├── chat/
│   │   ├── repository.go            # Save(ChatLog) / ListSessions(userID) / GetSession(userID, sessionID)
│   │   └── handler.go               # GET /api/sessions；GET /api/sessions/:session_id
│   └── model/
│       ├── repository.go            # ListActive：查询 is_active=true 的模型列表
│       └── handler.go               # GET /api/models（过滤有余量的模型）；GET /api/quota（全量 quota 详情）
├── db/migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_models.sql        # 同时预置 8 个初始模型记录
│   ├── 003_create_user_quotas.sql
│   └── 004_create_chat_logs.sql
├── web/
│   └── src/
│       ├── api/client.ts            # 封装所有后端 HTTP 调用，统一携带 Authorization header
│       ├── hooks/useStream.ts       # 消费 SSE 流，逐块累积到 buffer，streaming 状态控制
│       ├── components/
│       │   ├── ChatWindow.tsx       # 消息列表 + 输入框，流式打字机效果
│       │   ├── ModelSelector.tsx    # 下拉选择模型，展示剩余 token
│       │   └── SessionList.tsx      # 历史会话侧边栏，支持新建会话
│       └── pages/
│           ├── Login.tsx            # 点击按钮跳转 /auth/login
│           └── Chat.tsx             # 主页面，组合以上组件
├── config.yaml.example              # 配置模板（SSO / DB / JWT / Provider API Keys）
└── go.mod
```

---

## 请求处理流程

### 登录流程

```
用户点击"企业账号登录"
  → GET /auth/login
  → 服务端根据 cfg.SSO.Provider 构造 SSO 授权 URL，302 重定向
  → 用户在 SSO 完成认证
  → SSO 回调 GET /auth/callback?code=xxx
  → 服务端用 code 换取用户信息（ID / Email / Name）
  → UPSERT users 表
  → 调用 auth.SignToken 签发 JWT
  → 返回 { "token": "..." }，前端存入 localStorage
```

### 对话流程（POST /api/chat）

```
1. middleware.JWTAuth
     解析 Bearer Token → 验证签名和过期时间 → 将 userID 写入 gin.Context

2. proxy.Handler.Chat
     绑定请求 Body → ChatRequest{ model, messages, session_id, stream }

3. quota.Service.Check(userID, modelID)
     查 user_quotas 表 → remaining = quota_tokens - used_tokens
     remaining ≤ 0 → 返回 403 quota exceeded

4. proxy.Router.Get(modelID)
     按 modelID 返回对应 Provider 实例
     未知 model → 返回 400

5a. 非流式（stream=false）
     provider.Complete(ctx, userID, req)
       → 构造 Provider 格式请求体
       → HTTP POST 到 Provider API（携带后端 API Key）
       → 解析响应，转为统一 ChatResponse
     goroutine: quota.Deduct(userID, modelID, totalTokens)
     goroutine: chatRepo.Save(ChatLog{ status=success, ... })
     → 返回 JSON { content, usage }

5b. 流式（stream=true）
     provider.Stream(c, userID, req, quotaSvc, chatRepo)
       → 设置响应头 Content-Type: text/event-stream
       → 转发 Provider SSE 流到客户端（逐 chunk 写入）
       → 流结束后统计 token
       → quota.Deduct + chatRepo.Save
```

### Quota 扣减时序

```
请求进入 → Check（读） → 调用 Provider → 拿到实际 token 用量 → Deduct（写）

说明：Check 和 Deduct 之间不加锁，允许极小概率超额（初版可接受）。
      后续如需精确控制，可在 Deduct 改为 UPDATE ... WHERE used_tokens + $1 <= quota_tokens。
```

---

## 启动步骤

### 前置条件
- Go 1.22+
- Node.js 18+
- PostgreSQL 14+（本地或远程均可）

### 1. 初始化数据库

```bash
psql -U postgres -c "CREATE DATABASE llmgw;"
psql -U postgres -d llmgw -f db/migrations/001_create_users.sql
psql -U postgres -d llmgw -f db/migrations/002_create_models.sql
psql -U postgres -d llmgw -f db/migrations/003_create_user_quotas.sql
psql -U postgres -d llmgw -f db/migrations/004_create_chat_logs.sql
```

### 2. 配置后端

```bash
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入：
#   database.dsn
#   jwt.secret
#   sso.provider 及对应参数
#   providers.openai.api_key / providers.anthropic.api_key / 等
```

### 3. 启动后端

```bash
go mod tidy
go run ./cmd/server
# 默认监听 :8080
```

### 4. 启动前端

```bash
cd web
npm install
npm run dev
# 默认 http://localhost:5173，开发时 vite 代理 /api → localhost:8080
```

### 5. 为用户分配 Quota（手动 SQL，初版无管理界面）

```sql
-- 示例：给用户 user_001 分配 gpt-4o 的 1,000,000 token
INSERT INTO user_quotas (user_id, model_id, quota_tokens, reset_period, reset_date)
VALUES ('user_001', 'gpt-4o', 1000000, 'monthly', '2026-04-01');
```

---

## 待实现项（TODO）

| 模块 | 文件 | 内容 |
|------|------|------|
| SSO | `internal/auth/sso.go` | 实现 WechatWorkProvider / LDAPProvider |
| Proxy | `internal/proxy/providers/openai.go` | Complete 和 Stream 的 HTTP 调用逻辑 |
| Proxy | `internal/proxy/providers/anthropic.go` | messages 格式转换 + API 调用 |
| 前端 | `web/src/pages/Chat.tsx` | 安装 uuid 依赖（`npm i uuid @types/uuid`） |
| 前端 | `web/` | 配置 vite.config.ts 的 `/api` 代理 |
