# 阶段 4：Run 执行装配与唯一后端切换

这是整个方案唯一的后端生产语义切换点。切换前 Run 代码不接请求；切换后所有执行事实只写 Run/Message，不再写 Session 的 task/status/events。

本阶段暂不发布新的公开 Run 路由。现有 Session Chat/Stop/SSE 路由可保留一个阶段，但只能作为 RunService 的薄适配，阶段 5 随 UI 切换直接删除。

## 切换前置条件

- 阶段 1、2、3 已完成并在 `STATUS.md` 记录真实验证结果。
- Run 状态矩阵、active 唯一约束、取消条件更新、幂等和消息事务测试通过。
- RunExecutor 可执行现有 Engine，并在成功、失败、等待、取消和 panic 防护路径释放 `ToolSet`。
- 当前 Chat/Stop/SSE 行为有回归测试，能证明切换没有丢失用户可见能力。

## 目标调用链

```text
SessionHandler（临时适配）
  -> RunService.Create / SubmitInput / Cancel / GetEvents
  -> RunExecutor
  -> Planner/ReAct Engine
  -> RunService persistence boundary + RunEventStream
```

Handler 不直接创建 `RedisStreamTask`，Engine 和 RunExecutor 都不直接调用 Repository。RunExecutor 只负责编排一次运行，通过 RunService 提交状态、Plan 和最终消息，不对外提供业务 API。

## 代码范围

- `internal/bootstrap`：装配 Settings、PromptCatalog、ToolProvider、RunService、RunExecutor 和 RunEventStream。
- `internal/service`：新增 RunService/RunExecutor；旧 AgentService 执行入口缩减为适配或删除。
- `internal/agent`：提供稳定 Engine facade，复用现有 Planner/ReAct/Flow，不复制状态机。
- `internal/handler`：现有 Chat/Stop/SSE 只做请求转换和 RunService 调用，不再访问 Task。
- `internal/repository`：Run/Message 成为唯一执行事实写入口。
- `internal/runstream`：定义 RunEventStream 契约和 Redis 实现，使用 Run ID 作为 stream key，保留短期实时事件。

## 检查点

### 4A：装配但不接生产

实现 RunExecutor 与 Engine adapter，使用服务级测试验证 Settings/Prompt/ToolSet 快照、等待输入和释放路径。此检查点不得修改 Handler，不得产生生产 Run。

本检查点的 adapter 只负责把现有 `PlannerReActFlow` 的事件/状态转换为 `RunExecutionResult`，不负责把持久化的 `RunExecutionSnapshot` 重新灌回 Flow。因而它可以验证成功、失败、取消和本轮等待问题的转换，但**不能宣称已经完成 waiting_input 跨重启恢复**；跨重启恢复仍由 `RunService.SubmitInput` 的 `BuildResumeMessages` 契约承担，生产接入前必须补齐“从快照恢复计划并继续当前步骤”的 Engine 输入边界。

4A 还要求：工具集合获取返回错误或 `nil` 时，已进入 `running` 的 Run 必须立即执行 `RequestCancel -> ReconcileCancelling`；终态持久化失败时保留控制句柄，允许后续 reconciliation 收敛，不能报告执行已完成。

当前实现与验证：

- `RunExecutor` 在 Engine 退出后才调用 `ToolSet.Release`，再提交 waiting 或终态结果。
- `FinishTerminal` 将终态、执行快照和最终助手消息放在同一事务中；取消后的迟到成功/失败受条件更新拒绝。
- Engine panic、工具集合获取失败、`nil` 工具集合、取消无句柄和终态持久化失败均有服务测试。
- adapter 已覆盖成功、等待问题归属（`run_id/session_id`）、快照 revision 和 context cancel 转换。

建议提交：`refactor(api): add run executor lifecycle`。

### 4B：原子切换生产语义

在一个提交中完成 bootstrap、Handler、Service 和事件读取切换。旧 Session 路由继续存在时，只解析旧请求并调用 RunService；旧 SSE 路由根据当前 Run 定位 RunEventStream，不再读取 Task registry。切换从本提交开始写入 Run/Message，不回填旧 `sessions.events`。

建议提交：`refactor(api): switch execution semantics to runs`。

禁止拆成“先双写验证，再删除旧写入”的两个生产提交。项目尚未上线，直接用测试和数据库约束证明切换正确。

## 并发与取消

- 创建 Run 依赖数据库 active 部分唯一索引，不采用先读后写。
- Cancel 先通过条件更新把可取消 Run 原子迁移为 `cancelling`，再调用进程内 cancel function；迟到的 Engine 结果影响行数为 0 时停止写终态和终态消息。
- RunExecutor 的 `runID -> cancel` 映射只是进程内控制句柄，不是业务事实；进程重启后把遗留 `pending`、`running` Run 条件迁移为 `interrupted`，把 `cancelling` 收敛为 `cancelled`；`waiting_input` 只在执行快照完整时保持可恢复。
- 同一个等待输入 Run 通过 `SubmitInput` 继续执行；新的顶层用户请求创建新 Run。
- Engine goroutine 退出前不得释放其 ToolSet。取消路径的顺序固定为：Engine goroutine 退出 -> `ToolSet.Release()` -> 条件更新 `cancelling -> cancelled` -> 注销控制句柄。控制句柄必须保持注册，直到 Run 已持久化为 `waiting_input` 或终态，不能先注销再写终态。
- Cancel 已成功落库为 `cancelling`，但发现控制句柄不存在时，必须立即调用与启动恢复共用的 `ReconcileCancelling(runID)`；不能等待下一次进程重启。该收敛操作必须幂等，只有仍处于 `cancelling` 的 Run 才能迁移为 `cancelled`。
- 终态持久化失败时不得假装取消完成或提前注销句柄。Run 保持 `cancelling`，记录结构化错误和 reconciliation 指标，并由当前进程的有限重试/后台 reconciliation 或下次启动恢复继续收敛。进程被强制终止时不承诺 defer 执行，数据库中的 `cancelling` 才是恢复依据。
- 进入 `waiting_input` 表示本次 Engine 调用已经退出并释放 ToolSet；用户回答后重新 Acquire 新 ToolSet，不能跨等待长期持有外部连接。
- Cancel API 在 `cancelling` 落库并发出 cancel signal 后即可返回“取消处理中”；客户端通过 Run 查询或 SSE 观察最终 `cancelled`。服务关闭必须等待 Engine goroutine 收敛后再关闭依赖。

### 取消竞争测试矩阵

- Cancel 与 Engine 正常完成同时发生：只有一个条件更新成功，不能同时产生 succeeded 和 cancelled 终态消息。
- Cancel 发生在 Engine 返回后、终态持久化前：`cancelling` 阻止迟到的成功/失败覆盖。
- Cancel 落库后控制句柄已不存在：请求线程立即执行 `ReconcileCancelling`，最终收敛为 `cancelled`。
- `cancelling -> cancelled` 条件更新影响零行：读取当前状态并按幂等结果处理，不覆盖其他终态。
- 终态写入暂时失败：Run 保持 `cancelling`，句柄与资源状态可诊断，reconciliation 恢复后完成收敛。
- 重复 Cancel：返回同一稳定语义，不重复发送取消信号、终态消息或终态事件。

## 可观测性

生产切换必须同时增加以下指标和结构化日志：

- Run 状态迁移次数，按 from/to/result 分类。
- active Run 唯一索引冲突和创建幂等命中次数。
- waiting_input 数量、停留时间和恢复失败原因。
- cancel 请求到 Engine goroutine 退出的耗时。
- interrupted Run 数量及启动恢复结果。
- SSE 重连、游标过期和 Redis 发布失败次数。

日志必须包含 `session_id`、`run_id` 和状态 revision，不记录 Prompt、密钥或完整工具参数。

## 完成条件

- HTTP 成功、等待输入、提交输入、取消竞争、重复幂等、终态和 SSE 事件顺序测试通过。
- Session 的 `UpdateStatus`、`AppendEvent`、task ID 写入在生产调用链中为零。
- bootstrap/handler/RunService 不再调用 `NewRedisStreamTask`、`defaultTaskRegistry` 或 `taskBySession`。
- Redis 丢失不影响 Run 终态和最终消息查询。
- 取消测试证明：`cancelling` 先落库、迟到完成无法覆盖、Engine 退出后才释放 ToolSet、持久化 `cancelled` 后才注销句柄；无句柄和持久化失败路径都能通过 reconciliation 收敛。
- 上述关键指标在成功、等待、恢复、取消和启动中断路径均有观测验证。
- 通用 test、race、vet 和 diff 检查全部通过。

## 回滚边界

4A 可独立 revert。4B 必须整体 revert，不能只恢复 Handler 或只恢复 Executor。回滚到阶段 3 后，旧 Session 语义仍完整；不得保留 Run/Session 双写作为回滚手段。
