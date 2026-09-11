# go-manus 实战经验与面试考点沉淀（2026-09-04）

> 汇总近几轮对话中排查的真实问题，提炼出"资深 Go 后端工程师"面试常考点。
> 每个问题都按 **What → How → Why → Alternatives → Trade-offs** 五层框架展开。

---

## 目录

1. [JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱](#1-json-bytes-vs-jsonrawmessagepostgresql-jsonb-base64-编码陷阱)
2. [Go 死锁与 channel 关闭：一次性 ping-pong 协程模式](#2-go-死锁与-channel-关闭一次性-ping-pong-协程模式)
3. [Docker Desktop 替换：macOS 容器运行时选型](#3-docker-desktop-替换macos-容器运行时选型)
4. [SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作](#4-sse-流式响应从一直转圈看-sse-协议与前端协作)
5. [JSON 库替换踩坑：sonic 与 encoding/json 行为差异](#5-json-库替换踩坑sonic-与-encodingjson-行为差异)
6. [代码评审与回归测试：测试不是覆盖，而是契约](#6-代码评审与回归测试测试不是覆盖而是契约)

---

## 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱

### 1.1 现象

前端调用 `/api/sessions/{id}` 返回的 30 个 events 全部存在，但 UI 对话区为空。`/api/sessions/stream` 的 SSE 流也有数据，但页面不渲染。

### 1.2 What

`Event.Data` 字段类型为 `[]byte`，但其中存的是 JSON 字符串（如 `{"role":"user","message":"..."}`）。

### 1.3 How

`sonic.Marshal` 序列化时，发现 `Data` 是 `[]byte`（Go 二进制类型），而 JSON 规范没有 `[]byte` 类型，**只能 base64 编码**。结果：

```json
{
  "id": "...",
  "type": "message",
  "data": "eyJyb2xlIjoidXNlciIsIm1lc3NhZ2UiOiJHaXRIdQ=="
}
```

存入 PostgreSQL JSONB 列后，前端拿到的是这个对象，`events[i].data` 是个 base64 字符串，前端 `JSON.parse` 失败。

### 1.4 Why（根因）

| 字段 | 类型 | JSON 行为 |
|------|------|-----------|
| `string` | 文本 | `"value"` |
| `[]byte` | 二进制 | `"base64=="` |
| `json.RawMessage` | `[]byte` 别名 + `MarshalJSON` 实现 | 原样嵌入 |

`json.RawMessage` 的 `MarshalJSON` 方法：

```go
func (m RawMessage) MarshalJSON() ([]byte, error) {
    if m == nil { return []byte("null"), nil }
    return m, nil   // 直接返回，不编码
}
```

### 1.5 Alternatives

| 方案 | 优 | 劣 |
|------|----|----|
| `[]byte` | 简单 | ❌ base64 编码 |
| `string` | 简单 | ❌ 需要手动转义 |
| `json.RawMessage` | ✅ 延迟解析、类型安全、原样嵌入 | 需要 `encoding/json` |
| `map[string]interface{}` | 无依赖 | ❌ 失去类型、解析时性能差 |
| `interface{}` | 灵活 | ❌ 太宽泛 |

### 1.6 Trade-offs

| 维度 | 选 `json.RawMessage` |
|------|---------------------|
| 性能 | ✅ 延迟解析，反序列化只做一次 |
| 兼容性 | ✅ 前端不用 base64 解码 |
| 内存 | 字符串形式存储在内存中 |
| 调试 | `data` 字段在日志中可读 |

### 1.7 面试要点

> **Q：Go 中如何存储"未知结构"的 JSON 字段？**
>
> A：用 `json.RawMessage`。它延迟解析 + 直接嵌入，是 Go 生态处理半结构化 JSON 的标准做法。任何 JSON 库（encoding/json、jsoniter、sonic）都兼容。

### 1.8 适用场景

| 场景 | 选型 |
|------|------|
| 已知结构 | 强类型 struct |
| 完全未知 | `json.RawMessage` |
| 动态 key | `map[string]json.RawMessage` |
| 完全透传（如 HTTP body） | `io.Reader` |

### 1.9 类似风险点（已全量排查）

| 字段 | 修复 |
|------|------|
| `Event.Data []byte` | ✅ 改 `json.RawMessage` |
| `RequestPolicy.Extra map[string][]byte` | ✅ 改 `map[string]json.RawMessage` |
| `llmcore.protocol.Raw` | ✅ 仅内存不入库，无影响 |
| `llmcore.profile.Raw` | ✅ 仅内存不入库，无影响 |

---

## 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式

### 2.1 现象

`/chat` 接口偶发卡死，CPU 飙高，日志停止输出。

### 2.2 What

`streamSSE` 中两个 goroutine 通过 channel 通信，但 channel 关闭时序错误导致 `panic: send on closed channel`。

### 2.3 How

```go
// 错误模式
ch := make(chan Event, 100)
go func() {
    defer close(ch)
    for ev := range producer() {
        ch <- ev        // 写
    }
}()

for ev := range ch {     // 读
    if ev.Type == "done" { break }
}
```

问题：
- `producer` 写完后 `defer close(ch)`，**但读端 break 后，读协程退出，写协程可能还在写**，下次写就 panic
- 写端如果中途 panic，**读端会一直阻塞**

### 2.4 Why（设计原则）

| 原则 | 含义 |
|------|------|
| **谁创建谁关闭** | channel 由发送方创建，由发送方关闭 |
| **多发送方不关闭** | 多发送方共享 channel 时，只读不关 |
| **关闭用 sync.Once** | 防止重复关闭 panic |
| **done channel 模式** | 通过 `done chan struct{}` 通知退出，**不关闭数据 channel** |

### 2.5 Alternatives

#### 模式 A：defer close（适用于单发送方）

```go
func writeEvents(ch chan<- Event, events []Event) {
    defer close(ch)        // 单发送方，安全
    for _, ev := range events {
        ch <- ev
    }
}

func readEvents(ch <-chan Event) {
    for ev := range ch {   // 自动检测关闭
        process(ev)
    }
}
```

#### 模式 B：done channel（适用于多消费者）

```go
done := make(chan struct{})
go func() {
    defer close(done)      // 关闭 done，而非数据 channel
    for {
        select {
        case ch <- ev:     // 写
        case <-done:       // 退出
            return
        }
    }
}()

// 读端
select {
case ev := <-ch:
    process(ev)
case <-done:
    return
}
```

#### 模式 C：context 取消

```go
ctx, cancel := context.WithCancel(parent)
defer cancel()

go func() {
    for {
        select {
        case ch <- ev:
        case <-ctx.Done():  // 上游取消
            return
        }
    }
}()
```

### 2.6 Trade-offs

| 模式 | 优点 | 缺点 |
|------|------|------|
| defer close | 简单 | 仅限单发送方 |
| done channel | 通用 | 多一个 channel |
| context | 支持超时/取消链 | 略复杂 |

### 2.7 面试要点

> **Q：Go channel 关闭原则是什么？**
>
> A：**谁创建谁关闭**。多发送方共享 channel 不关闭，用 done channel 或 context 协调退出。
> 关闭已关闭的 channel 会 panic，向已关闭的 channel 发送也会 panic。

> **Q：goroutine 泄漏怎么排查？**
>
> A：`pprof` goroutine profile。常见场景：channel 无消费者、context 未取消、select 永远命中 default。

### 2.8 SSE 实现最佳实践

```go
func streamSSE(w http.ResponseWriter, events []Event) error {
    flusher, ok := w.(http.Flusher)
    if !ok { return errors.New("streaming unsupported") }
    
    // 1. 设置 SSE 头
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.WriteHeader(200)
    
    // 2. 用 done channel 模式
    ctx := r.Context()
    for _, ev := range events {
        select {
        case <-ctx.Done():
            return ctx.Err()    // 客户端断开，立即返回
        default:
        }
        
        if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, ev.Data); err != nil {
            return err
        }
        flusher.Flush()
    }
    return nil
}
```

---

## 3. Docker Desktop 替换：macOS 容器运行时选型

### 3.1 现象

公司内部禁用 Docker Desktop，寻求开源替代品。

### 3.2 What

Docker Desktop 是 Docker 公司专有软件（部分闭源），内部政策要求替换。开源替代方案对比。

### 3.3 主流方案对比

| 工具 | 授权 | 内存 | GUI | K8s | 适配性 |
|------|------|------|-----|-----|--------|
| **Colima** | MIT | ~350MB | ❌ | 可选 | ✅ 平替 |
| **Rancher Desktop** | Apache 2.0 | ~2GB | ✅ | 内置 k3s | ✅ 适合 K8s |
| **Podman Desktop** | Apache 2.0 | 中 | ✅ | 需配置 | ⚠️ Compose 部分不兼容 |
| **OrbStack** | 闭源付费 | ~250MB | ✅ | 可选 | ❌ 非开源 |

### 3.4 推荐方案：Colima

```bash
brew install colima docker docker-compose
colima start --cpu 4 --memory 8 --disk 100
docker ps
```

### 3.5 迁移关键点

| 关键点 | 解决方案 |
|--------|----------|
| **Docker Socket 路径不同** | `docker context use colima` |
| **凭证配置残留** | `rm ~/.docker/config.json` |
| **Compose 完全兼容** | ✅ 现有 `docker-compose.yml` 无需改 |
| **Docker-in-Docker** | ⚠️ `/var/run/docker.sock` 需挂 Colima socket |

### 3.6 资源分配建议（16GB MacBook）

| 服务 | CPU | 内存 |
|------|-----|------|
| Colima VM | 3-4 核 | 6-8 GB |
| IDE (GoLand) | 2 核 | 4 GB |
| 系统 | 1 核 | 2 GB |
| 浏览器 | - | 2 GB |

### 3.7 面试要点

> **Q：Docker Desktop 和 Colima 的本质区别是什么？**
>
> A：Docker Desktop 是 hypervisor + GUI + Docker CLI 一体化产品；
> Colima 是基于 Lima VM 的纯 CLI 工具，复用系统 Docker CLI。两者最终都跑在 LinuxKit VM 上。

> **Q：macOS 上为什么不能直接跑 Docker daemon？**
>
> A：Docker daemon 需要 Linux 内核特性（cgroups、namespaces、overlayfs），
> macOS 是 BSD 内核，必须通过 VM 间接运行（Docker Desktop/Colima/Podman Machine）。

---

## 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### 4.1 现象

第一段 AI 回复正常，后续每轮卡"正在思考中"，刷新能看到内容。

### 4.2 What

SSE（Server-Sent Events）流式响应在多轮对话中异常。

### 4.3 11 个 Bug 拆解

#### Bug 1：SSE `event:` 字段被钉死为 `message`

```go
// 错误
fmt.Fprintf(w, "data: %s\n\n", data)

// 正确
fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
```

**Why**：SSE 协议用 `event:` 字段标识消息类型，前端 `addEventListener('plan', ...)` 监听。

#### Bug 2：plan/step/title 事件被当作 message 渲染

前端 `if (data.role === 'assistant')` 判断逻辑覆盖了所有非 message 事件。

#### Bug 3：done 事件未发出

流式响应需要"显式结束信号"，前端靠 `done` 事件切回 idle 状态。

#### Bug 4：连接复用时残留 state

`EventSource` 默认会自动重连，但不会重置前端状态。

#### Bug 5：心跳机制缺失

Nginx 默认 60s 空闲超时，AI 长任务可能 90s 无输出，前端以为断开。

#### Bug 6：buffered data 未 flush

```go
w.Write(data)
flusher.Flush()  // 必须显式 flush
```

#### Bug 7：跨域 CORS 配置

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Credentials", "true")
```

#### Bug 8：Nginx 反代 buffering

```nginx
proxy_buffering off;
proxy_cache off;
```

#### Bug 9：前端 strict-origin 报错

EventSource 不支持自定义 header，无法用 Bearer Token；只能靠 cookie。

#### Bug 10：client-side 未处理 abort

`EventSource.close()` 必须在组件 unmount 时调用。

#### Bug 11：React 18 StrictMode 双调用

useEffect 在 dev 模式下双执行，stream 创建两次但只关一个。

### 4.4 SSE 协议规范

```
event: <event-type>
data: <event-data>
id: <event-id>
retry: <reconnect-time-ms>

(blank line terminates event)
```

### 4.5 SSE vs WebSocket vs 长轮询

| 特性 | SSE | WebSocket | 长轮询 |
|------|-----|-----------|--------|
| 协议 | HTTP | WS | HTTP |
| 方向 | 服务器→客户端 | 双向 | 拉模式 |
| 重连 | 自动 | 手动 | 手动 |
| 复杂度 | 低 | 中 | 高 |
| 适用 | AI 流式输出 | 实时聊天 | 通知 |

### 4.6 面试要点

> **Q：SSE 和 WebSocket 如何选型？**
>
> A：单向流式（AI 输出、日志、通知）选 SSE，简单、自动重连、HTTP 友好；
> 双向实时（IM、协作）选 WebSocket。SSE 不能发客户端消息。

> **Q：Nginx 反代 SSE 要注意什么？**
>
> A：
> 1. `proxy_buffering off;`（禁止缓冲）
> 2. `proxy_read_timeout 300s;`（拉长超时）
> 3. `proxy_set_header Connection '';`（禁用 keep-alive）
> 4. `add_header X-Accel-Buffering no;`（应用层标记）

---

## 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### 5.1 现象

`c1f5bfb perf: 替换 encoding/json → sonic` 时，把 `json.RawMessage` 改成 `[]byte`，引入 base64 bug。

### 5.2 What

sonic 兼容 encoding/json 的大部分行为，但**默认 unsafe** 和**类型推断**有差异。

### 5.3 sonic vs encoding/json 差异

| 维度 | encoding/json | sonic |
|------|---------------|-------|
| 性能 | 标准 | 2-5x 快（unsafe） |
| `[]byte` 处理 | base64 | base64 |
| `json.RawMessage` | ✅ 嵌入 | ✅ 嵌入 |
| `MarshalJSON` 接口 | ✅ 走 | ✅ 走 |
| `UnmarshalJSON` 接口 | ✅ 走 | ✅ 走（需注册） |
| 默认 unsafe | ❌ | ⚠️ `sonic.Config{}.Marshal()` |
| nil slice | `null` | `null` |
| 数字精度 | 完整 | 完整（默认 float64） |

### 5.4 sonic 正确使用姿势

```go
// 1. 默认配置
import "github.com/bytedance/sonic"

sonic.Marshal(v)
sonic.Unmarshal(data, &v)

// 2. 显式配置（生产环境推荐）
import "github.com/bytedance/sonic/encoder"

m := encoder.StdEncoder{}
m.Encode(v)  // 不走 unsafe

// 3. JSON 兼容性差异测试
// 标准库和 sonic 在大数字 / unicode 上有差异，需要回归测试
```

### 5.5 替换 JSON 库的检查清单

- [ ] 类型层面：`[]byte` 是否会被 base64 编码
- [ ] 接口层面：自定义 `MarshalJSON`/`UnmarshalJSON` 是否被识别
- [ ] 行为层面：nil/空值/数字精度
- [ ] 性能层面：benchmark 对比
- [ ] 兼容性：JSON 输出能否被其他语言正确解析
- [ ] 回归测试：替换前后输出 byte-level 对比

### 5.6 面试要点

> **Q：Go 高性能 JSON 库选型？**
>
> A：
> - **sonic**（字节）：JIT + unsafe，最快，但依赖 cgo/汇编
> - **jsoniter**（赵子义）：API 兼容标准库，无 cgo
> - **easyjson**：代码生成，无 runtime 反射
> - **goccy/go-json**：纯 Go，比标准库快 2-3x
>
> 选型标准：性能优先选 sonic，跨平台选 jsoniter，长期维护选 easyjson。

---

## 6. 代码评审与回归测试：测试不是覆盖，而是契约

### 6.1 Why：测试是行为的契约

| 错误认知 | 正确认知 |
|----------|----------|
| 测试是为了覆盖率 | 测试是定义"什么算正确" |
| 测试越多越好 | 测试越关键越好（核心路径） |
| 改完代码再说 | 改之前先写测试（红→绿） |
| 测试不写也行 | 回归测试防历史 bug |

### 6.2 历史 Bug 的回归测试

为 `Event.Data` base64 bug 写专门的回归测试：

```go
// TestEvent_Data_NotBase64 防御性回归测试
func TestEvent_Data_NotBase64(t *testing.T) {
    original := `{"role":"user","message":"hello"}`
    event := Event{
        Type: EventTypeMessage,
        Data: json.RawMessage(original),
    }
    out, err := sonic.Marshal(event)
    if err != nil { t.Fatalf("marshal: %v", err) }
    
    if contains(string(out), `"data":"eyJ`) {
        t.Errorf("Data was base64-encoded, JSON output:\n%s", out)
    }
}
```

### 6.3 测试分类

| 类型 | 目的 | 例子 |
|------|------|------|
| 单元测试 | 函数行为 | `TestEvent_Data_NotBase64` |
| 集成测试 | 模块协作 | API + DB |
| 端到端测试 | 用户场景 | Playwright/Cypress |
| 契约测试 | API 接口 | Pact |
| 性能测试 | benchmark | `go test -bench` |
| 混沌测试 | 异常场景 | kill -9、断网 |

### 6.4 Go 测试模式

#### 表格驱动测试

```go
func TestEvent(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"user", `{"role":"user"}`, "object"},
        {"empty", `{}`, "object"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

#### 子测试 + t.Helper

```go
func assertEqual(t *testing.T, got, want string) {
    t.Helper()    // 错误时报告调用方位置
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

#### 模糊测试（Go 1.18+）

```go
func FuzzEvent(f *testing.F) {
    f.Add(`{"role":"user"}`)
    f.Fuzz(func(t *testing.T, data string) {
        var e Event
        if err := sonic.Unmarshal([]byte(data), &e); err == nil {
            // round-trip
            out, _ := sonic.Marshal(e)
            var back Event
            sonic.Unmarshal(out, &back)
            if !reflect.DeepEqual(e, back) {
                t.Errorf("round-trip failed")
            }
        }
    })
}
```

### 6.5 面试要点

> **Q：Go 测试为什么推荐表格驱动？**
>
> A：1) 重复结构集中管理；2) 添加用例只需新增行；3) `t.Run(name)` 单独报告每个 case；
> 4) 易读，符合 Go 哲学"clear is better than clever"。

> **Q：怎么为遗留代码补测试？**
>
> A：1) 找到核心入口；2) 写"行为快照测试"（不判断对错，只锁定现状）；3) 逐步重构 + 修复预期；
> 4) 对每个修复写回归测试。

---

## 7. 综合面试题

### 7.1 资深 Go 后端

1. **Go channel 关闭原则是什么？defer close 在多发送方场景下有什么问题？**
2. **如何用 `json.RawMessage` 处理"未知结构"的 JSON 字段？它和 `map[string]interface{}` 的取舍？**
3. **SSE 和 WebSocket 的适用场景？Nginx 反代 SSE 的关键配置？**
4. **Go 中如何排查 goroutine 泄漏？channel 死锁？**

### 7.2 系统设计

5. **如何设计一个支持多 LLM 厂商的 Agent 系统？协议层、Adapter 层、控制面怎么分层？**
6. **JSONB 在 PostgreSQL 中的存储原理？和 MySQL JSON 类型有什么差异？**

### 7.3 工程实践

7. **JSON 库替换时如何做兼容性验证？需要关注哪些维度？**
8. **macOS 上为什么必须用 VM 跑容器？Colima 和 Docker Desktop 的本质区别？**
9. **代码评审看什么？哪些是"硬伤"？哪些是"风格"？**

---

## 8. 参考资料

- [PostgreSQL JSONB 文档](https://www.postgresql.org/docs/current/datatype-json.html)
- [Go encoding/json 源码](https://cs.opensource.google/go/go/+/refs/tags/go1.22.0:src/encoding/json/)
- [SSE 规范 (HTML Living Standard)](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [sonic GitHub](https://github.com/bytedance/sonic)
- [Colima GitHub](https://github.com/abiosoft/colima)

---

> 文档生成时间：2026-09-04
> 适用项目：go-manus（imooc-mas 慕课网多 Agent 实战项目）
> 后续维护：每个新发现的"踩坑点"按五层框架（What/How/Why/Alternatives/Trade-offs）追加章节
