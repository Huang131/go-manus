# 集成测试

本目录包含使用独立测试资源的集成测试，这些测试：

- 连接真实的 PostgreSQL、Redis、MinIO
- 测试真实的数据库操作和文件上传
- 能够发现 Mock 测试无法覆盖的问题（如 SQL 错误、存储问题）

## 测试数据隔离方案

默认使用 `docker-compose.test.yml` 启动独立容器、网络和 volume；CI 或并行测试可通过 `API_TEST_*` 环境变量改端口或指向独立资源。测试数据使用以下隔离约束：

| 资源 | 开发用 | 测试用 | 隔离方式 |
|------|--------|--------|----------|
| PostgreSQL | `localhost:5432` / `manus` | `localhost:15432` / `manus_test` | 独立容器、端口和 volume |
| Redis | `localhost:6379` / db 0 | `localhost:16379` / db 1 | 独立容器和端口 |
| MinIO | `localhost:9000` / `go-manus-files` | `localhost:19000` / `go-manus-test-files` | 独立容器、端口和 volume |

每个测试用例使用唯一标识创建数据，并通过 `defer CleanupXxx()` 清理；测试入口会拒绝生产数据库、Redis DB 0 和非测试 bucket。

可用环境变量：

```text
API_TEST_POSTGRES_DSN
API_TEST_REDIS_ADDR
API_TEST_REDIS_DB
API_TEST_S3_ENDPOINT
API_TEST_S3_BUCKET
```

## 快速开始

### 方式一：一键运行（推荐）

```bash
cd api
make test-integration
```

自动执行：启动独立环境 → 执行迁移 → 运行集成测试 → 删除测试环境。失败时会先输出容器日志。

### 方式二：分步操作

```bash
cd api

# 1. 初始化测试环境（创建测试数据库和 bucket）
make test-up

# 2. 运行集成测试
go test -tags=integration -v ./tests/...

# 3. 删除测试容器和 volume
make test-down
```

## 前置条件

1. Docker 和 Docker Compose 已安装
2. 默认端口 `15432`、`16379`、`19000`、`19001` 未被占用，或通过 `API_TEST_*_PORT` 覆盖

## 测试覆盖

| 测试文件 | 覆盖范围 |
|---------|---------|
| `session_test.go` | Session API：创建、获取、列表、删除、分页 |
| `file_test.go` | File API：上传、下载、获取信息（真实 MinIO，含并发上传） |
| `appconfig_test.go` | AppConfig API：LLM、Agent、MCP、A2A 配置 CRUD |
| `llm_model_test.go` | LLMModel API：CRUD + UnsetDefault + 健康上报 + default 切换 + 并发 |
| `sandbox_external_test.go` | 真实沙箱协议契约（`external` tag）：Shell 执行、文件读写/查找/删除、读取截断上限、业务错误映射、浏览器截图 |
| `internal/search/search_test.go` | 真实 Tavily/Bocha 与 fallback 协议（`external` tag，由 `make test-external` 显式运行） |

## 常见问题

### Q: 测试连接失败

```bash
# 检查独立测试资源状态
docker compose -f ../docker-compose.test.yml ps

# 重新初始化测试环境
make test-up
```

### Q: 如何只运行特定测试

```bash
# 只运行 Session 测试
go test -tags=integration -v -run TestSessionAPI ./tests/...

# 只运行 File 测试
go test -tags=integration -v -run TestFileAPI ./tests/...
```

### Q: 测试环境需要重新初始化吗

仅在以下情况需要重新初始化：
- 迁移脚本有更新（新增表或字段）
- MinIO bucket 被手动删除
- 数据库结构异常

`make test-integration` 每次都会创建全新的 volume，因此不会复用上一次的数据。

## Makefile 命令说明

| 命令 | 作用 |
|------|------|
| `make test-unit` | 运行无外部依赖的普通测试 |
| `make test-component` | 启动独立环境并运行 repository component 测试 |
| `make test-api` | 启动独立环境并运行 HTTP API 集成测试 |
| `make test-integration` | 启动独立环境并运行全部内部集成测试 |
| `make test-external` | 运行显式 `external` 标签的 SenseNova 与 Search 测试 |
| `make test-sandbox` | 运行真实沙箱 external smoke，复用已启动的 sandbox 服务 |
| `make test-race` | 使用 race detector 运行普通测试 |
| `make test-up` | 启动独立测试环境并执行 migration |
| `make test-down` | 删除独立测试容器、网络和 volume |

`make test-race` 不包含 `integration` 和 `external` build tag。真实存储或 HTTP
集成链路只有在出现明确并发风险时，才应启动测试环境后对目标用例单独运行 `-race`。

## 真实沙箱外部测试

`sandbox_external_test.go`（`external` build tag）验证 Go 客户端与真实沙箱部署之间的协议契约：
响应信封解析、读取截断上限、业务错误映射、二进制截图通道。它不进 `go test ./...` 门禁，
也不使用 `docker-compose.test.yml`，而是复用开发环境里已在运行的 sandbox：

```bash
# 1. 启动沙箱（宿主机 8090 → 容器 8080）
docker compose -f ../docker-compose.yml up -d sandbox

# 2. 运行
cd api && make test-sandbox
```

沙箱不可达时测试会明确 skip。改动 `sandbox/` 的路由或服务代码后需要重建镜像，否则测试会命中旧行为：

```bash
docker compose -f ../docker-compose.yml build sandbox
docker compose -f ../docker-compose.yml up -d sandbox
```

## CI 集成

```bash
#!/bin/bash
set -e

# Makefile 负责启动、日志和清理
make test-integration
```
