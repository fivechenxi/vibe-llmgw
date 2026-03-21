#!/bin/bash
# LLM Gateway Curl 命令集
# 可直接复制粘贴执行的 curl 命令
#
# 前置条件：
#   1. 启动测试环境: ./test/docker/setup-test-env.sh up
#   2. 确保 jq 已安装 (可选，用于格式化输出)

# ==============================================================================
# 配置
# ==============================================================================

export API_BASE="${API_BASE:-http://localhost:8080}"
export JWT_SECRET="${JWT_SECRET:-test-jwt-secret-for-blackbox-testing}"

# ==============================================================================
# Token 生成函数
# ==============================================================================

# 生成 JWT Token (需要 openssl)
gen_token() {
    local user="${1:-alice}"
    local exp=$(($(date +%s) + 3600))
    local header='{"alg":"HS256","typ":"JWT"}'
    local payload="{\"sub\":\"${user}\",\"email\":\"${user}@test.com\",\"name\":\"Test User\",\"exp\":${exp}}"
    local h=$(echo -n "$header" | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')
    local p=$(echo -n "$payload" | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')
    local s=$(echo -n "${h}.${p}" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')
    echo "${h}.${p}.${s}"
}

# ==============================================================================
# 常用命令
# ==============================================================================

# 获取 Token
export TOKEN=$(gen_token alice)
echo "Token: $TOKEN"

# ==============================================================================
# 1. 认证相关
# ==============================================================================

# 登录 (重定向到 SSO)
echo "\n# 登录"
curl -s -o /dev/null -w "%{http_code} %{redirect_url}\n" "${API_BASE}/auth/login"

# 登出
echo "\n# 登出"
curl -s -X POST "${API_BASE}/auth/logout" | jq '.'

# ==============================================================================
# 2. 模型 & Quota
# ==============================================================================

# 获取可用模型列表 (有剩余 quota 的模型)
echo "\n# 获取可用模型"
curl -s -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/models" | jq '.'

# 获取 Quota 详情
echo "\n# 获取 Quota 详情"
curl -s -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/quota" | jq '.'

# ==============================================================================
# 3. 聊天 - 非流式
# ==============================================================================

# 发送消息 (非流式)
echo "\n# 发送消息 - 非流式"
curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "mock",
        "messages": [{"role": "user", "content": "你好"}]
    }' \
    "${API_BASE}/api/chat" | jq '.'

# 带 session_id 的消息 (Session-Sticky)
echo "\n# 发送消息 - 带 Session ID"
curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "mock",
        "messages": [{"role": "user", "content": "你好"}],
        "session_id": "550e8400-e29b-41d4-a716-446655440000"
    }' \
    "${API_BASE}/api/chat" | jq '.'

# 多轮对话
echo "\n# 多轮对话"
curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "mock",
        "messages": [
            {"role": "user", "content": "我叫小明"},
            {"role": "assistant", "content": "你好小明！有什么可以帮你的吗？"},
            {"role": "user", "content": "我叫什么名字？"}
        ]
    }' \
    "${API_BASE}/api/chat" | jq '.'

# ==============================================================================
# 4. 聊天 - 流式 SSE
# ==============================================================================

# 发送消息 (流式)
echo "\n# 发送消息 - 流式 SSE"
curl -s -N -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "mock",
        "messages": [{"role": "user", "content": "讲个笑话"}],
        "stream": true
    }' \
    "${API_BASE}/api/chat"

echo ""

# ==============================================================================
# 5. 会话历史
# ==============================================================================

# 获取会话列表
echo "\n# 获取会话列表"
curl -s -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/sessions" | jq '.'

# 获取会话详情
echo "\n# 获取会话详情"
curl -s -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/sessions/550e8400-e29b-41d4-a716-446655440000" | jq '.'

# ==============================================================================
# 6. 错误场景测试
# ==============================================================================

# 缺少 Token
echo "\n# 缺少 Token - 应返回 401"
curl -s -o /dev/null -w "%{http_code}\n" "${API_BASE}/api/models"

# 过期 Token
echo "\n# 过期 Token - 应返回 401"
EXPIRED_TOKEN=$(gen_token_expired bob 2>/dev/null || echo "expired.token.here")
curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer expired" "${API_BASE}/api/models"

# Quota 耗尽的用户 (bob)
echo "\n# Quota 耗尽 - 应返回 403"
BOB_TOKEN=$(gen_token bob)
curl -s -X POST \
    -H "Authorization: Bearer $BOB_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"model": "mock", "messages": [{"role": "user", "content": "hi"}]}' \
    "${API_BASE}/api/chat" | jq '.'

# 不支持的模型
echo "\n# 不支持的模型 - 应返回 400"
curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"model": "unknown-model", "messages": [{"role": "user", "content": "hi"}]}' \
    "${API_BASE}/api/chat" | jq '.'

# 无效的 Session UUID
echo "\n# 无效的 Session UUID - 应返回 400"
curl -s -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/sessions/invalid-uuid" | jq '.'