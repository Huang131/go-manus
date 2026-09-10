# go-manus `api/` 代码 Review 与重构报告

> 项目路径：`/Users/huang/GolandProjects/study/go-manus/api`
> 分析时间：2026-09-10
> 分析范围：后端 `api/` 全模块（bootstrap / handler / service / agent / external / repository / llmcore / model / config）
> 分析方法：静态分析（`go vet`）+ 分层人工审阅 + 与 `docs/smell-report-2026-08-31.md` 对照复核
> 前提：**开发阶段，不考虑代码/数据历史兼容性**

---

## 一、Executive Summary

相较 2026-08-31 的旧报告，项目已明显演进：`service/` 层已建立（session/file/appconfig/llmmodel/status/filecleanup）、`SessionRepository` 的文件方法已拆分到 `FileRepository`、`json_parser_repair.go` 的死代码已清理、`go vet` 零告警、91 个源文件全部单测通过。

但仍存在**一个真实功能缺陷**与**多处结构性重复**，本次 review 重新核实后确认如下：

| 维度 | 评级 | 核心问题 |
|------|------|---------|
| 正确性 | ⚠️ 有隐患 | Anthropic 客户端错误未分类 → 模型 fallback 完全失效 |
| 重复代码 | ❌ 偏高 | 4 份 `WithTx`（已漂移）、2 套 LLM HTTP 脚手架、Scan 列表重复 |
| 复杂度 | ⚠️ 中 | `PlannerReActFlow.Invoke`(233行) / `TaskRunner.Invoke`(227行) 长方法仍在 |
| 分层/架构 | ⚠️ 中 | handler 直连 agent、AgentService 仍 13 依赖、bootstrap 多值返回 |

---

## 二、Findings（问题清单）

### 🔴 A. 正确性

| # | 位置 | 问题 | 影响 |
|---|------|------|------|
| A1 | `external/anthropic_llm.go` Invoke/Stream | 所有错误用裸 `fmt.Errorf` 返回，未归一化为 `llmcore.ProviderError` | `routed_llm.go:97` 的 fallback 依赖 `err.(*ProviderError)` 类型断言，Anthropic 错误断言必失败 → **Anthropic 模型永远无法 fallback**；且无 tool-call 超时（OpenAI 有 `toolCallTimeout`） |
| A2 | `repository/session_repository.go` 各 UPDATE | 不校验 `RowsAffected` | id 不存在时影响 0 行但返回 `nil` error，调用方无法感知"会话不存在"，静默失败 |
| A3 | `repository/session_repository.go` 写方法 | `deleted_at IS NULL` 过滤不一致 | 读方法过滤软删除，但 Update/AppendEvent/SaveMemory/UpdateStatus 等不过滤，已删除会话仍可被写入 |
| A4 | `external/routed_llm.go:250` `RecordFailure` | `_ = err` 丢弃错误类型 | 健康打点不区分 auth/瞬时故障，配置错误的模型被当瞬时故障处理 |

### 🟡 B. 重复代码

| # | 位置 | 问题 |
|---|------|------|
| B1 | 4 个仓储 `WithTx` | `file`/`app_config`/`llm_model` 三份逐字相同（copy-paste），`session` 版本独自演进出 `committed/rolledBack` 双标志兜底 → **行为漂移**，其余三个缺兜底 |
| B2 | `openai_llm.go` / `anthropic_llm.go` | Invoke+Stream 的 HTTP 脚手架（marshal→建请求→发送→错误分类→读体→非2xx→反序列化→归一化）大量重复；OpenAI 内部 Invoke/Stream 重复约 80 行；Anthropic 的 tools 转换循环在 Invoke/Stream 逐字重复 |
| B3 | `file_repository.go` / `session_repository.go` | Scan 字段列表内联重复（file 5 次、session 3 次）；`llm_model_repository.go` 已有正确范式（`scanLLMModel` + `llmModelColumns` 常量）却未推广 |
| B4 | `file_repository.go` | `ListBySessionID`/`GetExpiredFiles`/`GetFilesBySessionIDs` 三处相同的 rows 遍历样板 |

### 🟡 C. 长方法 / 复杂度

| # | 位置 | 行数 | 问题 |
|---|------|------|------|
| C1 | `agent/planner_react_flow.go` `Invoke` | 233 | goroutine 内 `for{switch 7-case}` 状态机，每 case 15-45 行，无法单测 |
| C2 | `agent/task_runner.go` `Invoke` | 227 | 状态机+重试+事件+序列化混合，深层嵌套 |
| C3 | `agent/base.go` `Invoke` / `handleToolCall` | 152 / 96 | 多重试+工具调用+消息构造混合 |

### 🔵 D. 分层 / 架构

| # | 位置 | 问题 |
|---|------|------|
| D1 | `handler/session_handler.go:25-29` | `SessionHandler` 同时依赖 `service.SessionService` 与 `agent.AgentService`，agent 编排未收口到 service 层 |
| D2 | `agent/service.go:20-56` | `AgentService` 13 依赖字段 + 构造函数 13 参数（God Object）；`taskBySession` 是进程内 map，多实例/重启即丢 |
| D3 | `bootstrap/app.go:517` `initExternalClients` | 返回 6 个值 → `initAgent` 收 7 参（Long Parameter List / 多值返回） |

### 🔵 E. 其他

- E1 `service/session_service.go:46,57` `vncPort = 5901` 硬编码，应进 config
- E2 `session_repository.List` 的 `COUNT(*)` 与数据查询不在同一事务，total 可能轻微不一致
- E3 `GetMemory` 用 `memories->>$2`（返回 text）再 Unmarshal，大 JSON 多一次转换

---

## 三、Refactoring 执行记录（本次已落地）

> 原则：每步保持 `go build` / `go vet` / `go test ./...` 全绿；开发阶段不考虑兼容性。

| 批次 | 项 | 处理 |
|------|----|------|
| 1 | B1 | 在 `repository/transaction.go` 提取泛型 `runInTx` helper，4 个仓储 `WithTx` 统一委托，消除重复并让全部仓储获得 `committed/rolledBack` 兜底 |
| 1 | B3/B4 | `file_repository.go` 引入 `fileColumns` 常量 + `scanFile` helper；`session_repository.go` 引入 `scanSession`；新增泛型 `collectRows[T]` 收敛 rows 遍历样板 |
| 1 | A1 | Anthropic 客户端新增 `classifyHTTPError`，把网络/超时/HTTP 错误归一化为 `llmcore.ProviderError`，补齐 tool-call 超时；与 OpenAI 行为对齐，恢复 fallback 能力。配套单测覆盖 8 个 HTTP 状态码分类、网络错误、tool-call 超时、协议错误、Stream 429、默认超时兜底 |
| 1 | A2/A3 | session 写方法统一加 `deleted_at IS NULL` 过滤 + `RowsAffected` 校验，命中 0 行返回 `ErrSessionNotFound` |
| 2 | C1 | `PlannerReActFlow.Invoke` 状态机每个 case 抽成独立 `handleXxx` 方法，主循环退化为调度 |
| 2 | D3 | `bootstrap` 引入 `externalClients` struct 替代 6 值返回 |

未落地（建议后续单独排期）：B2 完整共享 HTTP 脚手架（改动面大）、C2/C3 长方法、D1/D2 架构级拆分、E1-E3。

---

## 四、验证

- `go build ./...`：通过
- `go vet ./...`：通过
- `go test ./...`：全部包通过
- `gofmt -l internal/ cmd/`：无输出（批次 2 收尾复核）
