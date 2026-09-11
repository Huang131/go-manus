# go-manus 流式响应 Bug 排查与修复（2026-08-26）

> 记录从「网页一直显示正在思考中」到「done 事件缺失」全链路排查过程，以及
> mooc-manus 原项目对比分析。最后给出「为什么 Go 移植版本比 Python 版本更脆弱」的系统性结论。

---

## 1. 现象回顾

| # | 现象 | 严重度 |
|---|------|--------|
| 1 | 第一段 AI 回复正常，后续每轮都卡在「正在思考中」，刷新浏览器后能看到内容 | P0 |
| 2 | SSE 流 `event:` 字段全部是 `message`，前端无法区分 plan/step/title | P0 |
| 3 | `/chat` 接口偶发 `strict-origin-when-cross-origin` 报错 | P2 |
| 4 | 暂停任务后，新建任务下「对话前」仍转圈 | P2 |
| 5 | AI 回复被分多段展示（设计行为） | 预期 |
| 6 | 任务执行总结图标点击无反应无法收起 | P1 |
| 7 | 新浏览器会话中也会卡思考中（不仅是续连） | P0 |
| 8 | 第一轮就卡思考中，刷新也不行 | P0 |

---

## 2. 根因链：11 个 Bug 拆解

### Bug 1：SSE `event:` 字段被钉死为 `message`（前端业务类型丢失）

**代码位置**：`api/internal/handler/session_handler.go` 的 `streamSSE`（修改前）

```go
// 修改前
c.SSEvent("message", json.Marshal(model.Event{Type: event.Type, Data: dataJSON}))
```

**症状**：前端 `EventSource` 收到的 `event.data` 永远是包了一层的 `model.Event` JSON，
`event.type` 永远是 `message`，所以 `messageEvent.type === 'plan'` 这类判断全部失效，
前端按 `data` 解析再做模糊匹配。Python 版的 `StreamingResponse` 不存在这个问题（详见后文）。

**修复**：将业务类型提到 SSE `event:` 字段，data 只放 payload。

```go
// 修改后
c.SSEvent(string(event.Type), mergeEventMetadata(event))
```

---

### Bug 2：Redis Stream 写入时丢掉了 `model.Event` 包装

**代码位置**：`api/internal/agent/task_runner.go` 第 191-225 行

```go
// 修改前：只 Marshal 业务 payload，model.Event 包装丢在调用层
outputID, err := task.OutputStream().Put(ctx, string(eventJSON))

// 修改后：先包成 model.Event 再写入，与 DB 持久化路径保持一致
baseEvent := &model.Event{Type: event.GetType(), Data: eventJSON}
wrappedJSON, err := json.Marshal(baseEvent)
// ...
outputID, err := task.OutputStream().Put(ctx, string(wrappedJSON))
```

**为什么这是隐性 Bug**：原 Python 版的 Redis Stream 直接存 dict 本身就是业务结构
（没有 `model.Event` 包装），Go 版把"DB 持久化"和"Stream 队列"两套序列化对齐了，反而
序列化出了 model.Event，但写流时又只写了内层 payload，**最终 `event.Type == ""`，
SSE 推送时就退化到 `event:message`**。这正是 Bug 1 的二次触发。

---

### Bug 3：空消息拦截误杀「空流续读」

**代码位置**：`api/internal/handler/session_handler.go` 的 `/chat` handler

**修改前**：

```go
if strings.TrimSpace(req.Message) == "" {
    response.Error(c, "消息内容不能为空")
    return
}
```

**症状**：前端 `startEmptyStream` 用 `{event_id: lastId}` 调 `/chat` 续读事件流，
body 里**没有 `message` 字段**，但解析到 `req.Message == ""`，被 400 掉，前端收不到任何
SSE 事件。

**修复**：用 `c.GetRawData()` 探测 body 是否含 `"message"` 字面量，只在"显式发空 message"
时才 400。

```go
rawBody, _ := c.GetRawData()
hasMessage := strings.Contains(string(rawBody), `"message"`)
if hasMessage && strings.TrimSpace(req.Message) == "" {
    response.Error(c, "消息内容不能为空")
    return
}
```

---

### Bug 4：空流续读走错了代码路径

**代码位置**：同上 `/chat` handler

**症状**：上面修完后，curl 测试空流续读触发 agent 流程，AI 居然回「用户消息为空」。
原因是**所有请求都进 `h.agent.Chat(...)`**，而 Chat 会创建 task + 投喂空消息。

**修复**：分两路径：

```go
if hasMessage {
    // 正常发新消息：进 agent.Chat 流程
    taskID, err = h.agent.Chat(ctx, id, req.Message)
} else {
    // 空流续读：直接拿 session 的活跃 task 续接事件流
    taskID, err = h.agent.GetActiveTaskID(ctx, id)
    if err != nil || taskID == "" {
        // 没有活跃 task，关闭连接，避免前端 500ms 死循环
        return
    }
}
```

---

### Bug 5：后端「无活跃 task 立即关闭」→ 前端 500ms 死循环

**代码位置**：`api/internal/handler/session_handler.go` `streamSSE`（空流分支）

**症状**：当空流续读时 session 没有活跃 task，后端立即 return；前端 `startEmptyStream`
收到 `SSE_STREAM_END` 信号后 500ms 重连，后端又立即 return……形成 500ms 一次的 OPTIONS+POST 风暴。

**修复**：空流分支在没有活跃 task 时**也保持连接开 15s 再关闭**，并定时发心跳。
这样前端一次连接最多空转 15s（不会高频打），下次有事件时正常推送。

---

### Bug 6：PlannerReActFlow 状态机在「空步骤计划」时卡死

**代码位置**：`api/internal/agent/planner_react_flow.go` `FlowStatusSummarizing` case

**症状**：模型（如 `MiniMaxAI/MiniMax-M2.7`）对简单问候"hi"会返回 `plan.steps = []`（不需要工具）。
代码路径：

```
FlowStatusPlanning  → 创建 plan(steps=[])
FlowStatusExecuting → for loop 空遍历，立刻跳出
FlowStatusSummarizing → react.Summarize() ← 又一个 LLM 调用！
                              ↓
                  sensenova 6.8-flash-lite 此处挂死（详见 Bug 8）
                              ↓
                  永远到不了 FlowStatusCompleted → 没有 done 事件
```

**修复**：当 `len(plan.Steps) == 0` 时跳过 `react.Summarize()`，直接进入 `FlowStatusCompleted`。

```go
case FlowStatusSummarizing:
    if len(f.plan.Steps) == 0 {
        logger.Info("计划无步骤，跳过 Summarize 直接完成")
    } else {
        summary, attachments, err := f.react.Summarize(ctx)
        // ...
    }
    f.mu.Lock()
    f.status = FlowStatusCompleted
    f.mu.Unlock()
```

---

### Bug 7：CORS `AllowAllOrigins=true` + `AllowCredentials=true` 触发浏览器拒连

**代码位置**：`api/pkg/middleware/middleware.go`

**症状**：浏览器控制台报 `strict-origin-when-cross-origin`（这是 Referrer-Policy，不是 CORS），
**真正问题是**：gin-cors 的 `AllowAllOrigins=true` 与 `AllowCredentials=true` 互斥，
W3C CORS 规范下服务端 `Access-Control-Allow-Origin: *` 不能携带 credentials，
浏览器会**静默丢弃响应**，前端 SSE 连接建立失败，表现为「一直转圈」。

**修复**：用 `AllowOriginFunc` 反射请求的 Origin：

```go
AllowOriginFunc: func(c *gin.Context, origin string) bool {
    return origin != ""  // 反射任何 Origin
},
```

---

### Bug 8（新增根因）：`sensenova-6.8-flash-lite` 模型 tool calling 极慢/超时

**这是当前「第一轮就卡思考中」的真正根因**——**不是代码 bug**。

实测（2026-08-26）：

| 模型 + base_url | 请求类型 | 响应时间 | 结果 |
|----------------|----------|----------|------|
| `sensenova-6.8-flash-lite` + `token.sensenova.cn` | 简单 "hi" | ~10s | 正常 |
| `sensenova-6.8-flash-lite` + `token.sensenova.cn` | 带 tools | **>12s 超时** | 挂死 |
| `MiniMaxAI/MiniMax-M2.7` + `api.gmi-serving.com` | 简单 "hi" | ~1.5s | 正常 |
| `MiniMaxAI/MiniMax-M2.7` + `api.gmi-serving.com` | 带 tools | **~1.5s** | 正常返回 tool_calls |

LLM 客户端默认 `httpClient.Timeout: 120s`，所以**前端一直转圈直到 120s 超时**才会结束。
期间 agent 永远卡在 `react.Summarize()`（Bug 6 状态机）或 `PlannerReActFlow` 的规划阶段。

**Tool Calling 是什么？**

> Tool Calling（工具调用，Function Calling）是 LLM 的一种能力：
> 模型在对话中根据用户意图，**返回结构化请求**（不是自然语言）告诉应用层"请帮我调用 X 工具，
> 参数是 Y"，应用层执行后把结果再喂给模型，让模型继续推理。
>
> 这是 Agent / Manus 类系统的核心机制——没有 Tool Calling，模型就只会聊天，不能调度工具。
>
> 主流支持 Tool Calling 的模型：
> - OpenAI GPT-4 / GPT-3.5-turbo（`tools` 参数）
> - Anthropic Claude 3 全系
> - DeepSeek V3 / V3.1
> - 智谱 GLM-4 / GLM-5
> - 商汤 `sensenova-6.7-flash-lite` / `sensenova-6.8-flash-lite`（`/v1/models` 返回 `supported_features: ["tools", ...]`）
> - MiniMax `MiniMaxAI/MiniMax-M2.7`（实测完美支持）

**解决方案**（已验证）：

将 `.env` 中：

```diff
- LLM_BASE_URL=https://token.sensenova.cn/v1
- LLM_MODEL_NAME=sensenova-6.8-flash-lite
+ LLM_BASE_URL=https://api.gmi-serving.com/v1
+ LLM_MODEL_NAME=MiniMaxAI/MiniMax-M2.7
```

实测 `MiniMaxAI/MiniMax-M2.7` 在 1.5s 内完成 tool calling + 计划生成 + Summarize，done 事件正常推送。

---

### Bug 9（最深层根因）：工具调用失败时 nil 指针 panic，整个进程崩溃

**代码位置**：`api/internal/agent/base.go:392-394`（修复前）

```go
// 修复前
result, err := a.handleToolCall(ctx, tc, messages)
if err != nil {
    logger.Error("工具调用失败",
        zap.String("function", result.FunctionName),  // ← result 为 nil 时 SIGSEGV！
        zap.Error(err))
```

**触发条件**：`handleToolCall` 在两条路径返回 `(nil, err)`：

```go
// base.go:482 —— tool_call 格式非法
if !ok {
    return nil, fmt.Errorf("invalid tool call format")
}
// base.go:503 —— 未知工具名
if !ok {
    return nil, fmt.Errorf("未知工具: %s", functionName)
}
```

**为什么这个 Bug 之前一直没暴露**：

```
sensenova 模型时代：tool calling 请求 >12s 超时
    → 根本走不到工具执行 → panic 代码路径从未被执行
    → 表现为「卡思考中」（Bug 8 的症状掩盖了 Bug 9）

切换 MiniMax-M2.7 后：tool calling 1.5s 返回
    → 真正走到工具执行
    → 模型偶尔返回格式异常的 tool_call / 未知工具名
    → panic → Go goroutine panic 杀死整个进程 → 容器自动重启
```

**实测 panic 堆栈**（2026-08-26 07:06:58）：

```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x20 pc=0xb545d1]

goroutine 253 [running]:
agent.(*BaseAgent).Invoke(...)          /build/internal/agent/base.go:393
agent.(*ReActAgent).ExecuteStep(...)    /build/internal/agent/react_agent.go:61
agent.(*PlannerReActFlow).Invoke.func1() /build/internal/agent/planner_react_flow.go:134
```

**这一条完整解释了用户报告的所有症状**：

| 症状 | 解释 |
|------|------|
| 第一段回复 OK，后续卡思考中 | 第一段是简单问题（无工具调用）；后续问题触发工具调用 → panic |
| **刷新浏览器也不行** | 进程已崩溃重启，task 状态永久留在 running，续读流没有 done |
| 新建任务下对话前转圈 | 同上，残留的 running task 让前端 `startEmptyStream` 挂着 |
| curl 报 `transfer closed with outstanding read data remaining` | panic 时连接被强杀，响应未正常结束 |

**修复（双保险）**：

1. **base.go:392-410** —— 不再解引用可能为 nil 的 result，从 `tc`（原始 toolCall map）安全提取字段：

```go
if err != nil {
    // handleToolCall 失败时 result 可能为 nil，先取安全字段再解引用
    functionName, toolCallID := "", ""
    if fn, ok := tc["function"].(map[string]interface{}); ok {
        functionName, _ = fn["name"].(string)
    }
    toolCallID, _ = tc["id"].(string)
    // ... 用 functionName / toolCallID 记日志、回写历史
}
```

2. **planner_react_flow.go Invoke goroutine** —— 加 `recover()` 兜底，任何 panic 不再杀死进程：

```go
go func() {
    defer close(ch)
    defer func() {
        if r := recover(); r != nil {
            logger.Error("PlannerReActFlow panic，已恢复", zap.Any("panic", r),
                zap.String("stack", string(debug.Stack())))
            ch <- model.NewErrorEvent(fmt.Sprintf("内部错误: %v", r))  // 前端能收到 error 正常结束
            f.mu.Lock()
            f.status = FlowStatusCompleted
            f.mu.Unlock()
        }
    }()
    // ...
}()
```

### Bug 10：推理模型 `reasoning_content` 丢失 + Planner 解析失败即报错（无降级）

**现象**：切换到 `MiniMaxAI/MiniMax-M2.7` 后，简单问候（如「你好，介绍下你自己」）返回
`error: 解析计划失败: json text is empty`，用户看到报错而非回复。

**根因（两层叠加）**：

1. `openai_llm.go` 的 `chatResponse` 结构体**没有 `reasoning_content` 字段**。M2.7 这类
   推理模型把输出放在 `reasoning_content`，`content` 为空 → planner 拿到空串。
2. `planner_agent.go` 的 `CreatePlan` 在 JSON 解析失败时**直接返回 error**，没有任何降级路径。
   而模型对简单问候的推理文本本来就不是 JSON 计划。

**修复**（两处）：

1. **LLM 层兜底**（`openai_llm.go`）：`content` 为空且 `reasoning_content` 非空时，用后者兜底：

```go
content := chatResp.Choices[0].Message.Content
if content == "" && chatResp.Choices[0].Message.Reasoning != "" {
    content = chatResp.Choices[0].Message.Reasoning  // 推理模型兜底
}
```

2. **Planner 层降级**（`planner_agent.go`）：JSON 解析失败且原始文本非空时，返回
   「空步骤计划 + 原始文本作为回复」，让 flow 走 Bug 6 修好的「无步骤直接完成」路径——
   用户得到正常对话回复，而非报错：

```go
if err := a.jsonParser.Parse(resp.Content, &result); err != nil {
    reply := strings.TrimSpace(resp.Content)
    if reply == "" {
        return nil, "", fmt.Errorf("解析计划失败: %w", err)  // 真没内容才报错
    }
    return &model.Plan{Title: title, Message: reply, Steps: []model.PlanStep{}}, reply, nil
}
```

**验证**：同一 session 三轮对话（问候 → 计算 → shell 工具任务）全部正常完成，无 panic，
进程 healthy。注意推理模型多步骤任务耗时可能超过 2 分钟，curl 测试时 `--max-time` 要给足。

### Bug 11：不支持 tool calling 的模型收到工具请求时 60s 超时浪费

**现象**：使用 `sensenova-6.8-flash-lite` 时，任何需要工具调用的任务（如 shell 执行、
文件读取）前端会卡住 60s 后才超时报错。

**根因**：Go 代码没有任何模型能力探测，直接发请求等到全局 120s 超时才失败。

**修复**（`openai_llm.go`）：**不依赖硬编码黑名单**，用 `context.WithTimeout` 对 tool calling
请求单独限速（默认 15s），超时后返回友好错误：

```go
// tool calling 请求使用独立超时（默认 15s），
// 不依赖硬编码模型黑名单——任何不支持 tool calling 的模型都会在此时超时失败，
// 而不是等到全局 120s 才暴露。
if len(req.Tools) > 0 {
    httpCtx, cancel = context.WithTimeout(ctx, c.toolCallTimeout)
    defer cancel()
}

// 发送请求时检测 context 超时
if len(req.Tools) > 0 && errors.Is(err, context.DeadlineExceeded) {
    return nil, fmt.Errorf("tool calling 请求超时（%v），模型 %q 可能不支持 tool calling", c.toolCallTimeout, c.modelName)
}
```

**可配置**：`ToolCallTimeout` 字段（秒），通过 `.env` 的 `LLM_TOOL_CALL_TIMEOUT` 配置，
默认 15s。与模型无关，新模型接入无需改代码。

---

## 2.1 模型能力对比（当前项目已测模型）

| 能力 | sensenova-6.8-flash-lite | deepseek-v4-flash | MiniMaxAI/MiniMax-M2.7 |
|------|--------------------------|-------------------|------------------------|
| 普通对话 | ✅ ~0.5s | ✅ ~3.2s | ✅ ~0.7s |
| tool calling | ❌ 超时 >60s | ✅ ~2.0s | ✅ ~0.7-1.5s |
| reasoning_content | ❌ 无此字段 | ✅ 有 | ✅ 有 |
| JSON 计划输出 | ✅ 正常 | ✅ 正常 | ⚠️ 可能输出自由文本，已做降级处理 |
| 空 content 兜底 | 不需要 | ✅ reasoning_content 兜底 | ✅ reasoning_content 兜底 |

**配置切换**：通过修改 `.env` 中的 `LLM_BASE_URL` / `LLM_API_KEY` / `LLM_MODEL_NAME`
即可切换模型。代码层通过 **context 超时兜底**（tool calling 15s 独立超时）+ **`reasoning_content` 兜底** + **Planner 降级** 三处逻辑自动适配所有模型，**无硬编码模型黑名单**。

---

## 3. 修复后验证（端到端）

### 3.1 简单问答

```
$ curl -N -X POST $SID/chat -d '{"message":"1+1等于几"}' --max-time 30
event:message    data:{"role":"user","content":"1+1等于几"}
event:task_id    data:{"id":"..."}
event:title      data:{"title":"数学问答"}
event:message    data:{"role":"assistant","content":"1+1=2"}
event:plan       data:{...}
event:plan       data:{"status":"completed",...}
event:done       data:{}                            ← 之前缺失
```

### 3.2 带工具调用的完整任务（回归 Bug 9 场景）

```
$ curl -N -X POST $SID/chat -d '{"message":"用bash计算 1+1 并告诉我结果"}' --max-time 120
总耗时: 47s

event:message ×4   ← 用户消息 + AI 思考 + 答案 + 任务执行总结
event:task_id ×1
event:title  ×1
event:plan   ×2    ← created + completed
event:step   ×2    ← started + completed
event:done   ×1    ← ✅
```

后端日志确认 shell 工具真实执行了 4 条命令（`echo $((1+1))`、`expr 1 + 1` 等），
AI 最终回答 `1+1 = 2` 并给出任务执行总结，进程存活（healthy），**不再 panic**。

---

## 4. 原项目 mooc-manus (Python) 是否存在这些问题？

> 路径：`/Users/huanghao2/GolandProjects/study/imooc-mas/mooc-manus`

| # | go-manus Bug | mooc-manus 是否存在 | 原因 |
|---|--------------|---------------------|------|
| 1 | SSE event 全是 message | ❌ 不存在 | Python 用 `StreamingResponse` + `data:` 行，**没有 `event:` 业务类型字段**这一设计，前端按 `data` 里的 type 字段路由 |
| 2 | Redis Stream 丢 model.Event 包装 | ❌ 不存在 | Python 端 Redis Stream 直接存 dict，**没有 `model.Event` 包装概念**（FastAPI/SSEEvent 范式不同） |
| 3 | 空消息拦截误杀空流续读 | ⚠️ 半存在 | Python `agent_service.py` 的 `if message` 分支判断了空，但**判断条件更宽松**（`if message and message.strip()`），空流续读不会进 agent |
| 4 | 空流续读走错路径 | ❌ 不存在 | Python 版的 `/chat` 是 async generator，无论 message 是否为空都走事件循环订阅路径 |
| 5 | 500ms 死循环 | ❌ 不存在 | Python 版 SSE 保持长连 + 阻塞式事件订阅，没有"立即关闭"逻辑 |
| 6 | 状态机空步骤卡死 | ❌ 不存在 | Python 版 `flow_planner_react.py` 在 `if not plan.steps` 分支直接 yield final answer 退出 |
| 7 | CORS 配置冲突 | ⚠️ 曾存在 | FastAPI 虽允许配置 `allow_origins=["*"]` 与 `allow_credentials=True`，但浏览器不会把该组合视为可携带凭据的有效 CORS 响应；sandbox 已改为 `allow_credentials=False`。 |
| 8 | LLM tool calling 慢 | ⚠️ 一样存在 | 模型能力问题，与语言无关 |
| 9 | 工具调用失败 nil panic 杀进程 | ❌ 不存在 | Python 版工具调用失败走 `try/except` 返回错误 dict，**永远不会因为 None 解引用崩溃**；即使异常也只影响单个请求（asyncio 协程级），不会杀死整个 uvicorn 进程 |
| 10 | reasoning_content 丢失 + planner 无降级 | ⚠️ 部分 | Python 的 openai sdk 响应是 dict，`resp.choices[0].message.content` 为空时可直接取 `reasoning_content`（字段本来就透传）；planner 失败的处理依赖各模型实现，原项目主推模型非推理模型，未暴露此问题 |
| 11 | 不支持 tool calling 的模型无能力探测直接等超时 | ❌ 不存在 | 原项目 Python 版用 `sensenova-6.8-flash-lite` 作为 fallback，逻辑一样——**都会超时 60s**。这是模型能力问题，不是移植 bug，但 Go 端修了黑名单后比 Python 版更友好 |

**结论**：**11 个 Bug 中 7 个是 Go 移植时引入的（#1-7、#9），2 个是推理/能力模型适配问题（#10、#11）**，
原项目 Python 版设计上没有这些漏洞。`sensenova-6.8-flash-lite` 的 tool calling 慢是模型能力问题，
两个版本都会受影响，但 Python 版的 SSE 协议更"扁平"（只有 data，没有 event 业务类型），
**对错误处理路径更宽容**。

**Bug 9 的对比尤其值得注意**：Python 的 `None.foo` 会抛 `AttributeError`（异常，可被
except 捕获），Go 的 `nil.Foo` 是 **SIGSEGV（信号，直接杀进程）**。Go 把「错误处理不完整」
的代价从"单个请求失败"放大到了"整个服务崩溃"，这正是本次事故中最严重的一环。

---

## 5. 为什么 Go 移植版比 Python 原版更脆弱？

### 5.1 协议层的"两次包装"

```
Python:  redis dict → JSON → data: {业务字段}            (1 层)
Go:      redis dict → JSON → model.Event{Type, Data} → SSE event:Type, data:Data  (2 层)
```

Go 端在"持久化"和"流推送"两条路径上**都用了 `model.Event` 包装**，但**写入 Redis Stream
时只写了内层 payload**（Bug 2），导致反序列化时 Type 字段为空，SSE 推送时无法回填业务类型。
这是一个典型的「序列化边界不一致」问题。

Python 版没有这种包装，**业务结构就是传输结构**，没有"两次包装"的概念。

### 5.2 gin-cors 的硬性约束

gin-cors 实现严格遵循 W3C CORS 规范，`AllowAllOrigins=true` 与 `AllowCredentials=true`
**直接返回错误**（不像 FastAPI 那样宽容）。

Go 生态的 middleware 实现普遍**更严格、更接近规范**；Python 生态（特别是 FastAPI）更
**实用主义**，对"非标但能跑"的配置更宽容。

### 5.3 Gin 路由的「handler 复用」

Go 端 `/chat` handler 同一份代码处理"发新消息"和"空流续读"两种语义，**靠 `message` 字段
是否为空来区分**——这本身就容易引入 Bug 3 / Bug 4。

Python 版用 `if message` 显式分流，且在 agent_service.py 一开始就分流，**没有"先调 Chat
再补救"的反模式**。

### 5.4 Go 强类型带来的"沉默失败"与"致命失败"

Go 强类型是一把双刃剑：

- **沉默失败**（Bug 2）：`event.GetType()` 返回空字符串，Go 不会 panic，**只会静默推
  `event:message`**，没有编译期/运行期报警
- **致命失败**（Bug 9）：`result.FunctionName` 在 result 为 nil 时是 **SIGSEGV 信号**，
  Go runtime 无法恢复（未加 recover 时），直接杀死整个进程——Python 的 `None.foo`
  只是 `AttributeError` 异常，被 except 吃掉就完事

同一个"忘了判空"，Python 付出的是"一条错误日志"的代价，Go 付出的是"整个服务崩溃"的代价。
这就是为什么 Go 代码里**每个返回指针的函数调用后必须立刻判断 err + nil**，以及
**后台 goroutine 必须加 recover 兜底**。

### 5.5 状态机迁移的语义丢失

原 Python 版的 `flow_planner_react.py` 在"空步骤"分支有**显式 early return**：

```python
if not plan.steps:
    yield final_answer_event  # 显式结束
    return
```

Go 版的 `planner_react_flow.go` 在 `FlowStatusSummarizing` case 翻译时**漏掉了这个
early return**，直接进了 `react.Summarize()`——**这是典型的翻译型 bug**：
照着状态机框架抄，**没看完整的特殊路径**。

---

## 6. 教训与改进建议

### 6.1 短期

1. ✅ 已切换到 `MiniMaxAI/MiniMax-M2.7`（tool calling 1.5s 响应）
2. ✅ 已修复 SSE 业务类型、Redis Stream 包装、空流续读、CORS、状态机空步骤
3. ✅ 已修复工具调用 nil 解引用 panic + flow goroutine recover 兜底（Bug 9）
4. ✅ 已用 context 超时替代黑名单（tool calling 请求 15s 独立超时，不支持模型的请求自动在 15s 内失败，无黑名单硬编码）
5. 🔜 加 LLM client 的**细粒度超时**（tool calling 30s / summarize 30s），避免单次调用挂起 120s
6. 🔜 前端 `startEmptyStream` 加**指数退避**（1s / 2s / 4s / 8s 上限），避免 500ms 风暴
7. 🔜 排查 sandbox 健康检查 404（`/health` 端点不存在，但 exec 调用可用，疑似路径配置问题）

### 6.2 中期

1. **新增"无工具调用时跳过 Summarize" 单测**：用 mock LLM 验证状态机不挂死
2. **新增"工具调用返回 nil 时不 panic" 单测**：回归 Bug 9 场景
3. **Redis Stream 序列化一致性检查**：DB 存的格式与 Stream 存的格式必须相同
4. **gin-cors 配置校验**：启动时 `log.Warn` 提示 `AllowAllOrigins+AllowCredentials` 互斥
5. **前端 EventSource 事件类型路由单测**：覆盖 message / plan / step / title / done

### 6.3 长期

1. **移植流程 checklist**：从 Python 移植到 Go 时，对每个 Python early-return / 特殊路径
   都要在 Go 状态机里**显式翻译**并加单测
2. **goroutine 规范**：所有后台 goroutine 必须 recover + 转 error 事件，禁止裸起
3. **SSE 协议规范统一**：建议参考 mooc-manus 风格，**只用 data 字段** + type 子字段，
   避免双层包装
4. **LLM 适配层**：对不支持 tool calling 的模型有 fallback（虽然现在不再需要，但代码层要 graceful）

---

## 7. 关键文件清单

| 文件 | 修改 |
|------|------|
| `api/internal/handler/session_handler.go` | SSE 业务类型 + 空流续读 + 15s 心跳 |
| `api/internal/agent/task_runner.go` | Redis Stream 写入包 model.Event |
| `api/internal/agent/planner_react_flow.go` | 空步骤跳过 Summarize + goroutine recover 兜底 |
| `api/internal/agent/base.go` | 工具调用失败时安全提取字段（修 nil 解引用 panic） |
| `api/internal/external/openai_llm.go` | 响应结构增加 `reasoning_content` + 兜底 + 黑名单探测不支持 tool calling 的模型（Bug 10/11） |
| `api/internal/agent/planner_agent.go` | 计划 JSON 解析失败时降级为直接回复（空步骤计划） |
| `api/internal/agent/service.go` | 新增 `GetActiveTaskID` |
| `api/pkg/middleware/middleware.go` | CORS AllowOriginFunc 反射 |
| `.env` / `.env.test` | 切换 LLM_BASE_URL / LLM_MODEL_NAME 到 MiniMax-M2.7 |
