# 集成测试

本目录包含使用独立测试资源的集成测试，这些测试：

- 连接真实的 PostgreSQL、Redis、MinIO
- 测试真实的数据库操作和文件上传
- 能够发现 Mock 测试无法覆盖的问题（如 SQL 错误、存储问题）

## 测试数据隔离方案

当前默认配置兼容本地 Docker；CI 或并行测试应通过 `API_TEST_*` 环境变量指向独立资源。测试数据使用以下隔离约束：

| 资源 | 开发用 | 测试用 | 隔离方式 |
|------|--------|--------|----------|
| PostgreSQL | `localhost:5432` / `manus` | `localhost:5432` / `manus_test` | 不同数据库 |
| Redis | `localhost:6379` / db 0 | `localhost:6379` / db 1 | 不同 DB 编号 |
| MinIO | `localhost:9000` / `go-manus-files` | `localhost:9000` / `go-manus-test-files` | 不同 bucket |

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

自动执行：初始化测试环境 → 运行集成测试

### 方式二：分步操作

```bash
cd api

# 1. 初始化测试环境（创建测试数据库和 bucket）
make test-up

# 2. 运行集成测试
go test -tags=integration -v ./tests/...

# 3. 清理测试数据（可选，不会删除数据库本身）
make test-down
```

## 前置条件

1. Docker 和 Docker Compose 已安装（使用默认本地测试资源时）
2. 测试资源已启动并完成迁移；默认脚本仍使用 `docker compose -f ../docker-compose.yml up -d`

## 测试覆盖

| 测试文件 | 覆盖范围 |
|---------|---------|
| `session_test.go` | Session API：创建、获取、列表、删除、分页 |
| `file_test.go` | File API：上传、下载、获取信息（真实 MinIO，含并发上传） |
| `appconfig_test.go` | AppConfig API：LLM、Agent、MCP、A2A 配置 CRUD |
| `llm_model_test.go` | LLMModel API：CRUD + UnsetDefault + 健康上报 + default 切换 + 并发 |

## 常见问题

### Q: 测试连接失败

```bash
# 检查默认本地测试资源是否运行
docker ps | grep go-manus-postgres
docker ps | grep go-manus-minio

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

否则 `manus_test` 数据库会保留，下次直接运行测试即可。

## Makefile 命令说明

| 命令 | 作用 |
|------|------|
| `make test-up` | 初始化测试环境（调用 `../scripts/test-env-up.sh`） |
| `make test-integration` | 一键运行：初始化 + 测试（依赖 `test-up`） |
| `make test-down` | 清理测试数据（调用 `../scripts/test-env-down.sh`） |

## CI 集成

```bash
#!/bin/bash
set -e

# 确保开发 Docker 运行
docker compose -f docker-compose.yml up -d

# 初始化测试数据（创建 manus_test DB 等）
make test-up

# 运行测试
go test -tags=integration -cover ./tests/... || {
    # 测试失败时保留日志
    docker compose logs
    exit 1
}

# 清理测试数据
make test-down
```
