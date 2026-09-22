# 阶段 2：Engine 结果契约与 ContextBuilder

## 目标

把 Agent 执行内核从“用特殊 error 表达等待输入、用固定 10 条消息压缩上下文”改为显式结果契约和预算化 ContextBuilder。本阶段只替换 Engine 内部协作方式，仍由现有 AgentService、AgentTaskRunner 和 RedisStreamTask 驱动，外部 Session API 行为不变。

## 非目标

- 不新增或写入 Run。
- 不改变 Session 状态字段、Redis stream key 或 SSE 事件格式。
- 不重写 `PlannerReActFlow.Invoke`；该方法已经拆分为多个状态处理函数，本阶段只调整它们的输入输出契约。
- 不接入第三方 tokenizer，不做 LLM 自动摘要。

## 前置条件与唯一语义

- 阶段 1 的 1A/1B 检查点必须先在 `STATUS.md` 标记完成并有独立验证；当前状态仍是部分完成，不得提前实施本阶段。
- 唯一生产业务语义仍是 Session + RedisStreamTask。本阶段不得出现 Run 写路径。
- AgentSettings 和 PromptCatalog 必须直接复用阶段 1 类型，禁止新增配置副本。

## 文件清单

创建：

- `api/internal/agent/outcome.go`：`StepOutcome`、`StepOutcomeKind`、`Failure`。
- `api/internal/agent/context_builder.go`：预算计算、裁剪和消息配对。
- `api/internal/agent/token_estimator.go`：`TokenEstimator` 与保守估算实现。
- `api/internal/agent/context_builder_test.go`、`outcome_test.go`、`token_estimator_test.go`。

修改：

- `api/internal/agent/react_agent.go`：`ExecuteStep` 返回 `StepOutcome, error`，不再修改 Step 来传递等待语义。
- `api/internal/agent/planner_react_flow.go`：按 outcome 更新 Step/Flow；等待输入仍投影成现有 wait 事件和 Session waiting 状态。
- `api/internal/agent/base.go`：LLM 请求统一经 ContextBuilder 构建；删除 `LoadMemory`、`CompactMemory` 和固定 `keepCount=10`。
- `api/internal/agent/memory.go`：收敛为并发安全的运行期消息缓冲，不包含容量、Compact 或持久化承诺；如无独立价值可改名为 `conversation.go`。
- `api/internal/agent/planner_agent.go`、`react_agent.go`：规划、执行、更新计划、总结均使用同一 ContextBuilder 入口。
- `api/internal/agent/task_runner.go`：保持外部循环，适配 Flow 新结果；不得继续识别 `ErrWaitForUser`。
- 对应 Agent 单元测试与集成测试。

删除：

- `api/internal/agent/base.go` 中 `ErrWaitForUser`、`LoadMemory`、`CompactMemory`。
- `Memory.Compact`、`SimpleMemory.maxSize` 和只保留 10 条消息的实现。

阶段 6 才删除 SessionRepository 的 GetMemory/SaveMemory，因为本阶段不扩大仓储接口改动范围；从本阶段起生产执行代码不得调用它们。

## 关键契约

```go
type StepOutcomeKind string

const (
    StepOutcomeSucceeded    StepOutcomeKind = "succeeded"
    StepOutcomeFailed       StepOutcomeKind = "failed"
    StepOutcomeWaitingInput StepOutcomeKind = "waiting_input"
)

type Failure struct {
    Code      string `json:"code"`
    Message   string `json:"message"`
    Retryable bool   `json:"retryable"`
}

type StepOutcome struct {
    Kind        StepOutcomeKind
    Result      string
    Question    string
    Failure     *Failure
    Attachments []string
}

type TokenEstimator interface {
    EstimateMessages([]llmcore.Message) int
}

type ContextPolicy struct {
    MaxInputTokens     int
    ReservedOutputTokens int
}

type ContextBuilder interface {
    Build(ContextInput) ([]llmcore.Message, error)
}
```

`error` 只表示调用无法得到业务结果：LLM/工具基础设施失败、上下文无法满足最低保留集、协议解析失败等。业务 `success=false` 返回 `StepOutcomeFailed`；向用户提问返回 `StepOutcomeWaitingInput`。

`MaxInputTokens` 从所选模型的 `Capabilities.MaxContextTokens` 减去 `RequestPolicy.DefaultMaxTokens` 或模型默认输出预算得出，并设置安全余量。模型画像缺失时使用明确的保守默认值。估算规则为 ASCII 约 4 字符/token、非 ASCII 约 1 字符/token，再乘 1.2；实现必须可替换。

阶段 2 仍使用现有 Session/Task 链：AgentService 在创建任务前通过 LLMModelRepository 解析本次 `model_id` 或默认模型，把 `ContextPolicy` 作为任务快照传入 Engine；不允许从 `LLM.MaxTokens()` 的动态 fallback getter 猜测上下文窗口。阶段 4 创建 Run 时保存同一 model ID 和预算来源，确保后续可复现。

ContextBuilder 的最低保留集是 system、当前用户输入和当前步骤。tool call 与同 `tool_call_id` 的 tool result 形成不可拆分组。裁剪顺序：旧工具组、旧助手/用户轮次、较早步骤结果。若最低保留集已超预算，返回可识别的 context limit error，禁止静默截断当前输入。

## 数据库与 API 变更

无数据库和公开 API 变更。现有 wait/error/done/step SSE 事件继续输出相同结构，Session 的 `waiting/completed/failed` 投影暂时保留。此约束用于让阶段 2 可独立回滚和部署。

## 实施顺序与检查点

1. 检查点 2A：新增 TokenEstimator/ContextBuilder，将所有 LLM 输入切入 Builder，删除伪容量和固定压缩逻辑；运行 Agent 与现有 Session 契约测试，提交 `refactor(api): centralize agent context budgeting`。
2. 检查点 2B：新增 StepOutcome，一次性迁移 ReActAgent、PlannerReActFlow、TaskRunner 并删除 ErrWaitForUser；运行阶段全量门禁，提交 `refactor(api): make step outcomes explicit`。

每个检查点都只能存在一种生产行为：2A 完成后所有 LLM 请求必须走 Builder；2B 提交后所有等待语义必须走 outcome。

## 针对性测试

- StepOutcome：waiting 必须有 question，failed 必须有 failure，非法组合构造失败。
- ContextBuilder：中英文估算、最低保留集、tool call/result 成对保留、裁剪优先级、预算不足错误、输入切片不被修改。
- Flow：等待输入不再返回 `ErrWaitForUser`，仍发 message + wait 事件并停在 waiting；业务失败与系统错误分开。
- 回归：waiting 后下一条用户消息继续当前计划，不重新创建计划；总结流式输出和现有事件顺序不变。
- race：并发读取消息快照时无数据竞争。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/agent -run 'Test(ContextBuilder|StepOutcome|PlannerReActFlow)' -count=1
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
cd ..
git diff --check
```

## 提交与回滚

回滚使用 `git revert` 按 2B、2A 逆序执行，回滚任一点后都必须保持现有 Session API 可运行。

## 完成定义

- 生产代码不存在 `ErrWaitForUser`、固定 `keepCount=10` 和无效 maxSize。
- 所有 LLM 请求通过 ContextBuilder；工具消息不会被拆对。
- 业务失败、等待输入和系统错误具有不同类型。
- Session/Task 外部契约未改变，全量 test、race、vet 通过。
- `STATUS.md` 记录提交与结果；阶段 3 可与本阶段之后独立开始，阶段 4 必须等待阶段 2、3 都完成。

## 失败或中断恢复

读取 `STATUS.md` 后运行：

```bash
git log -4 --oneline
rg -n 'ErrWaitForUser|CompactMemory|keepCount|maxSize' api/internal/agent
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/agent -count=1
```

若 Outcome 迁移只完成一半，不得用适配器长期同时支持 error 和 outcome。回到最近通过测试的提交，继续一次性迁完调用链。
