# 测试说明

## 前置条件

- Go 1.22+（路径 `/opt/homebrew/bin/go`）
- 如需代理访问，配置 `HTTP_PROXY` 环境变量
- Provider 集成测试需要对应的 API Key 环境变量；未设置时自动 skip
- Repository 集成测试需要设置 `TEST_DATABASE_URL`；未设置时自动 skip

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
| `TestService_CheckThenDeduct_QuotaDecreases` | Check 通过后 Deduct，stub 正确记录用量 |

```bash
cd /Users/didi/Documents/workspace/RiderProject/llmgw

/opt/homebrew/bin/go test ./internal/quota/ -v -run "TestUserQuota|TestService" -timeout 30s
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
/opt/homebrew/bin/go test ./internal/quota/ -v -run TestRepository -timeout 30s
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
cd /Users/didi/Documents/workspace/RiderProject/llmgw

/opt/homebrew/bin/go test ./internal/proxy/ -v -timeout 30s
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
/opt/homebrew/bin/go test ./internal/proxy/ -v -run TestIntegration -timeout 30s
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
cd /Users/didi/Documents/workspace/RiderProject/llmgw

/opt/homebrew/bin/go test ./internal/proxy/providers/ -v -run TestMock -timeout 30s
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
cd /Users/didi/Documents/workspace/RiderProject/llmgw

ANTHROPIC_API_KEY="..." \
HTTP_PROXY="..." \
/opt/homebrew/bin/go test ./internal/proxy/providers/ \
  -v -run TestAnthropic -timeout 60s
```

### 只跑某一个

```bash
# 仅测试同步接口
ANTHROPIC_API_KEY="..." HTTP_PROXY="..." \
/opt/homebrew/bin/go test ./internal/proxy/providers/ \
  -v -run TestAnthropicComplete$ -timeout 60s

# 仅测试流式接口
ANTHROPIC_API_KEY="..." HTTP_PROXY="..." \
/opt/homebrew/bin/go test ./internal/proxy/providers/ \
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
/opt/homebrew/bin/go test ./internal/proxy/providers/ \
  -v -run TestOpenAI -timeout 60s
```

---

## 说明

- Anthropic 测试使用 `claude-haiku-4-5`，OpenAI 测试使用 `gpt-4o-mini`，均为最低价模型
- 未设置对应 API Key 时测试自动 skip，不影响 CI
- 代理通过 `HTTP_PROXY` 环境变量传入，不传则直连；生产环境在 `config.yaml` 的 `proxy` 字段配置
