# LLM Gateway 测试环境

本目录包含黑盒测试的 Docker 测试环境配置。

## 目录结构

```
test/docker/
├── Dockerfile                  # Go 应用构建镜像
├── docker-compose.test.yml     # 测试环境编排
├── config.test.yaml            # 测试配置文件
├── init-db.sql                 # 数据库初始化脚本
├── seed-test-data.sql          # 测试数据种子
├── setup-test-env.sh           # 环境启动脚本
└── README.md                   # 本文档
```

## 快速开始

### 1. 启动测试环境

```bash
./test/docker/setup-test-env.sh up
```

这将启动：
- PostgreSQL 数据库 (端口 5433)
- LLM Gateway 服务 (端口 8080)

### 2. 运行黑盒测试

```bash
./test/docker/setup-test-env.sh test
```

该脚本会使用当前环境的 `go` 命令运行测试（不再依赖固定路径）。

### 3. 查看日志

```bash
./test/docker/setup-test-env.sh logs
```

### 4. 停止环境

```bash
./test/docker/setup-test-env.sh down
```

## 测试账户

| 用户 ID | Email | 描述 |
|---------|-------|------|
| alice | alice@test.com | 有 mock 和 gpt-4o 的 quota |
| bob | bob@test.com | mock quota 已耗尽 |
| charlie | charlie@test.com | 无任何 quota |

## 数据库连接

```bash
# 使用 psql 连接
./test/docker/setup-test-env.sh psql

# 或直接连接
psql "postgres://llmgw:llmgw_test_password@localhost:5433/llmgw_test"
```

## 手动测试

### 获取可用模型列表

```bash
# 生成测试 JWT Token
TOKEN=$(go run -C . ./test/tools/sign-token.go alice)

# 调用 API
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/models
```

### 发送聊天请求

```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"mock","messages":[{"role":"user","content":"hello"}]}' \
  http://localhost:8080/api/chat
```

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| TEST_API_BASE | API 基础 URL | http://localhost:8080 |
| TEST_JWT_SECRET | JWT 签名密钥 | test-jwt-secret-for-blackbox-testing |
| TEST_DATABASE_URL | 测试数据库连接 | postgres://...@localhost:5433/llmgw_test |
| DATABASE_DSN | 应用数据库连接 (容器内) | postgres://...@postgres:5432/llmgw_test |

## 测试数据

初始化数据包含：
- 3 个测试用户
- 2 个用户的 quota 配置
- 3 个 mock 模型的凭证
- 3 条历史聊天记录

## 故障排查

### 端口被占用

```bash
# 检查端口占用
lsof -i :8080
lsof -i :5433

# 修改 docker-compose.test.yml 中的端口映射
```

### 数据库连接失败

```bash
# 检查数据库状态
docker compose -f test/docker/docker-compose.test.yml exec postgres pg_isready

# 查看数据库日志
docker compose -f test/docker/docker-compose.test.yml logs postgres
```

### 重置环境

```bash
# 完全清理并重建
./test/docker/setup-test-env.sh down
docker volume rm llmgw_postgres_data 2>/dev/null || true
./test/docker/setup-test-env.sh up
```