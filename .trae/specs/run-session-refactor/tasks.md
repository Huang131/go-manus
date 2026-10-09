# Run/Session 架构重构任务

任务状态以 `docs/refactor-run/STATUS.md` 为唯一恢复入口。

1. [完成] Settings 单一类型、严格校验与配置 migration。
2. [完成] PromptCatalog、版本和 hash。
3. [完成] StepOutcome 唯一信号并删除旧等待表达。
4. [完成] ContextPolicy：统一 context window、输出预留、工具 schema 预算和完整工具链裁剪。
5. [完成] MCP 动态路由、初始化与错误契约收敛。
6. [完成] Run、Message、execution snapshot、状态机、Repository 和纯 RunService（3A/3B/3C 完成，仍未接生产）。
7. [进行中] RunExecutor 生命周期、取消收敛、ToolSet 释放和可观测性。
8. 后端生产写路径原子切换。
9. Run API、SSE、Last-Event-ID 和 UI 切换。
10. 删除 RedisStreamTask、Session 执行字段与旧投影。

每次只实施 `STATUS.md` 中第一个 pending 或 in_progress 检查点，不提前修改后续阶段。
