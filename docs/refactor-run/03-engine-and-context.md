# 阶段 2：Engine、Outcome 与 ContextPolicy

## 当前事实

`PlannerReActFlow.Invoke` 已经由多个状态处理函数组成；本阶段不再拆 Invoke。检查点 2A 已完成：等待输入已从 `InvokeResult.WaitForUser`、`ToolCallResult.WaitForUser` 和 `ErrWaitForUser` 三套表达收敛为唯一的 `StepOutcome.Kind`。

## 目标

让 Engine 只负责计算并返回明确结果，把等待用户、业务失败、取消和系统错误区分开；让上下文预算成为可替换策略，而不是散落的常量。

## Engine 输出

建议引入内部值对象：

```go
type StepOutcome struct {
    Kind        OutcomeKind // completed / waiting_input / fatal_failure / cancelled
    Text        string
    Waiting     *WaitingInput
    Failure     error
    Attachments []string
}
```

- `StepOutcome.Kind` 是 Engine/Flow 唯一控制信号；`waiting_input` 是正常业务结果，不用 error 或布尔字段表达。
- LLM、工具、存储和协议错误继续返回 `error`，同时保留可诊断原因。
- 取消由 `context.Context` 传播，不把取消伪装成普通失败。

`StepOutcome` 不暴露 `retryable_failure`。当前 LLM 调用失败重试和空响应重试已经在 BaseAgent 内闭环，并由任务级 `MaxRetries` 快照限制次数；重试耗尽后才向外返回 `fatal_failure` 或 `error`。Flow、RunExecutor 和 RunService 不得再次重试同一次 LLM/工具调用，避免出现双重重试、重复工具副作用和悬空 Outcome 枚举。未来如需跨进程重派发，应由独立的 Run 调度契约设计，不能复用 Engine 内部重试语义。

迁移顺序固定为：工具调用结果在 BaseAgent 边界转换成 `StepOutcome`，ReAct 和 Flow 只判断 `Kind`，旧 TaskRunner 临时把 `StepOutcome` 映射到当前 Session/Task 行为。检查点 2A 已删除 `ErrWaitForUser`、`InvokeResult.WaitForUser` 和 `ToolCallResult.WaitForUser`；不得让过渡字段重新进入 Run 设计。

`WaitingInput` 至少包含稳定问题文本、附件、是否建议用户接管以及当前 step 标识；附件在 BaseAgent 边界归一为 `[]string`，避免工具参数的动态类型泄漏到 Flow。进入 Run 阶段后再由持久化层分配 message ID。

## ContextPolicy

把当前固定预算抽象为策略输入：

- 模型 context window
- 输出预留
- system/history/current message/tool schema 的预算
- token 估算器
- 工具调用与结果的成组保留规则

首期仍可使用近似估算；不要为没有 provider tokenizer 的场景伪造精确 token 数。必须保证裁剪不会拆散工具调用和结果配对。

## 工具边界前置治理

RunExecutor 会把任务级 `ToolSet` 作为 Engine 输入，因此在本阶段同时收紧 MCP 的动态调用契约：

- `MCPTool.GetTools` 在发现阶段生成稳定且唯一的 function name，并建立只读的 `functionName -> {server, tool}` 索引；`InvokeWithName` 只查索引，不再通过拼接字符串和遍历 map 反解。
- 名称编码必须可逆或由索引直接解释；发现重复 function name 时初始化失败，不允许依赖 map 迭代顺序选择目标。
- manager 初始化、工具发现、连接、超时和协议错误返回 `error`；远端工具明确返回 `IsError` 时使用失败 `ToolResult`，让 Engine 能区分系统错误与可交给模型处理的工具业务失败。
- 初始化失败必须释放已创建的 manager/client，`ToolProvider` 不注册半初始化工具。

这组修复独立于 Run 状态机，可以作为单独提交完成；不得借机重写 ToolRegistry 或 ToolSet 引用生命周期。

## 检查点

- Flow 的等待输入、成功、失败、取消行为测试保持通过。
- BaseAgent 的 LLM 错误和空响应重试仍在 Engine 内闭环；测试证明重试不跨越 Flow/RunExecutor 边界，且耗尽后只产生一个最终 Outcome。
- 全仓不再存在 `ErrWaitForUser`、`InvokeResult.WaitForUser` 或 `ToolCallResult.WaitForUser`；等待输入只由 `OutcomeWaitingInput` 表达。
- 旧 TaskRunner 的临时适配只消费 `StepOutcome`，不定义另一套 Outcome。
- ContextPolicy 在首次组装和每次真实 provider 调用前都生效；system、历史、当前请求、工具 schema 与输出预留共享同一窗口。
- 裁剪必须保留最后一个用户请求及其后的完整工具链原始顺序；不得把用户请求移动到 tool result 之后，也不得写入未实际执行的 tool call。
- ContextPolicy 对超预算、空历史、工具调用配对、工具 schema 预算和输出预留有行为测试。
- MCP 动态 function 无名称碰撞和随机路由；初始化错误、协议错误与远端业务失败的测试分别锁定契约。
- 旧 Session/RedisStreamTask 生产写路径完全不变。

建议提交：

- `refactor(tools): make mcp dispatch unambiguous`
- `refactor(agent): return explicit outcomes and context policy`

## 回滚

本阶段没有 migration 和路由切换。若新 Outcome 或 ContextPolicy 影响旧 Flow，回滚该提交即可；不得为了兼容而保留两套 Engine 结果长期并行。
