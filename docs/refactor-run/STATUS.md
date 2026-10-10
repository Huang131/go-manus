# Run/Session 重构实施状态

本文件是跨会话恢复入口。状态只反映当前代码，不把目标设计写成已完成。

## 当前状态

| 阶段 | 状态 | 提交/依据 | 说明 |
|---|---|---|---|
| 0. 事实基线与方案校准 | complete | `0382404` | 已核对当前 Session/Task 链、配置、Engine、ToolSet，并形成可恢复的阶段方案 |
| 1. Settings 与 Prompt | complete | `4df77fe`、`e9dcba2`、`9548b84`、`dd5c2cf`、`b0f5328` | Settings、严格校验、migration、UI 与 PromptCatalog/hash 已完成 |
| 2. Engine 与 Context | complete | `4650cc4`、`aba7f43`、`8fd0fa6` | Outcome、ContextPolicy 与 MCP 动态调用契约已完成 |
| 3. Run 领域与存储 | complete | `a8de838`、`f0ada6a`、`6055590`、`cb50565` | 3A 领域、3B Repository/migration、3C 纯 RunService 已完成；尚未接入生产执行路径 |
| 4. 执行生产切换 | in_progress | `cb50565`、`d7bb0cb`、`20b622f` | 4A 生命周期与 waiting_input 恢复前置已完成；尚未接入 Bootstrap、Handler、Chat 或 SSE |
| 5. API/UI/SSE | pending | - | 唯一公开契约切换点；完成后删除旧路由 |
| 6. 遗留删除 | pending | - | 依赖阶段 4/5 的引用扫描和回归 |

允许状态：`pending`、`in_progress`、`complete`、`blocked`。

## 当前生产语义

```text
SessionHandler -> AgentService -> RedisStreamTask -> AgentTaskRunner
              -> PlannerReActFlow -> LLM + ToolSet
```

当前不存在 Run 生产写入。`ToolRegistry`、MCP/A2A ToolSet 引用生命周期已完成，不属于本方案待办。

## 最近验证基线

实施新阶段前，在 `api/` 重跑：

```bash
go test ./...
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1
go vet ./...
```

受限环境的回环端口失败必须单独记录，不能改写为通过。

## 最近完成：检查点 4A RunExecutor 生命周期与 Engine adapter

本检查点代码尚未接入 Bootstrap、Handler、Chat 或 SSE。

验证结果（4A/恢复前置）：

```text
go test ./internal/model ./internal/service ./internal/agent ./internal/repository -count=1  PASS
go test ./internal/agent -run TestPlannerEngineAdapterResumesWaitingPlanWithoutReplanning -count=1  PASS
go vet ./...                                                             未完成：当前环境 Go 编译阶段长时间无输出
git diff --check                                                         PASS
go test ./... -count=1                                                   未完成：当前环境 Go 编译阶段长时间无输出，已停止
go test -tags=integration ./tests -run '^TestRunRepo_FinishTerminal' -count=1
  BLOCKED: 当前受限环境无法连接 localhost:15432（operation not permitted）
```

行为契约：RunExecutor 启动后冻结 Settings、消息和 ToolSet；Engine 退出后才释放 ToolSet，再提交 waiting 或终态。工具获取失败、`nil` ToolSet、Engine panic、取消无句柄和终态持久化失败均有测试。Repository 终态、执行快照和最终助手消息使用同一事务，迟到终态受条件更新拒绝。Planner adapter 已能从快照恢复计划和当前步骤，恢复时重建持久化上下文且不重新规划，最新回答只注入一次。

边界：Planner adapter 已能把持久化 `RunExecutionSnapshot` 重建为 Plan，恢复 Flow 并继续原步骤；恢复测试覆盖不重新规划、步骤摘要和用户回答单次注入。该能力仍是未接生产的前置实现，4B 仍需完成 Bootstrap、Handler、Chat、SSE 切换及取消竞争的 HTTP 验收。

## 检查点

| 检查点 | 状态 | 入口 |
|---|---|---|
| 1A Settings 单一类型、配置 migration 与校验 | complete | `4df77fe`、`e9dcba2`、`9548b84`、`dd5c2cf` |
| 1B PromptCatalog 与 hash | complete | `b0f5328` |
| 2A StepOutcome 唯一信号并删除旧等待表达 | complete | `4650cc4` |
| 2B ContextPolicy | complete | `aba7f43` |
| 2C MCP 动态路由、初始化与错误契约 | complete | `8fd0fa6` |
| 3A Run、执行快照与等待恢复契约 | complete | `a8de838` |
| 3B Run/Message Repository | complete | `f0ada6a` |
| 3C 纯 RunService | complete | `cb50565 refactor(api): add pure run service` |
| 4A RunExecutor 生命周期与观测，未接生产 | complete | `refactor(api): add run executor lifecycle` |
| 4A waiting_input 恢复前置 | complete | `20b622f feat(run): restore waiting input execution context` |
| 4B 唯一后端生产切换 | pending | `05-run-execution-cutover.md` |
| 5A Run API/SSE | pending | `06-api-ui-sse-cutover.md` |
| 5B UI 切换 | pending | `06-api-ui-sse-cutover.md` |
| 5C 删除旧路由 | pending | `06-api-ui-sse-cutover.md` |
| 6A 删除 Task 基础设施 | pending | `07-legacy-removal.md` |
| 6B 删除 Session 执行字段 | pending | `07-legacy-removal.md` |
| 6C 收紧 model 依赖 | pending | `07-legacy-removal.md` |

## 阶段验收归属

| 验收项 | 归属阶段 |
|---|---|
| Settings 单一真相、范围校验、Prompt hash | 1 |
| 等待输入不再依赖错误字符串、上下文裁剪可替换 | 2 |
| Run 状态、等待恢复、幂等、单活跃约束、消息存储 | 3 |
| 所有生产执行只写 Run/Message，取消收敛后 ToolSet 必然释放 | 4 |
| Run API、Last-Event-ID、UI 只消费 Run 语义 | 5 |
| 旧 Task、Session 执行字段和旧路由物理删除 | 6 |

## 决策记录

- 首期不引入 Worker、Lease、Outbox、永久事件溯源、Plan/Step 独立表或通用 UnitOfWork；达到 `01-architecture.md` 的演进触发条件后单独立项。
- `PlannerReActFlow.Invoke` 已拆分，后续只调整结果和状态边界。
- `StepOutcome.Kind` 是目标唯一等待控制信号；阶段 2 必须删除旧 error/boolean 双表达。Engine 内部完成有限重试，Flow/RunExecutor 不消费 `retryable_failure`，也不重复执行同一次调用。
- 2A 已完成：BaseAgent 在工具边界将等待输入归一为 `OutcomeWaitingInput`，`WaitingInput.Attachments` 固定为 `[]string`；ReAct 补充 `StepID`，Flow 只按 `Kind` 决定等待、取消、完成和失败。
- `waiting_input` 依靠 execution snapshot、已完成步骤摘要和问题/回答关联恢复，不依赖旧 goroutine 或 `SimpleMemory`。
- `20b622f` 已把快照中的计划元数据、步骤描述、完成结果和附件恢复为 `model.Plan`；Planner/ReAct memory 从持久化消息重建，最新回答仅作为当前步骤输入注入一次。
- 一次客户端提交只使用一个幂等键；创建 Run 与初始消息共享该键，后续输入在 Run 内去重。
- 阶段 4 不回填 `sessions.events` 或旧 Task 历史，从切换点开始写 Run/Message。
- Run input 只接受 `waiting_input`；`cancelling`、其他活跃状态和终态分别返回稳定的 409 业务错误，重复幂等输入返回原结果。
- 取消先条件写入 `cancelling`，再发出 cancel signal；Engine 退出并释放 ToolSet、持久化 `cancelled` 后才注销控制句柄。Cancel 发现句柄不存在或终态写入失败时，必须通过幂等 `ReconcileCancelling` 在当前进程或启动恢复中收敛。
- `MaxPlanSteps` 不作为伪配置加入；必须先有真实消费逻辑。
- `sessions.memories` 和 `Session.Memories` 当前不存在，不在方案中虚构。
- ToolRegistry 三份映射、MCP/A2A retired 无界释放问题已在工具重构提交中解决，不重复规划。
- 2C 已完成：MCP function name 在发现时建立不可变索引，名称碰撞拒绝初始化；连接、发现与协议错误返回 Go error，远端 `IsError` 保留为可交给模型处理的失败 `ToolResult`。
- Agent Settings 的持久化记录必须完整合法：记录缺失才使用 `10/3/10` 默认值；非法更新返回 400，非法存量记录由 `005_normalize_agent_settings.sql` 一次性修正，启动读取不再静默 clamp。
- Task/Run 创建必须只读取一次 Settings；Runner 与 ToolSet 必须使用同一快照，热更新只影响后续创建的执行。
- 3A 已完成：`RunStatus` 是唯一状态迁移矩阵；`RunExecutionSnapshot` 对 waiting_input 强制校验当前步骤、等待问题、恢复模式和已完成步骤摘要。`BuildResumeMessages` 是纯转换器，固定重建“初始用户输入 → 已完成步骤摘要 → 助手问题 → 用户回答”的 LLM 消息序列，不读取存储、不获取 ToolSet、不改变 Run 状态。

## 最近完成：检查点 3A Run 领域与等待恢复契约

提交：`a8de838 feat(api): define run domain contracts`。

验证结果：

```text
go test ./... -count=1                                                   PASS
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1  PASS
go vet ./...                                                             PASS
git diff --check                                                         PASS
```

行为契约：Run 的 active/terminal 分类和状态迁移集中在 `model.RunStatus`；损坏的等待快照在进入恢复前被拒绝；恢复消息保留输入和附件，并按快照中完成步骤的稳定顺序重建上下文。该检查点没有 migration、Repository、bootstrap 注入或 Chat/Stop/SSE 生产路径改动。

下一入口：检查点 3B Run/Message Repository 与 migration。先实现 PostgreSQL 表、窄事务入口、幂等和 active 唯一约束测试，仍不得接入生产写路径。

## 最近完成：检查点 3B Run/Message Repository 与 migration

提交：`f0ada6a feat(api): add run persistence`。

验证结果：

```text
make test-up（重复执行 001-006 migration）                         PASS
go test -tags=integration ./tests -run '^TestRunRepo_' -count=1 -v PASS
```

行为契约：创建 Run 与首条用户消息、进入等待状态与助手问题、恢复输入与状态切换均使用聚合内窄事务；数据库以唯一键、部分唯一索引和条件更新承担幂等与并发约束。新增的回滚测试确认问题消息 Session 与 Run 不匹配时，Run 保持 `running`，快照与 `waiting_message_id` 不写入，问题消息也不落库。该检查点没有 bootstrap、Handler、Chat、Session、Task 或 SSE 生产路径改动。

下一入口：检查点 3C 纯 RunService。先以 Repository 为依赖写领域用例测试，再实现最小服务方法；仍不得创建 Run 生产写入或改造现有 Chat 路径。

## 最近完成：检查点 3C 纯 RunService

提交：`cb50565 refactor(api): add pure run service`。

验证结果：

```text
go test ./internal/service -run '^TestRunService' -count=1 PASS
go test -race ./internal/service -count=1 PASS
go vet ./internal/service PASS
git diff --check PASS
```

行为契约：`RunService` 统一负责创建 Run、启动 pending、进入 waiting_input、提交回答恢复上下文、请求取消和 `cancelling -> cancelled` 幂等收敛。服务层拒绝 cancelling/终态输入，重复回答按幂等键返回同一恢复上下文；恢复前先验证 snapshot 和消息配对，避免无效快照改变 Run 状态。本检查点没有 bootstrap、Handler、Chat、Task、SSE 或 RunExecutor 改动。

下一入口：阶段 4A RunExecutor 生命周期与 Engine adapter。需要补齐终态消息原子写入、控制句柄、ToolSet 释放和取消竞争，再接入生产路径。

## 最近完成：检查点 2C MCP 动态路由、初始化与错误契约

提交：`8fd0fa6 refactor(mcp): enforce dynamic tool contracts`。

验证结果：

```text
go test ./... -count=1                                                   PASS
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1  PASS
go vet ./...                                                             PASS
git diff --check                                                         PASS
```

行为契约：MCP 工具发现阶段建立稳定的 `functionName -> {server, tool}` 索引，调用不再扫描或反解名称；碰撞、连接、发现和协议失败立即返回 Go error 并释放已创建资源；远端 `IsError` 仍作为失败 `ToolResult` 返回给 Engine。`ToolProvider` 只注册完整初始化的 MCP 工具。

下一入口：检查点 3A Run、执行快照与等待恢复契约；先只建立领域和存储契约，不接入生产写路径。

## 已完成：检查点 2B ContextPolicy

提交：`aba7f43 refactor(agent): enforce context policy per request`。

验证结果：

```text
go test ./... -count=1                                                   PASS
go test -race ./internal/agent/... -count=1                              PASS
go vet ./...                                                             PASS
git diff --check                                                         PASS
```

行为契约：`ContextPolicy` 在初始组装和每次 provider 调用前统一生效；工具 schema 与输出预留计入同一 token 窗口；裁剪从近到远回填完整历史组，并固定保留最后一个用户请求及其后续完整工具链。当前顺序执行模型仅把实际执行的第一个 tool call 写入历史，避免留下没有对应 tool result 的协议残片。

后续入口：阶段 2 已完成；下一步进入检查点 3A Run、执行快照与等待恢复契约。

## 已完成：检查点 2A StepOutcome

提交：`4650cc4 refactor(agent): return explicit step outcomes`。

验证结果：

```text
go test ./... -count=1                                                   PASS
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1  PASS
go vet ./...                                                             PASS
git diff --check                                                         PASS
```

行为契约：等待输入返回 `OutcomeWaitingInput` 而非 error；取消返回 `OutcomeCancelled` 且不会把 PlanStep 标记为失败；Flow 保证先发助手问题、后发 wait 事件；ReAct 为等待结果写入当前 `StepID`。

后续入口：检查点 2B 已完成，下一步进入 2C MCP 动态路由、初始化和错误契约。

## 阶段 1 完成记录

验证结果：

```text
go test ./...                                                        PASS
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1  PASS
go vet ./...                                                        PASS
npm run lint                                                        PASS（仅既有 warning）
npm run build                                                       PASS
```

独立测试环境中额外验证：

- Agent 配置合法值 HTTP 生命周期通过。
- 非法配置更新稳定返回 HTTP 400。
- `005_normalize_agent_settings.sql` 在真实 PostgreSQL 连续执行两次后结果一致。

PromptCatalog 额外验证：七个模板可枚举；每个模板 hash 等于内容 SHA-256；目录 hash 不受 map 顺序影响并随内容变化。

阶段 1 的下一入口已完成，当前按检查点 2B 推进。

## 恢复流程

```bash
cd /Users/huanghao2/GolandProjects/study/imooc-mas/go-manus
git status --short --branch
git log -8 --oneline --decorate
sed -n '1,220p' docs/refactor-run/STATUS.md
```

读取第一个 `pending` 或 `in_progress` 检查点，确认上一提交存在，运行该阶段恢复基线后再修改。任何中断都从最近一个通过门禁的提交继续。
