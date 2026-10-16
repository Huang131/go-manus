# Run 历史追踪回放 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 刷新新会话时恢复消息、计划、步骤和受控的工具调用过程，并让工具过程在 Redis 过期后仍可审计。

**Architecture:** `messages` 继续是最终对话消息的唯一事实；`runs.execution_snapshot` 继续是执行恢复快照。新增按 Run 排序的 `run_trace_events`，仅保存 `plan`、`step`、`tool_calling`、`tool_called`、`wait`、`error` 六类 UI 回放事件。执行器通过复合发布器先可靠记录追踪事件、再尽力发布 Redis SSE；历史 API 将该追踪事件随 Run 返回，前端使用它重建时间线。

**Tech Stack:** Go、PostgreSQL/JSONB、pgx、Gin、Next.js、TypeScript、现有 Redis Stream SSE。

---

## 文件边界

- `api/migrations/008_create_run_trace_events.sql`：首次 Docker 初始化和现有开发库建表的幂等 DDL。
- `api/internal/model/run_trace.go`：受控追踪事件的领域模型及可持久化事件筛选。
- `api/internal/repository/run_trace_repository.go`：PostgreSQL 批量查询、顺序分配和 JSONB 编解码。
- `api/internal/service/run_trace_recorder.go`：记录器和“持久化优先、SSE 尽力而为”的复合发布器。
- `api/internal/service/run_application_service.go`：历史响应在单次批量查询中带回 `events`。
- `api/internal/bootstrap/app.go`：装配追踪仓库和复合发布器。
- `ui/src/hooks/use-session-detail.ts`：将历史追踪事件与消息恢复为前端事件；对早期无追踪记录的 Run 回退到快照。
- 对应 `*_test.go`：仓储 component、服务 unit、Handler HTTP 和前端纯映射测试。

### Task 1: 用数据库契约锁定追踪记录

**Files:**
- Create: `api/migrations/008_create_run_trace_events.sql`
- Create: `api/internal/model/run_trace.go`
- Create: `api/internal/repository/run_trace_repository.go`
- Modify: `docker-compose.yml`
- Test: `api/internal/repository/run_trace_repository_component_test.go`

- [ ] **Step 1: 写失败的 PostgreSQL component 测试**

```go
func TestRunTraceRepositoryAppendAndListByRunIDs(t *testing.T) {
    repo := newComponentRunTraceRepository(t)
    require.NoError(t, repo.Append(context.Background(), "run-1", &model.RunTraceEvent{
        Type: model.EventTypeToolCalling,
        Data: json.RawMessage(`{"tool_call_id":"call-1","function_name":"shell","arguments":{"command":"pwd"}}`),
    }))
    events, err := repo.ListByRunIDs(context.Background(), []string{"run-1"})
    require.NoError(t, err)
    require.Equal(t, []int64{1}, []int64{events["run-1"][0].Sequence})
    require.Equal(t, model.EventTypeToolCalling, events["run-1"][0].Type)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./internal/repository -run '^TestRunTraceRepositoryAppendAndListByRunIDs$' -count=1`

Expected: FAIL，因为 `RunTraceRepository` 和测试建表 fixture 尚不存在。

- [ ] **Step 3: 实现最小数据库模型和仓储**

```sql
CREATE TABLE IF NOT EXISTS run_trace_events (
    run_id VARCHAR(255) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence BIGINT NOT NULL,
    event_type VARCHAR(32) NOT NULL CHECK (event_type IN ('plan','step','tool_calling','tool_called','wait','error')),
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    PRIMARY KEY (run_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_run_trace_events_run_sequence ON run_trace_events (run_id, sequence);
```

`Append` 在事务内为同一 Run 使用 `SELECT COALESCE(MAX(sequence), 0) + 1 ... FOR UPDATE` 的锁定行策略，并只接受 `IsPersistableRunTraceEvent` 白名单事件。`ListByRunIDs` 使用一个 `WHERE run_id = ANY($1)` 查询，并按 `run_id, sequence` 升序分组。

- [ ] **Step 4: 运行目标仓储测试确认通过**

Run: `GOCACHE=/private/tmp/go-manus-gocache make test-component`

Expected: 新 component 测试通过，且 SQL/JSONB/级联删除契约由真实 PostgreSQL 覆盖。

- [ ] **Step 5: 接入 Docker 首次初始化**

在 `docker-compose.yml` 的 Postgres `docker-entrypoint-initdb.d` 挂载列表末尾加入：

```yaml
- ./api/migrations/008_create_run_trace_events.sql:/docker-entrypoint-initdb.d/008_create_run_trace_events.sql
```

并在当前开发 PostgreSQL 直接执行该 SQL；不新增迁移服务或要求部署者手工执行迁移。

### Task 2: 将追踪写入执行路径并保持实时事件语义

**Files:**
- Create: `api/internal/service/run_trace_recorder.go`
- Modify: `api/internal/service/run_executor.go`
- Modify: `api/internal/bootstrap/app.go`
- Test: `api/internal/service/run_trace_recorder_test.go`
- Test: `api/internal/service/run_executor_test.go`

- [ ] **Step 1: 写失败的单元测试**

```go
func TestRunTraceEventPublisherPersistsTraceBeforePublishingSSE(t *testing.T) {
    trace := &traceStoreStub{}
    realtime := &eventPublisherStub{}
    publisher := NewRunTraceEventPublisher(trace, realtime)
    err := publisher.Publish(context.Background(), "run-1", model.NewToolCallingEvent("call-1", "shell", map[string]interface{}{"command": "pwd"}))
    require.NoError(t, err)
    require.Equal(t, []string{"trace:tool_calling", "realtime:tool_calling"}, append(trace.calls, realtime.calls...))
}

func TestRunTraceEventPublisherDoesNotPersistTokenDeltas(t *testing.T) {
    trace := &traceStoreStub{}
    publisher := NewRunTraceEventPublisher(trace, nil)
    require.NoError(t, publisher.Publish(context.Background(), "run-1", model.NewMessageDeltaEvent("m-1", "a", 1)))
    require.Empty(t, trace.calls)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service -run '^TestRunTraceEventPublisher' -count=1`

Expected: FAIL，因为复合发布器尚不存在。

- [ ] **Step 3: 写最小实现**

定义 `RunTraceStore.Append(context.Context, string, *model.RunTraceEvent) error` 和 `RunTraceEventPublisher`：对白名单事件先写 PostgreSQL；写入失败直接返回错误，调用方把 Run 收敛为失败，防止声明“可追溯”却静默丢审计。成功后调用 Redis 发布器；Redis 失败只记录告警、不撤销已记录的执行事实。`message_delta`、`message_done`、`shell_output` 不落表。

工具 payload 由 `model.NewRunTraceEvent` 在写入前执行：深度复制、敏感键（`password`、`token`、`secret`、`authorization`）替换为 `"[REDACTED]"`，JSON 编码后最多保留 64 KiB，超过时写入 `{ "truncated": true, "summary": "..." }`；对象存储产物只保留既有文件引用。

- [ ] **Step 4: 将追踪故障传播到执行结果**

让 `PlannerEngineAdapter.Execute` 对持久化追踪发布错误立即停止消费 Flow、返回失败结果；但 Redis 发布错误仍由复合发布器吞掉并记日志。补一条 `RunExecutor` 测试，断言追踪库失败后 `Finish` 收到 `RunExecutionFailed`。

- [ ] **Step 5: 运行服务与执行器测试确认通过**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service ./internal/agent -count=1`

Expected: PASS，且现有 Redis-only 发布器测试保持通过。

### Task 3: 在历史 API 返回完整回放所需数据

**Files:**
- Modify: `api/internal/service/run_application_service.go`
- Modify: `api/internal/handler/run_handler_test.go`
- Modify: `api/internal/service/run_application_service_test.go`
- Test: `api/internal/handler/run_handler_test.go`

- [ ] **Step 1: 写失败的应用服务与 HTTP 契约测试**

```go
func TestRunApplicationServiceListHistoryBatchesMessagesAndTraceEvents(t *testing.T) {
    page, err := service.ListBySessionID(context.Background(), "session-1", 20, 0)
    require.NoError(t, err)
    require.Len(t, page.Items[0].Events, 2)
    require.Equal(t, model.EventTypePlan, page.Items[0].Events[0].Type)
}
```

Handler 测试精确断言 `GET /sessions/:id/runs` 成功信封中的 `data.items[0].events[0].event_type == "tool_called"`。

- [ ] **Step 2: 运行测试确认失败**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service ./internal/handler -run 'TestRun(ApplicationServiceListHistoryBatchesMessagesAndTraceEvents|HandlerListBySession)' -count=1`

Expected: FAIL，因为 `RunHistoryItem.Events` 和批量读取尚未实现。

- [ ] **Step 3: 实现批量读取与 API 响应**

给 `runApplicationStore` 增加 `ListTraceEventsByRunIDs`，给 `RunHistoryItem` 增加 `Events []*model.RunTraceEvent`（JSON 字段 `events`）。`ListBySessionID` 与批量消息读取同级调用，避免 N+1；为空时返回空数组而不是 `null`。

- [ ] **Step 4: 运行 HTTP 和应用层测试确认通过**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./internal/service ./internal/handler -count=1`

Expected: PASS，分页、消息和回放事件均保持每个 Run 隔离。

### Task 4: 前端按持久化追踪回放

**Files:**
- Modify: `ui/src/lib/api/types.ts`
- Modify: `ui/src/hooks/use-session-detail.ts`
- Create: `ui/src/hooks/use-session-detail-history.test.ts`

- [ ] **Step 1: 写失败的纯映射测试**

```ts
it('按 Run 的持久化追踪恢复计划、步骤和已完成工具调用', () => {
  const events = historyToEvents(historyWithTrace, [])
  expect(events.map((event) => event.type)).toEqual([
    'message', 'plan', 'step', 'tool_calling', 'tool_called', 'step', 'message',
  ])
  expect(eventsToTimeline(events).find((item) => item.kind === 'step')).toMatchObject({
    tools: [expect.objectContaining({ function: 'shell', status: 'called' })],
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm test -- --runInBand ui/src/hooks/use-session-detail-history.test.ts`

Expected: FAIL，因为项目尚未配置可执行的前端单测脚本且类型中没有 `events`。

- [ ] **Step 3: 建立最小前端测试入口并实现映射**

使用项目现有 TypeScript/Next 依赖新增最小 Node test runner（不引入浏览器 E2E 框架）。将 `historyToEvents` 导出为纯函数：每个 Run 输出用户消息、持久化 `plan/step/tool_*` 事件、助手消息；历史有追踪时严格使用追踪顺序，早期无追踪时由 `execution_snapshot` 合成一个 plan 与步骤列表。保留附件解析、消息 ID 和运行列表的旧到新排序。

- [ ] **Step 4: 运行前端映射测试、lint 与生产构建**

Run: `npm run lint && npm run build`

Expected: PASS，类型声明和服务端 JSON 字段完全一致。

### Task 5: 端到端验证、文档与提交

**Files:**
- Modify: `.trae/specs/run-session-refactor/spec.md`
- Modify: `docs/refactor-run/STATUS.md`

- [ ] **Step 1: 记录架构决策**

将“Redis Stream 是短期实时通道；Run UI trace 是仅含六类事件的持久化审计投影，不保存 token 增量或二进制”的新决策写入规格和状态记录，替换“不保存永久事件溯源”的模糊表述。

- [ ] **Step 2: 在开发 PostgreSQL 直接创建表并验证 schema**

Run: `docker exec -i go-manus-postgres psql -U postgres -d manus < api/migrations/008_create_run_trace_events.sql`

Expected: `CREATE TABLE` / `CREATE INDEX` 或幂等成功；`\\d+ run_trace_events` 显示主键 `(run_id, sequence)`。

- [ ] **Step 3: 运行完整验证**

Run: `GOCACHE=/private/tmp/go-manus-gocache go test ./... -count=1 && cd ../ui && npm run lint && npm run build`

Expected: Go 单元测试、前端 lint 和构建全部通过。

- [ ] **Step 4: 运行组件/API/集成测试并检查容器日志**

Run: `GOCACHE=/private/tmp/go-manus-gocache make test-component && GOCACHE=/private/tmp/go-manus-gocache make test-api && GOCACHE=/private/tmp/go-manus-gocache make test-integration`

Expected: 全部通过；随后检查 `docker logs go-manus-api`、`docker logs go-manus-postgres`、`docker logs go-manus-redis` 是否存在新增错误。

- [ ] **Step 5: 更新代码图、检查影响范围并提交**

Run: `npx graphify hook-rebuild && git diff --check && git status --short`

随后运行 GitNexus `detect_changes`；若索引工具仍因 `Napi::Error` 无法运行，在提交说明和验证记录中保留该阻塞证据。精确暂存本计划涉及的文件，提交消息：`feat(run): persist replayable tool traces`。
