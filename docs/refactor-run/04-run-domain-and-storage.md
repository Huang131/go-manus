# 阶段 3：Run 领域模型与 PostgreSQL 存储

## 目标

建立 Run、Plan、Step 的领域状态、数据库表和仓储原语，为下一阶段一次性切换生产执行语义提供稳定基础。本阶段的 Run 能力只由单元/集成测试调用，生产 Handler、AgentService 和任务执行链不得创建或更新 Run。

## 非目标

- 不接生产请求，不新增 Run Handler/公开路由。
- 不修改 Session.Status、Events、task_id 或 memories。
- 不创建 Worker、lease、outbox、run_events 或永久 token 事件表。
- 不实现自动恢复 interrupted Run；首期只准确标记中断。

## 前置条件与唯一语义

- 阶段 1 的 1A/1B 检查点必须先在 `STATUS.md` 标记完成。阶段 2 可以已完成或尚未开始，但阶段 4 必须同时依赖阶段 2、3；当前状态仍是部分完成，不能把本阶段当作可直接开工。
- 生产唯一语义仍是 Session + RedisStreamTask。本阶段禁止在 bootstrap 注入 RunRepository 到生产写链路。
- 开始前运行全量 Go 测试，并在 `STATUS.md` 记录当前提交。

## 文件清单

创建：

- `api/internal/model/run.go`：Run、RunStatus、状态迁移和终态判断。
- `api/internal/model/plan.go`：新增 Run 使用的 PlanSnapshot、PlanStatus、StepSnapshot、StepStatus 与快照复制。
- `api/internal/model/message.go`：持久化对话消息模型，不包含 Redis 游标。
- `api/internal/repository/run_repository.go`：RunRepository 与 PostgreSQL 实现。
- `api/internal/repository/message_repository.go`：MessageRepository 与 PostgreSQL 实现。
- `api/internal/repository/run_unit_of_work.go`：在同一个 pgx 事务中绑定 Run 与 Message 仓储。
- `api/migrations/005_create_runs_and_messages.sql`。
- `api/internal/model/run_test.go`、`plan_test.go`。
- `api/tests/run_repository_integration_test.go`、`message_repository_integration_test.go`。

修改：

- `api/internal/repository/queryer.go`：只在确有公共扫描需求时扩展，不增加业务规则。
- `api/internal/bootstrap/app.go`：在 repositories 容器中构造 Run/Message Repository；本阶段不向 AgentService/Handler 注入。
- `api/migrations/Makefile`：`migrate-down` 按外键顺序增加 `messages`、`runs` 删除项。

本阶段不删除生产文件。

## 领域模型

```go
type Run struct {
    ID               string
    SessionID        string
    Status           RunStatus
    ModelID          string
    SettingsSnapshot AgentSettingsSnapshot
    PromptHashes     map[string]string
    Plan             *PlanSnapshot
    PlanRevision     int
    IdempotencyKey   string
    ErrorCode        string
    ErrorMessage     string
    StartedAt        *time.Time
    FinishedAt       *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

为避免 `model` 反向依赖 `agent`，`AgentSettingsSnapshot` 在 model 内定义为与 JSON 契约一致的值对象，阶段 4 的 RunService 从 `agent.AgentSettings` 显式转换。它没有默认值和运行时更新逻辑，不能成为第二份配置来源。

Run 状态：`pending/running/waiting_input/succeeded/failed/cancelled/interrupted`。合法迁移严格采用 `01-architecture.md` 的状态图。PlanSnapshot 状态为 `pending/running/succeeded/failed/cancelled`，StepSnapshot 为 `pending/running/succeeded/failed/skipped/cancelled`。

当前 `event.go` 中的 `Plan/PlanStep/ExecutionStatus` 暂时只服务旧 Session/Task 生产链。本阶段不迁移、不复用它们，避免在还没有切换 UI/SSE 时改变 wire status。阶段 4 的新 Engine 改用 PlanSnapshot；阶段 6 随旧 Task 文件一起删除这些 legacy 类型。这是有明确删除期限的迁移适配，不是两套生产事实来源：本阶段 RunPlan 仍不接生产，阶段 4 后 legacy Plan 不再被生产装配使用。

Plan 是 Run 当前快照。每次替换 Plan 必须携带 expected revision，SQL 仅在 revision 匹配时更新并加一；失败返回 conflict。首期不保留历史 Plan 版本。

## 数据库设计

`runs` 至少包含：

```sql
id VARCHAR(255) PRIMARY KEY,
session_id VARCHAR(255) NOT NULL REFERENCES sessions(id),
status VARCHAR(32) NOT NULL,
model_id VARCHAR(255) NOT NULL REFERENCES llm_models(id),
settings_snapshot JSONB NOT NULL,
prompt_hashes JSONB NOT NULL,
plan JSONB,
plan_revision INT NOT NULL DEFAULT 0,
idempotency_key VARCHAR(255),
error_code VARCHAR(64),
error_message TEXT,
started_at TIMESTAMP,
finished_at TIMESTAMP,
created_at TIMESTAMP NOT NULL,
updated_at TIMESTAMP NOT NULL
```

`messages` 至少包含 `id`、`session_id`、可空 `run_id`、`role`、`content`、`attachments JSONB`、单调递增 `ordinal`、`created_at`。`run_id` 可空是为了未来保留不触发执行的系统/导入消息；本轮由 Run 产生的消息必须填写 run_id。附件只保存文件 ID/引用，不能复制二进制内容。

创建 Run 时先解析用户指定模型或当前默认模型，并把解析后的实际 ID 固化到 `model_id`；若没有可用模型，创建请求失败，不创建 pending Run。后续默认模型变化不影响已创建 Run。

约束与索引：

- `CHECK` 限制 Run/role 状态枚举。
- `UNIQUE(session_id, idempotency_key) WHERE idempotency_key IS NOT NULL`。
- `UNIQUE(session_id) WHERE status IN ('pending','running','waiting_input')`，数据库保证单 Session 单活跃 Run。
- `runs(session_id, created_at DESC)`、`messages(session_id, ordinal)`、`messages(run_id, ordinal)`。
- migration 使用 `IF NOT EXISTS` 保持现有简易迁移器可重复执行。

## Repository 契约

```go
type RunRepository interface {
    Create(context.Context, *model.Run) error
    GetByID(context.Context, string) (*model.Run, error)
    GetByIdempotencyKey(context.Context, string, string) (*model.Run, error)
    ListBySession(context.Context, string, int, int) ([]*model.Run, int, error)
    Transition(context.Context, string, model.RunStatus, model.RunStatus, RunPatch) (bool, error)
    ReplacePlan(context.Context, string, int, *model.PlanSnapshot) (bool, error)
    MarkActiveInterrupted(context.Context) (int64, error)
    WithTx(context.Context, func(RunRepository) error) error
}
```

`Transition` 必须在 SQL 的 `WHERE id=$1 AND status=$expected` 中完成比较更新，不能先读后写。返回 `false,nil` 表示状态已变化，由服务层重读判断取消优先。`MarkActiveInterrupted` 覆盖 pending/running/waiting_input，并设置 finished_at。

MessageRepository 提供 `Create`、`ListBySession`、`ListByRun` 和事务绑定。阶段 4 创建 Run 和首条用户消息时必须使用同一数据库事务，因此两个仓储需支持在同一个 `pgx.Tx` 上构造；不要在 model 定义 Repository 接口。

```go
type RunUnitOfWork interface {
    WithTx(context.Context, func(RunRepository, MessageRepository) error) error
}
```

Service 只依赖该接口，不接触 `pgx.Tx`；实现复用现有 `runInTx` 的提交、回滚和 panic 语义。

## 实施顺序与检查点

1. 检查点 3A：新增 Run/PlanSnapshot/StepSnapshot/Message 模型、状态测试和 migration；在空测试库执行两次验证幂等，提交 `feat(api): define run domain and schema`。
2. 检查点 3B：实现 RunRepository、MessageRepository、RunUnitOfWork 与集成测试；在 bootstrap 的 repositories 容器构造仓储，但确认生产链路没有 Create/Transition，提交 `feat(api): add run persistence repositories`。

3A、3B 都不接生产请求，单独回滚不会改变现有业务语义。

## 针对性测试

- model：完整合法/非法迁移矩阵，终态不可迁移，快照复制不泄漏可变切片。
- repository：同 Session 第二个活跃 Run 触发可识别 conflict；终态历史 Run 不阻止新 Run。
- repository：running -> cancelled 后，expected running -> succeeded 更新行数为零，状态仍 cancelled。
- repository：同 idempotency key 只能存在一个 Run；不同 Session 可复用 key。
- plan：expected revision 不匹配时不覆盖更新。
- startup cleanup：三种活跃状态变 interrupted，终态不变。
- messages：按 ordinal 稳定排序，Session/Run 过滤正确，附件 round-trip。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/model ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go test -tags=integration ./tests -run 'Test(Run|Message)Repository' -count=1
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
rg -n 'RunRepository|MessageRepository' internal/handler internal/agent
cd ..
git diff --check
```

最后一条 `rg` 必须没有生产写调用；测试引用可以存在。

## 提交与回滚

回滚 3B 只需 `git revert <3B-sha>`。回滚 3A 时先停止应用，在开发数据库执行 `DROP TABLE IF EXISTS messages; DROP TABLE IF EXISTS runs;`，再 `git revert <3A-sha>` 并运行全量测试。因为本阶段没有生产 Run 数据，删除新表不会损失已生效业务语义。

## 完成定义

- Run 状态和数据库约束一致，取消优先由条件 SQL 保证。
- 单 Session 单活跃 Run 由部分唯一索引保证。
- Plan revision 和消息稳定顺序有集成测试。
- 生产请求仍只写 Session/RedisStreamTask，没有双写。
- 全量 test、integration、race、vet 通过，`STATUS.md` 已更新。

## 失败或中断恢复

```bash
git status --short --branch
psql "$DATABASE_URL" -c '\d runs'
psql "$DATABASE_URL" -c '\d messages'
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/model ./internal/repository -count=1
```

如果 migration 已执行而源码未完成，可以保留空表继续实现；生产链路没有写入，因此不会形成第二套语义。若要退回基线，按“提交与回滚”删除两表并 revert。
