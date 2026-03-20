# API Reference

Base URL: `http://localhost:8080`

认证接口无需 Token，其余所有 `/api/*` 接口均需在 Header 中携带：
```
Authorization: Bearer <JWT>
```

---

## 认证

### 登录

```
GET /auth/login
```

重定向至 SSO 登录页，无需请求体。

**Response:** `302 Found` → SSO 授权 URL

---

### SSO 回调

```
GET /auth/callback?code={code}&state={state}
```

SSO 认证完成后由 SSO Provider 回调，服务端用 code 换取用户信息并签发 JWT。

**Response `200`**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

### 登出

```
POST /auth/logout
```

无状态登出，客户端丢弃本地 Token 即可。

**Response `200`**
```json
{
  "message": "logged out"
}
```

---

## 模型与 Quota

### 获取可用模型列表

```
GET /api/models
```

返回当前用户有剩余 quota 的模型列表。

**Response `200`**
```json
{
  "models": [
    { "model_id": "gpt-4o",            "remaining_tokens": 980000 },
    { "model_id": "claude-3-5-sonnet", "remaining_tokens": 500000 },
    { "model_id": "deepseek-v3",       "remaining_tokens": 1000000 }
  ]
}
```

---

### 获取 Quota 详情

```
GET /api/quota
```

返回当前用户所有模型的完整 quota 记录（含已用量、重置周期）。

**Response `200`**
```json
{
  "quotas": [
    {
      "id": 1,
      "user_id": "user_001",
      "model_id": "gpt-4o",
      "quota_tokens": 1000000,
      "used_tokens": 20000,
      "reset_period": "monthly",
      "reset_date": "2026-04-01T00:00:00Z"
    }
  ]
}
```

---

## 对话

### 发送消息

```
POST /api/chat
```

**Request Body**
```json
{
  "model": "gpt-4o",
  "messages": [
    { "role": "user",      "content": "你好" },
    { "role": "assistant", "content": "你好！有什么可以帮你的？" },
    { "role": "user",      "content": "介绍一下你自己" }
  ],
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "stream": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型 ID，须在可用模型列表中 |
| messages | array | 是 | 完整的对话上下文，role 为 `user` 或 `assistant` |
| session_id | string(uuid) | 否 | 会话 ID，用于关联历史记录；不传则本次不归档 |
| stream | bool | 否 | 默认 false；true 时以 SSE 流式返回 |

**Response `200` (stream=false)**
```json
{
  "content": "我是一个 AI 助手...",
  "usage": {
    "input_tokens": 42,
    "output_tokens": 128,
    "total_tokens": 170
  }
}
```

**Response `200` (stream=true)**

`Content-Type: text/event-stream`，逐块返回文本片段：
```
data: 我是
data: 一个
data: AI 助手...
data: [DONE]
```

**错误响应**

| HTTP 状态码 | 场景 |
|------------|------|
| `400` | model 字段缺失，或 model 不在支持列表中 |
| `401` | JWT 缺失或已过期 |
| `403` | 该模型 quota 已耗尽 |
| `502` | 调用上游 Provider API 失败 |

---

## 历史记录

### 获取会话列表

```
GET /api/sessions
```

返回当前用户的所有历史会话 ID，按时间倒序。

**Response `200`**
```json
{
  "sessions": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660f9500-f39c-41d4-b827-557766551111"
  ]
}
```

---

### 获取会话详情

```
GET /api/sessions/{session_id}
```

返回某会话下所有消息记录，按请求时间升序。

**Path Params**

| 参数 | 类型 | 说明 |
|------|------|------|
| session_id | string(uuid) | 会话 ID |

**Response `200`**
```json
{
  "messages": [
    {
      "id": "a1b2c3d4-...",
      "user_id": "user_001",
      "session_id": "550e8400-...",
      "model_id": "gpt-4o",
      "request_at": "2026-03-20T10:00:00Z",
      "response_at": "2026-03-20T10:00:02Z",
      "request_messages": [
        { "role": "user", "content": "你好" }
      ],
      "response_content": "你好！有什么可以帮你的？",
      "input_tokens": 5,
      "output_tokens": 12,
      "status": "success",
      "error_message": ""
    }
  ]
}
```

**错误响应**

| HTTP 状态码 | 场景 |
|------------|------|
| `400` | session_id 不是合法 UUID |
| `401` | JWT 缺失或已过期 |
