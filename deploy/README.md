# 前端最小部署（本地点击测试）

## 你要访问的地址

| 方式 | 地址 | 说明 |
|------|------|------|
| **全栈 Docker（推荐）** | **http://localhost:3000** | 根目录 `docker-compose.yml`：Postgres + API + 前端 Nginx |
| 仅前端容器 + 本机 API | **http://localhost:3000** | `frontend-compose.yml`：Nginx 反代到宿主机 **8080** |
| 纯开发服务器 | **http://localhost:5173** | `cd web && npm run dev`，反代到本机 **8080**（见 `vite.config.ts`） |

**前提**：除「全栈 Docker」外，需本机 **8080** 已有 API。若改端口，同步改 `web/vite.config.ts` 的 `proxy`，或改 `deploy/nginx.frontend.conf` / `deploy/nginx.frontend.docker.conf`。

## Docker 一键

### 全栈（数据库 + API + 前端静态页）

仓库根目录：

```bash
docker compose up --build -d
```

浏览器打开 **http://localhost:3000**。需能拉取 `node`、`nginx` 等基础镜像（若 Docker Hub 超时，可先只起 API：`docker compose up -d postgres llmgw`，再用本机 `npm run dev` 访问 **http://localhost:5173**）。

### 仅前端容器（API 跑在宿主机 8080）

```bash
docker compose -f frontend-compose.yml up --build
# 或
docker compose -f deploy/docker-compose.frontend.yml up --build
```

浏览器打开 **http://localhost:3000**。构建参数含 `VITE_ENABLE_DEV_LOGIN=true`。

## 不用 Docker（更轻）

```bash
cd web
npm ci
# 可选：启用开发登录表单（与 Docker 构建参数一致）
export VITE_ENABLE_DEV_LOGIN=true
npm run dev
```

浏览器打开 **http://localhost:5173**。

## 联调说明

- 前端请求走同源 `/api`、`/auth`，由 Vite 或 nginx 转到后端，避免浏览器 CORS。
- 数据库、JWT、模型数据等仍由后端与配置文件决定；仅前端容器无法代替后端。
