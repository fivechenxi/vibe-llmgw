# LLM Gateway 系统设计文档

## 1. 项目概述

LLM Gateway 是一个企业内部 AI 聊天平台，通过统一的后端账号中转多种 LLM 模型的 API 请求，为企业员工提供安全、可管控的 AI 对话服务。

**核心目标：**
- 企业 SSO 统一认证接入
- 支持国内外多种 LLM 模型选择
- 按用户分配模型使用 quota
- 完整的请求/响应日志审计
- 后端统一管理 API 密钥，用户无感知

---

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                      用户浏览器                           │
│              Web Chatbot UI (React/Vue)                  │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS
┌──────────────────────▼──────────────────────────────────┐
│                   API Gateway Layer                      │
│              (认证校验 / 路由 / 日志)                      │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │  Auth 服务   │  │ Quota 服务  │  │   Log 服务       │ │
│  │  (SSO/JWT)  │  │ (用量管理)  │  │  (请求/响应记录) │ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
│                                                         │
│  ┌─────────────────────────────────────────────────────┐│
│  │              LLM Proxy 服务 (参考 LiteLLM)           ││
│  │   统一格式转换 → 路由到各 LLM Provider API            ││
│  └─────────────────────────────────────────────────────┘│
└──────────────────────┬──────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
   ┌────▼────┐   ┌─────▼────┐  ┌────▼──────┐
   │ OpenAI  │   │  Claude  │  │  国内模型  │
   │  GPT-4  │   │Anthropic │  │(通义/文心/ │
   └─────────┘   └──────────┘  │ DeepSeek) │
                               └───────────┘
```

---

## 3. 模块设计

### 3.1 认证模块（Auth Service）

**支持的 SSO 方式：**
- 企业微信 OAuth2.0
- LDAP/AD 企业统一账号
- SAML 2.0（对接第三方 IdP）

**流程：**
1. 用户访问系统，重定向至 SSO 登录页
2. SSO 认证成功后，回调 Gateway 换取 JWT Token
3. 后续所有请求携带 JWT，Gateway 验证并提取用户身份

**JWT Payload 示例：**
```json
{
  "sub": "user_id_001",
  "email": "user@company.com",
  "name": "张三",
  "exp": 1700000000
}
```

### 3.2 Quota 管理模块

每个用户按模型单独分配 quota，管理员通过后台配置。

**数据结构：**
```
user_quota:
  user_id: string
  model_id: string          // 如 "gpt-4o", "claude-3-5-sonnet", "deepseek-v3"
  quota_tokens: int         // 分配的 token 总量（或按请求次数）
  used_tokens: int          // 已使用量
  reset_period: enum        // monthly / never
  reset_date: date          // 下次重置日期
```

**Quota 检查流程：**
1. 请求到达时查询用户对应模型的剩余 quota
2. 若 quota 不足，返回 `403` + 提示信息
3. 请求成功后异步扣减实际消耗的 token 数

> 注：当前版本暂不实现实时限速（rate limiting），仅做 quota 总量控制。

### 3.3 LLM Proxy 模块

参考 [LiteLLM](https://github.com/BerriAI/litellm) 设计思路，将各 LLM Provider 的 API 格式统一为 OpenAI Chat Completions 格式。

**支持模型列表（初版）：**

| 类别 | 模型 | Provider |
|------|------|----------|
| 海外 | gpt-4o, gpt-4o-mini | OpenAI |
| 海外 | claude-3-5-sonnet, claude-3-haiku | Anthropic |
| 国内 | qwen-max, qwen-plus | 阿里通义 |
| 国内 | ERNIE-4.0 | 百度文心 |
| 国内 | deepseek-v3, deepseek-r1 | DeepSeek |

**Proxy 处理流程：**
```
用户请求 (统一格式)
  → 验证 JWT
  → 检查 Quota
  → 记录请求日志
  → 转换为目标 Provider 格式
  → 调用 Provider API（使用后端统一 API Key）
  → 转换响应为统一格式
  → 记录响应日志 + 扣减 token
  → 返回给用户
```

### 3.4 日志模块

记录每次对话的完整请求与响应，用于审计、成本核算和用量分析。

**日志字段：**
```
chat_log:
  id: uuid
  user_id: string
  session_id: string
  model_id: string
  request_at: timestamp
  response_at: timestamp
  request_messages: json       // 完整的 messages 数组
  response_content: text       // 模型回复内容
  input_tokens: int
  output_tokens: int
  total_tokens: int
  status: enum                 // success / quota_exceeded / error
  error_message: string?
```

**存储方案：** 写入关系型数据库（PostgreSQL），同时可异步同步至日志系统（如 Elasticsearch）供检索分析。

### 3.5 前端 Chatbot UI

**功能：**
- SSO 登录/登出
- 模型选择下拉菜单（仅展示用户有 quota 的模型）
- 文本对话界面（当前版本仅支持纯文字）
- 当前模型 quota 余量展示
- 历史会话列表（按 session 分组）

**技术选型：** React + Tailwind CSS，流式响应（SSE/streaming）支持打字机效果。

---

## 4. 数据库设计

```sql
-- 用户表（从 SSO 同步）
CREATE TABLE users (
  id          VARCHAR(64) PRIMARY KEY,
  email       VARCHAR(255) UNIQUE NOT NULL,
  name        VARCHAR(255),
  created_at  TIMESTAMP DEFAULT NOW()
);

-- 模型配置表（管理员维护）
CREATE TABLE models (
  id          VARCHAR(64) PRIMARY KEY,   -- e.g. "gpt-4o"
  name        VARCHAR(255),
  provider    VARCHAR(64),               -- openai / anthropic / alibaba ...
  is_active   BOOLEAN DEFAULT TRUE
);

-- 用户 Quota 表
CREATE TABLE user_quotas (
  id            SERIAL PRIMARY KEY,
  user_id       VARCHAR(64) REFERENCES users(id),
  model_id      VARCHAR(64) REFERENCES models(id),
  quota_tokens  BIGINT NOT NULL,
  used_tokens   BIGINT DEFAULT 0,
  reset_period  VARCHAR(16) DEFAULT 'monthly',
  reset_date    DATE,
  UNIQUE (user_id, model_id)
);

-- 对话日志表
CREATE TABLE chat_logs (
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

CREATE INDEX idx_chat_logs_user_id ON chat_logs(user_id);
CREATE INDEX idx_chat_logs_session_id ON chat_logs(session_id);
CREATE INDEX idx_chat_logs_request_at ON chat_logs(request_at);
```

---

## 5. API 设计

### 认证
```
GET  /auth/login          重定向至 SSO
GET  /auth/callback       SSO 回调，颁发 JWT
POST /auth/logout         登出
```

### 模型与 Quota
```
GET  /api/models          获取当前用户可用模型列表（含 quota 余量）
GET  /api/quota           获取当前用户所有模型的 quota 详情
```

### 对话
```
POST /api/chat
  Header: Authorization: Bearer <jwt>
  Body:
    {
      "model": "gpt-4o",
      "messages": [
        {"role": "user", "content": "你好"}
      ],
      "session_id": "uuid",   // 可选，用于关联历史
      "stream": true          // 是否流式返回
    }
  Response (stream=true): SSE text/event-stream
  Response (stream=false): { "content": "...", "usage": {...} }
```

### 历史记录
```
GET  /api/sessions                   获取会话列表
GET  /api/sessions/:session_id       获取某会话的完整消息记录
```

---

## 6. 技术选型

| 层次 | 技术 |
|------|------|
| 层次 | 技术 |
|------|------|
| 后端框架 | Go (Gin / Chi) |
| 数据库 | PostgreSQL |
| LLM 代理 | 自实现 Proxy（参考 LiteLLM 设计），或直接集成 LiteLLM |
| 前端 | React + Tailwind CSS |
| 日志收集 | 结构化日志写入 PostgreSQL（zap 输出） |

---

## 7. 安全考虑

- 所有 LLM Provider API Key 存储在后端环境变量或 Secret 管理服务，前端不可见
- JWT Token 设置合理过期时间（如 8 小时）
- 日志中敏感内容（如 PII）按需脱敏
- API 请求统一走 HTTPS
- 管理后台操作需要独立的管理员角色权限

---

## 8. 后续扩展方向（暂不实现）

- 单用户实时限速（Rate Limiting，如每分钟最多 N 次请求）
- 多模态支持（图片、文件上传）
- 同时向多个模型发送并行对话（参考 ChatALL）
- 团队/部门维度的 quota 管理
- 使用成本报表与管理员仪表盘