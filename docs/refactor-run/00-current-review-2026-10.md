# API 当前事实基线（2026-10）

本文记录 Run/Session 重构完成后的当前生产事实，以及后续维护必须遵守的边界。

## 核验基线

- 代码基线：`2e3fa27 refactor(api): standardize model JSON codec`
- 核验范围：`api/internal`、`api/migrations`、当前测试和 `docs/refactor-run`
- 当前服务形态：模块化单体，Agent 在进程内执行，PostgreSQL 保存会话数据，Redis Stream 提供实时输出。
- 工作区已有其他未提交修改时，实施者必须先用 `git status` 和 `git diff` 区分归属，不得覆盖。

## 当前生产链

```text
RunHandler / SessionHandler
  -> RunApplicationService
  -> RunService + RunExecutor
  -> PlannerEngineAdapter
  -> PlannerReActFlow
  -> LLM + ToolSet
  -> RunRepository + RunEventStream
```

SSE 通过 Run ID 读取 Redis Stream，`Last-Event-ID` 用于 Run 事件续读。`Session` 不再包含执行状态、TaskID 或事件 JSONB；会话列表的 `active_run_status` 是从 `runs` 读取的只读投影。

## 已完成能力

### 工具系统

`ToolRegistry` 已统一为 `ToolDescriptor`，普通工具和多函数工具共用同一注册索引；MCP 动态 function 已接通；`ToolProvider.Acquire()` 返回任务级 `ToolSet`，`Release()` 负责 MCP/A2A 外部资源引用释放。热重载旧实例会等待引用归零，不应再次把这些内容列为待做。

MCP 动态 function 在发现时建立稳定索引，名称碰撞会拒绝初始化；初始化、连接和协议错误返回 Go error 并清理半初始化资源；远端明确的 `IsError` 保留为失败 `ToolResult`。该契约已由工具重构完成。

### Agent 配置与搜索 limit

启动时从 `app_configs` 读取 Agent 配置，Handler 会传递 `max_search_results`，搜索 provider 使用该 limit。`AgentSettings` 已统一配置类型、范围校验和 revision；每个 Run 在创建时冻结 Settings 快照。不存在可直接引用的 `MaxPlanSteps` 实现。

### Engine 与上下文

`PlannerReActFlow.Invoke` 已拆为状态处理函数。`ContextPolicy` 在首次组装和每次 provider 调用前统一生效，预算仍可使用近似估算；没有 provider tokenizer 时不得伪造精确 token 数。等待输入只由 `StepOutcome.Kind == OutcomeWaitingInput` 表达。

`SimpleMemory`、当前 Plan 指针和步骤结果仍是进程内状态；`waiting_input` 恢复依靠持久化执行快照、步骤结果摘要和问题/回答配对重建上下文，不依赖存活 goroutine。

### 仍在使用、禁止提前删除

- `cfg.LLM`：用于开关、fallback、seed 和 timeout；不能整体判定为死配置。
- `SimpleMemory`：Agent 运行期消息缓存，和不存在的 Session 持久化 memory 不是一回事。
- `SandboxID`：文件、VNC 和沙箱能力仍按 Session 关联。
- MCP/A2A runtime 与 ToolProvider cleanup：服务热重载和关闭流程仍使用。

## 当前限制与演进条件

1. **真实依赖验收已完成**：`make test-integration` 会独立启动 PostgreSQL、Redis、MinIO，覆盖 Run 生命周期、取消竞争和 SSE 续读；后续涉及这条链路的改动必须重跑该门禁。
2. **执行仍是进程内模型**：当前不含 Worker、Lease、Outbox 或永久事件溯源；只有达到架构触发条件时才单独立项。
3. **上下文预算仍是近似值**：当前没有 provider tokenizer，新增模型能力时需评估是否需要精确计量。

## 重构原则

- Run 是一次请求的执行事实；Session 只保留会话元数据和 Sandbox 关联。
- PostgreSQL 保存 Run、消息和最终快照；Redis 只承担短期实时事件和 SSE 游标。
- 继续采用进程内 `RunExecutor`，不在无触发条件时引入 Worker、Lease、Outbox 或永久事件溯源。
- 禁止恢复 Session 执行字段、旧 Task registry 或长期双写。
- 后续改动必须可编译、可测试、可独立回滚。
