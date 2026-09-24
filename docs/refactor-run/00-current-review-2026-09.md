# API 当前实现审查（2026-09）

本文是后续 Run/Session 重构的源码事实基线。它只描述当前 `api/` 代码已经存在的生产链路、已核实的问题和实施约束；`Run`、`RunService`、`RunExecutor`、`PlanSnapshot` 等名称在本文中出现时，除非明确标注“目标”，都不能视为当前代码。

## 审查范围与验证

审查对象是 `api/` 当前分支代码，基线提交为 `911c89c refactor(llm/sse): 抽取通用 SSE 解析层，统一 LLM 适配器流式解析`。工作区另有用户未提交修改，本轮没有覆盖、重置或归因这些修改。

源码核验使用 `rg`、Graphify 查询和带行号阅读。API 普通测试命令为：

```bash
cd api
GOCACHE=/private/tmp/go-manus-review-cache go test ./...
GOCACHE=/private/tmp/go-manus-review-cache go vet ./...
```

本次在受限环境中，`go test ./...` 的 Go 编译和大多数包测试通过，但 `internal/handler` 的 `TestVNCProxy_ProxyEcho`、`internal/sandbox` 的 `httptest.NewServer` 因环境禁止监听回环临时端口失败。这是执行环境限制，不能据此判定生产逻辑失败；需要在允许本地监听的开发终端重跑。文档中的阶段完成不能只依据“部分包通过”。

同一次验证中，`go vet ./...` 发现用户工作区未提交的 `api/internal/llm/anthropic_llm_test.go:532-569` 存在 context cancel 未覆盖所有返回路径的警告。本轮不修改该用户改动，但后续提交前应单独修复并重跑 vet。

## 当前唯一生产执行链

当前请求链仍然是：

```text
HTTP SessionHandler
  -> AgentService.Chat / StopSession
  -> taskBySession[sessionID]
  -> RedisStreamTask
  -> AgentTaskRunner
  -> PlannerReActFlow
  -> PlannerAgent / ReActAgent
  -> LLM + tools
```

关键证据：

| 事实 | 证据 |
|---|---|
| Session 同时包含任务、事件和状态 | [`api/internal/model/session.go:21-35`](../../api/internal/model/session.go#L21-L35) |
| Session 表仍有 `task_id/events/status` | [`api/migrations/001_create_sessions.sql:7-20`](../../api/migrations/001_create_sessions.sql#L7-L20) |
| AgentService 按 Session 保存任务 | [`api/internal/agent/service.go:28-41`](../../api/internal/agent/service.go#L28-L41) |
| Chat 创建/复用 RedisStreamTask | [`api/internal/agent/service.go:64-143`](../../api/internal/agent/service.go#L64-L143)、[`267-306`](../../api/internal/agent/service.go#L267-L306) |
| SSE 按 task ID 读 Redis Stream | [`api/internal/agent/service.go:318-362`](../../api/internal/agent/service.go#L318-L362)、[`api/internal/handler/session_handler.go:337-393`](../../api/internal/handler/session_handler.go#L337-L393) |
| 全局注册表仍在生产读取路径 | [`api/internal/agent/task_redis.go:22-49`](../../api/internal/agent/task_redis.go#L22-L49)、[`api/internal/agent/service.go:323-341`](../../api/internal/agent/service.go#L323-L341) |

`agent.Task`、`Stream`、`RedisStreamTask` 是当前任务基础设施抽象，不是领域实体。它们在 Run 执行器切换前不能删除。

## 已经落地的能力，不能再按旧结论描述

### Agent 配置与搜索 limit 已接通，但还没有统一 Settings 模型

当前 `agent.AgentConfig` 只有 `MaxIterations`、`MaxRetries`、`MaxSearchResults`，默认值为 10、3、10（[`api/internal/agent/config.go:8-45`](../../api/internal/agent/config.go#L8-L45)）。启动时 `initAgent` 从 `AppConfigSvc.GetAgentConfig` 读取数据库值并覆盖默认值（[`api/internal/bootstrap/app.go:731-739`](../../api/internal/bootstrap/app.go#L731-L739)）；Handler 更新时也传递 `MaxSearchResults`（[`api/internal/handler/app_config_handler.go:75-86`](../../api/internal/handler/app_config_handler.go#L75-L86)）。搜索接口带有显式 `limit`，Tavily 和 Bocha 使用该值（[`api/internal/search/search.go:19-31`](../../api/internal/search/search.go#L19-L31)、[`61-75`](../../api/internal/search/search.go#L61-L75)、[`165-178`](../../api/internal/search/search.go#L165-L178)）。

剩余问题是配置治理，而不是断链：`model.AgentConfig` 与 `agent.AgentConfig` 重复；没有 SettingsManager、revision 和任务级不可变快照；`MaxPlanSteps` 尚不存在；规范化逻辑把非正值回退默认值，尚未形成严格的配置校验契约。阶段 1 应围绕这些剩余问题设计，不能重复实现“启动加载”和“搜索 limit 接通”。

### ContextBuilder 已接入，但仍是保守估算

`ContextBuilder` 已按 system、历史和当前问题构建输入，工具调用与结果成组保留（[`api/internal/agent/context_builder.go:19-60`](../../api/internal/agent/context_builder.go#L19-L60)、[`63-96`](../../api/internal/agent/context_builder.go#L63-L96)）。当前预算固定为 32,000，估算采用 rune/4 加消息 envelope（[`api/internal/agent/context_builder.go:11-17`](../../api/internal/agent/context_builder.go#L11-L17)、[`99-115`](../../api/internal/agent/context_builder.go#L99-L115)）。

尚未实现模型画像、输出预留、工具 schema 成本和真实 tokenizer。后续只需扩展策略接口，不要把当前近似值写成“已完成的模型级 token 预算”。

### PlannerReActFlow.Invoke 已经拆分

`Invoke` 现在是调度循环，状态逻辑由 `handleIdle`、`handlePlanning`、`handleExecuting`、`handleWaiting`、`handleUpdating`、`handleSummarizing` 等方法承载（[`api/internal/agent/planner_react_flow.go:100-175`](../../api/internal/agent/planner_react_flow.go#L100-L175)）。后续问题是三套状态的投影和持久化边界，不是 Invoke 尚未拆分。

### LLM 配置仍在使用

`cfg.LLM` 仍用于启动开关、env 模型首次 seed、fallback 配置和通用 timeout；数据库 `llm_models` 是主要动态模型来源（[`api/internal/bootstrap/app.go:442-518`](../../api/internal/bootstrap/app.go#L442-L518)、[`521-573`](../../api/internal/bootstrap/app.go#L521-L573)）。健康状态当前仅在进程内缓存，注释已说明重启后重新统计。不能把 LLM 配置整体标成死代码；后续可把它收敛为部署级 fallback/bootstrap 配置。

### 工具和清理能力的当前边界

工具已下沉到 `api/internal/agent/tools`，`ToolProvider` 负责工具组装和 MCP/A2A 热重载（[`api/internal/agent/tools/tool_provider.go:14-28`](../../api/internal/agent/tools/tool_provider.go#L14-L28)、[`115-190`](../../api/internal/agent/tools/tool_provider.go#L115-L190)）。MCP/A2A 的 retired 列表和 Cleanup 有真实生产调用，不能当作死代码；后续要补 reload 与执行并发的生命周期契约。

文件清理能力已收敛为 Scheduler -> `CleanExpiredFiles`，不要重新把已删除的管理接口列入当前清理清单。

## 主要问题与重构建议

### P0：Session 混合会话事实和执行事实

`Session.TaskID`、`Session.Events`、`Session.Status` 与 `sessions.task_id/events/status` 仍由 SessionRepository、AgentService、TaskRunner 多处读写（[`api/internal/repository/session_repository.go:19-40`](../../api/internal/repository/session_repository.go#L19-L40)、[`91-139`](../../api/internal/repository/session_repository.go#L91-L139)、[`221-253`](../../api/internal/repository/session_repository.go#L221-L253)、[`289-293`](../../api/internal/repository/session_repository.go#L289-L293)）。一个 Session 自然只能绑定一个活跃任务，历史执行、取消优先级和等待输入无法成为独立事实。

建议：Run 作为一次执行单元，Session 只保留会话元数据和 Sandbox 关联；消息与实时事件分别建模。阶段 4 切换前不修改旧写路径，禁止通过双写制造“兼容”。

### P0：任务生命周期依赖进程内两个可变映射

`AgentService.taskBySession` 限制了单实例和单 Session 单任务；`defaultTaskRegistry` 又被 SSE 直接访问。任务完成、SSE 续读、创建下一次任务依赖回调和注册表清理窗口（[`api/internal/agent/service.go:175-243`](../../api/internal/agent/service.go#L175-L243)、[`267-306`](../../api/internal/agent/service.go#L267-L306)）。

首期仍可保留它作为旧链路基础设施，但应把它明确隔离为待替换组件。Run 阶段只保留 `runID -> cancel` 运行期控制句柄，业务状态从 PostgreSQL 查询；当前不引入 Worker、Lease、Outbox 或多实例恢复。

### P1：三套状态模型并存

当前同时存在 `FlowStatus`、`SessionStatus` 和 `ExecutionStatus`；`FlowStatus.ToSessionStatus` 负责投影（[`api/internal/agent/flow.go:5-39`](../../api/internal/agent/flow.go#L5-L39)），TaskRunner 在结束时再次把 Flow 状态写回 Session（[`api/internal/agent/task_runner.go:312-341`](../../api/internal/agent/task_runner.go#L312-L341)）。这不是 Invoke 未拆分，而是终态语义集中度不足。

建议目标是 RunStatus 唯一执行事实，Plan/Step 使用独立快照状态；Engine 返回结果，RunService/Repository 负责条件状态迁移。取消后迟到的 succeeded 必须被 expected-status 条件更新挡住。

### P1：消息/事件的持久化边界不清

`sessions.events` JSONB 追加完整事件，Redis Stream 保存实时输出；`AppendEvent` 还承担 latest message 和 unread 投影（[`api/internal/repository/session_repository.go:221-253`](../../api/internal/repository/session_repository.go#L221-L253)）。token delta、业务事件、长期消息和页面恢复因此混在两条链路中。

建议 PostgreSQL 保存消息和最终业务快照，Redis 只保存短期实时事件；Redis 丢失不改变业务终态，SSE 通过快照校准。当前 SSE 已支持 `Last-Event-ID`/Redis ID，但仍服务旧 task 契约（[`api/internal/handler/session_handler.go:224-266`](../../api/internal/handler/session_handler.go#L224-L266)）。

### P1：AgentService 与 SessionRuntime 仍职责过宽

AgentService 同时负责聊天入口、任务创建/取消、事件读取、配置重载和附件解析；SessionRuntime 同时依赖 repository、service.FileStorage、sandbox 并写 Session 状态（[`api/internal/agent/session_runtime.go:23-65`](../../api/internal/agent/session_runtime.go#L23-L65)）。

建议按职责提取 RunService、RunExecutor、RunEventStream、AttachmentResolver 和 Engine facade，保留现有 `internal/` 骨架，不做大规模目录迁移。

### P2：配置与 Prompt 仍缺少可复现性

Prompt 仍是 Go 常量，七个模板集中在 [`api/internal/agent/prompts.go:3-165`](../../api/internal/agent/prompts.go#L3-L165)，没有文件版本、hash 或 Run 快照。配置存在 `model.AgentConfig`/`agent.AgentConfig` 两个类型和多个入口。阶段 1 应先做 PromptCatalog/SettingsManager，再在 Run 创建时保存快照。

### P2：ContextBuilder 的策略输入不足

当前固定 32k 和近似估算适合首期，但没有模型 context window、输出预留和 provider 差异。先定义可替换 `TokenEstimator`/`ContextPolicy`，不要立即强制引入 tokenizer。

### P2：包边界仍有泄漏

`model` 当前直接依赖 sonic；`event.go` 还依赖 logger（[`api/internal/model/session.go:3-7`](../../api/internal/model/session.go#L3-L7)、[`api/internal/model/event.go:1-10`](../../api/internal/model/event.go#L1-L10)）。这会让领域模型绑定序列化和日志实现。阶段 6 再按影响分析把序列化移到边界层，不应在 Run 切换前混改。

## 不应提前删除的候选

以下符号有当前生产消费者，不能只凭静态搜索删除：

- `agent.Task`、`Stream`、`RedisStreamTask`、`defaultTaskRegistry`：旧 Agent/SSE 生产链仍使用。
- `cfg.LLM`：用于 fallback、seed、timeout 和开关。
- `SandboxConfig.Address/HTTPTimeout`：用于 Sandbox、Browser、SessionService 和 health check；其余字段需逐字段核实。
- MCP/A2A ToolProvider 的 retired 工具和 Cleanup：服务热重载与 shutdown 使用。
- `SimpleMemory`：当前 Agent 实例的运行期消息缓存，虽不是持久化事实，仍被 BaseAgent 使用。

可以在阶段 6、旧生产链彻底不可达且测试通过后删除；不能在阶段 1/2 作为“清理死代码”提前移除。

## 推荐实施顺序与独立回滚边界

1. **校准文档和契约**：以本文为事实基线；不改生产语义。
2. **Prompt/Settings 内部治理**：PromptCatalog、唯一 Settings 类型、严格校验和任务快照；Session/Task API 不变。
3. **Engine/Context 内部治理**：扩展 ContextPolicy，之后再做 StepOutcome；仍只走 Session/RedisStreamTask。
4. **Run 领域与存储**：先写 Run spec、状态迁移矩阵、migration、Repository 和契约测试；Run 未接生产，不产生第二套写语义。
5. **唯一生产切换点**：一次切换 Handler/bootstrap/执行器到 Run；旧路由只能薄委托，禁止双写。
6. **API/UI/SSE 切换**：完成 Run API、Last-Event-ID、快照恢复和 UI 切换，再删除旧路由。
7. **物理清理**：最后删除 Task 基础设施、Session 执行字段和 model 序列化依赖。

每个检查点都必须满足：`go test ./...`、相关 race、`go vet ./...`、`git diff --check`；失败时回到最近通过提交，不临时保留两套长期业务语义。Run 阶段开始前必须先有明确的 Run 领域 spec；当前生产源码没有任何 Run 类型，不能先写“Run 测试”猜契约。

## 后续审查入口

- 当前事实与风险：本文。
- 目标架构：[`01-architecture.md`](./01-architecture.md)。
- Settings/Prompt：[`02-settings-and-prompts.md`](./02-settings-and-prompts.md)。
- Engine/Context：[`03-engine-and-context.md`](./03-engine-and-context.md)。
- Run 开始前的领域与存储：[`04-run-domain-and-storage.md`](./04-run-domain-and-storage.md)。
- 跨会话进度：[`STATUS.md`](./STATUS.md)。
