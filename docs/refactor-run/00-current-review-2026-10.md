# API 当前事实基线（2026-10）

本文只记录最新源码已经存在的生产行为，以及后续 Run/Session 重构必须遵守的边界。文中“目标”均不是现有代码。

## 核验基线

- 代码基线：`4c3d39d docs(tools): update registry and snapshot lifecycle comments`
- 核验范围：`api/internal`、`api/migrations`、当前测试和 `docs/refactor-run`
- 当前服务形态：模块化单体，Agent 在进程内执行，PostgreSQL 保存会话数据，Redis Stream 提供实时输出。
- 工作区已有其他未提交修改时，实施者必须先用 `git status` 和 `git diff` 区分归属，不得覆盖。

## 当前生产链

```text
SessionHandler
  -> AgentService.Chat / StopSession
  -> taskBySession[sessionID]
  -> RedisStreamTask
  -> AgentTaskRunner
  -> PlannerReActFlow
  -> PlannerAgent / ReActAgent
  -> LLM + ToolRegistry/ToolSet
```

SSE 通过 task ID 读取 Redis Stream；`Last-Event-ID` 已支持，但它仍然是旧 task 语义。`Session` 仍包含 `TaskID`、`Status`、`Events`，数据库 `sessions` 仍有对应列。

## 已完成能力

### 工具系统

`ToolRegistry` 已统一为 `ToolDescriptor`，普通工具和多函数工具共用同一注册索引；MCP 动态 function 已接通；`ToolProvider.Acquire()` 返回任务级 `ToolSet`，`Release()` 负责 MCP/A2A 外部资源引用释放。热重载旧实例会等待引用归零，不应再次把这些内容列为待做。

仍有三项明确的 MCP 边界债务，应在 RunExecutor 接入前独立修复：

- 动态 function name 由 `mcp_ + server + "_" + tool` 拼接，并在调用时遍历反解；当 server/tool 名含下划线且组合发生碰撞时，可能生成重复 function name。应在发现工具时建立不可歧义的 `functionName -> server/tool` 索引，并拒绝重复名称。
- `MCPTool.Initialize` 当前记录 manager 初始化或工具发现错误后仍返回 `nil`，调用方无法区分初始化失败和“没有工具”。初始化/发现失败应返回错误并清理半初始化资源。
- 连接、超时和协议错误当前被包装为失败 `ToolResult`。虽然 `ToolResult.Success=false` 仍会进入指标，但无法走 Go error 的诊断路径。目标契约应区分：基础设施/协议失败返回 `error`；远端工具明确返回的业务失败保留为 `ToolResult{Success:false}`。

### Agent 配置与搜索 limit

启动时从 `app_configs` 读取 Agent 配置，Handler 会传递 `max_search_results`，搜索 provider 使用该 limit。当前运行时配置字段为 `MaxIterations`、`MaxRetries`、`MaxSearchResults`。尚未完成的是统一 Settings 类型、范围校验、revision 和任务级快照；不存在可直接引用的 `MaxPlanSteps` 实现。

### Engine 与上下文

`PlannerReActFlow.Invoke` 已拆为状态处理函数：`handleIdle`、`handlePlanning`、`handleExecuting`、`handleWaiting`、`handleUpdating`、`handleSummarizing`。`ContextBuilder` 已进入 LLM 请求链路，但预算仍是近似值，尚无模型画像、输出预留和真实 tokenizer。等待输入同时由 `InvokeResult.WaitForUser`、`ToolCallResult.WaitForUser` 和 `ErrWaitForUser` 表达，属于必须在 Engine 阶段收敛的重复状态。

`SimpleMemory`、当前 Plan 指针和步骤结果均为进程内状态。服务重启后仅凭用户/助手消息无法无损恢复正在等待输入的步骤，因此目标方案必须持久化稳定的执行快照和等待检查点，并明确上下文重建规则。

### 仍在使用、禁止提前删除

- `agent.Task`、`Stream`、`RedisStreamTask`、`defaultTaskRegistry`：当前 Chat 和 SSE 生产链仍依赖。
- `cfg.LLM`：用于开关、fallback、seed 和 timeout；不能整体判定为死配置。
- `SimpleMemory`：Agent 运行期消息缓存，和不存在的 Session 持久化 memory 不是一回事。
- `SandboxID`：文件、VNC 和沙箱能力仍按 Session 关联。
- MCP/A2A runtime 与 ToolProvider cleanup：服务热重载和关闭流程仍使用。

## 需要解决的问题

1. **Session 混合了会话和执行事实**：一次 Session 只能绑定一个 task，历史执行、取消竞态、等待输入无法独立表达。
2. **运行状态有多份投影**：Flow、Session、ExecutionStatus 各自表达状态，迟到的完成结果可能覆盖取消结果。
3. **进程内映射承担业务事实**：`taskBySession` 和全局 registry 既负责控制句柄又被 SSE 查询，重启和多实例扩展困难。
4. **消息与实时事件边界不清**：`sessions.events` 保存完整事件，Redis Stream 保存实时输出，两者缺少明确的恢复契约。
5. **配置不可复现**：运行任务没有保存创建时的 Settings 和 Prompt 版本。
6. **职责集中**：AgentService 同时处理聊天、任务生命周期、事件读取、配置与附件，后续扩展成本高。
7. **MCP 动态协议仍有歧义**：function name 反解、初始化失败和错误分类尚未形成稳定契约。

## 重构原则

- 先定义契约，再实现 Run；当前没有 Run 类型，不能先写猜测性的 Run 测试。
- Run 是一次请求的执行事实；Session 只保留会话元数据和 Sandbox 关联。
- PostgreSQL 保存 Run、消息和最终快照；Redis 只承担短期实时事件和 SSE 游标。
- 首期只做进程内 `RunExecutor`，不引入 Worker、Lease、Outbox 或永久事件溯源。
- 只允许一个生产语义切换点；切换前不写 Run，切换后不再写 Session 执行字段，禁止长期双写。
- 每一步都必须可编译、可测试、可独立回滚。
