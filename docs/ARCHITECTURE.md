# 当前架构总览

本文只描述 `api/` 当前已经实现的生产架构。历史方案、阶段性 review 和已删除的 Task 链路统一放在 `docs/archive/` 或 `docs/refactor-run/`，不能用来解释当前代码。

## 核心边界

```text
HTTP Handler
  -> RunApplicationService
  -> RunService / RunExecutor
  -> PlannerEngineAdapter
  -> PlannerReActFlow
  -> LLM + ToolSet
  -> RunRepository + RunEventStream
```

- `Session` 是会话容器，只保存标题、未读数、Sandbox 关联和时间等元数据。
- `Run` 是一次顶层执行请求，负责状态、幂等键、设置快照、执行快照和错误事实。
- `RunMessage` 保存 Run 的持久化用户、助手和工具相关消息；PostgreSQL 中的 Run/Message 是执行事实来源。
- `RunApplicationService` 编排创建 Run、提交等待输入、取消、查询和历史列表。
- `RunExecutor` 管理一次 Engine 调用的控制句柄、取消、ToolSet 生命周期和终态持久化。
- `PlannerEngineAdapter` 把现有 Planner/ReAct Flow 适配到 Run 执行边界，等待输入恢复时从执行快照和持久化消息重建上下文。
- `AgentService` 负责 Settings、Prompt、ToolProvider 等运行时装配，不再承载按 Session 保存的任务状态。

## 请求与恢复

顶层请求通过 `POST /api/sessions/:id/runs` 创建 Run。服务在创建时冻结本次执行所需的设置和输入，使用数据库唯一约束保证同一 Session 的活跃 Run 和幂等创建语义。

Engine 返回等待用户输入时，服务将问题消息、`waiting_input` 状态和 `RunExecutionSnapshot` 一起持久化。快照包含计划元数据、当前步骤、已完成步骤摘要、等待问题和恢复模式。用户回答通过 `POST /api/runs/:runId/input` 继续同一 Run；服务重新获取 Settings 和 ToolSet，从快照及历史消息重建 Engine Context，回答只注入当前步骤一次。

进程重启后不依赖旧 goroutine 或 `SimpleMemory`：遗留 `pending/running` Run 收敛为 `interrupted`，`cancelling` 收敛为 `cancelled`；完整的 `waiting_input` 快照可以恢复，损坏快照返回明确领域错误并停止静默重规划。

## 状态与取消

Run 状态由 `model.RunStatus` 的迁移矩阵统一定义：`pending`、`running`、`waiting_input`、`cancelling`、`succeeded`、`failed`、`cancelled`、`interrupted`。

取消先通过数据库条件更新把可取消状态迁移到 `cancelling`，再通知进程内执行句柄。Engine goroutine 退出后才释放 ToolSet，并将 Run 收敛为 `cancelled`；迟到的成功或失败不能覆盖已取消状态。控制句柄保持到 waiting 或终态事实持久化完成，句柄缺失或持久化暂时失败时由幂等 reconciliation 收敛。

## 实时事件与 SSE

PostgreSQL 负责最终 Run、Message、状态和执行快照；Redis Stream 只负责短期实时事件和 SSE 续读，不是业务事实来源。事件以 Run ID 隔离，SSE 接受 `Last-Event-ID`，从游标之后读取，保留原事件 ID，允许网络边界下的至少一次重复投递。客户端按 `run_id + event_id` 去重，游标过期时查询 Run 快照而不是伪造历史事件。

Run SSE 的终止事件为 `done`、`error` 或 `wait`；终态快照以 PostgreSQL 查询为准。`GET /api/sessions/:id/stream` 仍然存在，但只推送会话目录变化，不承载 Run 执行事件。

## 工具与模型边界

- `internal/llmcore` 定义消息、工具调用、响应和计划 Schema 等稳定协议。
- `internal/external` 实现 OpenAI 兼容、Anthropic、动态模型和流式适配器。
- `internal/agent/tools` 统一 File、Shell、Browser、Search、Message、MCP 和 A2A 工具协议。
- 每个 Run 获取冻结的 ToolSet；MCP/A2A 连接由 ToolProvider 管理引用生命周期，不能由 Handler 直接创建或关闭。
- Planner 只生成结构化计划，不执行工具；ReAct 执行工具调用。Planner JSON 响应和 ReAct tool-call 响应是两种不同协议，不能混用。

## 主要代码入口

| 能力 | 入口 |
|---|---|
| Run API | `api/internal/handler/run_handler.go` |
| Session API | `api/internal/handler/session_handler.go` |
| Run 应用编排 | `api/internal/service/run_application_service.go` |
| Run 状态与持久化边界 | `api/internal/service/run_service.go`、`api/internal/repository/run_repository.go` |
| Run 执行生命周期 | `api/internal/service/run_executor.go` |
| 等待恢复 | `api/internal/service/run_recovery.go`、`api/internal/agent/run_engine_adapter.go` |
| Planner/ReAct | `api/internal/agent/planner_react_flow.go`、`planner_agent.go`、`react_agent.go` |
| 工具系统 | `api/internal/agent/tools/` |
| LLM 协议 | `api/internal/llmcore/` |
| Provider 适配器 | `api/internal/external/` |
| 路由装配 | `api/internal/router/router.go`、`api/internal/bootstrap/app.go` |

## 测试与运行入口

普通单元测试不依赖外部服务：

```bash
cd api
make test-unit
go vet ./...
```

真实 PostgreSQL、Redis、MinIO 组件测试和 Run HTTP/SSE 集成测试使用独立环境：

```bash
cd api
make test-component
make test-api
make test-integration
```

SenseNova、Search 和真实 Sandbox 是显式 external smoke，不属于普通回归门禁。测试分层和环境变量说明见 [`API_TEST_PLAN.md`](./API_TEST_PLAN.md) 与 [`api/tests/README.md`](../api/tests/README.md)。

## 当前限制

- Planner 仍依赖模型遵守结构化 JSON 契约；repair 只能修复有限语法问题，不能把普通文本可靠转换为计划。
- 不同模型的 reasoning、工具调用和 JSON 模式能力由 `llm_models.request_policy` 与 `capabilities` 控制。
- Redis 丢失不会改变 Run 最终事实，但会影响短期实时事件和断线续读；客户端必须回查 Run 快照。
- 管理接口默认不做鉴权，这是当前部署决策，不在本文档中假设已有权限边界。
