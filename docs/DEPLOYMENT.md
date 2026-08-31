# 部署指南

## 1. 部署方式

统一使用 [docker-compose.yml](docker-compose.yml)，通过 nginx 网关统一接入前端、API 和沙箱服务，适用于标准部署、演示和小规模生产。

## 2. 架构总览

```
浏览器 ──► nginx (80) ──┬─► /api/      ──► api:8080 (Go)
                        ├─► /sandbox/  ──► sandbox:8080 (Python)
                        └─► /          ──► ui:3000 (Next.js)
                                   │
                    api ──► postgres / redis / minio
```

| 容器 | 内部端口 | 外部端口 | 说明 |
|------|---------|---------|------|
| nginx | 80 | 80 | 统一网关，路由 `/api/`、`/sandbox/`、`/` |
| ui | 3000 | 3000 | Next.js 前端 |
| api | 8080 | 8080 | Go API 服务 |
| sandbox | 8080 | 8090 | Python 沙箱服务（Shell / 文件） |
| postgres | 5432 | 5432 | PostgreSQL 16 |
| redis | 6379 | 6379 | Redis 7（含消息队列） |
| minio | 9000/9001 | 9000/9001 | 对象存储（S3 兼容，本地替代 COS） |

## 3. 前置要求

- Docker >= 20.10
- Docker Compose >= 2.0
- 至少 4GB 内存
- `docker-compose.yml` 中 `sandbox` 使用 `privileged: true` 并挂载 Docker Socket，请确认宿主 Docker 已启动

## 4. 快速启动（nginx 网关模式，推荐）

### 4.1 准备环境变量

```bash
cd go-manus

# 复制环境变量模板
cp api/.env.example .env

# 编辑 .env，至少配置 LLM 相关变量（见第 5 节）
vim .env
```

### 4.2 一键启动

```bash
# 构建并启动所有服务（含 nginx 网关）
docker-compose up -d --build

# 查看服务状态（等待全部 healthy）
docker-compose ps
```

等待 `postgres`、`redis`、`minio`、`api`、`ui` 全部变为 `healthy`，`nginx` 变为 `Up`。

### 4.3 访问入口（统一走 nginx）

| 入口 | 说明 |
|------|------|
| `http://localhost/` | 前端 UI（nginx 反代到 ui:3000） |
| `http://localhost/api/...` | API 接口（反代到 api:8080） |
| `http://localhost/sandbox/...` | 沙箱服务（反代到 sandbox:8080） |
| `http://localhost/health` | API 健康检查（经网关） |

> **⚠️ 重要提示**：浏览器必须通过 **80 端口 nginx**（`http://localhost/`）访问。若直连 `http://localhost:3000` 或 `http://localhost:8080`，前端请求的 `/api` 相对路径缺少网关前缀，会出现 404。

## 5. 环境变量配置

### 5.1 `.env` 模板（`api/.env.example`）

```bash
# 数据库（docker-compose 内会覆盖为容器地址）
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DATABASE=manus

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=

# COS 对象存储（本地用 MinIO，生产用腾讯云 COS）
COS_ENDPOINT=http://localhost:9000
COS_SECRET_ID=minioadmin
COS_SECRET_KEY=minioadmin
COS_REGION=us-east-1
COS_BUCKET=go-manus-files

# 服务配置
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
```

### 5.2 必须在 `.env` 中补充的变量

以下变量由 [docker-compose.yml](docker-compose.yml) 从 `.env` 读取并注入 API 容器，**按需填写**：

```bash
# LLM 配置（必需，否则 Agent 无法调用模型）
LLM_BASE_URL=https://api.openai.com   # 或其他 OpenAI 兼容 API
LLM_API_KEY=your_api_key
LLM_MODEL_NAME=gpt-4

# 可选：覆盖默认端口
API_PORT=8080
UI_PORT=3000
SANDBOX_PORT=8090
NGINX_PORT=80
POSTGRES_PORT=5432
REDIS_PORT=6379
```

> 说明：docker-compose 内 API 容器地址固定为 `postgres` / `redis` / `minio` / `sandbox`（容器间通信），`.env` 中的 `localhost` 地址仅用于本地直接运行时。

## 6. 常用运维命令

### 6.1 通过 Makefile

```bash
make up              # 启动所有服务（含 nginx 网关）
make down            # 停止所有服务
make down-v          # 停止并删除数据卷（慎用，清空数据）
make ps              # 查看服务状态
make health          # 检查 API /health、/api/status + 容器状态
make logs            # 查看所有日志
make logs-api        # API 日志（同理 logs-ui / logs-sandbox / logs-nginx / logs-postgres / logs-redis）
make build           # 构建镜像
make restart-api     # 重启 API（同理 restart-ui / restart-sandbox）
make rebuild-api     # 重建并启动 API（同理 rebuild-ui / rebuild-sandbox）
make shell-api       # 进入 API 容器
make shell-postgres  # 进入 PostgreSQL 命令行
make clean           # 清理未使用的 Docker 资源
```

### 6.2 直接使用 docker-compose

```bash
docker-compose up -d --build
docker-compose down
docker-compose logs -f api
docker-compose restart nginx
```

## 7. 健康检查

| 检查项 | 命令 | 预期 |
|--------|------|------|
| API 健康检查 | `curl http://localhost:8080/health` | 200，返回 JSON |
| API 状态 | `curl http://localhost:8080/api/status` | 200，返回 JSON |
| 经网关访问 | `curl http://localhost/api/sessions` | 200，返回会话列表 |
| UI | `curl -I http://localhost/` | 200 |

容器级健康检查在 [docker-compose.yml](docker-compose.yml) 中定义：

- `postgres`：`pg_isready`
- `redis`：`redis-cli ping`
- `minio`：`mc ready local`
- `api`：`wget /health`
- `ui`：`wget http://127.0.0.1:3000`

## 8. 数据持久化

数据卷定义于 [docker-compose.yml](docker-compose.yml)：

| 卷 | 挂载点 | 说明 |
|----|--------|------|
| `go-manus-postgres-data` | `/var/lib/postgresql/data` | PostgreSQL 数据 |
| `go-manus-redis-data` | `/data` | Redis AOF 持久化 |
| `go-manus-minio-data` | `/data` | MinIO 对象数据 |

```bash
# 备份 PostgreSQL（进入容器后执行）
docker exec -it go-manus-postgres pg_dump -U postgres -d manus > backup_$(date +%Y%m%d).sql
```

## 9. 安全与网络

- 所有容器位于 `go-manus-network` 桥接网络，容器间通过服务名通信。
- `sandbox` 以 `privileged: true` 运行并挂载 `/var/run/docker.sock`（支持 Docker in Docker），仅用于可信环境。
- 如需 HTTPS，在 [docker-compose.yml](docker-compose.yml) 中取消 `nginx` 的 `443` 端口注释并挂载证书，同时在 `nginx/conf.d/default.conf` 中配置 TLS。
- 生产环境应将 `minio` 的默认账号 `minioadmin/minioadmin` 与 `.env` 中的 COS 密钥替换为强密码。

## 10. 故障排查

### 10.1 常见问题

| 问题 | 解决方案 |
|------|---------|
| 前端页面 404 | 确认通过 `http://localhost/`（80 端口）访问，不要直连 3000/8080 |
| `/api/sessions/stream` 404 | 前端 `NEXT_PUBLIC_API_BASE_URL` 必须为 `/api`（见 ui/Dockerfile），不要写成绝对地址 |
| nginx 502 Bad Gateway | 服务重启后 nginx 缓存的 upstream IP 可能过期，执行 `docker-compose restart nginx` |
| API 无法连接数据库 | 查看 `docker-compose logs api`，确认 `postgres` 已 healthy |
| 容器一直重启 | 查看 `docker-compose ps` 的健康状态，`docker-compose logs <service>` 定位原因 |
| LLM 调用失败 | 确认 `.env` 中 `LLM_API_KEY` / `LLM_BASE_URL` / `LLM_MODEL_NAME` 已正确配置并 `docker-compose up -d api` 重启 |

### 10.2 日志查看

```bash
make logs          # 全部服务
make logs-api      # 仅 API
make logs-nginx    # 仅 nginx
docker-compose logs -f nginx | grep -i error   # nginx 错误
```

### 10.3 完全重置

```bash
# 停止并删除所有容器 + 数据卷（会清空数据库、Redis、MinIO 数据）
docker-compose down -v

# 重新部署
docker-compose up -d --build
```
