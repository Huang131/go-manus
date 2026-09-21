# API 测试方案

本文是 `api/` 服务当前测试编写、分层和重构的基线。目标不是提高测试数量或覆盖率数字，而是让每个测试守护一个清晰的业务、存储或协议契约，并能在失败时快速定位所属层。

## 1. 当前判断

当前 API 已有较好的测试基础：

- `internal/llmcore`、LLM provider、fallback、SSE payload 等纯逻辑测试较完整。
- `api/tests` 已有真实 PostgreSQL repository、MinIO 文件和 HTTP 路由测试。
- PostgreSQL 测试已经覆盖部分 JSONB、软删除、事务和唯一约束。

当前主要问题不是测试太少，而是测试层级和业务边界混杂：

- [tests/integration_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/integration_test.go:51) 使用一个 `TestMain` 同时初始化 PostgreSQL、Redis、MinIO 和 Gin 路由。
- [tool_event_flow_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/internal/agent/tool_event_flow_test.go:29) 使用内存队列替身验证 Agent flow 编排；Redis Stream 协议由独立 component 测试验证。
- [session_handler_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/internal/handler/session_handler_test.go:17) 的 Handler mock 维护完整 Session map，重实现了部分业务状态。
- [task_redis_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/internal/agent/task_redis_test.go:402) 等测试依赖固定 `Sleep` 和轮询。
- Redis component 测试已迁移到 [redis_component_integration_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/redis_component_integration_test.go:1)，使用独立 Redis 验证 `XADD/XREAD`、cursor、blocking cancel、retention 和清空；Agent 事件编排仍需后续补充续读链路。
- Chat 集成测试只覆盖 Agent 未启用时的 412，[session_routes_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/session_routes_test.go:25) 没有成功执行链路。
- [tests/README.md](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/README.md:19) 描述了不存在的 `truncateTables()`，且把 Redis 描述为已被真实测试使用，文档与实现不一致。

SenseNova 测试密钥是产品决策保留的验证资源，不列为本方案的清理项。它属于真实外部服务验证，不应被当作普通单测或本地组件测试的替代品。

## 2. 测试分层

| 层级 | 外部依赖 | 应验证的内容 | 不应验证的内容 |
|---|---|---|---|
| Unit | 无数据库、Redis、MinIO、真实 LLM | 状态迁移、配置校验、错误映射、纯编排、边界条件、内存并发 | SQL、事务、Redis 协议、真实第三方可用性 |
| Repository component | 独立 PostgreSQL | SQL、JSONB、软删除、分页、事务回滚、唯一索引、并发更新 | Handler DTO、第三方模型协议 |
| Redis component | 独立 Redis | `XADD/XREAD`、cursor、blocking cancel、retention、断线续读、重复/丢失事件 | Agent 业务状态机的全部逻辑 |
| Storage component | 独立 MinIO/S3 | 上传、下载、删除、bucket 隔离、session ownership、清理 | 业务错误补偿的全部组合 |
| HTTP integration | Gin + 真实内部依赖 + deterministic fake 外部服务 | 路由、DTO、错误码、事务边界、SSE 事件顺序、核心业务生命周期 | 真实 LLM 稳定性、第三方限流 |
| External smoke | 显式启用的 SenseNova、Search、Sandbox 等 | provider 协议兼容性、部署环境连通性 | 普通回归门禁、数据库正确性 |
| Server smoke | 独立 build tag、回环临时端口 | 真实进程启动、健康检查、优雅关闭、超时 | 大量业务场景、固定端口 |

Service 单测可以使用 Repository fake，但 fake 只能验证调用参数、调用顺序和错误传播，不能模拟 SQL 分页、事务或并发约束。真实数据库行为必须由 repository/component integration 验证。

## 3. 环境与命令

默认命令必须快速、稳定、无外部依赖：

```bash
go test ./...
go vet ./...
```

集成测试使用集中式 `internal/testsupport.IntegrationEnv`，测试函数不直接散落读取环境变量。当前已实现以下连接覆盖：

```text
API_TEST_POSTGRES_DSN
API_TEST_REDIS_ADDR
API_TEST_REDIS_DB
API_TEST_S3_ENDPOINT
API_TEST_S3_BUCKET
```

当前 [Makefile](../api/Makefile) 已提供以下分层入口：

```bash
make test-unit
make test-component
make test-api
make test-external
make test-race
```

配置加载器会校验测试资源：数据库名称必须包含独立的 `test` 段，Redis 禁止使用 DB 0，bucket 名称必须包含独立的 `test` 段。当前实现还没有消费 Redis prefix；在 prefix 真正接入读写链路前，不把它列为可用配置。CI 使用独立 service container 或每个 job 的唯一 namespace，不依赖固定开发容器名。

Phase 1 已落地独立 `docker-compose.test.yml`：不使用固定容器名，使用独立端口、网络和 volume，并在成功或失败后清理。CI 并行任务可通过 `API_TEST_COMPOSE_PROJECT` 和 `API_TEST_*_PORT` 使用唯一 namespace。启动脚本负责 migration 和 MinIO bucket 初始化；Makefile 在测试失败时输出容器日志，并始终调用 `test-down` 删除环境。

SenseNova 测试保留现有验证用途；后续若调整入口，应使用独立 external profile 和显式命令，不能让普通 `go test ./...` 触发外部调用。

## 4. 高收益测试契约

### Unit

- 当前 Session/RedisStreamTask 的状态、终态不可回改和取消优先。
- 当前消息提交、错误映射和任务生命周期边界。
- ContextBuilder 的上下文预算、tool call/result 成对裁剪。
- Agent、MCP、A2A 配置校验和 runtime snapshot。
- LLM fallback 条件、协议错误分类和工具副作用边界。
- SSE payload 合并、事件类型映射和 cursor 写入。
- 工具参数校验、shell watcher 和 goroutine 生命周期。

### PostgreSQL component

- Session 事件追加、最新消息和未读数投影。
- JSONB、软删除、分页排序、foreign key。
- transaction commit/rollback。
- LLM model 的 partial unique default 和并发切换。
- 当前 Session 事件与任务状态的持久化边界。

### Redis component

- 空 cursor、明确 cursor 和 `$` 的区别。
- blocking read 在 context cancel 后及时返回。
- stream retention、TTL、完成后缩短保留窗口。
- Last-Event-ID 续读、断线重连、重复和丢失事件边界。

### HTTP integration

- Session、文件、AppConfig、LLM model 主流程和错误码。
- Agent 已启用时的当前 Chat/Session 成功链路。
- Agent 未启用时的依赖错误链路。
- SSE 首事件、tool 事件顺序、done 事件和取消。
- 外部 LLM 使用 deterministic fake，不连接真实模型。

## 5. 应精简的测试

以下测试没有足够的运行时价值，应在确认生产引用后逐步删除或合并：

- [memory_test.go:10](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/internal/agent/memory_test.go:10) 仅验证容器 Add/Get/Clear，且 Add 只断言长度非零，没有验证内容和顺序；当前若保留 SimpleMemory，应补齐 `MergeMessages`、内容和顺序契约。
- [llm_model_test.go:223](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/llm_model_test.go:223) 的 NotFound 测试只断言非 200，应精确断言 404 和业务错误码。
- [llm_model_test.go:229](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/llm_model_test.go:229) 允许任意非 500 的宽松断言。
- [appconfig_test.go:190](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/tests/appconfig_test.go:190) 只断言空更新返回 200 的测试，需先明确业务契约。
- [tool_event_flow_test.go](/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/internal/agent/tool_event_flow_test.go:313) 使用统一 context 超时等待事件，避免额外维护固定 deadline。

不要为了减少数量直接删除仍覆盖生产语义的测试。每次删除前先确认生产调用、替代测试和当前业务契约。

## 6. 分阶段实施

每个阶段必须独立可编译、可运行测试、可回滚，并且不长期保留两套业务语义。

### Phase 0：基线

只记录测试清单、build tag、外部依赖、运行时间和已知失败，不改行为。

### Phase 1：环境入口

建立 `IntegrationConfig/TestEnv`、测试资源安全校验和 `test-unit/test-component/test-api/test-external/test-race` Makefile 目标。新增独立 Compose test profile 或 Testcontainers 启动器，替代固定开发容器名，并同步修正 `api/tests/README.md`。此阶段不删除现有测试。

### Phase 2：边界归位

把 Handler mock 改为最小 stub；把 PostgreSQL、MinIO 和 HTTP 测试按职责归类；为 Redis Stream 增加真实 component integration；删除接口断言和 mock 自测。

### Phase 3：并发稳定性

用 channel/barrier 替代固定 Sleep；测试使用独立 registry；为 registry 注销完成提供可等待信号；用 `go test -race -count=100` 验证关键生命周期测试。

### Phase 4：当前 API HTTP 契约

Run 生产代码尚不存在，本阶段不设计、不实现、也不提前编写 Run 测试。基于当前 Session/RedisStreamTask/Agent 链路，使用 deterministic fake LLM 补齐一条 HTTP Chat 成功链路：请求绑定、任务创建、事件顺序、SSE done、取消和错误映射。HTTP integration 使用真实内部依赖，外部模型保持 fake。

### Phase 5：当前低收益测试清理

当前 HTTP 契约稳定后，删除已确认无生产价值的接口断言、mock 自测、宽松断言和重复生命周期测试。暂不删除仍服务当前 Session/RedisStreamTask 生产链路的测试。

### Future R：Run 重构后的测试

Run 的生产代码重构不属于当前测试重构。等 Run spec、领域模型、migration 和生产执行链路进入实施阶段后，再单独执行：

- 冻结状态迁移、幂等、单 Session 单活跃 Run、取消竞争和 interrupted 恢复契约；
- 增加 Run repository、executor、HTTP/SSE integration；
- Run 生产语义切换完成后，再删除只服务旧 Session/RedisStreamTask 语义的测试。

在 Future R 开始前，当前测试方案不对 Run 的状态、字段或 API 做任何假设。

每阶段建议独立提交，例如：

```text
test(api): establish test layer baseline
test(api): isolate component test environments
test(api): align tests with infrastructure boundaries
test(agent): make lifecycle tests deterministic
test(api): cover the current chat lifecycle
test(api): remove low-value legacy tests
```

## 7. 完成标准与阶段归属

测试重构完成的判断依据不是总覆盖率，而是：

1. `go test ./...` 不依赖外部服务并稳定通过：Phase 0 建立基线，之后每个阶段持续验证。
2. PostgreSQL、Redis、MinIO component 测试可以独立启动和清理：Phase 1 完成后验收。
3. HTTP integration 覆盖至少一条当前 Session/Agent 完整业务链路：Phase 4 完成后验收；Run 链路不属于当前标准。
4. 真实外部服务测试显式启用、可单独运行：Phase 1 完成 external 命令后验收。
5. 当前事务、唯一约束、取消竞争和 SSE 契约明确：事务和唯一约束归 Phase 2，Redis 续读和 retention 归 Phase 2，取消竞争归 Phase 3，HTTP/SSE 业务链路归 Phase 4。
6. 测试失败能定位到 unit、repository、Redis、storage、HTTP 或 external 层：Phase 1 建立命令和目录边界后验收。

Run 的状态迁移、幂等、单活约束和 interrupted 恢复属于 Future R，不作为当前测试重构的完成标准。
