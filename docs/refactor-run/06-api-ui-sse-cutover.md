# 阶段 5：Run API、UI 与 SSE 契约切换

## 目标

发布面向 Run 的最终 HTTP 契约，把 UI 的状态、输入、停止和断线恢复全部切到 Run，并删除阶段 4 的旧 chat/stop 适配路由。本阶段结束后 Session API 只返回会话信息，执行状态只通过 Run API 表达。

## 非目标

- 不改变 Run 的状态机和执行内核。
- 不删除已不可达的 Task/Registry 源文件；阶段 6 统一清理。
- 不持久化逐 token 事件，不引入第二套 SSE 游标。
- 不为旧客户端保留版本化兼容 API。

## 前置条件与唯一语义

- 阶段 4 complete，生产执行路径已经只写 Run。
- 旧 chat/stop 即使存在也只委托 RunService，因此本阶段短暂路由并存不构成双业务语义。
- 开始前分别运行 API 全量测试和 UI `npm run lint`、`npm run build`。

## 最终 HTTP 契约

```text
POST /api/sessions/{sessionID}/runs
GET  /api/sessions/{sessionID}/runs?limit=20&offset=0
GET  /api/runs/{runID}
POST /api/runs/{runID}/input
POST /api/runs/{runID}/cancel
GET  /api/runs/{runID}/events

GET /api/settings/agent
PUT /api/settings/agent
```

创建请求：

```json
{
  "message": "用户消息",
  "attachments": ["file-id"],
  "model_id": "可省略的模型 ID"
}
```

`Idempotency-Key` 通过 HTTP header 传递，空值由服务端生成但客户端重试必须复用原值。创建成功返回 201；同一 key 返回原 Run 的 200；同 Session 有另一活跃 Run 返回 409。input 只接受 waiting_input，返回 202；cancel 幂等，已 cancelled 返回 200，其他终态返回 409。不存在或不属于当前 Session 的资源返回 404。

`GET /runs/{id}` 返回 `RunSnapshot`：Run 元数据、当前 Plan/revision、按 ordinal 排序的 messages、prompt hashes 和 settings snapshot。不得返回密钥、内部 cancel 句柄或 Redis key。Session 列表 DTO 可携带可空的 `active_run_summary`，但 `model.Session` 本身不增加执行状态。

## 文件清单

创建：

- `api/internal/handler/run_handler.go`、`run_handler_test.go`。
- `api/internal/handler/settings_handler.go`、`settings_handler_test.go`。
- `ui/src/lib/api/run.ts`：Run CRUD、input/cancel、SSE。
- `ui/src/hooks/use-run-detail.ts`：RunSnapshot 和增量事件的唯一客户端状态入口。

修改：

- `api/internal/router/router.go`：注册 Run/Settings 路由，最终删除 chat/stop 和旧 agent settings 路由。
- `api/internal/handler/sse.go`：只保留通用 SSE 编码；Run handler 使用标准 `Last-Event-ID`。
- `api/internal/handler/session_handler.go`：移除 Chat、Stop、执行事件订阅和兼容 status/events DTO。
- `api/internal/service/run_service.go`：增加 RunSnapshot 查询和 Session 下分页列表。
- `api/internal/service/session_service.go`：返回纯 Session DTO 与可选 active_run_summary，不返回 events/status。
- `api/internal/bootstrap/app.go`：注入 RunHandler 和 SettingsHandler。
- `ui/src/lib/api/types.ts`：新增 RunStatus、Run、RunSnapshot、Plan/Step；删除 SessionStatus 和 ChatParams。
- `ui/src/lib/api/session.ts`：删除 chat/stop/空流续读。
- `ui/src/hooks/use-session-detail.ts`：改为会话元数据组合 `use-run-detail`，或删除后由新 Hook 替代。
- `ui/src/components/session-detail-view.tsx`、`session-item.tsx`、`chat-input.tsx`、`plan-panel.tsx`：状态判断改为 active Run。
- `ui/src/app/sessions/[id]/page.tsx`、`ui/src/app/page.tsx`：创建 Session 后通过 Run API 发首条消息。
- `ui/src/lib/session-events.ts`：支持 Snapshot 初始化和 Run SSE 增量，不从 Session.events 恢复。
- `ui/src/components/settings/CommonSetting.tsx`、`ui/src/lib/api/config.ts`：Agent settings 改用 GET/PUT `/settings/agent`。

删除：

- 路由 `POST /api/sessions/:id/chat`、`POST /api/sessions/:id/stop`。
- 路由 `GET/POST /api/app-config/agent`。
- UI 中 `startEmptyStream`、`event_id` body 字段和 Session status 本地伪更新。

## SSE 契约

`GET /api/runs/{runID}/events` 返回标准 `text/event-stream`：

- 客户端通过 `Last-Event-ID` header 续读，服务端不再接受 body `event_id`。
- 每条 Redis 事件输出 `id: <redis-stream-id>`、`event: <event-type>`、`data: <json>`。
- 每 15 秒发送无 id 的心跳注释。
- Run 到终态且积压事件发送完毕后关闭连接；客户端随后以 RunSnapshot 校准最终状态。

Redis stream 过期、游标早于 stream 首条记录或 Redis 重启时，服务端先输出无 id 的 `snapshot_required` 事件，然后从当前最新位置订阅仍活跃 Run；终态 Run 直接关闭。UI 收到该事件或网络重连后无法续读时，调用 `GET /runs/{id}`，用 messages + Plan 完整替换页面快照，再以新的空游标订阅。`snapshot_required` 不带 SSE id，避免把数据库快照伪装成 Redis 游标。

SSE Handler 不拼装业务状态，只调用 RunService 获取权限/终态并调用 RunEventStream 读取事件。Redis 不可用时 Run 查询仍正常，SSE 返回明确 503，UI 采用短退避轮询 RunSnapshot，终态后停止。

## UI 状态规则

- Session 列表运行标记来自 `active_run_summary.status`，没有 active Run 即空闲。
- 页面初次加载：取 Session 元数据和 Run 列表，选择 URL 指定 Run 或最新 Run，再取 RunSnapshot。
- 发送消息：无活跃 Run 时 Create；waiting_input 时 SubmitInput；pending/running 时禁用输入或显示 409。
- 停止按钮只在 pending/running/waiting_input 显示，调用 cancel 后以服务端响应覆盖状态。
- UI 不预写 succeeded/cancelled，不把 SSE 断开当 completed。
- 历史 Run 可选择查看；历史终态 Run 不建立持续 SSE。

## 实施顺序与提交检查点

### A. 新 API 和 SSE（旧 UI 仍可运行）

新增 RunHandler、SettingsHandler、RunSnapshot 和新路由。保留旧 chat/stop 薄适配，完成 Handler/SSE 契约测试。此时两组路由调用同一个 RunService，没有双写。建议提交：`feat(api): expose run lifecycle and event APIs`。

### B. UI 切换并删除旧路由

先修改类型和 API client，再切 Hook/组件，最后删除旧前后端入口。一个提交内保持 `npm run build` 通过；提交前确认源码不存在旧 URL。建议提交：`refactor(ui): switch session execution to runs`。

B 固定拆为两个提交：B1 新增 API client + Hook，但生产页面尚未启用；B2 切换页面并删除旧前后端路由。B1 不得并行发起旧、新请求，生产页面仍只走旧薄适配；B2 完成后才能标记阶段 complete。

## 针对性测试

- Handler：状态码、Idempotency-Key、分页边界、Session/Run 归属校验、敏感字段不输出。
- SSE：Last-Event-ID 续读、严格大于游标、心跳无 id、积压后终态关闭、过期游标发 snapshot_required、Redis 故障 503。
- RunSnapshot：Plan/messages 顺序稳定，终态与条件更新一致。
- UI 静态检查：不再引用 SessionStatus、`/chat`、`/stop`、`event_id`。
- UI 手工契约场景：首条消息、运行中刷新、等待输入后继续、取消、断网重连、Redis stream 过期后恢复、查看历史 Run。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/handler ./internal/service ./internal/router ./internal/bootstrap -count=1
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
cd ../ui
npm run lint
npm run build
cd ..
rg -n '/sessions/.*/(chat|stop)|event_id|SessionStatus|startEmptyStream' api/internal ui/src
git diff --check
```

最终 `rg` 不得命中生产代码；测试名称或迁移说明若命中需记录并说明。

## 回滚

检查点 A 可独立 revert，阶段 4 旧适配仍可服务 UI。B 尚未发布时可 revert B 回到旧 UI；后端执行仍是 Run 语义。B 已部署后若回滚，必须同时回滚 UI 与路由删除，恢复的旧路由仍只能薄委托 RunService，禁止恢复 Session 执行写入。开发数据无需转换。

## 完成定义

- UI 只通过 Run API 控制执行，Session 不承载执行状态。
- 旧 chat/stop 和 app-config/agent 路由已删除。
- Last-Event-ID、快照恢复和 Redis 故障行为有测试。
- API 全量 test/race/vet 与 UI lint/build 均通过。
- `STATUS.md` 写入 A/B 提交、验证结果和阶段 6 入口。

## 失败或中断恢复

```bash
git log -6 --oneline
rg -n '/sessions/.*/(chat|stop)|/runs/|event_id|SessionStatus' api/internal ui/src
(cd api && GOCACHE=/private/tmp/go-manus-gocache go test ./internal/handler ./internal/service -count=1)
(cd ui && npm run build)
```

根据页面实际调用的 URL 判断当前入口。若 UI 已调用 Run API，旧路由是否存在不影响语义，但必须完成删除后才能进入阶段 6。
