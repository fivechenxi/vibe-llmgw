# 测试说明

## 前置条件

- Go 1.22+（路径 `/opt/homebrew/bin/go`）
- 如需代理访问，配置 `HTTP_PROXY` 环境变量
- 集成测试需要对应的 API Key 环境变量；未设置时自动 skip

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
