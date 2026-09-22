# 当前架构总览

本文档只记录当前代码已经实现的架构。历史设计、阶段性 review 和实验性方案统一放入 `docs/archive/`，不作为实现依据。

## 请求链路

```text
HTTP API -> service -> agent flow -> Planner -> ReAct -> Tool/MCP/A2A
                         |                 |
                         +-> LLM adapter <-+
```

- `api/internal/handler` 负责 HTTP 入参、鉴权错误映射和响应 envelope。
- `api/internal/service` 负责配置、会话和模型管理，不直接拼装 provider 请求。
- `api/internal/agent` 负责规划、工具执行、记忆和事件流。
- `api/internal/llmcore` 定义统一消息、工具、响应、模型画像和计划 Schema。
- `api/internal/external` 实现 OpenAI 兼容、Anthropic、动态模型和流式适配器。

## Planner 与 ReAct 边界

Planner 的职责是生成结构化计划，使用 `response_format=json_object`；它不执行工具。ReAct 阶段才携带工具定义，并处理 `tool_calls`。

这一区分是业务契约，不是针对某个模型的临时规避：结构化计划请求和工具调用请求属于不同响应协议，不能把两者混成一个请求。

## 流式事件

规划和工具调用响应先完整聚合，避免半截 JSON 被执行；总结阶段可以使用文本流式输出。HTTP SSE 使用合法 Redis Stream ID 续读，业务 `event_id` 仅用于去重。

当前一次执行仍由 `Session.Status`、进程内 `AgentService.taskBySession` 和 `RedisStreamTask` 共同承载。Run 领域模型、Run 表和 Run API 尚未实现；相关内容只存在于 `docs/refactor-run/` 的未来方案中。

## 重要代码入口

| 能力 | 入口 |
|---|---|
| Agent 编排 | `api/internal/agent/planner_react_flow.go` |
| Planner | `api/internal/agent/planner_agent.go` |
| ReAct | `api/internal/agent/react_agent.go` |
| LLM 协议 | `api/internal/llmcore/` |
| Provider 适配器 | `api/internal/external/` |
| 模型配置 API | `api/internal/handler/llm_model_handler.go` |

## 当前限制

- Planner 结构化输出依赖模型遵守 JSON 契约；本地 repair 只能修复语法，不能把普通回答转换成计划。
- 不同模型的 reasoning 参数、工具调用和 JSON 模式能力必须通过 `llm_models.request_policy` 与 `capabilities` 配置。
- 内部 component/API 集成测试使用独立 PostgreSQL、Redis 和 MinIO；Sandbox 与 SenseNova 使用显式 external smoke 入口。普通 `go test ./...` 不依赖这些服务。
