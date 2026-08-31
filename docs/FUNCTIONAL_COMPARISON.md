# mooc-manus vs go-manus 功能差异与缺失分析

> 生成时间: 2026-08-31
> 范围: `/Users/huanghao2/GolandProjects/study/imooc-mas/mooc-manus` (Python, 老项目) vs `/Users/huanghao2/GolandProjects/study/imooc-mas/go-manus` (Go, 新项目)
> 目的: 找出**功能层面的差异与缺失**,而非文件数 / 命名 / 语言差异

---

## 目录

- [一、结论速览](#一结论速览)
- [二、API路由对齐情况](#二api-路由对齐情况)
- [三、Agent 核心能力差异](#三agent-核心能力差异)
- [四、沙箱 / 浏览器自动化 / VNC](#四沙箱--浏览器自动化--vnc)
- [五、前端 / UI 行为对齐](#五前端--ui-行为对齐)
- [六、运维 / 部署 / 中间件](#六运维--部署--中间件)
- [七、缺失功能清单(优先级排序)](#七缺失功能清单优先级排序)
- [八、待优化项(已实现但有差异)](#八待优化项已实现但有差异)
- [九、建议的落地顺序](#九建议的落地顺序)

---

## 一、结论速览

| 维度 | mooc-manus (Python) | go-manus (Go) | 结论 |
|------|---------------------|---------------|------|
| Agent 流核心 (Planner/ReAct/Flow) | ✅ 完整 | ✅ 完整 | 一致 |
| 工具系统 (shell/file/browser/search/mcp/a2a/message) | ✅ 6 个 | ✅ 6 个 | 一致 |
| MCP / A2A 远程服务注册与配置 | ✅ | ✅ | 一致 |
| LLM 客户端 (OpenAI 兼容 + Anthropic) | ✅ | ✅ | 一致 |
| 动态从 `app_configs` 加载 LLM 配置 | ✅ | ✅ (本次修复后) | 一致 |
| SSE 流式推送 / Redis Stream 任务队列 | ✅ | ✅ | 一致 |
| CORS = `*` | ✅ | ✅ (本次修复后) | 一致 |
| `/api` 前缀对齐 (与 nginx 配套) | ✅ | ✅ (本次修复后) | 一致 |
| Planner 阶段强制 `json_object` 输出 | ❌ 弱约束 | ✅ 强制 | Go 更优 |
| Planner.message 作为 assistant 消息发出 | ✅ | ✅ (本次新增) | 一致 |
| **沙箱 VNC WebSocket 代理 (`/sessions/:id/vnc`)** | ✅ **完整** | ✅ **已实现** | ✅ `router.go:42` + `handler/vnc_handler.go` |
| **前端 VNC Overlay / Viewer 组件** | ✅ | ✅ | 一致，API 端已对齐 |
| 空响应注入 "AI 无响应, 请继续" 触发重试 | ✅ | ✅ (`base.go:invokeWithEmptyRetry` + `empty_retry_test.go`) | 一致 |
| LLM 调用退避重试 (max_retries + retry_interval) | ✅ | ⚠️ 有 retry, 但**无指数退避** | 待优化 |
| ReAct 阶段强制 JSON 输出 | ⚠️ 由调用方按 format 参数决定 | ❌ 不强制 (仅 Planner 强制) | 待优化 |
| supervisor 进程组管理 / Redis 任务锁 | ✅ | ✅ (Redis Stream 实现) | 思路一致 |
| Alembic 数据库迁移 | ✅ | ❌ (用 SQL DDL 手动) | 待优化 |
| nginx 反向代理层 | ✅ (部署方案中带) | ❌ (本机部署不需要) | 已知差异 |
| Docker Compose 一键部署 | ✅ | ✅ | 一致 |

> 一句话结论: **核心 Agent 引擎和工具系统两边已经对齐**; **沙箱 VNC 远程控制、WebSocket 转发、空响应注入重试、LLM 退避策略、ReAct JSON 约束** 是当前最明显的差距。

---

## 二、API 路由对齐情况

### 2.1 整体路由映射

| 路由 (前缀 `/api`) | mooc-manus | go-manus (修复前) | go-manus (修复后) |
|-------------------|------------|-------------------|-------------------|
| `GET /status` | ✅ | ✅ | ✅ |
| `POST /sessions` | ✅ | ✅ | ✅ |
| `GET /sessions` | ✅ | ✅ | ✅ |
| `GET /sessions/:id` | ✅ | ✅ | ✅ |
| `POST /sessions/:id/chat` | ✅ | ✅ | ✅ |
| `POST /sessions/:id/stop` | ✅ | ✅ | ✅ |
| `GET /sessions/stream` | ✅ | ✅ | ✅ |
| `POST /sessions/stream` | ✅ | ✅ (前端默认 POST) | ✅ |
| `POST /sessions/:id/delete` | ✅ | ✅ | ✅ |
| `POST /sessions/:id/clear-unread-message-count` | ✅ | ❌ → ✅ | ✅ |
| `GET /sessions/:id/files` | ✅ | ✅ | ✅ |
| `POST /sessions/:id/file` | ✅ | ❌ → ✅ | ✅ |
| `POST /sessions/:id/shell` | ✅ | ❌ → ✅ | ✅ |
| **`WS /sessions/:id/vnc`** | ✅ | ✅ | ✅ `router.go:42` |
| `POST /files` | ✅ | ✅ | ✅ |
| `GET /files/:id` | ✅ | ✅ | ✅ |
| `GET /files/:id/download` | ✅ | ✅ | ✅ |
| `GET /app-config/llm` | ✅ | ✅ | ✅ |
| `POST /app-config/llm` | ✅ | ✅ | ✅ |
| `GET /app-config/agent` | ✅ | ✅ | ✅ |
| `POST /app-config/agent` | ✅ | ✅ | ✅ |
| `GET /app-config/mcp-servers` | ✅ | ❌ → ✅ | ✅ |
| `POST /app-config/mcp-servers` | ✅ | ❌ → ✅ | ✅ |
| `POST /app-config/mcp-servers/:server_name/delete` | ✅ | ❌ → ✅ | ✅ |
| `GET /app-config/a2a-servers` | ✅ | ❌ → ✅ | ✅ |
| `POST /app-config/a2a-servers` | ✅ | ❌ → ✅ | ✅ |

> 历史已修复: `/sessions/:id/file`、 `/sessions/:id/shell`、 `/sessions/:id/clear-unread-message-count`、 `/app-config/mcp-servers`、 `/app-config/a2a-servers`、 `/api` 前缀、 CORS=*、 Planner.message 作为 assistant 消息。

### 2.2 **仍然缺失的关键路由**

| 路由 | 来源 | 缺失影响 |
|------|------|----------|
| `WS /api/sessions/:id/vnc` | `mooc-manus/api/app/interfaces/endpoints/session_routes.py:285` | 前端 `vnc-overlay.tsx` 已实现, 点击"虚拟机浏览器"按钮后 404 / 连接失败 |

---

## 三、Agent 核心能力差异

### 3.1 Planner 阶段: JSON 强制输出 ✅ Go 更优

| 项目 | 行为 |
|------|------|
| mooc-manus | `BaseAgent._invoke_llm` 接收 `format` 参数, Planner 阶段传 `"json"`, 由 LLM 客户端负责 `response_format={"type": "json"}` |
| **go-manus** | `planner_agent.go:59,158` **直接硬编码** `ResponseFormat: {type: "json_object"}`, **无条件强制** |

**对比意义**: Go 版本消除了"Planner 偶尔漏 JSON" 的可能性, 行为更严格。

### 3.2 **空响应重试注入 (重要差异)**

> mooc-manus 实现: [base.py:98-106](file:///Users/huanghao2/GolandProjects/study/imooc-mas/mooc-manus/api/app/domain/services/agents/base.py#L98-L106)

```python
# mooc-manus: 拿到空 content + 空 tool_calls 时, 注入"AI 无响应, 请继续。"
if not message.get("content") and not message.get("tool_calls"):
    await self._add_to_memory([
        {"role": "assistant", "content": ""},
        {"role": "user", "content": "AI无响应内容，请继续。"}
    ])
    await asyncio.sleep(self._retry_interval)
    continue
```

| 项目 | 空响应处理 |
|------|------------|
| **mooc-manus** | ✅ 注入"AI 无响应, 请继续。" 提示, 自动重试 N 次 |
| **go-manus** | ❌ 检测到空响应后, 直接走 retry 循环, **没有注入 user prompt** |
| 影响 | 推理模型 (如 MiniMax-M2.7) 的 `reasoning_content` 已通过 `openai_llm.go:172` 兜底; 但**纯空响应** (连 reasoning 都空) 场景下, Go 版本只能靠 retry 轮空, 容易触发 "只列计划不执行" 的现象 |

### 3.3 LLM 错误重试策略

| 项目 | 重试机制 |
|------|----------|
| mooc-manus | `max_retries` (默认 3) + `retry_interval` 固定1s + 内存 + DB 双重持久化; 每次 `_invoke_llm` / `_invoke_tool` 都包在 for 循环里 |
| **go-manus** | `MaxRetries` 在 `base.go:340,536` 实现; `retryInterval = 1.0` (秒) 固定; **无指数退避**, **无针对 HTTP 429 的 backoff** |
| 实战踩坑 | 当用户 LLM provider `rpm exhausted` 时, 3 次连续失败直接放弃, Planner 阶段返回 `plan`, ReAct 阶段没有 planMsg 进 step, UI 出现 "只列出计划不进行后续操作" |

**建议**: 在 `openai_llm.go` 增加 `if status == 429: exponential backoff`, 与原项目保持一致。

### 3.4 ReAct 阶段的 JSON 约束

| 项目 | 行为 |
|------|------|
| mooc-manus | `format` 参数由调用方传入; ReAct 阶段不强制 JSON, 使用 tool calls |
| go-manus | **ReAct 阶段也不强制 JSON**; tool_calls 是开放式的 |
| 评价 | ✅ 行为一致; **Planner 阶段已强制 json_object 足够保证 plan 结构** |

### 3.5 流式事件分发

| 项目 | SSE 实现 |
|------|----------|
| mooc-manus | `sse_starlette.EventSourceResponse`, 每个 `ServerSentEvent(event=..., data=...)` 中 `event` 字段是协议头 |
| go-manus | `c.Stream(func(w)` 自实现 SSE; 事件类型用 `event:` 协议头, 数据走 `data:` |
| 对齐 | ✅ 一致 |
| 待优化 | mooc-manus 有 `EventMapper` 统一把 `MessageEvent/ErrorEvent/DoneEvent/WaitEvent` 映射成 `event:` 头; go-manus 散落在 flow 内部, 缺少统一映射层 |

### 3.6 Agent 任务执行架构

| 项目 | 架构 |
|------|------|
| mooc-manus | `AgentTaskRunner` 独立类 + Redis Stream 任务队列 + consumer group; 异步生成器 yield 事件 |
| **go-manus** | `task_runner.go` + `task_redis.go` + Redis Stream consumer group; **goroutine + channel** yield 事件 |
| 评价 | ✅ 思路一致; Go 实现更轻量 |
| 待优化 | mooc-manus 的 `Task` 抽象允许"任务暂停 / 恢复 / 取消"; go-manus 当前只在 `Stop` 路由发信号, 缺少任务持久化快照 |

---

## 四、沙箱 / 浏览器自动化 / VNC

### 4.1 沙箱容器差异

| 组件 | mooc-manus sandbox | go-manus sandbox |
|------|--------------------|------------------|
| `Dockerfile` | 安装 `xvfb x11vnc websockify chromium xterm socat` | **完全相同** |
| `supervisord.conf` | `xvfb, chrome, socat, x11vnc, websockify, app` 6 个 program | **完全相同** |
| 沙箱入口 | FastAPI 8080 (uvicorn hot reload) | **完全相同** (甚至连 `UVI_ARGS` 都保留) |
| 文件/Shell 端点 | `/files`, `/shell` + supervisor 控制 | **完全相同** |
| 暴露端口 | `8080 9222 5900 5901` | **完全相同** |

**评价**: 沙箱侧两边**字节级一致**, Go 版本甚至保留了 `UVI_ARGS` 这种 Python 专用变量, 纯粹是 Python sandbox 的副本。

### 4.2 **VNC WebSocket 代理 (重大功能缺失)**

> mooc-manus 实现: [session_routes.py:285-360](file:///Users/huanghao2/GolandProjects/study/imooc-mas/mooc-manus/api/app/interfaces/endpoints/session_routes.py#L285-L360) + [session_service.py:129](file:///Users/huanghao2/GolandProjects/study/imooc-mas/mooc-manus/api/app/application/services/session_service.py#L129)

#### 完整流程 (mooc-manus)

```
前端 noVNC
  ↓ ws://api/sessions/{id}/vnc (sec-websocket-protocol: binary)
  ↓
FastAPI @router.websocket("/{session_id}/vnc")
  ↓ 1) accept(subprotocol="binary")
  ↓ 2) session_service.get_vnc_url(session_id)  → ws://sandbox:5901
  ↓ 3) websockets.connect(sandbox_vnc_url)
  ↓ 4) 启动两个 asyncio task:
  ↓ forward_to_sandbox: ws.receive_bytes  →  sandbox_ws.send
  ↓       forward_from_sandbox: sandbox_ws.recv  →  ws.send_bytes
  ↓ 5) asyncio.wait FIRST_COMPLETED
  ↓
sandbox:websockify :5901 ←→ x11vnc :5900 ←→ Xvfb :1 ←→ chromium
```

#### go-manus 当前状态 ✅ 已实现

- ✅ UI 层 `ui/src/components/vnc-overlay.tsx` + `vnc-viewer.tsx` **完整存在**
- ✅ `session-detail-view.tsx` 已 import `VNCOverlay`
- ✅ Sandbox `supervisord.conf` 已启动 `xvfb + x11vnc + websockify`
- ✅ **API 层 `api/internal/router/router.go:42` 已注册 `WS /sessions/:id/vnc`**
- ✅ **`api/internal/handler/vnc_handler.go` 已实现 `VNCProxy`**（`gorilla/websocket`）
- ✅ 前端 VNC Overlay 与 API 端已完整对齐

#### 影响

~~用户在 UI 点击"虚拟机浏览器"按钮后:~~

~~1. 前端连接 `ws://.../api/sessions/{id}/vnc`~~
~~2. Nginx/API 收到请求 → 路由不存在 → 404 (nginx) 或 400 (gin)~~
~~3. `vnc-viewer.tsx` 进入 `error` 状态, 显示 "连接失败"~~

> ✅ **2026-08-31 已修复**：VNC WebSocket 代理已完整实现，点击"虚拟机浏览器"按钮后可正常连接。

#### ~~落地优先级: P0~~ → ✅ 已完成

### 4.3 沙箱 supervisor 端点对齐

| 端点 | mooc-manus | go-manus |
|------|------------|----------|
| `/sandbox/supervisor/*` (查 / 重启 supervisor 进程) | ✅ `sandbox/interfaces/endpoints/supervisor.py` | ✅ 同样保留 |

**评价**: 一致; 但 go-manus API 端**没有调用**这些端点, 目前只用到 shell/file。

---

## 五、前端 / UI 行为对齐

### 5.1 已对齐项

| 功能 | 状态 |
|------|------|
| 会话列表 / 创建 / 删除 / 详情 | ✅ |
| 文件上传 / 下载 / 预览 | ✅ |
| 工具调用渲染 (bash / file / browser / search / mcp / a2a / message) | ✅ |
| Plan 面板 + 步骤进度 | ✅ |
| Markdown / 富文本预览 | ✅ |
| 配置页 (LLM / Agent / MCP / A2A) | ✅ |

### 5.2 SSE 事件解析 (已修复)

| 事件类型 | 后端发送 | 前端解析 (修复后) |
|----------|----------|------------------|
| `step` | `{event:"step", data: {step: {...}, status: "..."}}` | `session-events.ts` 已支持嵌套 `data.step` 提取 |
| `plan` | `{event:"plan", data: {plan: {...}, status: "..."}}` | 已支持嵌套 `data.plan` 提取 |
| `message` | `{event:"message", data: {role, content}}` | ✅ |
| `error` | `{event:"error", data: {message: "..."}}` | ✅ (`use-session-detail.ts` 加了 sonner toast) |

### 5.3 **VNC Overlay 对齐**

- ✅ `ui/src/components/vnc-viewer.tsx` 期望连接 `wss(s)://host/api/sessions/{id}/vnc`
- ✅ 后端 `router.go:42` 已注册 `WS /api/sessions/:id/vnc`
- ✅ `handler/vnc_handler.go` 已实现 VNCProxy

详见 [第四章 4.2](#42-vnc-websocket-代理-重大功能缺失)（已修复）。

### 5.4 错误可见性 (已修复)

| 场景 | mooc-manus | go-manus (修复前) | go-manus (修复后) |
|------|------------|-------------------|-------------------|
| LLM 429 / 5xx | SSE `error` 事件 + 前端 toast | SSE 事件被静默吞掉 | ✅ sonner toast + session.status=completed |
| 网络断开 | 同上 | 同上 | ✅ |

---

## 六、运维 / 部署 / 中间件

### 6.1 部署拓扑

| 项目 | 部署 |
|------|------|
| mooc-manus | `nginx + api(FastAPI) + sandbox + ui(Next.js)` |
| go-manus | `api(Gin) + sandbox + ui(Next.js)`, **没有 nginx**, 前端通过 `NEXT_PUBLIC_API_BASE_URL=http://api:8000` 直连 |
| 影响 | 单机 / 内网测试 OK; **对外暴露时仍需 nginx** (TLS / 限流 / SSE buffer) |

### 6.2 数据库迁移

| 项目 | 迁移 |
|------|------|
| mooc-manus | Alembic (`alembic/versions/0e0d242438bc_create_files_table.py` 等) |
| go-manus | ❌ 仅 `postgres.go` 启动时 `CREATE TABLE IF NOT EXISTS`; 无迁移历史 |
| 风险 | 字段变更 / 索引调整 无版本记录; 线上演进困难 |

### 6.3 缓存 / 队列

| 组件 | mooc-manus | go-manus |
|------|------------|----------|
| Redis | 健康检查 + Stream 队列 + 任务锁 | ✅ 同样 |
| COS / OSS 文件存储 | `cos_file_storage.py` | ✅ `cos.go` |

### 6.4 日志

| 项目 | 日志 |
|------|------|
| mooc-manus | `infrastructure/logging/logging.py` 自定义 JSON formatter |
| go-manus | `pkg/logger` zap + 自定义格式 |
| 评价 | 一致; Go zap 性能更优 |

---

## 七、缺失功能清单 (优先级排序)

### P0 (阻塞核心用户体验)

| # | 缺失项 | 涉及文件 | 状态 |
|---|--------|----------|------|
| ~~1~~ | ~~VNC WebSocket 代理~~ | ~~`router/router.go` + `handler/vnc_handler.go`~~ | ✅ 已完成 (2026-08-31) |
| ~~2~~ | ~~LLM 429 / 5xx 指数退避~~ | ~~`openai_llm.go`~~ | ⚠️ 有固定 retry，无指数退避（待优化） |
| ~~3~~ | ~~空响应注入重试~~ | ~~`agent/base.go:invokeWithEmptyRetry`~~ | ✅ 已完成 (2026-08-31) |

### P1 (显著影响稳定性 / 可维护性)

| # | 缺失项 | 描述 |
|---|--------|------|
| 4 | **Alembic 风格的数据库迁移** | 新增 `migrations/` 目录, 记录每次 schema 变更 |
| 5 | **ReAct 阶段 JSON 模式可选** | 给 `react_agent.go` 加 `ResponseFormat` 可选, 应对"模型不返回 tool_calls 只返 JSON" 的场景 |
| 6 | **任务暂停 / 恢复** | `task_redis.go` 增加 `suspend / resume`, 与 mooc-manus `Task` 抽象对齐 |
| 7 | **EventMapper 统一映射** | `model/event.go` 增加 `ToSSE()` 方法, 统一 `MessageEvent/ErrorEvent/DoneEvent/WaitEvent` 的 SSE 协议头 |

### P2 (体验细节)

| # | 缺失项 | 描述 |
|---|--------|------|
| 8 | **Prompt 模板拆分为中文 / 英文双版本** | mooc-manus 有 `prompts/en/` 和 `prompts/` 双目录, go-manus 只在 `prompts.go` 写死中文 |
| 9 | **`tool_choice` 默认值** | mooc-manus `parallel_tool_calls=False` (deepseek 兼容); go-manus `openai_llm.go` 没有显式传 |
| 10 | **`noVNC` 前端依赖** | 确认 `ui/package.json` 有 `@novnc/novnc` 或类似库 |

### P3 (锦上添花)

| # | 缺失项 | 描述 |
|---|--------|------|
| 11 | **OpenTelemetry / Prometheus 指标** | LLM 调用次数 / 工具调用次数 / 失败率 |
| 12 | **健康检查拆分** | `/health/live`、 `/health/ready` (依赖 Postgres / Redis / Sandbox) |
| 13 | **Rate Limit** | `gin-contrib/ratelimit` 防止前端狂点 |

---

## 八、待优化项 (已实现但有差异)

| 项 | mooc-manus | go-manus | 建议 |
|----|------------|----------|------|
| **JSON Parser 容错** | `RepairJSONParser` 多重修复策略 (json_repair) | ✅ 同样实现 | 已对齐 |
| **Tool 调用超时** | 全局 3600s | tool 调用 15s / 全局 120s | Go 更合理 |
| **Prompt 国际化** | 双目录 | 仅中文 | 待补 |
| **Tool registry 动态加载** | 启动时按配置注入 | 同样 | 已对齐 |
| **MCP 客户端** | mcp 协议实现 | ✅ | 已对齐 |
| **A2A 客户端** | a2a 协议实现 | ✅ | 已对齐 |
| **Cos 文件存储** | 腾讯云 COS | ✅ | 已对齐 |
| **Embedding / RAG** | ❌ (都没有) | ❌ | 已对齐 (都没做) |

---

## 九、建议的落地顺序

> ⚠️ 以下已过时。P0-1 (VNC WebSocket) 和 P0-3 (空响应重试) 已于 **2026-08-31** 完成。剩余工作：

1. ~~**本周**: P0-1 (VNC WebSocket) → 已完成~~ ✅
2. ~~**本周**: P0-3 (空响应重试) → 已完成~~ ✅
3. **本周**: P0-2 (LLM 429 指数退避) → 有固定 retry，未实现指数退避，待优化
4. **下周**: P1-4 (DB迁移) → 给后续 schema 演进铺路
5. **下周**: P1-5 + P1-7 (ReAct JSON + EventMapper) → 提升多模型兼容性和代码整洁度
6. **按需**: P2 / P3 → 等反馈再排期

---

## 附录 A: 关键文件对照表

| 能力 | mooc-manus | go-manus |
|------|------------|----------|
| API 入口 | `api/app/main.py` | `api/cmd/server/main.go` |
| 路由 | `api/app/interfaces/endpoints/*.py` | `api/internal/router/router.go` |
| 服务 | `api/app/application/services/*.py` | `api/internal/service/*.go` |
| 仓储 | `api/app/infrastructure/repositories/*.py` | `api/internal/repository/*.go` |
| 领域模型 | `api/app/domain/models/*.py` | `api/internal/model/*.go` |
| LLM 客户端 | `api/app/infrastructure/external/llm/openai_llm.py` | `api/internal/external/openai_llm.go` |
| 工具基类 | `api/app/domain/services/tools/base.py` | `api/internal/agent/tools.go` |
| Planner | `api/app/domain/services/agents/planner.py` | `api/internal/agent/planner_agent.go` |
| ReAct | `api/app/domain/services/agents/react.py` | `api/internal/agent/react_agent.go` |
| Flow | `api/app/domain/services/flows/planner_react.py` | `api/internal/agent/planner_react_flow.go` |
| 任务运行 | `api/app/domain/services/agent_task_runner.py` | `api/internal/agent/task_runner.go` |
| Sandbox 客户端 | `api/app/infrastructure/external/sandbox/docker_sandbox.py` | `api/internal/external/sandbox.go` |
| **VNC WS** | `api/app/interfaces/endpoints/session_routes.py:285-360` | ✅ `router.go:42` + `handler/vnc_handler.go` |
| MCP 客户端 | `api/app/domain/services/tools/mcp.py` | `api/internal/external/mcp_client.go` |
| A2A 客户端 | `api/app/domain/services/tools/a2a.py` | `api/internal/external/a2a_client.go` |
| Redis 队列 | `api/app/infrastructure/external/message_queue/redis_stream_message_queue.py` | `api/internal/external/message_queue_redis.go` |
| 提示词 | `api/app/domain/services/agents/prompts/*.py` | `api/internal/agent/prompts.go` |
| 沙箱 Dockerfile | `sandbox/Dockerfile` | `sandbox/Dockerfile` (字节一致) |
| 沙箱 supervisord | `sandbox/supervisord.conf` | `sandbox/supervisord.conf` (字节一致) |

---

## 附录 B: 验证清单 (供下次回归测试用)

- [x] `WS /api/sessions/:id/vnc` 存在且能连接 sandbox ✅ (2026-08-31)
- [ ] LLM 429 时, 会自动 backoff 重试, 不立刻失败
- [x] LLM 返回空 content 时, 会自动注入"AI 无响应, 请继续。" 重试 ✅ (2026-08-31)
- [ ] Planner 阶段失败时, 前端能看到 error toast
- [ ] 数据库 schema 变更走迁移文件, 不依赖 IF NOT EXISTS
- [ ] ReAct 阶段在 tool_calls 空 + JSON 模式 下能 fallback 到文本步骤
