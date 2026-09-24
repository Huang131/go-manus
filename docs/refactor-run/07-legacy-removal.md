# 阶段 6：遗留执行基础设施与 Session 执行字段清理

> 当前源码校准（2026-09）：本阶段尚未具备删除条件。`AgentService`、`RedisStreamTask`、全局 registry、Session 执行字段和旧 SSE 路由仍在生产路径使用；只有阶段 4/5 完成并通过引用扫描后，下面的删除清单才可执行。

## 目标

在阶段 5 的 Run API/UI 已稳定运行后，物理删除不可达的旧任务模型、全局任务注册表和 Session 执行字段，收紧包依赖并更新架构文档。当前不存在独立的 Session 记忆持久化接口，消息持久化由阶段 3 的 MessageRepository 承担。完成后代码库只保留一套 Run 业务语义。

## 非目标

- 不改变 Run API、状态机、SSE 事件类型和 UI 行为。
- 不迁移目录到 domain/application/adapter/transport。
- 不删除仍被文件、VNC、LLM 模型管理使用的 Session 元数据。
- 不删除 `docs/重构.md` 等历史提案；在文档顶部标记其历史属性，并以本目录和 `STATUS.md` 为实施依据。

## 前置条件与恢复基线

- 阶段 5 complete，`STATUS.md` 有 API/UI 提交和 lint/build/test 结果。
- `rg` 确认生产代码无旧 chat/stop、Session.status 写入和 `NewRedisStreamTask` 调用。
- 开始前在开发数据库备份 sessions 表；本阶段允许破坏式 migration，不考虑历史兼容。

## 删除清单

删除源码：

- `api/internal/agent/task.go`：`Task`、`Stream` 接口。
- `api/internal/agent/task_redis.go`：`TaskStream`、`RedisStreamTask`、`DefaultTaskRegistry` 及相关注册表逻辑。
- `api/internal/agent/task_runner.go`：旧 Runner。阶段 4 的 `agent.Engine` 已承接仍需保留的 Flow 驱动逻辑，本阶段直接删除该文件。
- `api/internal/agent/service.go`：仅在阶段 4 已从生产装配移除、阶段 5 旧路由已删除且 Run 链路回归通过后，删除整个文件及专用测试。
- `api/internal/agent/deps.go` 中旧 Repository 聚合和 TaskMessageQueue 字段；保留 EngineFactory 真正消费的能力接口，并移到使用方附近。
- `api/internal/agent/session_runtime.go` 中 Session 状态/事件更新方法；保留文件能力协作者或并入 RunExecutor。

删除或收敛接口：

- `agent.Service` 中 `taskBySession`、`GetActiveTaskID`、`StopSession`、`CleanupCompleted` 等任务注册表方法。
- `AgentTaskRunner.Done/GetStatus/GetPlan`，以及只为这些方法存在的 Flow 投影函数。
- `external.TaskMessageQueue` 别名和仅服务旧 Task 的队列方法；保留 RunEventStream 所需最小 Redis 接口。
- `SessionRepository.UpdateStatus/AppendEvent`；当前仓储没有 `GetMemory/SaveMemory` 方法，保留消息、标题、未读数、软删除和文件查询。

删除模型字段：

- `model.Session.Status`、`Events`、`TaskID`。`model.Session` 当前没有 `Memories` 字段。`SandboxID` 保留为 Session 级资源关联，因为现有 VNC 与文件工具仍按 Session 定位沙箱。
- `model.ExecutionStatus` 及 legacy `Plan`/`PlanStep`。Run 继续使用阶段 3 的 `PlanSnapshot`/`StepSnapshot` 与各自状态类型。
- `model.BaseEvent.ToJSON`、`toJSON`、model 对 `sonic` 和 logger 的依赖；事件序列化移到 external/handler 边界。

删除数据库列与仓储方法：

- 不新增或迁移不存在的 `sessions.memories` 列；当前仓储没有 `GetMemory/SaveMemory` 方法。消息持久化由阶段 3 的 MessageRepository 负责。
- `sessions.status/events/task_id` 列以及对应的状态、事件和任务查询/更新 SQL。

删除配置与路由：

- `model.AgentConfig` 的残留测试和文档引用；旧 `/api/app-config/agent` 生产路由已在阶段 5 删除。
- 任何 Session 状态投影、task_id 过滤和 session events JSON 编解码。

## 数据库破坏式收敛

创建 `api/migrations/006_drop_session_execution_columns.sql`，在开发阶段执行：

```sql
ALTER TABLE sessions
    DROP COLUMN IF EXISTS task_id,
    DROP COLUMN IF EXISTS events,
    DROP COLUMN IF EXISTS status;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_task_id;
```

`sandbox_id` 本轮保留，并在 `Session` 中标注为纯资源关联，不代表执行生命周期。migration 的 down 仅用于本地回滚，开发阶段可以重建 sessions 表；禁止新增触发器把 Run 状态写回 Session。

## 包边界收紧

最终允许的依赖：

```text
handler -> service -> model
service -> repository + agent.Engine
agent -> model + external capability interfaces
repository -> model + infrastructure
bootstrap -> concrete constructors
model -> stdlib only
```

用 `go list -deps` 或静态搜索确认：model 不导入 logger/sonic/repository/external；agent 不导入 service/repository；handler 不直接操作 Repository；全局可变任务 registry 不存在；RunService 是唯一状态写入口。

## 实施顺序与提交检查点

1. 用静态搜索再次确认阶段 4 已删除旧 Task 的全部生产引用；缺失能力从现有 RunService/RunExecutor/Engine 接口补齐。
2. 删除 task.go/task_redis.go/task_runner.go，更新 bootstrap 和测试；提交 A：`refactor(api): remove legacy task infrastructure`。
3. 删除 Session 执行字段和仓储方法，新增 migration；在空数据库和备份数据库各执行一次；提交 B：`refactor(api): make session a conversation container`。
4. 移除 model 的 JSON/logger 依赖和旧 ExecutionStatus，更新事件序列化边界；提交 C：`refactor(api): tighten model package boundary`。
5. 更新本方案集的 `01-architecture.md`、`README.md`、`STATUS.md`，并同步更新 `docs/ARCHITECTURE.md` 和 `docs/README.md` 中的当前实现与导航。

每个检查点都必须形成可编译、可测试、可独立 revert 的状态。A/B/C 之间不允许恢复旧业务写路径；回滚只恢复代码/表结构，不恢复 Session/Run 双写。

## 针对性测试

- 编译引用扫描：`go test ./...` 确认无 Task/ExecutionStatus 残留。
- Session Repository：只测试会话元数据、未读数、标题、软删除和文件关联。
- Run 生命周期回归：创建、等待、输入、取消、终态、SSE 快照在删除旧设施后行为不变。
- migration：新数据库执行全部 migration；旧开发库执行 006 后 Session 查询和 Run 查询均正常。
- model 边界：`go list -deps ./internal/model` 输出不包含 sonic/logger。
- race/vet：确认注册表删除后没有共享可变状态回归。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
if go list -deps ./internal/model | rg -q 'sonic|logger'; then exit 1; fi
if rg -q 'internal/(service|repository)' internal/agent --glob '*.go'; then exit 1; fi
rg -n 'RedisStreamTask|TaskRegistry|taskBySession|SessionStatus|ExecutionStatus|ErrWaitForUser|UpdateStatus|AppendEvent|\.Events|\.TaskID' internal ../ui/src
cd ../ui
npm run lint
npm run build
cd ..
git diff --check
```

最终 `rg` 允许只命中历史文档、阶段方案和明确迁移测试；生产 Go/TS 文件不得命中。命令中的 `go list` 检查应以退出码 0 表示无匹配，执行模型需保存实际输出。

## 回滚

- A：`git revert <A>`，恢复旧文件但不恢复旧路由或 Session 写入；若编译依赖已删，必须同时 revert 依赖提交。
- B：停止服务，执行本地 down/rebuild sessions，再 `git revert <B>`；Run 表和消息表保留或清空均可，不能向 Session 回填状态。
- C：`git revert <C>` 恢复 model 序列化依赖；Run 业务语义保持不变。

任何回滚都要重新运行 API 全量 test/race/vet 和 UI lint/build。开发阶段不做在线兼容迁移，确保回滚边界清楚。

## 完成定义

- 生产代码无 Task/Stream/RedisStreamTask、全局任务 registry、Session 执行字段；不存在的 `memories` 伪链路不应重新引入。
- model 仅依赖标准库，RunService 是状态唯一写入口。
- 开发数据库已应用 006，Run/Message 数据模型和 API/UI 通过完整回归。
- A/B/C 提交、验证结果、migration 版本和最终依赖检查已写入 `STATUS.md`。

## 失败或中断恢复

```bash
git log -8 --oneline
sed -n '1,220p' docs/refactor-run/STATUS.md
rg -n 'RedisStreamTask|TaskRegistry|taskBySession|SessionStatus|ExecutionStatus' api/internal ui/src
GOCACHE=/private/tmp/go-manus-gocache go test ./... -count=1
```

按最后一个已完成检查点继续。若删除文件后编译失败，优先恢复该检查点提交，不要复制回旧 Task 作为临时长期方案；缺失能力应从 RunService/RunExecutor 提取最小接口。
