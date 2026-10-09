# 阶段 5：唯一 Run API、UI 与 SSE 契约

阶段 4 已完成后端执行切换。本阶段发布唯一的 Run 对外契约、切换 UI，并删除 Session Chat/Stop/SSE 路由。项目未上线，不保留版本化兼容 API，也不设置新旧客户端并行期。

## API 契约

- `POST /sessions/:sessionId/runs`：携带客户端幂等键创建顶层 Run，返回 `run_id`、初始状态和必要快照。
- `POST /runs/:runId/input`：携带消息幂等键，只对 `waiting_input` Run 提交输入并继续同一 Run。
- `POST /runs/:runId/cancel`：条件迁移到 `cancelling`，重复取消返回稳定结果；`cancelled` 只在执行 goroutine 退出和 ToolSet 释放后出现。
- `GET /runs/:runId`：查询 Run、Plan 快照、错误和最终消息。
- `GET /runs/:runId/events`：按 Run ID 订阅 SSE。
- `GET /sessions/:sessionId/runs`：按时间分页查询历史 Run，供会话恢复和 UI 展示。

响应 DTO 与数据库 model 分离，字段和错误码在 Handler 契约测试中锁定。Handler 只负责鉴权、输入校验和协议转换，不自行拼装状态机。

`POST /runs/:runId/input` 的状态冲突必须使用稳定协议错误，不能由实现细节决定返回 500：

- Run 为 `cancelling`：HTTP 409，错误码 `run_cancelling`。
- Run 为 `pending` 或 `running`：HTTP 409，错误码 `run_not_waiting_input`。
- Run 为 `succeeded`、`failed`、`cancelled` 或 `interrupted`：HTTP 409，错误码 `run_terminal`。
- Run 为 `waiting_input` 且消息幂等键已处理：返回原提交结果，不创建第二条回答消息，也不启动第二个 Engine goroutine。

## SSE 契约

- `id` 使用 Redis Stream 游标或等价的稳定可排序游标，服务端接受 `Last-Event-ID`。
- SSE 采用至少一次投递语义。服务端重放同一事件时必须保留原 `event_id`，不能为重放生成新 ID。
- 客户端按 `run_id + event_id` 幂等去重；重复的终态事件必须内容一致，不得重复追加最终消息。
- 正常续读边界应从 `Last-Event-ID` 之后返回，不跳过紧随游标的下一条事件；网络边界仍允许客户端收到尚未确认的重复事件。
- Redis Stream 存在时从游标继续；游标过期时返回明确的 `event_stream_expired` 协议结果，并要求客户端查询 Run 快照，不能伪造逐 token 历史。
- 终态事件发送后关闭流；客户端以 `GET /runs/:id` 的 PostgreSQL 快照作为最终事实。
- `cancelling` 是可展示的过渡态但不是终态；只允许 `cancelled` 事件关闭取消中的流。
- 事件只包含 UI 所需增量和展示数据，不把完整内部 ToolResult、Prompt 或敏感参数写入流。

## UI 切换

UI 以 `run_id` 作为执行标识，Session 仅作为会话容器。会话详情通过消息和历史 Run 展示，不再解析 task ID、SessionStatus 或旧 Chat 响应。

断线恢复顺序固定为：先按 event ID 去重并落本地状态 -> 保存最后游标 -> 使用 `Last-Event-ID` 重连 -> 过期时查询 Run 快照 -> Run 非终态时重新订阅。UI 不自行推断后端状态迁移。

## 代码范围

- `internal/handler`、`internal/router`：增加 Run DTO/路由并删除 Session 执行路由。
- `internal/service`：仅补查询组合，不新增第二套状态迁移。
- `ui/src`：API client、store/hook、事件解析、取消和等待输入交互全部改用 Run。
- HTTP/SSE 测试：成功链、输入状态冲突与稳定错误码、Last-Event-ID、过期游标、终态关闭和取消竞争。

## 检查点

### 5A：Run API/SSE

发布 Run Handler/Router 和契约测试。阶段结束前旧 Session 路由仍可短暂存在，但必须调用同一 RunService，不能保留 Task 读取路径。

建议提交：`feat(api): expose run lifecycle and event APIs`。

### 5B：UI 切换

切换 API client、状态存储、SSE 重连、等待输入和取消交互；前端测试、lint、build 通过。

建议提交：`refactor(ui): consume run lifecycle`。

### 5C：删除旧路由

删除 Session Chat/Stop/SSE 路由、DTO 和 UI 旧解析。因为项目未上线，不增加 deprecated 标记或兼容版本。

建议提交：`refactor(api): remove legacy session execution routes`。

## 完成条件

- UI 和 API 只出现 Run 执行语义；全仓不存在旧 Session Chat/Stop/SSE 调用。
- Last-Event-ID 的重复、丢失、过期、稳定 event ID、客户端幂等和终态重复边界有 HTTP 层真实测试。
- input API 对 `cancelling`、非等待活跃状态、终态和重复幂等输入的 HTTP 状态码与业务错误码有契约测试。
- Run 查询在 Redis 不可用时仍返回终态、错误、Plan 和最终消息。
- API test/race/vet、UI test/lint/build 和 diff 检查全部通过。

## 回滚边界

5A、5B 可分别 revert。5C 只能在 UI 已切换且引用扫描为零后执行；回滚 5C 只恢复薄路由，不得恢复旧 Task 语义。若回滚 UI，后端仍以 Run 为唯一事实来源。
