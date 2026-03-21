# 黑盒测试设计文档

## 1. 测试范围

基于 `api.md` 和 `Design.md` 文档，对 LLM Gateway 的所有 API 端点进行黑盒测试，验证功能正确性、错误处理和边界条件。

**测试类型：**
- 功能测试（正向用例）
- 异常测试（负向用例）
- 边界测试
- 安全测试（认证/授权）

**排除范围：**
- OpenAI/Anthropic Provider 真实 API 调用（需真实 API Key）
- 数据库集成测试（需 PostgreSQL 环境）

---

## 2. 测试用例设计

### 2.1 认证模块 (Auth)

#### 2.1.1 GET /auth/login

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| AUTH-001 | 登录重定向成功 | 无 | GET /auth/login | 302 Found, Location header 指向 SSO URL |
| AUTH-002 | 登录重定向包含必要参数 | 无 | GET /auth/login | Location 包含 client_id, redirect_uri, response_type 等参数 |

#### 2.1.2 GET /auth/callback

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| AUTH-003 | 回调成功返回Token | SSO 返回有效 code | GET /auth/callback?code=valid&state=xyz | 200 OK, body 包含 token 字段 |
| AUTH-004 | 回调缺少code参数 | 无 | GET /auth/callback?state=xyz | 400 Bad Request 或特定错误响应 |
| AUTH-005 | 回调无效code | 无 | GET /auth/callback?code=invalid | 返回错误（具体行为取决于 SSO 实现） |

#### 2.1.3 POST /auth/logout

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| AUTH-006 | 登出成功 | 无 | POST /auth/logout | 200 OK, body 包含 "logged out" 消息 |

---

### 2.2 JWT 中间件

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| JWT-001 | 有效Token通过认证 | 存在有效用户 | Authorization: Bearer {valid_token} | 请求到达下游 handler |
| JWT-002 | 缺少Authorization头 | 无 | 无 Authorization header | 401 Unauthorized |
| JWT-003 | Token格式错误 | 无 | Authorization: InvalidFormat | 401 Unauthorized |
| JWT-004 | Token前缀错误 | 无 | Authorization: Basic xxx | 401 Unauthorized |
| JWT-005 | 无效Token签名 | 无 | Authorization: Bearer {invalid_sig} | 401 Unauthorized |
| JWT-006 | 过期Token | 无 | Authorization: Bearer {expired_token} | 401 Unauthorized |
| JWT-007 | 错误密钥签名 | 无 | Authorization: Bearer {wrong_secret} | 401 Unauthorized |

---

### 2.3 模型与 Quota 模块

#### 2.3.1 GET /api/models

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| MODEL-001 | 获取可用模型列表 | 用户有多个模型 quota | Authorization: Bearer {token} | 200 OK, models 数组包含 remaining_tokens > 0 的模型 |
| MODEL-002 | 空模型列表 | 用户无任何 quota | Authorization: Bearer {token} | 200 OK, models 数组为空 |
| MODEL-003 | 过滤耗尽模型 | 用户有耗尽的模型 | Authorization: Bearer {token} | 返回列表不包含 remaining_tokens <= 0 的模型 |
| MODEL-004 | 未认证访问 | 无 | 无 Authorization header | 401 Unauthorized |
| MODEL-005 | 服务端错误 | Mock repo 返回错误 | Authorization: Bearer {token} | 500 Internal Server Error |

#### 2.3.2 GET /api/quota

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| QUOTA-001 | 获取Quota详情 | 用户有多个模型 quota | Authorization: Bearer {token} | 200 OK, quotas 数组包含完整信息 |
| QUOTA-002 | 包含耗尽模型 | 用户有耗尽的模型 | Authorization: Bearer {token} | 返回列表包含 quota_tokens == used_tokens 的记录 |
| QUOTA-003 | 空Quota列表 | 用户无任何 quota | Authorization: Bearer {token} | 200 OK, quotas 数组为空 |
| QUOTA-004 | 未认证访问 | 无 | 无 Authorization header | 401 Unauthorized |

---

### 2.4 对话模块

#### 2.4.1 POST /api/chat (非流式)

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| CHAT-001 | 完整对话成功 | 用户有 quota, 模型有效 | model, messages, stream=false | 200 OK, content 非空, usage 包含 token 数 |
| CHAT-002 | 多轮对话 | 用户有 quota | 多条 messages | 200 OK, 响应与最后一条用户消息相关 |
| CHAT-003 | Session-Sticky路由 | 相同 session_id | 相同 session_id 多次请求 | 每次使用相同的 credential_id |
| CHAT-004 | Round-Robin路由 | 无 session_id | 无 session_id 多次请求 | 分散到不同 credential（Mock 环境验证行为） |
| CHAT-005 | 缺少model字段 | 无 | 请求体无 model | 400 Bad Request |
| CHAT-006 | 缺少messages字段 | 无 | 请求体无 messages | 400 Bad Request |
| CHAT-007 | 不支持的模型 | 无 | model="unknown-model" | 400 Bad Request |
| CHAT-008 | Quota耗尽 | 用户 quota=0 | 有效请求 | 403 Forbidden |
| CHAT-009 | 无可用Credential | 无 active credential | 有效请求 | 503 Service Unavailable |
| CHAT-010 | Provider错误 | Mock 返回错误 | 有效请求 | 502 Bad Gateway |
| CHAT-011 | 未认证访问 | 无 | 无 Authorization header | 401 Unauthorized |

#### 2.4.2 POST /api/chat (流式)

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| CHAT-012 | 流式响应成功 | 用户有 quota | stream=true | 200 OK, Content-Type: text/event-stream |
| CHAT-013 | 流式数据格式 | 用户有 quota | stream=true | body 包含 "data:" 前缀的分块数据 |
| CHAT-014 | 流式结束标记 | 用户有 quota | stream=true | body 包含 "data: [DONE]" |
| CHAT-015 | 流式Quota扣减 | 用户有 quota | stream=true | 流式完成后 quota 被扣减 |

---

### 2.5 历史记录模块

#### 2.5.1 GET /api/sessions

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| SESS-001 | 获取会话列表 | 用户有多个 session | Authorization: Bearer {token} | 200 OK, sessions 数组按时间倒序 |
| SESS-002 | 空会话列表 | 用户无 session | Authorization: Bearer {token} | 200 OK, sessions 数组为空 |
| SESS-003 | 未认证访问 | 无 | 无 Authorization header | 401 Unauthorized |

#### 2.5.2 GET /api/sessions/{session_id}

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| SESS-004 | 获取会话详情 | session 存在 | 有效 session_id | 200 OK, messages 数组按时间升序 |
| SESS-005 | 空会话详情 | session 不存在 | 不存在的 session_id | 200 OK, messages 数组为空 |
| SESS-006 | 无效UUID格式 | 无 | session_id="invalid-uuid" | 400 Bad Request |
| SESS-007 | 未认证访问 | 无 | 无 Authorization header | 401 Unauthorized |

---

### 2.6 路由模块

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| ROUTE-001 | 已知模型路由 | 无 | model="gpt-4o" | 返回 OpenAI Provider |
| ROUTE-002 | 已知模型路由 | 无 | model="claude-3-5-sonnet" | 返回 Anthropic Provider |
| ROUTE-003 | 已知模型路由 | 无 | model="deepseek-v3" | 返回 DeepSeek Provider |
| ROUTE-004 | 已知模型路由 | 无 | model="qwen-max" | 返回 Alibaba Provider |
| ROUTE-005 | 未知模型 | 无 | model="unknown-model" | 返回错误 |
| ROUTE-006 | 注册自定义模型 | 调用 Register | Register("custom", provider) | Get("custom") 返回注册的 provider |
| ROUTE-007 | 覆盖已有模型 | 已注册的模型 | Register("gpt-4o", custom) | Get 返回新 provider |

---

### 2.7 Credential 选择模块

| 用例ID | 用例名称 | 前置条件 | 输入 | 预期结果 |
|--------|----------|----------|------|----------|
| CRED-001 | Session-Sticky一致性 | 多个 credential | 相同 session_id 多次请求 | 返回相同 credential_id |
| CRED-002 | 不同Session可能不同 | 多个 credential | 不同 session_id | 可能返回不同 credential_id |
| CRED-003 | RoundRobin循环 | 多个 credential | 无 session_id 连续请求 | 循环分配 credential |
| CRED-004 | 单Credential | 仅一个 credential | 任意请求 | 始终返回该 credential |
| CRED-005 | 无Credential | 无 active credential | 任意请求 | 返回错误 |
| CRED-006 | 并发安全 | 多个 credential | 并发请求 | 所有请求返回有效 credential，无竞态 |

---

## 3. 测试矩阵

### 3.1 认证 + API 组合矩阵

| API 端点 | 无Token | 过期Token | 无效Token | 有效Token |
|----------|---------|-----------|-----------|-----------|
| GET /auth/login | ✓ | ✓ | ✓ | ✓ |
| GET /auth/callback | ✓ | ✓ | ✓ | ✓ |
| POST /auth/logout | ✓ | ✓ | ✓ | ✓ |
| GET /api/models | 401 | 401 | 401 | 200/500 |
| GET /api/quota | 401 | 401 | 401 | 200/500 |
| POST /api/chat | 401 | 401 | 401 | 200/400/403/502/503 |
| GET /api/sessions | 401 | 401 | 401 | 200/500 |
| GET /api/sessions/{id} | 401 | 401 | 401 | 200/400/500 |

### 3.2 Chat 请求参数矩阵

| model | messages | session_id | stream | 预期结果 |
|-------|----------|------------|--------|----------|
| 有效 | 有效 | 有效 UUID | false | 200 OK |
| 有效 | 有效 | 有效 UUID | true | 200 SSE |
| 有效 | 有效 | 无 | false | 200 OK |
| 有效 | 有效 | 无 | true | 200 SSE |
| 无 | 有效 | - | - | 400 |
| 有效 | 无 | - | - | 400 |
| 无效 | 有效 | - | - | 400 |
| 有效 | 有效 | - | - | 403 (quota=0) |

---

## 4. 测试环境要求

### 4.1 Mock 环境（无需外部依赖）

- 使用 Mock Provider 替代真实 LLM API
- 使用内存数据结构替代数据库
- 可在 CI 环境直接运行

### 4.2 集成环境（需外部依赖）

- PostgreSQL 数据库（设置 `TEST_DATABASE_URL`）
- 真实 API Key（设置 `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`）

---

## 5. 测试统计

| 模块 | 正向用例 | 负向用例 | 边界用例 | 总计 |
|------|----------|----------|----------|------|
| Auth | 4 | 2 | 0 | 6 |
| JWT Middleware | 1 | 6 | 0 | 7 |
| Models | 3 | 2 | 0 | 5 |
| Quota | 3 | 1 | 0 | 4 |
| Chat (非流式) | 4 | 7 | 0 | 11 |
| Chat (流式) | 4 | 0 | 0 | 4 |
| Sessions | 4 | 1 | 0 | 5 |
| Router | 7 | 0 | 0 | 7 |
| Credential | 5 | 1 | 0 | 6 |
| **总计** | **35** | **20** | **0** | **55** |