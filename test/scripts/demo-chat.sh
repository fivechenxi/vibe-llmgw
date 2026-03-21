#!/bin/bash
# LLM Gateway API Demo Script
# 演示用户与 Mock 模型的完整对话流程
#
# 前置条件：
#   1. 启动测试环境: ./test/docker/setup-test-env.sh up
#   2. 确保服务健康: curl -s http://localhost:8080/auth/login
#
# 使用方法：
#   ./test/scripts/demo-chat.sh              # 使用默认用户 alice
#   ./test/scripts/demo-chat.sh bob          # 指定用户
#   ./test/scripts/demo-chat.sh alice "hi"   # 指定用户和消息

set -e

# ==============================================================================
# 配置
# ==============================================================================

API_BASE="${TEST_API_BASE:-http://localhost:8080}"
JWT_SECRET="${TEST_JWT_SECRET:-test-jwt-secret-for-blackbox-testing}"
USER_ID="${1:-alice}"
MESSAGE="${2:-你好，请介绍一下你自己}"
SESSION_ID="${3:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==============================================================================
# 工具函数
# ==============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
}

# 生成 JWT Token（需要 jq 和 openssl）
generate_jwt() {
    local user_id="$1"
    local email="${user_id}@test.com"
    # Capitalize first letter of user_id for name
    local name="$(echo ${user_id:0:1} | tr '[:lower:]' '[:upper:]')${user_id:1} Test"

    # JWT Header
    local header='{"alg":"HS256","typ":"JWT"}'
    local header_b64=$(echo -n "$header" | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')

    # JWT Payload (expires in 1 hour)
    local exp=$(($(date +%s) + 3600))
    local payload="{\"sub\":\"${user_id}\",\"email\":\"${email}\",\"name\":\"${name}\",\"exp\":${exp}}"
    local payload_b64=$(echo -n "$payload" | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')

    # JWT Signature
    local signature=$(echo -n "${header_b64}.${payload_b64}" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')

    echo "${header_b64}.${payload_b64}.${signature}"
}

# 检查服务是否可用
check_service() {
    log_info "检查服务状态..."
    if curl -sf "${API_BASE}/auth/login" > /dev/null 2>&1; then
        log_success "LLM Gateway 服务可用"
        return 0
    else
        log_error "LLM Gateway 服务不可用，请先启动测试环境"
        echo "  运行: ./test/docker/setup-test-env.sh up"
        exit 1
    fi
}

# 格式化 JSON 输出
format_json() {
    if command -v jq &> /dev/null; then
        jq '.'
    else
        cat
    fi
}

# ==============================================================================
# 主流程
# ==============================================================================

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║             LLM Gateway API Demo Script                       ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "API Base: ${API_BASE}"
echo "User:     ${USER_ID}"
echo ""

# 检查服务
check_service

# 生成 Token
log_section "1. 生成 JWT Token"
TOKEN=$(generate_jwt "$USER_ID")
log_success "Token 生成成功"
echo "  User ID: ${USER_ID}"
echo "  Email:   ${USER_ID}@test.com"
echo "  Token:   ${TOKEN:0:50}..."

# 生成 Session ID (UUID格式)
if [ -z "$SESSION_ID" ]; then
    SESSION_ID=$(uuidgen 2>/dev/null || python3 -c "import uuid; print(uuid.uuid4())" 2>/dev/null || echo "550e8400-e29b-41d4-a716-446655440000")
fi
log_info "Session ID: ${SESSION_ID}"

# 查看可用模型
log_section "2. 获取可用模型列表 (GET /api/models)"
log_info "请求中..."
MODELS_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${API_BASE}/api/models")
MODELS_CODE=$(echo "$MODELS_RESPONSE" | tail -n1)
MODELS_BODY=$(echo "$MODELS_RESPONSE" | sed '$d')

if [ "$MODELS_CODE" = "200" ]; then
    log_success "响应码: ${MODELS_CODE}"
    echo "$MODELS_BODY" | format_json
else
    log_error "响应码: ${MODELS_CODE}"
    echo "$MODELS_BODY"
fi

# 查看 Quota
log_section "3. 获取用户 Quota (GET /api/quota)"
log_info "请求中..."
QUOTA_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${API_BASE}/api/quota")
QUOTA_CODE=$(echo "$QUOTA_RESPONSE" | tail -n1)
QUOTA_BODY=$(echo "$QUOTA_RESPONSE" | sed '$d')

if [ "$QUOTA_CODE" = "200" ]; then
    log_success "响应码: ${QUOTA_CODE}"
    echo "$QUOTA_BODY" | format_json
else
    log_error "响应码: ${QUOTA_CODE}"
    echo "$QUOTA_BODY"
fi

# 发送聊天消息（非流式）
log_section "4. 发送聊天消息 - 非流式 (POST /api/chat)"
log_info "用户消息: \"${MESSAGE}\""
log_info "请求中..."
CHAT_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"mock\",
        \"messages\": [{\"role\": \"user\", \"content\": \"${MESSAGE}\"}],
        \"session_id\": \"${SESSION_ID}\",
        \"stream\": false
    }" \
    "${API_BASE}/api/chat")
CHAT_CODE=$(echo "$CHAT_RESPONSE" | tail -n1)
CHAT_BODY=$(echo "$CHAT_RESPONSE" | sed '$d')

if [ "$CHAT_CODE" = "200" ]; then
    log_success "响应码: ${CHAT_CODE}"
    echo "$CHAT_BODY" | format_json

    # 提取回复内容
    if command -v jq &> /dev/null; then
        CONTENT=$(echo "$CHAT_BODY" | jq -r '.content // empty')
        TOKENS=$(echo "$CHAT_BODY" | jq -r '.usage.total_tokens // 0')
        if [ -n "$CONTENT" ]; then
            echo ""
            echo -e "${GREEN}AI 回复:${NC} ${CONTENT}"
            echo -e "${BLUE}Token 消耗:${NC} ${TOKENS}"
        fi
    fi
elif [ "$CHAT_CODE" = "403" ]; then
    log_error "响应码: ${CHAT_CODE} - Quota 已耗尽"
    echo "$CHAT_BODY" | format_json
else
    log_error "响应码: ${CHAT_CODE}"
    echo "$CHAT_BODY"
fi

# 发送聊天消息（流式）
log_section "5. 发送聊天消息 - 流式 SSE (POST /api/chat, stream=true)"
log_info "请求中..."
log_info "流式响应:"
echo ""

curl -s -N \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"mock\",
        \"messages\": [{\"role\": \"user\", \"content\": \"用一句话介绍Go语言\"}],
        \"session_id\": \"${SESSION_ID}\",
        \"stream\": true
    }" \
    "${API_BASE}/api/chat" 2>&1 | while IFS= read -r line; do
        if [[ "$line" =~ ^data:\ (.*)$ ]]; then
            data="${BASH_REMATCH[1]}"
            if [ "$data" = "[DONE]" ]; then
                echo -e "\n${GREEN}[DONE]${NC}"
            else
                echo -n "$data "
            fi
        fi
    done

echo ""

# 查看会话列表
log_section "6. 获取会话列表 (GET /api/sessions)"
log_info "请求中..."
SESSIONS_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${API_BASE}/api/sessions")
SESSIONS_CODE=$(echo "$SESSIONS_RESPONSE" | tail -n1)
SESSIONS_BODY=$(echo "$SESSIONS_RESPONSE" | sed '$d')

if [ "$SESSIONS_CODE" = "200" ]; then
    log_success "响应码: ${SESSIONS_CODE}"
    echo "$SESSIONS_BODY" | format_json
else
    log_error "响应码: ${SESSIONS_CODE}"
    echo "$SESSIONS_BODY"
fi

# 查看会话详情
log_section "7. 获取会话详情 (GET /api/sessions/{session_id})"
log_info "Session ID: ${SESSION_ID}"
log_info "请求中..."
DETAIL_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${API_BASE}/api/sessions/${SESSION_ID}")
DETAIL_CODE=$(echo "$DETAIL_RESPONSE" | tail -n1)
DETAIL_BODY=$(echo "$DETAIL_RESPONSE" | sed '$d')

if [ "$DETAIL_CODE" = "200" ]; then
    log_success "响应码: ${DETAIL_CODE}"
    echo "$DETAIL_BODY" | format_json
else
    log_error "响应码: ${DETAIL_CODE}"
    echo "$DETAIL_BODY"
fi

# 完成
log_section "演示完成"
echo ""
echo -e "${GREEN}✓ API 流程演示完成${NC}"
echo ""
echo "常用命令:"
echo "  # 查看 JWT Token 内容"
echo "  echo '${TOKEN}' | cut -d. -f2 | base64 -d 2>/dev/null | jq '.'"
echo ""
echo "  # 手动调用 API"
echo "  curl -H \"Authorization: Bearer ${TOKEN}\" ${API_BASE}/api/models"
echo ""
echo "  # 停止测试环境"
echo "  ./test/docker/setup-test-env.sh down"
echo ""