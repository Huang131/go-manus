# Run/Session 架构重构任务

任务状态以 `docs/refactor-run/STATUS.md` 为唯一恢复入口。

1. [完成] Settings 单一类型、严格校验与配置 migration。
2. [完成] PromptCatalog、版本和 hash。
3. [完成] StepOutcome 唯一信号并删除旧等待表达。
4. ContextPolicy 与 MCP 契约收敛。
5. Run、Message、execution snapshot、状态机和 Repository。
6. RunExecutor 生命周期、取消收敛、ToolSet 释放和可观测性。
7. 后端生产写路径原子切换。
8. Run API、SSE、Last-Event-ID 和 UI 切换。
9. 删除 RedisStreamTask、Session 执行字段与旧投影。

每次只实施 `STATUS.md` 中第一个 pending 或 in_progress 检查点，不提前修改后续阶段。
