# API 测试方案

本文描述 `api/` 当前测试分层、运行入口和维护边界。它是现状基线，不是尚未实现的 Run 规划；Run/Session 重构已经完成，新增测试应围绕当前 Run API 和已证实的业务契约展开。

## 当前测试分层

| 层级 | 依赖 | 主要验证 | 入口 |
|---|---|---|---|
| Unit | 无数据库、Redis、MinIO 或真实第三方服务 | 状态迁移、配置校验、错误映射、纯编排、工具参数、消息和上下文边界 | `cd api && make test-unit` |
| PostgreSQL component | 独立 PostgreSQL | SQL、JSONB、事务、分页、软删除、唯一索引、Run/Message 并发约束 | `cd api && make test-component` |
| Redis component | 独立 Redis | Stream 游标、exclusive read、blocking cancel、retention、清理 | `cd api && make test-component` |
| Storage component | 独立 MinIO | 文件上传、下载、删除、bucket 和 session ownership | `cd api && make test-component` |
| HTTP integration | Gin、真实内部存储、deterministic fake LLM | Run 创建、等待输入、取消、错误码、SSE 顺序和续读 | `cd api && make test-api` |
| External smoke | 显式 external tag、真实 SenseNova/Search/Sandbox | 第三方协议和部署连通性 | `make test-external`、`make test-sandbox` |

Unit 测试可以使用最小 fake 验证调用参数和错误传播，但不得在 fake 中重实现 Repository、Run 状态机或 Redis 协议。SQL、事务和并发约束必须由真实组件测试证明。

## 当前核心契约

### Run 生命周期

真实 PostgreSQL/Redis HTTP 测试应持续守护以下行为：

- 创建 Run 与初始用户消息使用同一幂等请求语义。
- 同一 Session 不能同时存在两个活跃 Run。
- `waiting_input` 保存完整执行快照；提交回答继续原 Run，不能重新规划或重复注入回答。
- `cancelling` 先落库，Engine 退出并释放 ToolSet 后才进入 `cancelled`。
- 迟到的成功或失败不能覆盖取消终态；重复取消和重复输入具有稳定结果。
- PostgreSQL 保留最终 Run、Message、Plan/执行快照；Redis 丢失只影响短期事件读取，不影响终态查询。

### SSE

`GET /api/runs/:runId/events` 使用 Redis Stream ID 或等价游标，并接受 `Last-Event-ID`：

- 从游标之后读取，不跳过紧随游标的事件。
- 重放保留原事件 ID；客户端按 `run_id + event_id` 去重。
- 终止事件为 `done`、`error` 或 `wait`，最终状态以 `GET /api/runs/:runId` 为准。
- 游标过期返回明确错误，客户端回查 Run 快照，不伪造逐 token 历史。

当前真实 HTTP 用例位于 `api/tests/chat_lifecycle_test.go`，覆盖成功事件顺序、取消和 `Last-Event-ID` 断线续读的重复/丢失边界。

## 环境与命令

普通测试不读取外部资源：

```bash
cd api
make test-unit       # go test ./...
go vet ./...
make test-race       # 普通测试 race detector
```

组件和 HTTP 测试使用 `docker-compose.test.yml` 的独立 PostgreSQL、Redis、MinIO，由 `test-up` 启动、迁移和初始化，由 `test-down` 清理。测试配置集中由 `internal/testsupport.IntegrationEnv` 读取，并允许环境变量覆盖：

```text
API_TEST_POSTGRES_DSN
API_TEST_REDIS_ADDR
API_TEST_REDIS_DB
API_TEST_S3_ENDPOINT
API_TEST_S3_BUCKET
```

加载器拒绝生产数据库、Redis DB 0 和非测试 bucket。CI 或并行任务应使用不同端口、数据库/bucket 名称或 compose project，不依赖开发容器名。

常用命令：

```bash
cd api
make test-component   # Repository、Redis、Storage component
make test-api         # Run/Session/File/LLM 等 HTTP integration
make test-integration # 全部内部 integration tag 测试
make test-external    # SenseNova、Search；显式外部服务
make test-sandbox     # 真实 Sandbox smoke；需要 sandbox 已启动
make test-down        # 清理独立测试环境
```

失败时 `test-component`、`test-api` 和 `test-integration` 会输出 compose 日志并清理环境。外部服务测试不是普通回归门禁；SenseNova 测试凭据是产品保留的验证资源，运行前必须显式选择 external 入口。

## 测试文件职责

- `api/internal/model`、`api/internal/service`、`api/internal/agent`：状态、恢复、Engine/ToolSet 生命周期和纯业务边界。
- `api/internal/handler`：请求绑定、分页、稳定错误映射和 Run DTO 契约。
- `api/tests/*_repository_integration_test.go`：PostgreSQL/Repository 行为和事务。
- `api/tests/redis_component_integration_test.go`：真实 Redis Stream 协议。
- `api/tests/chat_lifecycle_test.go`：真实 HTTP Run 成功、取消和 SSE 续读。
- `api/tests/file_test.go`、`session_test.go`、`llm_model_test.go`、`appconfig_test.go`：HTTP 资源和配置生命周期。
- `api/tests/*_external_test.go`、`internal/search/*_test.go`：真实外部服务 smoke。

## 维护规则

新增测试前先回答三个问题：生产代码是否消费该行为、现有测试是否已有更高层覆盖、失败后能否定位到一个明确契约。不能仅为 getter、接口实现断言、mock 自身固定返回值或覆盖率数字增加测试。

并发测试必须使用完成 channel、context 或 barrier 等待生命周期结束，禁止用固定 `Sleep` 和包级全局 registry 轮询作为同步手段。涉及 channel/goroutine 的改动至少运行：

```bash
cd api
go test -race -count=100 -run 'Test目标名称' ./internal/agent/...
```

HTTP 测试可以使用 `httptest` 的回环临时端口；规则禁止的是外部网络、固定端口和真实第三方调用。真实第三方调用只能出现在 `external` build tag 测试中。

## 完成基线

每次 API 代码变更至少验证：

```bash
cd api
go test ./... -count=1
go vet ./...
git diff --check
```

涉及存储、Run、SSE 或取消时，再运行对应的 `make test-component`、`make test-api` 或 `make test-integration`。涉及 goroutine、ToolSet、Redis Stream 生命周期时增加定向 race 重复测试。

当前不预先引入不存在的 Worker、Lease、Outbox 或永久事件溯源测试；只有生产契约落地后，才为新行为增加对应测试。
