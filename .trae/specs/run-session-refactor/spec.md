# Run/Session 架构重构规格

## 目标

将当前由 Session 同时承载会话元数据和一次执行状态的模型，重构为 Session 管理长期会话、Run 管理单次执行。PostgreSQL 保存业务事实，Redis Stream 只承担短期实时事件。

## 实施依据

- 总体方案：`docs/refactor-run/README.md`
- 当前事实：`docs/refactor-run/00-current-review-2026-10.md`
- 目标架构：`docs/refactor-run/01-architecture.md`
- 实施状态：`docs/refactor-run/STATUS.md`

本规格不复制各阶段的详细契约。发生冲突时，以 `docs/refactor-run` 中对应阶段文档和最新决策记录为准。

## 核心约束

1. 每个检查点必须能编译、能运行相关测试并可独立回滚。
2. 阶段 4 是后端唯一生产语义切换点，禁止 Session/Task 与 Run 双写。
3. `waiting_input` 必须依赖持久化 execution snapshot 恢复，不依赖存活 goroutine、ToolSet 或 SimpleMemory。
4. 同一 Session 同时只能存在一个 active Run，由数据库约束保证。
5. 当前不引入 Worker、Lease、Outbox、永久事件溯源、Plan/Step 独立表或通用 UnitOfWork；达到目标架构定义的触发条件后另行立项。
6. 项目未上线，不保留旧 API、旧字段或非法配置的长期兼容语义。
