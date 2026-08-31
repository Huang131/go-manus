# Go Manus 项目 Review 报告

> 对比原始 Python 版本 (mooc-manus) 与 Go 版本 (go-manus) 的功能差异

## 问题概览

| 优先级 | 问题 | 状态 |
|--------|------|------|
| ~~🔴 P0~~ | ~~工具注册不完整~~ | ✅ 已完成 |
| ~~🔴 P0~~ | ~~缺少 JSON 解析器~~ | ✅ 已完成 |
| ~~🔴 P0~~ | ~~MCP 工具未实现~~ | ✅ 已完成 |
| ~~� P0~~ | ~~缺少完整 ReAct 循环~~ | ✅ 已完成 |
| ~~� P1~~ | ~~缺少消息队列~~ | ✅ 已完成 |
| ~~🟡 P1~~ | ~~缺少异步能力~~ | ✅ 已完成 |
| ~~� P1~~ | ~~工具调用数量限制~~ | ✅ 已完成 |
| ~~� P1~~ | ~~空响应处理和重试~~ | ✅ 已完成 |
| ~~🟢 P2~~ | ~~测试覆盖不足~~ | ✅ 已完成 |
| ~~🔴 P0~~ | ~~缺少 Rollback 机制~~ | ✅ 已完成 |
| ~~🟡 P1~~ | ~~MessageTool 缺少交互能力~~ | ✅ 已完成 |
| ~~🟢 P2~~ | ~~JSON 解析实现差异~~ | ✅ 已完成 |
| ~~🟢 P2~~ | ~~消息队列方法缺失~~ | ✅ 已完成 |
| ~~🟢 P2~~ | ~~MCP 环境变量未传递~~ | ✅ 已完成 |

---

## 一、已完成的功能

### 1. 工具注册 ✅

Go 版本已注册全部 7 个工具：
- ShellTool
- FileTool
- BrowserTool
- SearchTool
- MessageTool
- MCPTool
- A2ATool

### 2. JSON 解析器 ✅

使用自定义正则修复实现 `RepairJSONParser`。

### 3. MCP 工具 ✅

实现了基于 stdio 的 MCP 客户端，支持 JSON-RPC 通信。

### 4. 消息队列 ✅

基于 Redis Stream 实现了完整的消息队列接口。

### 5. 异步能力 ✅

使用 goroutine + channel 实现了等价的非阻塞模型。

### 6. 测试覆盖 ✅

包含单元测试和集成测试。

---

## 二、待修复问题详情

### P0-1: 缺少完整 ReAct 循环

**问题描述**

Go 版本的 ReActAgent 只做了一次 LLM 调用，缺少迭代循环处理多轮工具调用。

**Python 版本实现**

```python
# base.py invoke() 方法
async def invoke(self, query: str) -> AsyncGenerator[BaseEvent, None]:
    # 1. 调用 LLM 获取响应
    message = await self._invoke_llm([{"role": "user", "content": query}])

    # 2. 循环遍历直到最大迭代次数
    for _ in range(self._agent_config.max_iterations):
        # 3. 如果无工具调用则表示 LLM 生成文本回答
        if not message or not message.get("tool_calls"):
            break

        # 4. 循环遍历工具参数并执行
        for tool_call in message["tool_calls"]:
            result = await self._invoke_tool(tool, function_name, function_args)
            tool_messages.append({...})

        # 5. 工具执行完成后再次调用 LLM
        message = await self._invoke_llm(tool_messages)
```

**Go 版本当前实现**

```go
// react_agent.go ExecuteStep() 只做了一次 LLM 调用
resp, err := a.llm.Invoke(ctx, &external.LLMRequest{
    Messages: messages,
    Tools:    a.GetToolsForLLM(),
})
// 没有处理 tool_calls，没有迭代循环
```

**修复方案**

重构 `ExecuteStep` 方法，实现完整的 ReAct 循环：
1. 循环调用 LLM 直到达到最大迭代次数或 LLM 不再调用工具
2. 解析 LLM 响应的 tool_calls
3. 执行工具调用并收集结果
4. 将工具结果作为消息继续调用 LLM

---

### P0-2: Memory 未持久化

**问题描述**

Python 版本的 Agent 会将 Memory 持久化到数据库，支持会话中断恢复。Go 版本只有内存存储。

**Python 版本实现**

```python
async def _add_to_memory(self, messages: List[Dict[str, Any]]) -> None:
    await self._ensure_memory()
    self._memory.add_messages(messages)
    # 持久化到数据库
    async with self._uow:
        await self._uow.session.save_memory(self._session_id, self.name, self._memory)
```

**修复方案**

1. 在 `SessionRepository` 添加 `SaveMemory` 和 `GetMemory` 方法
2. 在 `BaseAgent` 的 `AddMemory` 方法中调用持久化
3. 启动时从数据库恢复 Memory

---

### P0-3: 缺少 Rollback 机制

**问题描述**

Python 版本有 `roll_back` 方法，用于在用户发送新消息时修正 Agent 的消息列表状态。

**Python 版本实现**

```python
async def roll_back(self, message: Message) -> None:
    """用于确保消息列表状态正确，用于发送新消息、暂停/停止任务"""
    last_message = self._memory.get_last_message()
    if last_message.get("tool_calls"):
        tool_call = last_message["tool_calls"][0]
        function_name = tool_call.get("function", {}).get("name")

        if function_name == "message_ask_user":
            # 特殊处理：添加用户回复作为工具响应
            self._memory.add_message({...})
        else:
            # 删除最后一条消息
            self._memory.roll_back()
```

**修复方案**

在 `BaseAgent` 添加 `Rollback` 方法：
```go
func (a *BaseAgent) Rollback(message *model.Message) error
```

---

### P1-4: 工具调用数量限制

**问题描述**

Python 版本限制 LLM 一次只能调用一个工具，避免并发问题。

**Python 版本实现**

```python
# base.py _invoke_llm() 方法
filtered_message["tool_calls"] = message.get("tool_calls")[:1]  # 限制为1个
```

**修复方案**

在解析 tool_calls 时，只处理第一个工具调用。

---

### P1-5: 空响应处理和重试

**问题描述**

Python 版本有完整的空响应检测和自动重试机制。

**Python 版本实现**

```python
if not message.get("content") and not message.get("tool_calls"):
    logger.warning("LLM回复了空内容，执行重试")
    await self._add_to_memory([
        {"role": "assistant", "content": ""},
        {"role": "user", "content": "AI无响应内容，请继续。"}
    ])
    continue  # 重试
```

**修复方案**

在 LLM 调用后检测空响应，增加重试逻辑和 max_retries 配置。

---

### P1-6: MessageTool 缺少交互能力

**问题描述**

Python 版本的 `message_ask_user` 工具可以暂停 Agent 等待用户输入，Go 版本缺少 `WaitEvent` 机制。

**Python 版本实现**

```python
# react.py execute_step() 方法
if event.function_name == "message_ask_user":
    if event.status == ToolEventStatus.CALLING:
        yield MessageEvent(role="assistant", message=event.function_args.get("text"))
    elif event.status == ToolEventStatus.CALLED:
        yield WaitEvent()  # 等待用户输入
        return
```

**修复方案**

1. 在 `model/event.go` 添加 `WaitEvent` 类型
2. 在 `ReActAgent` 中处理 `message_ask_user` 工具调用
3. 在 `PlannerReActFlow` 中处理 `FlowStatusWaiting` 状态

---

### P2-7: JSON 解析实现差异

**问题描述**

Python 使用 `json_repair` 库，Go 使用自定义正则修复。

**修复方案**

考虑引入第三方库或增强正则修复逻辑。

---

### P2-8: 消息队列方法缺失

**问题描述**

Python 版本有以下额外方法：
- `get_range()` - 批量获取消息
- `get_latest_id()` - 获取最新消息 ID

**修复方案**

在 `MessageQueue` 接口添加这两个方法。

---

### P2-9: MCP 环境变量未传递

**问题描述**

Go 版本的 `StdioMCPClient.getConfigEnv()` 返回空 map。

**修复方案**

在配置中支持 MCP 服务器的环境变量传递。

---

## 三、架构差异说明

### 为什么 Go 版本不使用 Task + TaskRunner + MessageQueue 架构？

| 原因 | 解释 |
|------|------|
| **Go/Go 习惯不同** | Go 更倾向于用 goroutine + channel 直接实现异步，不需要 Task 抽象层 |
| **过早简化** | 初始实现时可能为了快速出 demo，做了过度简化 |
| **没有完整对照** | 可能没有深入分析 Python 版本就进行了移植 |
| **Redis Stream 未充分利用** | Go 版本虽然实现了消息队列，但只是用了基础的 API，没有像 Python 那样构建分布式 Task 框架 |

Python 版本的 `Task + TaskRunner + MessageQueue` 架构优势在于：
- **分布式支持**：Task 可以跨进程、跨机器
- **状态持久化**：任务状态存在 Redis Stream 中，进程重启可恢复
- **解耦**：任务创建者不需要等待任务完成

Go 版本当前设计更适合**单体应用**，如果要支持分布式，需要重构。

---

## 四、修复计划

```
P0 (阻塞功能)
├── P0-1: 完整 ReAct 循环 → ✅ 已完成
├── P0-2: Memory 持久化 → ✅ 已完成
└── P0-3: Rollback 机制 → ✅ 已完成

P1 (重要功能)
├── P1-4: 工具调用数量限制 → ✅ 已完成
├── P1-5: 空响应处理和重试 → ✅ 已完成
└── P1-6: MessageTool 交互能力 → ✅ 已完成

P2 (优化项)
├── P2-7: JSON 解析实现差异 → ✅ 已完成
├── P2-8: 消息队列方法缺失 → ✅ 已完成
└── P2-9: MCP 环境变量未传递 → ✅ 已完成
```

**所有问题已修复完成！**

---

## 五、修复完成总结

### 已完成 ✅

所有 P0、P1、P2 问题均已修复：

**P0 (阻塞功能)**
- 工具注册不完整
- 缺少 JSON 解析器
- MCP 工具未实现
- 缺少完整 ReAct 循环
- Memory 持久化
- Rollback 机制

**P1 (重要功能)**
- 缺少消息队列
- 缺少异步能力
- 工具调用数量限制
- 空响应处理和重试
- MessageTool 交互能力

**P2 (优化项)**
- 测试覆盖不足
- JSON 解析实现差异（使用 json-repair 第三方库）
- 消息队列方法缺失（GetRange、GetLatestID）
- MCP 环境变量未传递

### 架构演进说明

Go 版本当前设计更适合**单体应用**。如需支持分布式架构（如 Python 版本的 Task + TaskRunner + MessageQueue），需要进行以下重构：

1. **Task 抽象层**：基于 Redis Stream 构建分布式任务
2. **TaskRunner 重构**：支持跨进程任务执行
3. **状态持久化**：任务状态存入 Redis Stream，支持进程重启恢复
4. **消息队列增强**：利用已实现的 GetRange/GetLatestID 支持批量操作

---

## 三、RedisStreamTask 架构对齐 ✅

> 完成时间：2026-08-25

### 1. 对齐目标

将 Go 版本的 Agent 执行架构与 Python 版本的 RedisStreamTask 架构对齐，实现 Agent 执行解耦到后台 goroutine。

### 2. 关键设计决策

| 决策项 | Python 版本 | Go 版本 | 说明 |
|--------|------------|---------|------|
| TaskRegistry | `Dict[str, Task]` | `sync.RWMutex + map` | Go 风格更直接 |
| input_stream.pop() | `asyncio` 阻塞 | `XREAD BLOCK` | 对齐 Redis XREAD |
| output_stream.put() | 写后触发 | 写后轮询 | 前端 SSE 轮询 |
| goroutine 生命周期 | `asyncio.CancelledError` | `context.Context` | Go 惯用模式 |
| 任务注册 | 类变量存储 | 全局变量 + RWMutex | 对齐设计 |

### 3. 实现组件

#### 3.1 TaskRunner 接口

```go
type TaskRunner interface {
    Invoke(ctx context.Context, task *RedisStreamTask) error
    Destroy() error
    OnDone(task *RedisStreamTask)
}
```

#### 3.2 RedisStreamTask 架构

```
RedisStreamTask
├── id: string
├── runner: TaskRunner
├── inputStream: MessageQueue (Redis Stream)
├── outputStream: MessageQueue (Redis Stream)
├── cancelFunc: context.CancelFunc
├── done: atomic.Bool
└── 方法:
    ├── Invoke(ctx) - 启动后台 goroutine 执行
    ├── execute(ctx) - 执行逻辑
    ├── onDone() - 完成回调
    ├── Cancel() - 取消任务
    ├── PutInput() - 往 input_stream 放消息
    └── GetOutput() - 从 output_stream 取消息
```

#### 3.3 执行流程

```
AgentService.Chat(sessionID, message)
  ↓
获取或创建 RedisStreamTask(sessionID)
  ↓
task.Invoke(ctx) - 启动后台 goroutine
  ↓
AgentTaskRunner.Invoke(ctx, task) - 后台执行
  ↓
循环: task.InputStream().Pop() - 阻塞获取输入
  ↓
Flow.Invoke(message) - 执行 Agent Flow
  ↓
task.OutputStream().Put(event) - 写入输出
  ↓
前端轮询 task.GetOutput() - SSE 推送
```

### 4. 文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `task_redis.go` | 完善 | TaskRegistry、RedisStreamTask、TaskStream 实现 |
| `task_runner.go` | 完善 | AgentTaskRunner.Invoke() 对接 Task 架构 |
| `service.go` | 重构 | Chat() 方法对接 Task 架构 |
| `main.go` | 扩展 | 添加 MessageQueue 依赖 |
| `event.go` | 扩展 | MessageEvent 对齐 Python 版本 |
| `task_redis_test.go` | 新增 | TaskRegistry 单元测试 |

### 5. 向后兼容

- 保留 `AgentTaskRunner.Run()` 方法（向后兼容）
- `AgentService.Chat()` 返回 `taskID` 而非 `eventChan`
- 前端通过轮询 `GetTaskEvents()` 获取输出事件

### 6. 使用示例

```go
// 1. Chat 方法现在返回 taskID
taskID, err := agentService.Chat(ctx, sessionID, message)

// 2. 前端轮询获取事件
events, err := agentService.GetTaskEvents(ctx, taskID, "")

// 3. 停止会话
err := agentService.StopSession(ctx, sessionID)
```

---

## 四、架构对比总结

| 功能模块 | Python 版本 | Go 版本 | 状态 |
|----------|------------|---------|------|
| 工具注册 | ✅ 完整 | ✅ 完整 | 对齐 |
| JSON 解析 | ✅ json-repair | ✅ 自定义正则修复 | 对齐 |
| MCP 工具 | ✅ 完整 | ✅ 完整 | 对齐 |
| 消息队列 | ✅ Redis Stream | ✅ Redis Stream | 对齐 |
| ReAct 循环 | ✅ 完整 | ✅ 完整 | 对齐 |
| Task 架构 | ✅ RedisStreamTask | ✅ RedisStreamTask | 对齐 |
| 测试覆盖 | ✅ 完整 | ✅ 基础 | 持续完善 |