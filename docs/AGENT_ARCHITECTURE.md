# Agent 核心架构

## 1. 整体架构

go-manus 采用 **分层架构** 和 **ReAct (Reasoning + Acting)** 模式：

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP API Layer                           │
│                      (api/internal/handler/)                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Service Layer                             │
│                      (api/internal/service/)                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Agent Layer                               │
│                     (api/internal/agent/)                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Planner   │───►│   Flow      │───►│    ReAct    │         │
│  │   Agent     │    │  Orchestr.  │    │   Agent     │         │
│  └─────────────┘    └─────────────┘    └──────┬──────┘         │
│                                                │                 │
│                                                ▼                 │
│                                         ┌─────────────┐         │
│                                         │   Tools     │         │
│                                         │  (MCP/A2A)  │         │
│                                         └─────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     External Services                           │
│            (LLM / Sandbox / Browser / File System)              │
└─────────────────────────────────────────────────────────────────┘
```

## 2. 核心组件

### 2.1 BaseAgent

所有 Agent 的基类，提供通用能力：

```go
// api/internal/agent/base.go
type BaseAgent struct {
    name      string
    sessionID string
    config    *AgentConfig
    llm       external.LLM
    tools     []Tool
    memory    *Memory
}
```

**核心能力**：
- 工具注册与管理
- 记忆系统（对话历史）
- LLM 调用封装
- 消息构建与解析

### 2.2 PlannerAgent

任务规划 Agent，负责将复杂任务分解为可执行的步骤：

```go
// api/internal/agent/planner_agent.go
type PlannerAgent struct {
    BaseAgent
}
```

**职责**：
1. 分析用户输入，理解任务目标
2. 分解为多个执行步骤
3. 确定每个步骤需要的工具
4. 输出执行计划（Plan）

### 2.3 ReActAgent

执行 Agent，遵循 ReAct 模式循环执行：

```
┌────────────────────────────────────────────────────────────┐
│                      ReAct 循环                            │
│                                                            │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐            │
│  │ Reason   │───►│  Act     │───►│ Observe  │            │
│  │ 思考      │    │ 执行工具  │    │ 观察结果  │            │
│  └──────────┘    └──────────┘    └──────────┘            │
│       ▲                                       │            │
│       └───────────────────────────────────────┘            │
│                    循环直到完成                             │
└────────────────────────────────────────────────────────────┘
```

**ExecuteStep 流程**：

```go
func (a *ReActAgent) ExecuteStep(ctx context.Context, plan *model.Plan, 
    step *model.PlanStep, message *model.Message) error {
    
    // 1. 构建提示词（包含任务、步骤、历史上下文）
    prompt := a.BuildPrompt(plan, step, message)
    
    // 2. 调用 LLM 获取执行决策
    resp, err := a.llm.Invoke(ctx, &external.LLMRequest{
        Messages: messages,
        Tools:    a.GetToolsForLLM(),
    })
    
    // 3. 解析 LLM 响应
    var result struct {
        Success     bool     `json:"success"`
        Result      string   `json:"result"`
        Attachments []string `json:"attachments"`
    }
    json.Unmarshal([]byte(resp.Content), &result)
    
    // 4. 更新步骤状态
    step.Success = result.Success
    step.Result = result.Result
    step.Attachments = result.Attachments
    
    return nil
}
```

### 2.4 Flow 编排

```go
// api/internal/agent/flow.go
type BaseFlow interface {
    Invoke(message *model.Message) <-chan model.BaseEvent
    Done() bool
}
```

> ⚠️ 接口签名与实现不一致：`PlannerReActFlow.Invoke` 实际签名是
> `Invoke(ctx context.Context, message *model.Message) <-chan model.BaseEvent`，
> 多了 `ctx` 参数。`BaseFlow` 接口目前没有调用方在使用，属于遗留代码，
> 建议统一签名后恢复使用或删除该抽象。

**PlannerReActFlow** 是主要的执行流程：

```
Invoke()
    │
    ▼
┌─────────────────┐
│ 创建 Planner    │
│ Agent           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 调用 Planner    │
│ 生成 Plan       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 遍历 Plan Steps │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 创建 ReAct      │
│ Agent           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ ExecuteStep     │ ◄────┐
│ (ReAct 循环)    │      │
└────────┬────────┘      │
         │               │
         ▼               │
┌─────────────────┐      │
│ 发送 SSE Event  │──────┘
│ (逐步推送结果)  │  (循环直到完成)
└─────────────────┘
```

## 3. Memory 系统

记忆系统管理对话历史，支持上下文理解：

```go
// api/internal/agent/memory.go
type Memory struct {
    sessionID string
    messages  []Message  // 对话历史
    summary   string     // 摘要（节省 token）
}
```

**记忆管理策略**：
- 最近 N 条消息完整保留
- 早期消息压缩为摘要
- 基于 token 数量限制

## 4. 会话管理

```go
// api/internal/model/session.go
type Session struct {
    ID        string
    Status    SessionStatus  // planning/executing/completed/failed
    Plan      *Plan
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## 5. 事件驱动

通过 SSE（Server-Sent Events）实时推送执行进度：

```go
// api/internal/model/event.go
type BaseEvent struct {
    Type    EventType  // status/plan/step/tool/result/error
    Payload interface{}
}
```

**事件类型**：
- `status` - 会话状态变化
- `plan` - 计划生成完成
- `step` - 步骤开始/完成
- `tool` - 工具调用
- `result` - 最终结果
- `error` - 错误信息

## 6. 代码结构

```
api/internal/agent/
├── base.go                 # Agent 基类
├── config.go               # Agent 配置
├── flow.go                 # Flow 接口定义
├── memory.go               # 记忆系统
├── planner_agent.go        # 规划 Agent
├── react_agent.go          # ReAct 执行 Agent
├── planner_react_flow.go   # 规划-执行流程
├── service.go              # Agent 服务层
├── task.go                 # 任务定义
├── task_runner.go          # 任务运行器
├── tools.go                # 工具注册中心
├── prompts.go              # 提示词模板
│
├── tool_*.go               # 各种工具实现
│   ├── tool_file.go
│   ├── tool_shell.go
│   ├── tool_browser.go
│   ├── tool_search.go
│   ├── tool_message.go
│   ├── tool_a2a.go         # Agent 间通信
│   └── tool_mcp.go         # MCP 集成
│
└── *_test.go               # 单元测试
```

## 7. 扩展点

### 7.1 添加新工具

1. 实现 `Tool` 接口：

```go
type Tool interface {
    GetDefinition() ToolDefinition
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}
```

2. 在 `RegisterTools()` 中注册：

```go
func RegisterTools(/* 依赖注入 */) []Tool {
    return []Tool{
        NewFileTool(fileService),
        NewShellTool(sandboxClient),
        // 添加新工具
    }
}
```

### 7.2 自定义 Agent

1. 继承 `BaseAgent`：
```go
type CustomAgent struct {
    BaseAgent
    // 自定义字段
}
```

2. 实现核心方法：
```go
func (a *CustomAgent) ExecuteStep(ctx context.Context, ...) error {
    // 自定义执行逻辑
}
```

### 7.3 集成新 LLM

实现 `external.LLM` 接口：

```go
type LLM interface {
    Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
    GetName() string
}
```