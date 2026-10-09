# 阶段 3：Run 领域与存储（未接生产）

> 当前源码没有 Run、RunStatus、RunRepository、MessageRepository 或 Run migration。本阶段先锁定领域和事务契约，再写代码与测试。

## 目标与边界

建立最小 Run 领域模型和 PostgreSQL 存储，不接入现有 Chat/Stop/SSE 生产路径。阶段结束时可以独立创建、迁移、查询和测试 Run；生产请求仍只使用 Session/RedisStreamTask。

本阶段不引入 Worker、Lease、Outbox、永久事件表、Plan/Step 表或通用 UnitOfWork。首期 Run 由当前进程执行，PostgreSQL 保存业务事实，Redis 只承载短期实时事件。这些能力并非永久排除，只有达到 [目标架构](./01-architecture.md#7-演进边界) 中的触发条件后才单独立项。

## Run 领域模型

建议最小字段：

| 字段 | 语义 |
|---|---|
| `id`、`session_id` | Run 标识与所属会话 |
| `status` | 唯一执行状态 |
| `idempotency_key` | 顶层创建请求幂等键 |
| `settings_snapshot` | 创建时冻结的运行配置 |
| `prompt_hash` | 本次运行使用的 Prompt 版本标识 |
| `execution_snapshot`、`snapshot_revision` | 可恢复计划、步骤摘要、当前位置与乐观修订号 |
| `waiting_message_id` | 当前等待问题；非 waiting 状态为空 |
| `error_code`、`error_message` | 稳定错误分类与可展示错误 |
| `started_at`、`finished_at` | 实际执行时间 |
| `created_at`、`updated_at` | 持久化时间 |

首期状态固定为：

```text
pending -> running -> waiting_input -> running
pending -> cancelling | interrupted
running -> succeeded | failed | cancelling | interrupted
waiting_input -> cancelling | running
cancelling -> cancelled
```

`succeeded`、`failed`、`cancelled`、`interrupted` 是终态。状态迁移必须集中在 Run 领域/服务中，并由条件更新执行；Handler 和 Engine 不得各自维护另一份迁移规则。

### 3A 领域检查点（已完成）

`a8de838` 已实现但尚未接入存储的最小领域契约：

- `model.RunStatus` 集中定义 active、terminal 和允许的状态迁移；后续 Repository 只能复用该矩阵，不能自行发明状态转换。
- `model.RunExecutionSnapshot`、`RunStepSnapshot` 和 `WaitingCheckpoint` 是 JSONB 的 Go 契约。进入 `waiting_input` 恢复前必须验证正修订号、当前步骤、稳定问题消息、`continue_step` 恢复模式、唯一步骤 ID，以及每个 completed 步骤的非空结果摘要。
- `service.BuildResumeMessages` 是无副作用转换器：它按“初始用户输入 → 已完成步骤摘要（原步骤顺序）→ 助手问题 → 用户回答”构造 `llmcore.Message`，并保留输入、摘要和回答上的附件。问题 ID 与回答的 `reply_to_message_id` 不匹配时拒绝恢复。

该实现只锁定模型与行为测试，未新增表、Repository、Handler、bootstrap 注入或生产双写；3B 必须在这组模型之上实现存储约束。

## 状态与重启语义

- 服务重启时，`pending`、`running` 没有持久化队列可接管，统一条件迁移为 `interrupted`。
- `waiting_input` 不持有执行 goroutine 或旧 ToolSet。执行快照与等待检查点完整时，重启后保持该状态；快照缺失或损坏时条件迁移为 `interrupted` 并记录稳定错误码。
- 同一 Session 最多存在一个 `pending`、`running`、`waiting_input` 或 `cancelling` Run。
- 取消只允许从前三个可取消状态进入 `cancelling`；该状态阻止新的输入和 Engine 终态写入。执行 goroutine 退出并释放 `ToolSet` 后才进入 `cancelled`。
- 服务重启时不存在存活 goroutine，遗留 `cancelling` 可直接条件收敛为 `cancelled`。

## 等待输入恢复契约

`execution_snapshot` 首期使用 Run 内 JSONB，结构至少包含：

~~~text
snapshot_revision
plan_id / plan_revision
current_step_id
steps[]
  - id
  - status
  - result_summary
  - artifact_refs
waiting_checkpoint
  - question_message_id
  - step_id
  - resume_mode = continue_step
~~~

只保存恢复后继续执行所需的稳定摘要，不保存逐 token 输出、完整内部 ToolResult、外部连接句柄或 Provider 特定的 tool-call 协议对象。已完成步骤的 `result_summary` 必须足以支持后续步骤；不能只保存 completed 状态。

恢复流程固定为：

1. 对 `waiting_input` Run 幂等写入用户回答，并通过 `reply_to_message_id` 绑定当前 `waiting_message_id`。
2. 校验 Run 状态、snapshot revision、current step 和等待消息仍一致。
3. 读取初始用户消息、已完成步骤摘要、当前问题和本次回答，按固定转换器构造新的 `llmcore.Message` 序列。
4. 重新获取当前 Run 创建时冻结的 Settings/Prompt 标识和新的 `ToolSet`。
5. 从原 `current_step_id` 继续执行，不默认重规划；快照无法解释时返回领域错误并转 `interrupted`。

转换器必须是纯函数并有行为测试，保证同一持久化输入生成稳定的 role、顺序和内容。恢复不要求重造已经过期的 Redis 逐 token 事件。

## Messages 契约

`messages` 保存会话中需要长期查询的用户/助手消息，不保存逐 token 增量、内部 ToolResult 或完整 Prompt。建议最小字段：

- `id`、`session_id`、`run_id`、`idempotency_key`、`reply_to_message_id`
- `role`：首期持久化 `user`、`assistant`
- `content`、`attachments`
- `created_at`

初始用户输入、等待输入时的助手问题、后续用户输入和最终助手回复都关联同一个 Run。一次客户端提交只生成一个幂等键：创建 Run 时该键同时用于 Run 和初始用户消息；后续输入使用新的键并关联当前问题。客户端不需要生成两套键。

助手问题由服务端分配稳定 message ID，并写入 `runs.waiting_message_id`；回答必须设置 `reply_to_message_id`。不要使用 `run_id + role` 唯一约束，因为一个等待输入 Run 可以包含多轮用户/助手消息。

工具调用明细继续通过实时事件展示；只有确有长期审计和查询需求时才增加独立持久化模型。

## 数据库约束

- `UNIQUE(session_id, idempotency_key)` 保证创建 Run 幂等。
- `UNIQUE(run_id, idempotency_key)` 保证初始消息和后续每次客户端输入幂等；Run 创建事务内的初始消息复用创建请求的同一个键。
- 对 `pending`、`running`、`waiting_input`、`cancelling` 建 PostgreSQL 部分唯一索引，保证一个 Session 只有一个 active Run，取消收敛完成前不能创建新 Run。
- 状态迁移使用 `WHERE id = ? AND status IN (...)` 条件更新，并检查影响行数。
- 执行快照更新同时校验 Run 状态和 `snapshot_revision`，避免迟到写入覆盖新的等待检查点或步骤结果。
- `messages.session_id`、`messages.run_id` 建外键和按时间查询索引；客户端消息建立幂等唯一约束。
- Run/Message 的 ID、时间精度、软硬删除策略与现有项目规范保持一致。

首期 Plan/Step 保存在 Run 的 JSONB 快照中。只有独立查询、局部更新或审计需求被真实证明后才拆表。

## 原子事务

以下写入必须在同一 PostgreSQL 事务中完成：

1. 创建 Run + 写入初始用户消息。
2. 进入 `waiting_input` + 更新执行快照和等待检查点 + 写入助手提问消息。
3. 幂等写入用户输入 + `waiting_input -> running`。
4. 写入最终助手消息 + Run 终态迁移 + 更新 Session 摘要投影。

Session 继续保存标题、未读数和最新消息等会话摘要，但不再保存执行状态。Session 摘要由最终持久化消息投影更新，不从 Redis 增量事件反推。

不要先抽象通用 UnitOfWork。实现时使用面向 Run 聚合的窄事务入口，让 Run、Message 和 Session 投影共享同一个数据库事务；Repository 只做持久化和条件更新，不执行 Engine，也不发布 Redis 事件。

## 历史数据策略

阶段 3 的 `runs/messages` 不接生产，因此表最初为空。阶段 4 从切换点开始写入，不回填 `sessions.events`、旧 task 状态或 Redis 历史；开发环境旧执行数据可以通过 migration 重置。不得为回填历史而引入双写、兼容读取或一次性事件解释器。

## Redis 与恢复边界

- 流式文本、工具进度和状态通知写入按 Run ID 分区的 Redis Stream。
- Redis 事件可过期，不作为终态、最终消息或 Plan 的事实来源。
- Redis 丢失后，客户端通过 Run、Message 和执行快照恢复；服务端不重造逐 token 历史。
- 最终助手消息只在内容完整且 Run 终态条件更新成功时写入，避免取消后出现伪成功回复。

## 实施顺序

1. 评审状态矩阵、执行快照、消息转换、幂等、active 唯一约束和重启语义。
2. 新增 `runs`、`messages` migration 和 model。
3. 实现 Repository、聚合事务入口和数据库约束测试。
4. 实现纯 RunService 测试，但不从 bootstrap 注入、不改 Handler、不写生产 Run。

建议提交拆分：

- `feat(api): define run domain contracts`
- `feat(api): add run persistence`

## 完成条件与回滚

- 状态迁移、幂等、active 唯一索引、消息去重、事务回滚和 waiting_input 恢复测试通过。
- 恢复测试覆盖已完成步骤摘要、问题回答配对、同一步继续、重复回答幂等及损坏快照转 interrupted。
- `go test ./...`、Repository component 测试、migration 测试和 `go vet ./...` 通过。
- 旧生产链行为不变，bootstrap/Handler 没有 Run 依赖。

回滚只删除本阶段新增的 Run/Message 文件和 migration，不触碰 `sessions` 现有执行列；阶段 4 切换前不得清理旧列。
