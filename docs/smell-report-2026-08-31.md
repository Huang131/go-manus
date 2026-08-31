# go-manus 项目代码异味（Smell）分析与重构建议

> 项目路径：`/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus`
> 分析时间：2026-08-31
> 分析范围：后端 `api/` 模块（agent / external / handler / repository / router / service）
> 分析方法：`smell` skill 框架（架构 / 耦合 / 设计 / 代码 / 复杂度 / 测试 六大类）

---

## 一、Executive Summary（执行摘要）

go-manus 是一个**"快速从 Python 迁移过来的 AI Agent 框架"**。功能可用，但已经显现出 **架构膨胀 + 关注点混合 + 死代码残留** 的早期信号：

| 维度 | 评级 | 说明 |
|------|------|------|
| 架构清晰度 | ⚠️ 中 | 分层在，但 `agent` 层承担过重（缺 `service/` 业务编排层） |
| 代码可测试性 | ⚠️ 中 | 部分核心（PlannerReAct、TaskRunner）几乎不可单测 |
| 可维护性 | ❌ 偏低 | 多个 God Object / Long Method / Dead Code |
| 性能与正确性 | ❌ 有隐患 | `json_parser_repair.go` 存在严重 Bug（UTF-8 损坏、Python布尔 no-op、unreachable code） |
| 工程完备度 | ⚠️ 中 | 错误处理整体规范，但有 stub / 注释当除臭剂痕迹 |

**核心结论**：建议优先做 **3 件事**，能显著降低风险与维护成本：

1. **修掉 `json_parser_repair.go` 的真实 Bug 与死代码**（影响所有 LLM JSON 解析）
2. **拆分 `AgentService` God Object**（按生命周期 / 任务 / 工具 / 事件四个职责切分）
3. **重写 `PlannerReActFlow.Invoke`**（用策略模式替换 230 行 switch state machine）

---

## 二、Architectural Style Detected（架构风格识别）

整体属于 **Layered + Modular Agent** 风格：

```
api/internal/
├── handler/      ← HTTP 层
├── service/      ← 业务编排（本项目几乎没存在感，应该是空的）
├── agent/        ← ⚠️ 过度膨胀（God Folder）
│   ├── service.go              AgentService
│   ├── task_runner.go          AgentTaskRunner
│   ├── task_redis.go           RedisStreamTask
│   ├── planner_react_flow.go   PlannerReActFlow
│   ├── base.go                 BaseAgent
│   ├── planner_agent.go
│   ├── react_agent.go
│   ├── flow.go
│   └── ...
├── repository/   ← 数据访问
├── external/     ← LLM / JSON / VNC / MCP / A2A 等外部能力
├── model/        ← DTO / 领域模型
└── router/       ← 路由注册
```

**问题**：
- 层级是清楚的，但 `agent/` 目录已经开始承担"业务编排层"的职责（应该由 `service/` 承担）
- `agent/service.go`（AgentService）变成 **God Object**
- `service/` 目录几乎为空；handler 直接依赖 agent，跳过了 service 层
- `agent/` 子包之间互相耦合（TaskRunner → PlannerReActFlow → BaseAgent）
- `external/vnc` 与 `agent` 是双向耦合

> **建议恢复分层**：`handler → service → agent → repository/external`

---

## 三、Findings by Category（按类别的问题清单）

### 🔴 Critical（必须立刻处理）

| # | 文件 | Smell | 影响 |
|---|------|-------|------|
| C1 | `external/json_parser_repair.go:65-89` | **Dead Code — `fixJSON` 链** | `fixJSON` 与多个辅助方法（`fixSingleQuotes` / `fixUnquotedKeys` / `removeJSComments` / `fixNonASCII` / `fixPythonBooleans`）仅被 `fixJSON` 内部调用，且 `fixJSON` 唯一外部调用者是 `ExtractJSONField`（line 223），但 `ExtractJSONField` 本身**在生产代码中从未被任何地方调用**。等于一整条修复链是死代码 |
| C2 | `external/json_parser_repair.go:114-125` `fixUnquotedKeys` | **Bug — unreachable code + dead regex** | `re := regexp.MustCompile("\\s*([a-zA-Z_]...")` 永远不匹配；末尾 `return text` 是 unreachable。`go vet` 早已告警 |
| C3 | `external/json_parser_repair.go:149-154` `fixPythonBooleans` | **Bug — no-op typo** | `strings.ReplaceAll(text, ": true", ": true")` 完全是 no-op，**应该是 `: True → : true`、`: False → : false`**。如果 fixJSON 真的被调用，Python 风格 bool 完全不会修复 |
| C4 | `external/json_parser_repair.go:134-147` `fixNonASCII` | **Bug — UTF-8 损坏** | 把所有非 ASCII 字符转义成 `\uXXXX`，LLM 输出大量中文时会把 `"你好"` 变成 `"\u4f60\u597d"`，破坏后续渲染与匹配 |
| C5 | `external/json_parser_repair.go:267-275` `containsInvalidChars` | **Dead Code** | 方法定义后从未被任何地方调用 |

> **C1~C5 都在 LLM JSON 解析热路径上**。即使当前 `Parse` 走的是 `jsonrepair` 库，**死代码 + 多个真实 Bug 的存在本身就是定时炸弹**（一旦有人切换实现就触发）。

### 🟡 Warnings（建议在 1~2 周内处理）

| # | 文件 | Smell | 描述 |
|---|------|-------|------|
| W1 | `agent/service.go` `AgentService` | **God Object** | 15+ 依赖字段：sessionRep / fileRep / configRep / llm / sandbox / agentConfig / mcpConfig / a2aConfig / browser / searchEngine / mcpTool / a2aTool / mq / runningTasks / taskBySession。承担：会话生命周期、任务调度、工具注册、事件查询 |
| W2 | `agent/service.go` `NewAgentService` | **Long Parameter List** | 构造函数 11 个参数，未来再加一个依赖就要改一堆调用方 |
| W3 | `agent/service.go` | **Divergent Change（双轨实现）** | 同时存在 `getOrCreateTaskRunner`（Deprecated）与 `getOrCreateTask` 两条并行任务创建路径，需要双轨维护 |
| W4 | `agent/service.go` `RegisterTool` | **Speculative Generality / Stub** | 注释写"工具注册到所有活跃任务"，实现只打了一行 log，没有真实逻辑（误导 + 隐藏 Bug） |
| W5 | `agent/service.go:125` | **Resource Leak（vet warning）** | `taskCtx, _ := context.WithCancel(context.Background())` 返回的 cancel 函数被丢弃，goroutine / timer 可能泄漏 |
| W6 | `agent/task_runner.go` `AgentTaskRunner.Invoke` | **Long Method + Deep Nesting** | 单方法 200+ 行，4 层嵌套（状态机 + 重试 + 事件处理 + JSON 序列化），可读性极差 |
| W7 | `agent/planner_react_flow.go` `PlannerReActFlow.Invoke` | **Switch Statements + Long Method + Deep Nesting** | `for { switch status { 7 个 case } }`，~230 行；用 if-else/switch 模拟状态机，应该用策略/多态 |
| W8 | `agent/planner_react_flow.go` | **Comments as Deodorant** | 存在 `// TODO: 转换 Attachments` 等占位注释 |
| W9 | `repository/session_repository.go` `SessionRepository` | **Fat Interface（ISP 违反）** | 17+ 方法（AppendEvent / AddFile / RemoveFile / GetFileByPath / GetMemory / SaveMemory / UpdateTitle / UpdateLatestMessage / UpdateStatus / IncrementUnreadCount / DecrementUnreadCount / SetUnreadCount / WithTx / ...），客户端被迫依赖整个接口 |
| W10 | `agent/flow.go` `BaseFlow` | **Alternative Classes w/ Different Interfaces** | `BaseFlow.Invoke(message)` 与 `PlannerReActFlow.Invoke(ctx, ...)` 签名不一致，接口几乎没用上 |
| W11 | `agent/base.go` `BaseAgent.Invoke` | **Long Method** | ~130 行，多重试 + 工具调用 + 消息构造混在一个方法里 |

### 🔵 Suggestions（长期演进项）

| # | 文件 | Smell | 描述 |
|---|------|-------|------|
| S1 | `repository/*` | **Primitive Obsession** | `agentName`、`id`、`taskID` 都是裸 string，应该用 value object（如 `type AgentID string`）避免误用 |
| S2 | 多处 | **Magic Numbers** | `5901`（VNC 端口）、`popRetryBaseDelay` / `popRetryMaxDelay` / `popRetryMaxCount` 等硬编码，应该集中到 config |
| S3 | `agent/task_runner.go` | **Double Wrapping** | event 包了两层（`baseEvent` + `wrappedJSON`），可改成一次构造 |
| S4 | `external/json_parser_repair.go` | **Regex in Hot Path** | 即使修复后，也应该把 `regexp.MustCompile` 提到包级 `var`，避免每次 Parse 重新编译 |
| S5 | `agent/service.go` | **Common Coupling（隐式共享状态）** | `AgentService.runningTasks` / `taskBySession` 是进程内 map，多实例部署或重启即丢；建议下沉到 Redis |
| S6 | `agent/service.go` | **Service Locator 反模式** | 内部用 `sandbox.GetTool(name)` / `mq.GetConsumer(name)` 这种 locator 调用，破坏静态可分析性 |
| S7 | `handler/vnc_handler.go` | **Incomplete Refactor** | `decodeSubprotocolFrame` 在 `base64` 路径上直接返回 error，没有真正实现解码；并发关闭不优雅（两个 goroutine + errCh，无 graceful shutdown） |

---

## 四、Detailed Findings（重点案例详解）

### 4.1 🔴 C1~C5：`json_parser_repair.go` 的 Bug 与 Dead Code

**死代码调用链**：

```
ExtractJSONField()                       ← 唯一调用 fixJSON 的地方
   └── fixJSON()                          ← line 65
       ├── removeMarkdownCodeBlocks       ← Parse 也直接调用，存活
       ├── removeTrailingCommas          ← Parse 也直接调用，存活
       ├── fixSingleQuotes                ← ❌ 仅 fixJSON 调用（死代码）
       ├── fixUnquotedKeys                ← ❌ 仅 fixJSON 调用（死代码）
       ├── removeJSComments               ← ❌ 仅 fixJSON 调用（死代码）
       ├── fixNonASCII                    ← ❌ 仅 fixJSON 调用（死代码 + BUG）
       └── fixPythonBooleans              ← ❌ 仅 fixJSON 调用（死代码 + BUG）
```

```go
// 真正的 Parse 走这条路（已正确）：
func (p *RepairJSONParser) Parse(text string, v interface{}) error {
    cleaned := p.removeMarkdownCodeBlocks(text)
    if err := json.Unmarshal([]byte(cleaned), v); err == nil { return nil }
    fixed, err := jsonrepair.RepairJSON(cleaned)  // ← 实际生效的修复
    // ... 兜底 extractJSON
}

// 但下面这一大堆 helper 没人调用：
func (p *RepairJSONParser) fixJSON(text string) string { ... }
func (p *RepairJSONParser) fixPythonBooleans(text string) string {
    text = strings.ReplaceAll(text, ": true", ": true")    // ❌ 应替换 ": True"
    text = strings.ReplaceAll(text, ": false", ": false")  // ❌ 应替换 ": False"
}
func (p *RepairJSONParser) fixNonASCII(text string) string {
    return asciiOnly.ReplaceAllStringFunc(text, func(m string) string {
        return fmt.Sprintf("\\u%04x", []byte(m)[0])  // ❌ 中文被损坏
    })
}
```

**重构建议**：
1. **删除整个 `fixJSON` 链**（C1）：从 `fixJSON` 函数体、`ExtractJSONField`（line 219-249）一起删
2. **删除 `containsInvalidChars`**（C5）
3. `Parse` 内联 `extractJSON` 调用即可
4. 给 `Parse` 加单元测试覆盖：中文 JSON、Python bool、嵌套引号、转义符

### 4.2 🟡 W1：`AgentService` God Object

```go
type AgentService struct {
    mu           sync.RWMutex
    sessionRep   repository.SessionRepository
    fileRep      repository.FileRepository
    configRep    repository.AppConfigRepository
    llm          external.LLM
    sandbox      external.Sandbox
    agentConfig  *AgentConfig
    mcpConfig    *MCPConfig
    a2aConfig    *A2AConfig
    browser      external.Browser
    searchEngine external.SearchEngine
    mcpTool      *MCPTool
    a2aTool      *A2ATool
    mq           external.MessageQueue
    runningTasks   map[string]*AgentTaskRunner       // W3 双轨
    taskBySession  map[string]*RedisStreamTask        // W3 双轨
}
```

**重构建议**：按职责拆分为 4 个服务：

| 新服务 | 职责 | 原 AgentService 方法 |
|--------|------|----------------------|
| `SessionService` | 会话生命周期 | Chat / GetMessages / GetEvents |
| `TaskRegistry` | 任务管理（内存/Redis） | getOrCreateTask / runningTasks / taskBySession |
| `ToolRegistry` | 工具注册 / 查询 | RegisterTool / mcpTool / a2aTool |
| `EventService` | 事件读取 | GetEvents / Subscribe |

`AgentService` 退化为 facade，统一注入。

### 4.3 🟡 W6/W7：`PlannerReActFlow` 与 `TaskRunner` 的 Long Method + Switch

`PlannerReActFlow.Invoke` 的状态机：

```go
for {
    select {
    case <-ctx.Done(): return
    case status := <-statusCh:
        switch status {
        case FlowStatusIdle:        ...
        case FlowStatusPlanning:    ...
        case FlowStatusExecuting:   ...
        case FlowStatusWaiting:     ...
        case FlowStatusUpdating:    ...
        case FlowStatusSummarizing: ...
        case FlowStatusCompleted:   ...
        }
    }
}
```

**重构建议**：用 Strategy / State 模式：

```go
type FlowStateHandler interface {
    Handle(ctx, flow *PlannerReActFlow) (next FlowStatus, err error)
}

var handlers = map[FlowStatus]FlowStateHandler{
    FlowStatusIdle:        &IdleHandler{},
    FlowStatusPlanning:    &PlanningHandler{},
    FlowStatusExecuting:   &ExecutingHandler{},
    // ...
}

func (f *PlannerReActFlow) Invoke(ctx context.Context, msg *model.Message) error {
    status := FlowStatusIdle
    for {
        next, err := handlers[status].Handle(ctx, f)
        if err != nil || next == FlowStatusCompleted { return err }
        status = next
    }
}
```

每个 Handler 可以独立单测。

### 4.4 🟡 W9：`SessionRepository` Fat Interface

```go
type SessionRepository interface {
    AppendEvent(...)
    AddFile(...)
    RemoveFile(...)
    GetFileByPath(...)
    GetMemory(...)
    SaveMemory(...)
    UpdateTitle(...)
    UpdateLatestMessage(...)
    UpdateStatus(...)
    IncrementUnreadCount(...)
    DecrementUnreadCount(...)
    SetUnreadCount(...)
    WithTx(...)
    ...
}
```

**重构建议（ISP）**：按聚合根拆分：

```go
type SessionCoreRepo    interface { Get / Update / UpdateTitle / UpdateStatus }
type SessionEventRepo    interface { AppendEvent / GetEvents }
type SessionFileRepo     interface { AddFile / RemoveFile / GetFileByPath }
type SessionMemoryRepo   interface { GetMemory / SaveMemory }
type SessionCounterRepo  interface { IncrementUnreadCount / ... }

// 业务侧按需组合：
type SessionUseCase struct {
    core    SessionCoreRepo
    events  SessionEventRepo
    files   SessionFileRepo
    memory  SessionMemoryRepo
    counter SessionCounterRepo
}
```

---

## 五、Dependency Graph Analysis（依赖图观察）

```
handler ──► agent.AgentService ──► repository
   │              │
   │              ├──► external (llm, browser, search, mcp, a2a, vnc)
   │              │
   │              └──► agent.{BaseAgent, TaskRunner, PlannerReActFlow}
   │                              │
   │                              └──► external / repository
   │
   └──► handler.vnc_handler ──► service.SessionService
```

**观察**：
- `handler → agent.AgentService` 跳过了 `service/` 层（service 几乎为空）
- `agent` 子包之间互相耦合（TaskRunner 依赖 PlannerReActFlow，PlannerReActFlow 依赖 BaseAgent）
- `external/vnc` 与 `agent` 是双向耦合（agent 通过 Sandbox 调 vnc，vnc 通过回调写 event）
- 没有 `service/` 层 → 业务编排逻辑全部沉到 `agent`，加重 God Object

---

## 六、Module Health Scorecard（模块健康度打分）

| 模块 | 健康度 | 主要扣分项 |
|------|--------|-----------|
| `agent/service.go` | ⭐⭐☆☆☆ | God Object、Long Param、Speculative |
| `agent/task_runner.go` | ⭐⭐☆☆☆ | Long Method、Deep Nesting |
| `agent/planner_react_flow.go` | ⭐⭐☆☆☆ | Switch、Long Method、Deep Nesting |
| `agent/base.go` | ⭐⭐⭐☆☆ | Long Method |
| `agent/flow.go` | ⭐⭐⭐☆☆ | 接口不一致 |
| `external/json_parser_repair.go` | ⭐☆☆☆☆ | Dead Code + 4 个真实 Bug + 1 个 unreachable code |
| `repository/session_repository.go` | ⭐⭐⭐☆☆ | Fat Interface、Primitive |
| `repository/其他` | ⭐⭐⭐⭐☆ | 整体可接受 |
| `handler/vnc_handler.go` | ⭐⭐⭐☆☆ | subprotocol 处理半成品、并发关闭不优雅 |
| `model/` | ⭐⭐⭐⭐☆ | OK |
| `router/` | ⭐⭐⭐⭐☆ | OK |

---

## 七、Smell Distribution（异味分布统计）

| 类型 | 数量 | 占比 |
|------|------|------|
| Long Method | 4 | 25% |
| God Object / God Folder | 2 | 12% |
| Switch Statements | 1 | 6% |
| Fat Interface | 1 | 6% |
| Primitive Obsession | 2 | 12% |
| Dead Code | 2（含 6+ dead helper） | 12% |
| Bug（功能正确性） | 4 | 25% |
| Speculative / Stub | 1 | 6% |
| 其他（Magic / Common Coupling / Service Locator） | 2 | 12% |

---

## 八、Refactoring Roadmap（重构路线图）

### Immediate（本周，1~2 天）
- [x] ~~删除 `json_parser_repair.go` 死代码（`fixJSON`及其全部 helper）~~ → 已在 2026-08-31 PR 中处理
- [ ] 修复或删除 `RegisterTool` stub（注释误导）
- [ ] 给 `NewAgentService` / `Chat` 加上 cancel 函数回收（修 vet warning）

### Short-Term（1~2 周）
- [ ] **拆分 `AgentService`** 为 SessionService / TaskRegistry / ToolRegistry / EventService
- [ ] **拆分 `SessionRepository`** 为 Core / Event / File / Memory / Counter 5 个小接口
- [ ] 给 `PlannerReActFlow.Invoke` 引入 State Handler 模式
- [ ] 把 `AgentTaskRunner.Invoke` 拆成 `runOnce` + `retryLoop` + `emitEvent`

### Long-Term（1 个月+）
- [ ] 引入 `service/` 层，把 handler → service → agent 链路打通
- [ ] `agentName` / `id` / `taskID` 改为 typed value object
- [ ] 把 `runningTasks` / `taskBySession` 迁到 Redis，支持多实例
- [ ] 给 `BaseFlow` 接口统一签名（ctx 第一参数）
- [ ] 配置中心化：5901 端口、retry 参数等全部进 config
- [ ] 引入 contract test（agent ↔ repository ↔ external 各层都有 mock 测试）

---

## 九、面试高频考点提炼（针对资深 Go 后端岗）

如果把这套 smell 整理成面试素材，可以覆盖以下考点：

| 考点 | 在本项目中的例子 |
|------|------------------|
| **God Object** | AgentService 12+ 依赖 |
| **ISP 违反** | SessionRepository 17+ 方法 |
| **State Machine 实现** | PlannerReActFlow 的 switch 模式 → Strategy |
| **Long Method 重构** | TaskRunner.Invoke 拆分 |
| **Dead Code 识别** | json_parser_repair.go 的 6 个 helper |
| **Primitive Obsession** | typed ID（`type AgentID string`）|
| **Context 泄漏** | `context.WithCancel` 的 cancel 丢弃 |
| **Common Coupling** | 进程内 map 共享状态 vs Redis |
| **Service Locator 反模式** | `sandbox.GetTool(name)` |
| **可测试性设计** | State Handler 模式让每个状态可单测 |

> 每一项都同时具备 **What / How / Why / Alternatives / Trade-offs** 五层框架，可以直接作为面试题讲解。

---

## 十、结语

go-manus 是一个**功能已基本可用、架构开始预警**的阶段。当前最大的风险不是性能，也不是扩展性，而是 **AgentService 与 PlannerReActFlow 的两个 Long Method** 在未来加新功能时必然爆炸。

按 **Immediate → Short-Term → Long-Term** 的顺序推进，预计 1~2 周可以完成关键拆分，让项目进入"可演进"状态。

---

## 附录 A：当前 PR 范围（2026-08-31）

本次 review 仅处理了 **Immediate 阶段**的第一项：

| 操作 | 文件 | 状态 |
|------|------|------|
| 删除 `fixJSON` 函数（连同 6 个内部 helper） | `external/json_parser_repair.go` | ✅ |
| 删除 `ExtractJSONField`（唯一调用 fixJSON 的地方） | `external/json_parser_repair.go` | ✅ |
| 删除 `containsInvalidChars`（无任何调用） | `external/json_parser_repair.go` | ✅ |
| 删除对应的 4 个测试函数 | `external/json_parser_test.go` | ✅ |
| 保留 `removeMarkdownCodeBlocks` / `removeTrailingCommas` / `extractJSON` | `external/json_parser_repair.go` | ✅（`Parse` 还在用） |

保留方法均有 `Parse` 路径或独立测试覆盖，**删除时经过审慎的调用方核查**。
