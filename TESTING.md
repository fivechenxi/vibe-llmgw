# 测试说明

## 前置条件

- Go 1.22+（路径 `/opt/homebrew/bin/go`）
- 有效的 Anthropic API Key
- 如需代理访问，配置 `HTTP_PROXY` 环境变量

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

## 说明

- 测试使用 `claude-haiku-4-5`（最低价模型），每次调用消耗 token 极少
- 未设置 `ANTHROPIC_API_KEY` 时测试自动 skip，不影响 CI
- 代理地址通过环境变量 `HTTP_PROXY` 传入，不传则直连；生产环境在 `config.yaml` 的 `proxy` 字段配置
