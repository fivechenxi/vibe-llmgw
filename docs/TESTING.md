# 测试说明

## 前置条件

- Go 1.22+（路径 `/opt/homebrew/bin/go`）
- 如需代理访问，配置 `HTTP_PROXY` 环境变量
- 集成测试需要对应的 API Key 环境变量；未设置时自动 skip

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
