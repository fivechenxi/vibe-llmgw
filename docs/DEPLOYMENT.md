# Docker 部署指南

本文档描述如何使用 Docker Compose 部署 LLM Gateway。

## 快速开始

### 测试环境部署

```bash
# 启动测试环境（PostgreSQL + LLM Gateway）
./test/docker/setup-test-env.sh up

# 查看状态
./test/docker/setup-test-env.sh status

# 停止环境
./test/docker/setup-test-env.sh down
```

### 生产环境部署

```bash
# 使用生产配置启动
docker compose -f deploy/docker-compose.yml up -d

# 查看日志
docker compose -f deploy/docker-compose.yml logs -f llmgw
```

---

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Network                        │
│                                                         │
│  ┌─────────────┐      ┌─────────────────────────────┐  │
│  │  PostgreSQL │◀────▶│      LLM Gateway            │  │
│  │   :5432     │      │  (Gin HTTP Server :8080)    │  │
│  └─────────────┘      └─────────────────────────────┘  │
│                                 │                       │
└─────────────────────────────────┼───────────────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
               ┌────▼────┐  ┌─────▼────┐  ┌────▼──────┐
               │ OpenAI  │  │Anthropic │  │  国内模型  │
               │   API   │  │   API    │  │(DeepSeek) │
               └─────────┘  └──────────┘  └───────────┘
```

---

## 环境变量

### 必需变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `DATABASE_DSN` | PostgreSQL 连接字符串 | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key-here` |

### 可选变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `HTTP_PROXY` | 出站代理 | 空 |

### Provider 配置

通过 `config.yaml` 或环境变量配置各 LLM Provider：

```yaml
providers:
  openai:
    api_key: "sk-xxx"
    base_url: "https://api.openai.com/v1"
  anthropic:
    api_key: "sk-ant-xxx"
  deepseek:
    api_key: "sk-xxx"
```

---

## Docker Compose 配置

### 测试环境 (`test/docker/docker-compose.test.yml`)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: llmgw
      POSTGRES_PASSWORD: llmgw_test_password
      POSTGRES_DB: llmgw_test
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
      - ./seed-test-data.sql:/docker-entrypoint-initdb.d/02-seed.sql:ro

  llmgw:
    build:
      context: ../..
      dockerfile: test/docker/Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_DSN: "postgres://llmgw:llmgw_test_password@postgres:5432/llmgw_test?sslmode=disable"
    depends_on:
      postgres:
        condition: service_healthy
```

### 生产环境 (`deploy/docker-compose.yml`)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${DB_USER:-llmgw}
      POSTGRES_PASSWORD: ${DB_PASSWORD:?DB_PASSWORD required}
      POSTGRES_DB: ${DB_NAME:-llmgw}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-llmgw}"]
      interval: 10s
      timeout: 5s
      retries: 5

  llmgw:
    image: ${LLMGW_IMAGE:-llmgw:latest}
    ports:
      - "${SERVER_PORT:-8080}:8080"
    environment:
      DATABASE_DSN: "postgres://${DB_USER:-llmgw}:${DB_PASSWORD}@postgres:5432/${DB_NAME:-llmgw}?sslmode=disable"
      JWT_SECRET: ${JWT_SECRET:?JWT_SECRET required}
      HTTP_PROXY: ${HTTP_PROXY:-}
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

---

## 常用命令

### 构建镜像

```bash
# 构建测试环境镜像
docker compose -f test/docker/docker-compose.test.yml build

# 构建生产镜像
docker build -t llmgw:latest -f test/docker/Dockerfile .
```

### 启动/停止

```bash
# 后台启动
docker compose -f test/docker/docker-compose.test.yml up -d

# 停止并保留数据
docker compose -f test/docker/docker-compose.test.yml stop

# 停止并删除容器（保留数据卷）
docker compose -f test/docker/docker-compose.test.yml down

# 停止并删除所有（包括数据卷）
docker compose -f test/docker/docker-compose.test.yml down -v
```

### 日志查看

```bash
# 查看所有日志
docker compose -f test/docker/docker-compose.test.yml logs

# 实时跟踪日志
docker compose -f test/docker/docker-compose.test.yml logs -f llmgw

# 查看最近 100 行
docker compose -f test/docker/docker-compose.test.yml logs --tail=100 llmgw
```

### 数据库操作

```bash
# 连接到数据库
docker compose -f test/docker/docker-compose.test.yml exec postgres psql -U llmgw -d llmgw_test

# 备份数据库
docker compose -f test/docker/docker-compose.test.yml exec postgres pg_dump -U llmgw llmgw_test > backup.sql

# 恢复数据库
cat backup.sql | docker compose -f test/docker/docker-compose.test.yml exec -T postgres psql -U llmgw llmgw_test
```

---

## 健康检查

服务启动后自动进行健康检查：

- **PostgreSQL**: `pg_isready` 命令
- **LLM Gateway**: `curl -f http://localhost:8080/auth/login`

检查服务状态：

```bash
docker compose -f test/docker/docker-compose.test.yml ps
```

输出示例：
```
NAME              STATUS
llmgw-test-db     Up 5 minutes (healthy)
llmgw-test-server Up 5 minutes (healthy)
```

---

## 故障排查

### 端口冲突

```bash
# 检查端口占用
lsof -i :8080
lsof -i :5433

# 修改端口映射
# 编辑 docker-compose 文件中的 ports 配置
```

### 数据库连接失败

```bash
# 检查数据库状态
docker compose -f test/docker/docker-compose.test.yml exec postgres pg_isready

# 查看数据库日志
docker compose -f test/docker/docker-compose.test.yml logs postgres
```

### 镜像拉取失败（国内网络）

Docker Desktop 中配置代理：
1. Settings → Resources → Proxies
2. 开启 Manual proxy configuration
3. 设置 HTTP/HTTPS Proxy: `http://host.docker.internal:10809`
4. Apply & Restart

或在 Dockerfile 中使用国内镜像源：
```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
```

---

## API 演示

启动测试环境后，可以使用演示脚本快速体验 API 功能。

### 使用演示脚本

```bash
# 完整 API 流程演示（彩色输出）
./test/scripts/demo-chat.sh

# 指定用户
./test/scripts/demo-chat.sh alice

# 指定用户和消息
./test/scripts/demo-chat.sh alice "你好，介绍一下你自己"

# 指定用户、消息和会话ID
./test/scripts/demo-chat.sh alice "继续" "550e8400-e29b-41d4-a716-446655440000"
```

演示脚本执行以下操作：

| 步骤 | API | 说明 |
|------|-----|------|
| 1 | - | 生成 JWT Token |
| 2 | GET /api/models | 获取可用模型列表 |
| 3 | GET /api/quota | 获取用户 Quota |
| 4 | POST /api/chat | 发送聊天消息（非流式） |
| 5 | POST /api/chat | 发送聊天消息（流式 SSE） |
| 6 | GET /api/sessions | 获取会话列表 |
| 7 | GET /api/sessions/{id} | 获取会话详情 |

### 使用 curl 命令

```bash
# 运行 curl 命令集合
./test/scripts/curl-commands.sh
```

### 手动 curl 示例

```bash
# 设置变量
API_BASE="http://localhost:8080"
TOKEN="your-jwt-token-here"

# 1. 获取可用模型
curl -H "Authorization: Bearer $TOKEN" "${API_BASE}/api/models"

# 2. 发送聊天消息
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"mock","messages":[{"role":"user","content":"你好"}]}' \
  "${API_BASE}/api/chat"

# 3. 流式聊天
curl -N -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"mock","messages":[{"role":"user","content":"讲个笑话"}],"stream":true}' \
  "${API_BASE}/api/chat"
```

### 测试用户

| 用户 ID | Email | Quota 状态 |
|---------|-------|-----------|
| alice | alice@test.com | mock: 100万, gpt-4o: 49万 |
| bob | bob@test.com | mock: 已耗尽 |
| charlie | charlie@test.com | 无 quota |

### 生成 JWT Token

```bash
# 使用工具脚本生成
go run test/tools/sign-token.go alice

# 或使用 openssl 手动生成
header='{"alg":"HS256","typ":"JWT"}'
payload='{"sub":"alice","email":"alice@test.com","name":"Alice","exp":'$(($(date +%s)+3600))'}'
secret="test-jwt-secret-for-blackbox-testing"

h=$(echo -n "$header" | base64 | tr -d '=' | tr '/+' '_-')
p=$(echo -n "$payload" | base64 | tr -d '=' | tr '/+' '_-')
s=$(echo -n "${h}.${p}" | openssl dgst -sha256 -hmac "$secret" -binary | base64 | tr -d '=' | tr '/+' '_-')

echo "${h}.${p}.${s}"
```

---

## 生产部署建议

1. **使用 secrets 管理敏感信息**
   ```yaml
   secrets:
     jwt_secret:
       file: ./secrets/jwt_secret.txt
   ```

2. **配置 HTTPS 反向代理**
   - 使用 Nginx 或 Traefik 作为反向代理
   - 配置 SSL 证书

3. **设置资源限制**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '2'
         memory: 1G
   ```

4. **配置日志收集**
   - 使用 ELK 或 Loki 收集日志
   - 配置日志轮转

5. **数据库备份**
   - 定期备份 PostgreSQL
   - 使用云服务商的托管数据库服务