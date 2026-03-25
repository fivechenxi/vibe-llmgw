# 测试说明

## 前置条件

- Go 1.24+（确保 `go` 在 `PATH` 中可用）
- 如需代理访问，配置 `HTTP_PROXY` 环境变量
- Provider 集成测试需要对应的 API Key 环境变量；未设置时自动 skip
- Repository 集成测试需要设置 `TEST_DATABASE_URL`；未设置时自动 skip

---

## 测试覆盖率汇总

| 包 | 覆盖率 | 状态 |
|---|--------|------|
| internal/auth | 100.0% | ✅ PASS |
| internal/domain | 100.0% | ✅ PASS |
| internal/proxy | 94.8% | ✅ PASS |
| internal/middleware | 88.2% | ✅ PASS |
| internal/db | 83.3% | ✅ PASS |
| internal/config | 80.0% | ✅ PASS |
| internal/model | 56.7% | ✅ PASS (handler 100%, repo 需 DB) |
| internal/credential | 48.3% | ✅ PASS (selector 100%, repo 需 DB) |
| internal/chat | 38.1% | ✅ PASS (handler 100%, repo 需 DB) |
| internal/quota | 25.0% | ✅ PASS (service 100%, repo 需 DB) |
| internal/proxy/providers | 7.0% | ✅ PASS (mock 100%, OpenAI/Anthropic 需 API) |

> **说明**：Repository 层需要 PostgreSQL 数据库（设置 `TEST_DATABASE_URL`），OpenAI/Anthropic Provider 需要真实 API Key。

---

## 运行全部测试

```bash
cd /Users/didi/Documents/workspace/RiderProject/llmgw

# 运行所有测试
go test ./... -v

# 带竞态检测
go test -race ./...

# 带覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Auth 单元测试（无需 API Key，无需数据库）

测试文件：`internal/auth/handler_test.go`、`internal/auth/jwt_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestAuthHandler_Login_Redirects` | Login 重定向到 SSO |
| `TestAuthHandler_Callback_ReturnsTODO` | Callback 返回 token 占位符 |
| `TestAuthHandler_Logout_OK` | Logout 返回成功 |
| `TestAuthHandler_SSOProvider_Interface` | SSOProvider 接口编译通过 |
| `TestSignToken_ValidToken` | JWT 签名和验证 |
| `TestSignToken_ClaimsMatchAllFields` | claims 字段正确嵌入 |
| `TestSignToken_WrongSecretFails` | 错误密钥验证失败 |
| `TestSignToken_ExpiredToken` | 过期 token 验证失败 |
| `TestSignToken_SigningMethodIsHS256` | 签名算法为 HS256 |
| `TestSignToken_DifferentUsersProduceDifferentTokens` | 不同用户生成不同 token |

```bash
go test ./internal/auth/ -v -timeout 30s
```

---

## Middleware 单元测试（无需 API Key，无需数据库）

测试文件：`internal/middleware/auth_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestJWTAuth_ValidToken_Passes` | 有效 JWT 通过认证 |
| `TestJWTAuth_ValidToken_SetsUserIDKey` | userID 正确设置到 context |
| `TestJWTAuth_MissingAuthorizationHeader` | 缺少 token 返回 401 |
| `TestJWTAuth_WrongPrefix` | 错误前缀返回 401 |
| `TestJWTAuth_InvalidTokenString` | 无效 token 返回 401 |
| `TestJWTAuth_ExpiredToken` | 过期 token 返回 401 |
| `TestJWTAuth_WrongSecret` | 错误密钥返回 401 |
| `TestJWTAuth_AbortsPipeline` | 认证失败阻止下游 handler |
| `TestJWTAuth_UserIDKey_IsConstant` | UserIDKey 常量正确 |

```bash
go test ./internal/middleware/ -v -timeout 30s
```

---

## Credential 模块测试

### 单元测试（无需数据库）

测试文件：`internal/credential/selector_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestSelector_StickySession_SameSessionSameCred` | 同一 session 始终选同一 credential |
| `TestSelector_StickySession_DifferentSessionsMayDiffer` | 不同 session 可能选不同 credential |
| `TestSelector_RoundRobin_EmptySession_Cycles` | 空 session 轮询选择 |
| `TestSelector_RoundRobin_SingleCred` | 单 credential 始终返回它 |
| `TestSelector_RepoError_Propagates` | DB 错误透传 |
| `TestSelector_NoCredentials_ReturnsError` | 无 credential 返回错误 |
| `TestSelector_MultiModel_IndependentCounters` | 多模型独立计数器 |
| `TestSelector_ConcurrentPick_AllValid` | 并发选择线程安全 |
| `TestSelector_SessionSticky_ConcurrentSameSession` | 并发 sticky 一致性 |

```bash
go test ./internal/credential/ -v -run TestSelector -timeout 30s
```

### Repository 集成测试（需要 PostgreSQL）

测试文件：`internal/credential/repository_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestRepository_ListActive_ReturnsActiveOnly` | 只返回 is_active=true 的记录 |
| `TestRepository_ListActive_OrderedByID` | 按 id 排序 |
| `TestRepository_ListActive_NoActiveCredentials_ReturnsError` | 无活跃记录返回错误 |
| `TestRepository_ListActive_UnknownModel_ReturnsError` | 未知模型返回错误 |

```bash
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/llmgw_test" \
go test ./internal/credential/ -v -run TestRepository -timeout 30s
```

---

## Chat 模块测试

### 单元测试（无需数据库）

测试文件：`internal/chat/handler_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestChatHandler_ListSessions_OK` | 返回 session 列表 |
| `TestChatHandler_ListSessions_PassesUserID` | userID 正确传递给 repo |
| `TestChatHandler_ListSessions_EmptySessions` | 空 session 列表正常返回 |
| `TestChatHandler_ListSessions_RepoError` | repo 错误返回 500 |
| `TestChatHandler_GetSession_OK` | 返回 session 日志 |
| `TestChatHandler_GetSession_PassesUserIDAndSessionID` | 参数正确传递 |
| `TestChatHandler_GetSession_InvalidUUID` | 无效 UUID 返回 400 |
| `TestChatHandler_GetSession_RepoError` | repo 错误返回 500 |
| `TestChatHandler_GetSession_EmptyLogs` | 空 session 正常返回 |

```bash
go test ./internal/chat/ -v -run TestChatHandler -timeout 30s
```

### Repository 集成测试（需要 PostgreSQL）

测试文件：`internal/chat/repository_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestChatRepository_Save_And_ListSessions` | Save 后 ListSessions 能找到 |
| `TestChatRepository_GetSession_ReturnsLogs` | GetSession 返回正确日志 |
| `TestChatRepository_ListSessions_Deduplicates` | DISTINCT 去重 |
| `TestChatRepository_GetSession_EmptyForUnknownSession` | 未知 session 返回空 |
| `TestChatRepository_ListSessions_EmptyForUnknownUser` | 未知用户返回空 |

```bash
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/llmgw_test" \
go test ./internal/chat/ -v -run TestChatRepository -timeout 30s
```

---

## Model 模块测试

### 单元测试（无需数据库）

测试文件：`internal/model/handler_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestModelHandler_ListModels_ReturnsModelsWithRemainingQuota` | 只返回 remaining>0 的模型 |
| `TestModelHandler_ListModels_AllExhausted_ReturnsEmpty` | 全耗尽返回空 |
| `TestModelHandler_ListModels_RemainingCorrect` | Remaining 计算正确 |
| `TestModelHandler_ListModels_RepoError` | repo 错误返回 500 |
| `TestModelHandler_ListModels_NegativeRemaining_Excluded` | 负剩余被排除 |
| `TestModelHandler_ListQuota_ReturnsAll` | 返回所有 quota |
| `TestModelHandler_ListQuota_IncludesExhausted` | 包含耗尽的 quota |
| `TestModelHandler_ListQuota_RepoError` | repo 错误返回 500 |

```bash
go test ./internal/model/ -v -run TestModelHandler -timeout 30s
```

### Repository 集成测试（需要 PostgreSQL）

测试文件：`internal/model/repository_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestModelRepository_ListActive_ReturnsActiveModels` | 只返回 is_active=true |
| `TestModelRepository_ListActive_AllModelsHaveIsActiveTrue` | 所有返回的 IsActive=true |
| `TestModelRepository_ListActive_ScansAllFields` | 所有字段正确扫描 |

```bash
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/llmgw_test" \
go test ./internal/model/ -v -run TestModelRepository -timeout 30s
```

---

## Domain 单元测试

测试文件：`internal/domain/types_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestUserQuota_Remaining` | Remaining 计算正确 |
| `TestChatLog_IDIsUUID` | ID 为 UUID 类型 |
| `TestChatLog_CredentialID_NilByDefault` | CredentialID 默认 nil |
| `TestChatLog_CredentialID_Settable` | CredentialID 可设置 |
| `TestChatLog_RequestMessagesIsBytes` | RequestMessages 为 JSONB |
| `TestChatLog_StatusValues` | Status 值正确 |
| `TestTokenUsage_Fields` | TokenUsage 字段正确 |
| `TestTokenUsage_TotalIsIndependent` | TotalTokens 独立存储 |
| `TestMessage_RoleAndContent` | Message 字段正确 |
| `TestChatRequest_StreamDefaultsFalse` | Stream 默认 false |
| `TestChatRequest_JSONRoundTrip` | ChatRequest JSON 序列化 |
| `TestChatResponse_JSONRoundTrip` | ChatResponse JSON 序列化 |
| `TestModel_IsActiveField` | Model.IsActive 正确 |
| `TestModelCredential_Fields` | ModelCredential 字段正确 |
| `TestUser_Fields` | User 字段正确 |

```bash
go test ./internal/domain/ -v -timeout 30s
```

---

## Config 单元测试

测试文件：`internal/config/config_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestLoad_MinimalConfig` | 最小配置加载 |
| `TestLoad_EnvField` | Env 字段正确 |
| `TestLoad_SSOWechatWork` | SSO 微信配置加载 |
| `TestLoad_ProvidersConfig` | Providers 配置加载 |
| `TestLoad_ProxyField` | Proxy 字段加载 |
| `TestLoad_EmptyProxy` | 空 Proxy 正确处理 |
| `TestLoad_BaiduConfig` | 百度配置加载 |
| `TestLoad_MultipleLoads` | 多次加载互不干扰 |

```bash
go test ./internal/config/ -v -timeout 30s
```

---

## DB 单元测试

测试文件：`internal/db/db_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestConnect_UnreachableHost` | 不可达主机返回错误 |
| `TestConnect_InvalidScheme` | 无效 DSN 返回错误 |
| `TestConnect_Success` | 连接成功（需 TEST_DATABASE_URL） |

```bash
go test ./internal/db/ -v -timeout 30s
```

---

## Quota 单元测试（无需 API Key，无需数据库）

测试文件：`internal/quota/service_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestUserQuota_Remaining` | `Remaining()` 计算正确（含边界：0、负值） |
| `TestService_Check_HasQuota` | 有余量时 Check 返回 nil |
| `TestService_Check_ExactlyZeroRemaining` | 余量恰好为 0 → ErrQuotaExceeded |
| `TestService_Check_NegativeRemaining` | used > quota → ErrQuotaExceeded |
| `TestService_Check_RepoError` | repo DB 错误被原样透传 |
| `TestService_Check_NotFoundIsRepoError` | 无 quota 行 → 返回错误，且不是 ErrQuotaExceeded |
| `TestService_Deduct_DelegatesToRepo` | Deduct 将 tokens 传给 repo |
| `TestService_Deduct_AccumulatesMultipleCalls` | 多次调用累加到 stub |
| `TestService_Deduct_RepoError` | repo 写错误被透传 |
| `TestService_Deduct_ZeroTokens` | 0 token 不报错 |
| `TestService_TryDeduct_HasQuota` | TryDeduct 成功扣减 |
| `TestService_TryDeduct_QuotaExhausted` | quota 耗尽时 TryDeduct 失败 |
| `TestService_TryDeduct_NoRow` | 无行时 TryDeduct 返回 ErrQuotaExceeded |
| `TestService_TryDeduct_RepoError` | repo 错误透传 |
| `TestService_CheckThenDeduct_QuotaDecreases` | Check 通过后 Deduct，stub 正确记录用量 |

```bash
go test ./internal/quota/ -v -run "TestUserQuota|TestService" -timeout 30s
```

---

## Quota 模块测试（无需数据库）

测试文件：`internal/quota/module_test.go`

使用 in-memory fake repo 验证完整业务流程。

| 测试名 | 验证内容 |
|--------|---------|
| `TestModule_FullFlow` | 完整流程：check → deduct → exhaust |
| `TestModule_MultiUser` | 用户间 quota 隔离 |
| `TestModule_MultiModel` | 模型间 quota 隔离 |
| `TestModule_NoQuotaRow` | 无 quota 行返回错误 |
| `TestModule_DeductBeyondQuota` | Deduct 不强制上限 |
| `TestModule_ConcurrentDeduct` | 并发 Deduct 线程安全 |
| `TestModule_CheckDeductMultiRound` | 多轮请求正确扣减 |
| `TestModule_DeductNoRow` | 无行 Deduct 返回错误 |
| `TestModule_LargeQuota` | 大数值 quota 正确处理 |
| `TestModule_UsersIsolation` | 多用户隔离表驱动测试 |
| `TestModule_ConcurrentCheckAndDeduct` | TOCTOU 竞态演示 |
| `TestModule_TryDeduct_ConcurrentSafe` | TryDeduct 并发安全 |

```bash
go test ./internal/quota/ -v -run TestModule -timeout 30s
```

---

## Quota Repository 集成测试（需要 PostgreSQL）

测试文件：`internal/quota/repository_test.go`

需要一个已应用全部迁移的测试数据库，通过 `TEST_DATABASE_URL` 传入。

| 测试名 | 验证内容 |
|--------|---------|
| `TestRepository_Get_Exists` | 插入行后 Get 返回正确字段和 Remaining |
| `TestRepository_Get_NotFound` | 不存在的行返回 error |
| `TestRepository_Deduct_UpdatesUsedTokens` | Deduct 后 Get 确认 used_tokens 更新 |
| `TestRepository_Deduct_Accumulates` | 多次 Deduct 累加正确 |
| `TestRepository_ListByUser_ReturnsAll` | 多个模型的 quota 全部返回 |
| `TestRepository_ListByUser_NoRows` | 无记录时返回空 slice，不报错 |

```bash
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/llmgw_test?sslmode=disable" \
go test ./internal/quota/ -v -run TestRepository -timeout 30s
```

> 每个测试使用时间戳生成唯一 `user_id`，测试结束后自动清理插入的数据。

---

## Proxy 单元测试（无需 API Key，无需数据库）

测试文件：`internal/proxy/handler_test.go`、`internal/proxy/router_test.go`

所有依赖均通过 stub/interface 注入，无外部调用。

| 测试名 | 验证内容 |
|--------|---------|
| `TestHandlerChat_Complete` | 完整非流式路径：200、正确内容、quota 扣减、CredentialID 写入 ChatLog |
| `TestHandlerChat_CredentialUnavailable` | credential 池为空 → 503 |
| `TestHandlerChat_BadRequest_MissingModel` | model 缺失 → 400 |
| `TestHandlerChat_QuotaExceeded` | quota 耗尽 → 403 |
| `TestHandlerChat_QuotaServiceError` | quota 服务异常 → 500 |
| `TestHandlerChat_UnsupportedModel` | 未注册模型 → 400 |
| `TestHandlerChat_Stream` | 流式路径：SSE Content-Type、body 含内容 |
| `TestRouterGet_KnownModels` | 所有内置模型均可解析到 Provider |
| `TestRouterGet_UnknownModel` | 未知模型返回 error |
| `TestRouterRegister` | Register 后可通过 Get 取到 |
| `TestRouterRegister_Override` | Register 可覆盖已有映射 |

```bash
go test ./internal/proxy/ -v -timeout 30s
```

---

## Proxy 集成测试（无需 API Key，无需数据库）

测试文件：`internal/proxy/proxy_integration_test.go`

使用真实 Handler + 真实 Router + MockProvider 端到端验证完整请求路径，quota 和 ChatLog 用内存实现替代数据库。

| 测试名 | 验证内容 |
|--------|---------|
| `TestIntegration_CompleteFlow` | 全路径 200、内容正确、quota 扣减、credential_id 绑定写入 |
| `TestIntegration_StreamFlow` | SSE 流式路径端到端 |
| `TestIntegration_QuotaEnforced` | quota=0 时在 provider 调用前返回 403 |
| `TestIntegration_UnknownModelRejected` | 未注册模型 → 400 |
| `TestIntegration_MultiTurn` | 多轮消息正确转发，mock echo 最后一条 |
| `TestIntegration_RouterRegisterCustomProvider` | Router.Register 注入的 provider 可通过完整 Handler 路径访问 |

```bash
go test ./internal/proxy/ -v -run TestIntegration -timeout 30s
```

---

## Mock Provider 测试（无需 API Key，本地可直接运行）

测试文件：`internal/proxy/providers/mock_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestMockComplete` | 同步调用返回 echo 内容，token 计数非零 |
| `TestMockCompleteCustomResponse` | 自定义固定响应正确返回 |
| `TestMockStream` | 流式分块、quota 扣减、日志保存 |

```bash
go test ./internal/proxy/providers/ -v -run TestMock -timeout 30s
```

---

## Anthropic Provider 集成测试

测试文件：`internal/proxy/providers/anthropic_test.go`

包含三个测试用例：

| 测试名 | 验证内容 |
|--------|----------|
| `TestAnthropicComplete` | 同步调用，检查返回内容非空、token 计数正确 |
| `TestAnthropicCompleteWithSystem` | system prompt 被正确提取并传给 Anthropic |
| `TestAnthropicStream` | 流式调用，检查 chunk 收到、`[DONE]` 出现、quota 扣减、日志保存 |

### 运行全部测试

```bash
ANTHROPIC_API_KEY="..." \
HTTP_PROXY="..." \
go test ./internal/proxy/providers/ \
  -v -run TestAnthropic -timeout 60s
```

### 只跑某一个

```bash
# 仅测试同步接口
ANTHROPIC_API_KEY="..." HTTP_PROXY="..." \
go test ./internal/proxy/providers/ \
  -v -run TestAnthropicComplete$ -timeout 60s

# 仅测试流式接口
ANTHROPIC_API_KEY="..." HTTP_PROXY="..." \
go test ./internal/proxy/providers/ \
  -v -run TestAnthropicStream -timeout 60s
```

### 期望输出

```
=== RUN   TestAnthropicComplete
    anthropic_test.go:57: content:  "OK"
    anthropic_test.go:58: usage:    in=10 out=3 total=13
--- PASS: TestAnthropicComplete (1.23s)

=== RUN   TestAnthropicCompleteWithSystem
    anthropic_test.go:90: content: "你好！..."
--- PASS: TestAnthropicCompleteWithSystem (1.45s)

=== RUN   TestAnthropicStream
    anthropic_test.go:141: streamed content: "2"
    anthropic_test.go:142: usage: in=12 out=2
--- PASS: TestAnthropicStream (1.67s)

PASS
```

---

## OpenAI Provider 集成测试

测试文件：`internal/proxy/providers/openai_test.go`

| 测试名 | 验证内容 |
|--------|---------|
| `TestOpenAIComplete` | 同步调用，检查返回内容和 token 计数 |
| `TestOpenAIStream` | 流式调用，检查 chunk、`[DONE]`、quota 扣减、日志保存 |

```bash
OPENAI_API_KEY="..." \
HTTP_PROXY="..." \
go test ./internal/proxy/providers/ \
  -v -run TestOpenAI -timeout 60s
```

---

## 系统测试

测试文件：`test/system/system_test.go`

端到端 HTTP 请求流测试，覆盖所有 API 端点。

| 测试名 | 验证内容 |
|--------|---------|
| `TestSystem_Auth_Login` | Login 重定向 |
| `TestSystem_Auth_Callback` | Callback 返回 |
| `TestSystem_Auth_Logout` | Logout 成功 |
| `TestSystem_Middleware_ValidToken` | 有效 token 通过 |
| `TestSystem_Middleware_MissingToken` | 缺少 token 拒绝 |
| `TestSystem_Middleware_InvalidToken` | 无效 token 拒绝 |
| `TestSystem_Middleware_ExpiredToken` | 过期 token 拒绝 |
| `TestSystem_Middleware_WrongSecret` | 错误密钥拒绝 |
| `TestSystem_Router_KnownModels` | 已知模型路由 |
| `TestSystem_Router_UnknownModel` | 未知模型错误 |
| `TestSystem_Router_Register` | 自定义模型注册 |
| `TestSystem_Router_Override` | 模型覆盖 |
| `TestSystem_MockProvider_Complete` | Mock 同步调用 |
| `TestSystem_MockProvider_CustomResponse` | Mock 自定义响应 |
| `TestSystem_Quota_Remaining` | Quota 剩余计算 |
| `TestSystem_Quota_ErrQuotaExceeded` | Quota 耗尽错误 |
| `TestSystem_Domain_*` | Domain 类型测试 |
| `TestSystem_Config_*` | Config 字段测试 |
| `TestSystem_Integration_*` | 集成测试 |

```bash
go test ./test/system/... -v -timeout 60s
```

---

## 黑盒测试（API 契约测试）

测试文件：`test/blackbox/blackbox_test.go`、`test/blackbox/TESTCASES.md`

黑盒测试基于 `docs/api.md` 定义的 API 契约，验证 HTTP 接口的功能正确性，不依赖内部实现细节。所有外部依赖（数据库、LLM Provider）均使用 Mock 实现。

### 测试范围

| 模块 | 测试项 |
|------|--------|
| Auth | Login 重定向、Callback 返回、Logout 成功 |
| JWT Middleware | 有效 Token 通过、无效/过期/错误签名 Token 拒绝 |
| Models | 获取可用模型列表、过滤耗尽模型、Repo 错误处理 |
| Quota | 获取 Quota 详情、包含耗尽模型 |
| Chat (非流式) | 完整对话、多轮对话、Session-Sticky、参数校验、Quota 耗尽、无 Credential |
| Chat (流式) | SSE 响应格式、data 格式、[DONE] 标记 |
| Sessions | 获取会话列表、获取会话详情、无效 UUID |
| Router | 已知模型路由、未知模型错误、注册自定义模型、覆盖模型 |
| Credential | Session-Sticky 一致性、Round-Robin 循环、并发安全 |

### 运行黑盒测试

```bash
# 运行所有黑盒测试
go test ./test/blackbox/ -v -timeout 60s

# 运行特定模块测试
go test ./test/blackbox/ -v -run TestAuth -timeout 30s
go test ./test/blackbox/ -v -run TestJWT -timeout 30s
go test ./test/blackbox/ -v -run TestChat -timeout 30s
go test ./test/blackbox/ -v -run TestRouter -timeout 30s
go test ./test/blackbox/ -v -run TestCredential -timeout 30s
```

### 测试用例设计文档

详细的测试用例设计见 `test/blackbox/TESTCASES.md`，包含：
- 55 个测试用例定义
- 认证 + API 组合矩阵
- Chat 请求参数矩阵
- 边界条件和错误处理覆盖

---

## E2E 端到端测试（需 Docker 环境）

测试文件：`test/e2e/e2e_test.go`

E2E 测试验证完整的 HTTP 请求流程，需要运行中的 LLM Gateway 服务和 PostgreSQL 数据库。使用 Docker Compose 搭建测试环境。

### 测试范围

| 模块 | 测试项 |
|------|--------|
| Health Check | 服务健康检查 |
| Auth | Login 重定向、Logout 成功 |
| JWT Middleware | 缺少 Token、过期 Token |
| Models | 获取可用模型列表、无 Quota 用户 |
| Quota | 获取 Quota 详情 |
| Chat (非流式) | 完整对话、Quota 耗尽 |
| Chat (流式) | SSE 响应格式 |
| Sessions | 获取会话列表、获取会话详情、无效 UUID |
| Session-Sticky | 相同 Session 路由一致性 |
| Database | 测试数据验证 |

### 测试用例详情

| 用例ID | 测试名 | 验证内容 |
|--------|--------|----------|
| E2E-001 | TestE2E_HealthCheck | 服务可达，返回 302 重定向 |
| E2E-002 | TestE2E_Auth_Login | Login 返回 302 重定向到 SSO |
| E2E-003 | TestE2E_Auth_Logout | Logout 返回 200 |
| E2E-004 | TestE2E_JWT_MissingToken | 无 Token 返回 401 |
| E2E-005 | TestE2E_JWT_ExpiredToken | 过期 Token 返回 401 |
| E2E-006 | TestE2E_Models_ListModels | 返回有剩余 quota 的模型列表 |
| E2E-007 | TestE2E_Models_ListModels_EmptyForNoQuota | 无 quota 用户返回空列表 |
| E2E-008 | TestE2E_Quota_List | 返回所有 quota 记录 |
| E2E-009 | TestE2E_Chat_Complete | 非流式对话返回正确内容和 token 统计 |
| E2E-010 | TestE2E_Chat_QuotaExceeded | quota 耗尽返回 403 |
| E2E-011 | TestE2E_Chat_Stream | 流式对话返回 SSE 格式 |
| E2E-012 | TestE2E_Sessions_List | 返回用户会话列表 |
| E2E-013 | TestE2E_Sessions_Get | 返回会话详细消息 |
| E2E-014 | TestE2E_Sessions_InvalidUUID | 无效 UUID 返回 400 |
| E2E-015 | TestE2E_Chat_SessionSticky | 相同 session_id 多次请求路由一致 |
| E2E-016 | TestE2E_Database_HasTestData | 数据库包含测试数据 |

### 环境搭建

```bash
# 启动测试环境
./test/docker/setup-test-env.sh up

# 或手动启动
docker compose -f test/docker/docker-compose.test.yml up -d
```

### 运行 E2E 测试

```bash
# 使用脚本运行
./test/docker/setup-test-env.sh test

# 或手动运行
export TEST_API_BASE="http://localhost:8080"
export TEST_JWT_SECRET="test-jwt-secret-for-blackbox-testing"
export TEST_DATABASE_URL="postgres://llmgw:llmgw_test_password@localhost:5433/llmgw_test?sslmode=disable"
go test ./test/e2e/... -v -timeout 60s
```

### 测试环境

| 服务 | 端口 | 说明 |
|------|------|------|
| LLM Gateway | 8080 | HTTP API 服务 |
| PostgreSQL | 5433 | 测试数据库 |

### 测试数据

| 用户 ID | Email | Quota 状态 |
|---------|-------|-----------|
| alice | alice@test.com | mock: 100万, gpt-4o: 49万 |
| bob | bob@test.com | mock: 已耗尽 |
| charlie | charlie@test.com | 无 quota |

### 停止测试环境

```bash
./test/docker/setup-test-env.sh down
```

---

## 测试统计

| 类型 | 数量 |
|------|------|
| 单元测试 | 92 |
| 模块测试 | 23 |
| 集成测试 | 20 (需 DB/API) |
| 系统测试 | 30 |
| 黑盒测试 | 55 |
| E2E 测试 | 17 |
| **总计** | **237** |

---

## 说明

- Anthropic 测试使用 `claude-haiku-4-5`，OpenAI 测试使用 `gpt-4o-mini`，均为最低价模型
- 未设置对应 API Key 时测试自动 skip，不影响 CI
- 代理通过 `HTTP_PROXY` 环境变量传入，不传则直连；生产环境在 `config.yaml` 的 `proxy` 字段配置
- Repository 层测试需要 PostgreSQL 数据库，通过 `TEST_DATABASE_URL` 传入