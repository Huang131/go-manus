# 阶段 4：后端生产执行语义切换

## 目标

这是唯一的生产业务语义切换点。把创建、继续、等待、完成、失败、取消和重启中断全部切到 RunService + 进程内 RunExecutor。切换完成后 PostgreSQL 的 Run 是执行状态唯一事实来源，Session 只保存对话容器信息，Redis 只保存 `run:{runID}:events` 短期流。

本阶段保留旧 `/sessions/:id/chat` 和 `/stop` 路由以便现有 UI 继续工作，但它们必须是 RunService 的薄适配器，不得创建 RedisStreamTask 或更新 `sessions.status/events/memories/task_id`。

## 非目标

- 不改 UI 和最终公开 API 路径；这些在阶段 5 完成。
- 不删除旧 Task/Registry 源文件；只让生产装配不再引用，阶段 6 再物理删除。
- 不做多实例调度、自动恢复或 lease。
- 不同时写 Session 和 Run 的执行状态。

## 前置条件

- 阶段 2、3 均为 complete，提交 SHA 已记录在 `STATUS.md`。
- Run/Message migration 已应用，仓储集成测试通过。
- 开始前确认 `git grep 'ErrWaitForUser'` 无生产引用，Run 条件迁移和部分唯一索引已存在。

## 文件清单

创建：

- `api/internal/service/run_service.go`：创建、查询、输入、取消、列举 Run 的唯一用例入口。
- `api/internal/service/run_executor.go`：进程内调度、runID->cancel/engine 注册、outcome 消费和状态提交。
- `api/internal/service/run_event_stream.go`：Run stream 命名、写入、订阅和 TTL 的窄接口。
- `api/internal/agent/engine.go`：Run 级 Engine facade，内部使用 PlanSnapshot/StepSnapshot；不包含持久化逻辑。
- `api/internal/service/run_service_test.go`、`run_executor_test.go`、`run_event_stream_test.go`。

修改：

- `api/internal/agent/planner_react_flow.go`：为新 Engine 提供开始/继续和只读 PlanSnapshot；保持现有拆分后的状态处理函数。旧 Task 所需 legacy 类型留在不可达旧文件中，阶段 6 删除。
- `api/internal/agent/tool_provider.go`：`Tools` 接收本 Run 的 Settings 快照，并用 `MaxSearchResults` 构造 SearchTool，避免热更新改变运行中的 Run。
- `api/internal/agent/session_runtime.go`：替换为只负责附件、沙箱和文件能力的运行上下文；删除 Session 状态/事件写入。
- `api/internal/external/message_queue.go`、`message_queue_redis.go`：增加读取现有 Run stream、TTL 和删除所需的最小接口；删除 Task 语义别名可留到阶段 6。
- `api/internal/service/session_service.go`：Session 详情/列表的兼容响应从最新 Run 和 messages 投影 status/events，禁止读取旧执行字段作为事实来源。
- `api/internal/handler/session_handler.go`：Chat/Stop/Delete 改为调用 RunService；Chat 新消息根据活跃 Run 状态选择 CreateRun 或 SubmitInput，并订阅对应 Run stream。
- `api/internal/bootstrap/app.go`：构造 RunService、RunExecutor、RunEventStream，启动中断清理；生产装配不再创建 AgentService/RedisStreamTask。
- `api/internal/router/router.go`：本阶段路由地址不变，只替换 Handler 依赖。
- 相关 Handler、bootstrap、SessionService 测试。

不修改 `sessions` 表结构；阶段 6 再删列。

## 服务契约

```go
type CreateRunCommand struct {
    SessionID     string
    Message       string
    Attachments   []string
    ModelID       string
    IdempotencyKey string
}

type RunService interface {
    Create(context.Context, CreateRunCommand) (*model.Run, error)
    SubmitInput(context.Context, string, string, []string) (*model.Run, error)
    Cancel(context.Context, string) (*model.Run, error)
    Get(context.Context, string) (*model.Run, error)
    GetActiveBySession(context.Context, string) (*model.Run, error)
    ListBySession(context.Context, string, int, int) ([]*model.Run, int, error)
}

type RunDispatcher interface {
    Start(context.Context, string) error
    Resume(context.Context, string, model.Message) error
    Cancel(string)
    Shutdown(context.Context) error
}
```

Handler 依赖 RunService 接口，不依赖 Agent Engine 或 Repository。RunExecutor 依赖 RunRepository、MessageRepository、RunEventStream、EngineFactory 和 service.AgentSettingsManager；Engine 不得反向调用 RunService。

## 创建、继续和状态提交

Create 使用 RunUnitOfWork 在一个事务中完成：

1. 校验 Session 存在且未删除、消息和附件合法。
2. 若 `Idempotency-Key` 已存在，返回原 Run，不重复消息或派发。
3. 读取 AgentSettings、Prompt hash、model ID 的完整快照。
4. 创建 pending Run 和首条 user message；唯一索引冲突映射为 409。
5. 提交后调用 Dispatcher.Start。派发失败时用 expected pending 条件迁移为 failed；不能留下假 running。

Executor 启动后以 `pending -> running` 条件迁移取得执行权，加载用户消息/附件并创建一个 Engine。Engine 每次产生 Plan/Step 变化时更新 Run 当前 Plan 和 revision，同时发布 Redis 事件。终态提交顺序为：先持久化最终 assistant message/Plan，再以 expected running 更新 succeeded/failed，最后发布 done/error 事件并缩短 TTL。

waiting_input 时保存包含问题的 Plan 快照，再 `running -> waiting_input`，最后发布 message/wait 事件。SubmitInput 仅接受 waiting_input：在事务内写 user message并 `waiting_input -> running`，提交后 Resume 同一个进程内 Engine。若 Engine 注册项丢失，状态改为 interrupted 并返回可重试业务错误，不尝试从 Redis 重建执行内核。

## 取消优先与并发

- Cancel 接受 pending/running/waiting_input，用各自 expected status 条件迁移到 cancelled，然后调用内存 cancel。
- Executor 写 succeeded/failed 时必须带 expected running；更新 0 行后重读，若已 cancelled/interrupted 立即丢弃迟到终态。
- 注册表只保存运行期控制句柄，不保存业务状态；读请求永远查 PostgreSQL。
- 删除 Session 时先取消其活跃 Run，再软删 Session；两步中的取消失败必须阻止删除，避免孤儿执行继续消耗资源。
- App 启动、接收流量前执行 `MarkActiveInterrupted`；App 关闭按设置的超时取消所有执行，再把仍活跃 Run 标为 interrupted。

## Redis Run Event Stream

- key 固定为 `run:{runID}:events`，禁止包含 sessionID 或 taskID。
- 事件内容沿用当前 `model.Event` 协议，Redis Stream ID 作为 SSE id。
- 运行期 TTL 24h，每次写刷新；终态后 TTL 30min。
- Redis 写失败不回滚 PostgreSQL 业务状态，但记录结构化错误；客户端可在阶段 5 通过快照恢复。
- 不把 Redis 游标写入 Run/Message 表，不把 Redis 当状态查询来源。

## 旧 API 薄适配规则

`POST /sessions/:id/chat`：

- 有 message 且无活跃 Run：调用 Create。
- 有 message 且活跃 Run 为 waiting_input：调用 SubmitInput。
- 有 message 且活跃 Run 为 pending/running：返回 409。
- 无 message：订阅该 Session 的最新活跃 Run；没有活跃 Run时保留心跳兼容。

`POST /sessions/:id/stop` 查活跃 Run 后调用 Cancel。`GET /sessions/:id` 和 Session 列表中的旧 status/events 只是从 Run/Message 组装的兼容 DTO，不能写回 Session 表。该读适配会在阶段 5 随 UI 切换删除。

旧 UI 仍识别 `completed`，新 PlanSnapshot/StepSnapshot 使用 `succeeded`。兼容 chat Handler 在输出旧 SSE 时只做 wire 映射 `succeeded -> completed`；Redis Run stream 和 PostgreSQL 始终保存新状态。阶段 5 UI 切换后删除这层映射。

## 实施顺序与提交检查点

本阶段较大，拆为三个必须按序完成的检查点。每个检查点都必须编译并通过相关测试；只有 C 完成后才允许标记阶段 complete。

### A. 新执行组件（尚未接生产）

实现 Engine facade、RunEventStream、RunService、RunExecutor 和纯单元测试。bootstrap/Handler 仍使用旧路径，因此没有业务双写。提交建议：`feat(api): add run execution services`。

### B. 原子切换生产装配

在同一个提交中修改 bootstrap、SessionHandler、SessionService：停止构造旧 AgentService，将旧路由全部委托 RunService，并删除所有生产 `UpdateStatus/AppendEvent/SaveMemory/TaskID` 调用。提交建议：`refactor(api): switch execution state to runs`。这个提交是唯一切换边界，不允许拆成一个双写中间提交。

### C. 并发、重启与回归加固

增加取消竞态、幂等、waiting resume、启动 interrupted、删除 Session 的测试，修复但不改变契约。提交建议：`test(api): cover run lifecycle cutover`。

## 针对性测试

- Create：Run + user message 原子成功；仓储/派发失败路径状态正确；同 key 重试不重复。
- 并发：同 Session 两次创建只有一个成功；cancel 与 LLM 成功并发时最终必为 cancelled 或已先成功的合法终态，不出现回改。
- waiting：问题和 Plan 先持久化，输入只在 waiting 接受，resume 后能完成。
- restart/shutdown：遗留 pending/running/waiting 全变 interrupted；终态保持不变。
- stream：key/TTL/游标正确，Redis 失败不改变 Run 终态。
- compatibility：旧 chat/stop 只调用 RunService；Session 表 mock 断言 `UpdateStatus/AppendEvent/SaveMemory` 调用次数为 0。
- 删除 Session：活跃 Run 成功取消后才能软删。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service ./internal/agent ./internal/handler ./internal/bootstrap -count=1
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
rg -n 'UpdateStatus|AppendEvent|SaveMemory|getOrCreateTask|defaultTaskRegistry|taskBySession|NewRedisStreamTask|NewAgentService' internal --glob '*.go'
cd ..
git diff --check
```

`rg` 可以命中阶段 6 待删的旧实现和测试，但不得命中 bootstrap、handler、RunService、RunExecutor 的生产调用。必须人工记录允许命中的文件清单到 `STATUS.md`。

## 回滚

部署前保留阶段 A 提交。若 B/C 尚未产生用户 Run 数据，可按 C、B 逆序 `git revert`，旧 Session 语义恢复。若已产生 Run 数据，开发阶段允许破坏性回滚：停止服务，清空 `messages/runs`，revert C/B，重新初始化开发数据；禁止把 Run 状态反向同步到 Session 以制造兼容双写。阶段 A 可以保留或独立 revert。

## 完成定义

- 所有生产执行状态只写 runs，消息只写 messages，实时事件只写 Run stream。
- bootstrap 不构造 AgentService/RedisStreamTask；旧路由仅薄委托。
- Session 表执行字段即使仍存在也不再读写为业务事实。
- 幂等、取消优先、waiting resume、重启 interrupted 测试通过。
- 全量 test、race、vet 通过，三个检查点 SHA 写入 `STATUS.md`。

## 失败或中断恢复

```bash
git log -6 --oneline
sed -n '1,180p' docs/refactor-run/STATUS.md
rg -n 'NewRedisStreamTask|NewAgentService|RunService|RunExecutor' api/internal/bootstrap api/internal/handler
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service ./internal/handler ./internal/bootstrap -count=1
```

如果中断在检查点 B 的未提交工作中，以“bootstrap 是否仍构造旧 AgentService”判断唯一语义。不要让两条生产链同时可达；选择完成 B 或恢复到 A 后重新切换。
