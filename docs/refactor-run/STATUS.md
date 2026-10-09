# Run/Session 重构实施状态

本文件是跨会话恢复入口。状态只反映当前代码，不把目标设计写成已完成。

## 当前状态

| 阶段 | 状态 | 提交/依据 | 说明 |
|---|---|---|---|
| 0. 事实基线与方案校准 | complete | 文档工作区；代码基线 `4c3d39d` | 已核对当前 Session/Task 链、配置、Engine、ToolSet；文档完成后再记录独立文档提交 |
| 1. Settings 与 Prompt | in_progress | `4df77fe`、`e9dcba2`、`9548b84`、`dd5c2cf` | Settings、严格校验、migration 和 UI 已完成；PromptCatalog/hash 待实现 |
| 2. Engine 与 Context | pending | - | Flow 已拆状态处理；Outcome 和 ContextPolicy 未完成 |
| 3. Run 领域与存储 | pending | - | 当前源码没有 Run 类型、表或 Repository；必须先评审契约 |
| 4. 执行生产切换 | pending | - | 唯一后端生产语义切换点；Session 路由只剩薄适配 |
| 5. API/UI/SSE | pending | - | 唯一公开契约切换点；完成后删除旧路由 |
| 6. 遗留删除 | pending | - | 依赖阶段 4/5 的引用扫描和回归 |

允许状态：`pending`、`in_progress`、`complete`、`blocked`。

## 当前生产语义

```text
SessionHandler -> AgentService -> RedisStreamTask -> AgentTaskRunner
              -> PlannerReActFlow -> LLM + ToolSet
```

当前不存在 Run 生产写入。`ToolRegistry`、MCP/A2A ToolSet 引用生命周期已完成，不属于本方案待办。

## 最近验证基线

实施新阶段前，在 `api/` 重跑：

```bash
go test ./...
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1
go vet ./...
```

受限环境的回环端口失败必须单独记录，不能改写为通过。

## 检查点

| 检查点 | 状态 | 入口 |
|---|---|---|
| 1A Settings 单一类型、配置 migration 与校验 | complete | `4df77fe`、`e9dcba2`、`9548b84`、`dd5c2cf` |
| 1B PromptCatalog 与 hash | pending | `02-settings-and-prompts.md` |
| 2A StepOutcome 唯一信号并删除旧等待表达 | pending | `03-engine-and-context.md` |
| 2B ContextPolicy | pending | `03-engine-and-context.md` |
| 2C MCP 动态路由、初始化与错误契约 | pending | `03-engine-and-context.md` |
| 3A Run、执行快照与等待恢复契约 | pending | `04-run-domain-and-storage.md` |
| 3B Run/Message Repository | pending | `04-run-domain-and-storage.md` |
| 4A RunExecutor 生命周期与观测，未接生产 | pending | `05-run-execution-cutover.md` |
| 4B 唯一后端生产切换 | pending | `05-run-execution-cutover.md` |
| 5A Run API/SSE | pending | `06-api-ui-sse-cutover.md` |
| 5B UI 切换 | pending | `06-api-ui-sse-cutover.md` |
| 5C 删除旧路由 | pending | `06-api-ui-sse-cutover.md` |
| 6A 删除 Task 基础设施 | pending | `07-legacy-removal.md` |
| 6B 删除 Session 执行字段 | pending | `07-legacy-removal.md` |
| 6C 收紧 model 依赖 | pending | `07-legacy-removal.md` |

## 阶段验收归属

| 验收项 | 归属阶段 |
|---|---|
| Settings 单一真相、范围校验、Prompt hash | 1 |
| 等待输入不再依赖错误字符串、上下文裁剪可替换 | 2 |
| Run 状态、等待恢复、幂等、单活跃约束、消息存储 | 3 |
| 所有生产执行只写 Run/Message，取消收敛后 ToolSet 必然释放 | 4 |
| Run API、Last-Event-ID、UI 只消费 Run 语义 | 5 |
| 旧 Task、Session 执行字段和旧路由物理删除 | 6 |

## 决策记录

- 首期不引入 Worker、Lease、Outbox、永久事件溯源、Plan/Step 独立表或通用 UnitOfWork；达到 `01-architecture.md` 的演进触发条件后单独立项。
- `PlannerReActFlow.Invoke` 已拆分，后续只调整结果和状态边界。
- `StepOutcome.Kind` 是目标唯一等待控制信号；阶段 2 必须删除旧 error/boolean 双表达。Engine 内部完成有限重试，Flow/RunExecutor 不消费 `retryable_failure`，也不重复执行同一次调用。
- `waiting_input` 依靠 execution snapshot、已完成步骤摘要和问题/回答关联恢复，不依赖旧 goroutine 或 `SimpleMemory`。
- 一次客户端提交只使用一个幂等键；创建 Run 与初始消息共享该键，后续输入在 Run 内去重。
- 阶段 4 不回填 `sessions.events` 或旧 Task 历史，从切换点开始写 Run/Message。
- Run input 只接受 `waiting_input`；`cancelling`、其他活跃状态和终态分别返回稳定的 409 业务错误，重复幂等输入返回原结果。
- 取消先条件写入 `cancelling`，再发出 cancel signal；Engine 退出并释放 ToolSet、持久化 `cancelled` 后才注销控制句柄。Cancel 发现句柄不存在或终态写入失败时，必须通过幂等 `ReconcileCancelling` 在当前进程或启动恢复中收敛。
- `MaxPlanSteps` 不作为伪配置加入；必须先有真实消费逻辑。
- `sessions.memories` 和 `Session.Memories` 当前不存在，不在方案中虚构。
- ToolRegistry 三份映射、MCP/A2A retired 无界释放问题已在工具重构提交中解决，不重复规划。
- MCP function name 碰撞、初始化吞错和基础设施错误分类仍是待办，归入 2C；这不否定已经完成的 ToolRegistry/ToolSet 生命周期重构。
- Agent Settings 的持久化记录必须完整合法：记录缺失才使用 `10/3/10` 默认值；非法更新返回 400，非法存量记录由 `005_normalize_agent_settings.sql` 一次性修正，启动读取不再静默 clamp。
- Task/Run 创建必须只读取一次 Settings；Runner 与 ToolSet 必须使用同一快照，热更新只影响后续创建的执行。

## 最近完成：1A Settings

验证结果：

```text
go test ./...                                                        PASS
go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1  PASS
go vet ./...                                                        PASS
npm run lint                                                        PASS（仅既有 warning）
npm run build                                                       PASS
```

独立测试环境中额外验证：

- Agent 配置合法值 HTTP 生命周期通过。
- 非法配置更新稳定返回 HTTP 400。
- `005_normalize_agent_settings.sql` 在真实 PostgreSQL 连续执行两次后结果一致。

下一入口：检查点 1B `PromptCatalog` 与稳定 hash，不修改 Session/Task API。

## 恢复流程

```bash
cd /Users/huanghao2/GolandProjects/study/imooc-mas/go-manus
git status --short --branch
git log -8 --oneline --decorate
sed -n '1,220p' docs/refactor-run/STATUS.md
```

读取第一个 `pending` 或 `in_progress` 检查点，确认上一提交存在，运行该阶段恢复基线后再修改。任何中断都从最近一个通过门禁的提交继续。
