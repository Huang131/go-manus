# Run/Session 重构方案集

本目录记录 Run/Session 重构的实施决策与完成依据。当前生产语义是 `Session -> RunApplicationService -> RunExecutor -> PlannerEngineAdapter`：Session 只保存会话元数据，Run/Message 是执行事实来源，Redis Stream 只承担 Run 的短期实时事件与 SSE 续读。

阶段 1–6 均已完成，且已在独立 PostgreSQL、Redis、MinIO 环境完成完整集成验收。后续维护应遵循现有 Run 契约，不再恢复旧 Session Chat/Stop 或 Task 执行链；详见 [STATUS.md](./STATUS.md)。

事实基线见 [00-current-review-2026-10.md](./00-current-review-2026-10.md)，实施状态见 [STATUS.md](./STATUS.md)。`docs/重构.md` 只作为历史提案，不作为实施依据。

## 目标

- Session 只管理会话元数据、消息查询入口和 Sandbox 关联。
- Run 表示一次用户请求对应的一次执行，允许一个 Session 拥有多个历史 Run，但同一时间只有一个 active Run。
- `waiting_input` 必须依靠 PostgreSQL 中的执行快照恢复，不能依赖存活 goroutine 或 `SimpleMemory`。
- Engine 负责计算，RunService/Repository 负责状态持久化，Handler 不直接操作 Engine 或数据库。
- PostgreSQL 是业务状态事实来源，Redis Stream 只承载短期实时事件。
- 保留当前 `internal/model`、`agent`、`service`、`repository`、`handler`、`bootstrap` 骨架，不做目录大迁移。

## 阶段总览

| 阶段 | 文档 | 交付物 | 是否改变生产语义 |
|---|---|---|---|
| 1 | [02-settings-and-prompts.md](./02-settings-and-prompts.md) | 唯一 Settings、校验、Prompt 版本和快照接口 | complete |
| 2 | [03-engine-and-context.md](./03-engine-and-context.md) | Engine 输入输出契约、ContextPolicy、StepOutcome | complete |
| 3 | [04-run-domain-and-storage.md](./04-run-domain-and-storage.md) | Run 领域模型、可恢复执行快照、migration、Repository 和契约测试 | complete |
| 4 | [05-run-execution-cutover.md](./05-run-execution-cutover.md) | RunExecutor 装配、取消与 ToolSet 生命周期、Session 到 Run 的唯一后端切换 | complete |
| 5 | [06-api-ui-sse-cutover.md](./06-api-ui-sse-cutover.md) | 唯一 Run API、SSE、UI 切换和旧路由删除 | complete |
| 6 | [07-legacy-removal.md](./07-legacy-removal.md) | 删除旧 Task、Session 执行字段和不可达代码 | complete |

这些阶段按顺序完成，当前不得恢复旧 Session 执行写路径、旧 Task registry 或 `sessions.events` 双写。后续功能应以 Run/Message 为唯一执行事实来源；达到架构文档定义的触发条件后，再单独评估 Worker、Lease 或 Outbox。

## 每个阶段的完成标准

- `cd api && go test ./...`
- `cd api && go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1`
- `cd api && go vet ./...`
- `git diff --check`
- 涉及 UI 时，再运行 `cd ui && npm run lint && npm run build`
- 新增的状态、存储、HTTP 或 SSE 契约均有行为测试；不为 getter、常量或 mock 自身重复写低收益测试。
- 新增生产切换时必须同时提供状态迁移、恢复、取消收敛和关键指标，不能只验证成功路径。
- 一个阶段对应一个或多个独立 Conventional Commit，每个提交都能单独 revert。
- `STATUS.md` 记录提交 SHA、真实验证结果、生产语义和维护决策。

受限环境不能监听回环端口时，必须记录失败原因并在允许本地监听的开发环境补跑，不能把环境失败写成通过。

## 禁止双轨长期存在

这是阶段 3–6 实施期间采用的约束：阶段 3 可以有未接生产的 Run 代码；阶段 4 一次性切换所有创建、继续、等待、完成、失败和取消写入；阶段 5 完成 UI 切换后，阶段 6 删除旧实现。当前这些阶段均已完成，生产代码不得恢复旧 Session 执行写路径、旧 Task registry 或 `sessions.task_id/status/events` 双写。

## 中断恢复

```bash
cd /Users/huanghao2/GolandProjects/study/imooc-mas/go-manus
git status --short --branch
git log -8 --oneline --decorate
sed -n '1,240p' docs/refactor-run/STATUS.md
```

实施阶段已经结束。进行后续维护时，先读取 `STATUS.md` 的当前生产语义与决策记录；不要通过复制旧 Task 或新增双写绕过编译错误。
