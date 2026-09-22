# Run/Session 重构方案集

本文档集把一次大范围重构拆成六个可以独立实施、验证、提交和回滚的工作包。设计基于当前单实例部署，不引入分布式 Worker、Lease、Outbox 或永久事件溯源。

本目录描述未来目标和实施顺序，不代表当前代码已经采用 Run 模型。真实进度只看 [STATUS.md](./STATUS.md)；当前架构只看 [`../ARCHITECTURE.md`](../ARCHITECTURE.md)。

## 目标

- `Session` 只表示长期对话容器。
- `Run` 表示一次用户请求触发的 Agent 执行。
- `Plan` 是 Run 内的执行快照，通过 `revision` 标识修订次数。
- PostgreSQL 保存业务状态，Redis Stream 只保存短期实时事件。
- 配置只有一个运行时模型和一个生效入口。
- 等待输入、业务失败、取消和系统错误具有不同语义。
- 保留现有 `model/service/agent/repository/external/handler/bootstrap` 工程骨架。

完整目标架构和决策见 [01-architecture.md](./01-architecture.md)。

## 工作包

| 阶段 | 文档 | 交付结果 | 依赖 |
|---|---|---|---|
| 1 | [02-settings-and-prompts.md](./02-settings-and-prompts.md) | Agent 配置单一来源，搜索数量真实生效，Prompt 可版本化 | 无 |
| 2 | [03-engine-and-context.md](./03-engine-and-context.md) | 显式 StepOutcome，ContextBuilder 替换伪记忆容量 | 阶段 1 |
| 3 | [04-run-domain-and-storage.md](./04-run-domain-and-storage.md) | Run 状态机、表和 Repository 完成，但暂不接生产请求 | 阶段 1 |
| 4 | [05-run-execution-cutover.md](./05-run-execution-cutover.md) | 后端生产写路径全部切到 RunService/RunExecutor | 阶段 2、3 |
| 5 | [06-api-ui-sse-cutover.md](./06-api-ui-sse-cutover.md) | UI 和公开 API 切到 Run 契约，旧路由删除 | 阶段 4 |
| 6 | [07-legacy-removal.md](./07-legacy-removal.md) | 删除 RedisStreamTask、Session 执行字段和死代码 | 阶段 5 |

阶段 2 和阶段 3 在阶段 1 完成后可以分别实施，但不要并行修改同一工作树。阶段 4 是唯一的业务语义切换点。

## 每个工作包的硬门禁

每个阶段都必须满足以下条件才能标记完成：

1. 在 `api/` 运行 `go test ./...` 通过。
2. 在 `api/` 运行 `go test -race ./internal/agent ./internal/service ./internal/repository -count=1` 通过。
3. 在 `api/` 运行 `go vet ./...` 通过。
4. 涉及 UI 的阶段在 `ui/` 运行 `npm run lint` 和 `npm run build` 通过。
5. 在项目根目录运行 `git diff --check` 通过。
6. 阶段内新增的契约测试通过。
7. 每个文档定义的检查点形成独立 Conventional Commit，可单独 `git revert`。
8. [STATUS.md](./STATUS.md) 已写入检查点 SHA、测试结果和下一阶段入口。

在受限沙箱中，如果现有 `httptest` 因禁止监听 `127.0.0.1:0` 失败，必须记录完整失败命令，并在允许本地监听的开发终端补跑。不能把环境失败记录为测试通过。

## 不保留两套长期业务语义

“可增量实施”不等于“双写”。本方案使用以下约束：

- 阶段 1、2 只替换内部组件，外部行为不变。
- 阶段 3 新增 Run 表和纯领域能力，但不接生产请求，不写生产 Run 数据。
- 阶段 4 一次性把所有生产创建、等待、恢复、完成、失败和取消写入切到 `RunService`。
- 阶段 4 后，旧 `/chat` 和 `/stop` 如暂时存在，只能委托 `RunService`，禁止更新 `sessions.status`。
- 阶段 5 删除旧路由，前端只读取 Run 状态。
- 阶段 6 删除旧 Task 基础设施和 Session 执行字段。

## 后续模型的恢复步骤

任何新会话开始时执行：

```bash
cd /Users/huanghao2/GolandProjects/study/imooc-mas/go-manus
git status --short --branch
git log -8 --oneline --decorate
sed -n '1,240p' docs/refactor-run/STATUS.md
```

然后：

1. 找到 `STATUS.md` 中第一个 `pending` 或 `in_progress` 阶段；若为 `in_progress`，从第一个未完成检查点继续。
2. 阅读本 `README`、[01-architecture.md](./01-architecture.md) 和该阶段文档。
3. 核对上一阶段记录的提交 SHA 是否仍在当前分支。
4. 运行上一阶段的“恢复基线命令”。
5. 只实施当前工作包，不提前修改后续阶段文件。

如果工作区有未提交代码，先用 `git diff` 判断它属于当前阶段还是用户已有修改。不得覆盖或重置用户修改。

## 文档与代码的更新规则

- 当前实现完成一个阶段后，更新 `STATUS.md`，不要改写已完成阶段的目标。
- 实际实现与方案有合理偏差时，在 `STATUS.md` 的“决策偏差”记录原因和最终接口。
- 如果发现设计前提错误，停止当前阶段，新增决策记录；不要静默引入另一套状态或配置模型。
- `docs/重构.md` 是历史原始提案，仅用于追溯；本目录是 Run 重构的实施依据。
