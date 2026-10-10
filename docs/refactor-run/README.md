# Run/Session 重构方案集

本目录是基于当前 `api` 源码整理的实施方案。当前生产语义仍是 `Session + AgentService + RedisStreamTask`；Run 领域、Repository、RunService 和 RunExecutor 已分阶段实现，但尚未接入 Bootstrap、Handler、Chat 或 SSE 生产路径。Run API 仍是后续目标，不代表已经实现。

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
| 1 | [02-settings-and-prompts.md](./02-settings-and-prompts.md) | 唯一 Settings、校验、Prompt 版本和快照接口 | 否 |
| 2 | [03-engine-and-context.md](./03-engine-and-context.md) | Engine 输入输出契约、ContextPolicy、StepOutcome | 否 |
| 3 | [04-run-domain-and-storage.md](./04-run-domain-and-storage.md) | Run 领域模型、可恢复执行快照、migration、Repository 和契约测试 | 否，暂不接请求 |
| 4 | [05-run-execution-cutover.md](./05-run-execution-cutover.md) | RunExecutor 装配、取消与 ToolSet 生命周期、可观测性、Session 薄适配 | 是，唯一后端语义切换点 |
| 5 | [06-api-ui-sse-cutover.md](./06-api-ui-sse-cutover.md) | 发布唯一 Run API、至少一次 SSE、切换 UI 并删除旧路由 | 是，唯一对外契约切换点 |
| 6 | [07-legacy-removal.md](./07-legacy-removal.md) | 删除旧 Task、Session 执行字段和无用投影 | 否，清理已不可达代码 |

阶段 1、2 可以独立实施；阶段 3 必须先完成领域契约；阶段 4 之前不得让 Run 写入生产请求；阶段 4 之后禁止恢复旧 Session 执行写路径。阶段 4 暂存的 Session 路由只是调用 RunService 的适配层，阶段 5 必须随 UI 切换直接删除，不设置兼容期。

## 每个阶段的完成标准

- `cd api && go test ./...`
- `cd api && go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1`
- `cd api && go vet ./...`
- `git diff --check`
- 涉及 UI 时，再运行 `cd ui && npm run lint && npm run build`
- 新增的状态、存储、HTTP 或 SSE 契约均有行为测试；不为 getter、常量或 mock 自身重复写低收益测试。
- 新增生产切换时必须同时提供状态迁移、恢复、取消收敛和关键指标，不能只验证成功路径。
- 一个阶段对应一个或多个独立 Conventional Commit，每个提交都能单独 revert。
- `STATUS.md` 记录提交 SHA、真实验证结果、生产语义和下一入口。

受限环境不能监听回环端口时，必须记录失败原因并在允许本地监听的开发环境补跑，不能把环境失败写成通过。

## 禁止双轨长期存在

阶段 3 可以有未接生产的 Run 代码；阶段 4 一次性切换所有创建、继续、等待、完成、失败和取消写入。旧路由若暂时保留，只能薄委托 RunService，不得继续写 `sessions.task_id/status/events`。阶段 5 完成 UI 切换后，阶段 6 才删除旧实现。

## 中断恢复

```bash
cd /Users/huanghao2/GolandProjects/study/imooc-mas/go-manus
git status --short --branch
git log -8 --oneline --decorate
sed -n '1,240p' docs/refactor-run/STATUS.md
```

从 `STATUS.md` 中第一个 `pending` 或 `in_progress` 检查点继续；先运行该检查点的恢复基线命令，再修改对应文档列出的代码范围。不要通过复制旧 Task 或新增双写来绕过编译错误。
