# 文档导航

主目录只保留当前代码对应的长期文档：

- [ARCHITECTURE.md](./ARCHITECTURE.md)：当前 Agent、LLM、Planner、ReAct 和 SSE 架构。
- [LLM_INTEGRATION.md](./LLM_INTEGRATION.md)：Provider、SenseNova、模型测试和 Planner JSON 契约。
- [DEPLOYMENT.md](./DEPLOYMENT.md)：Docker、服务依赖和部署方式。
- [MAINTENANCE_STATUS.md](./MAINTENANCE_STATUS.md)：验证基线、已知限制和维护规则。
- [API_TEST_PLAN.md](./API_TEST_PLAN.md)：API 测试分层、已完成改造和剩余高收益缺口。
- [API_TEST_RULES.md](./API_TEST_RULES.md)：测试规则入口，正文位于 `.trae/rules/api-testing.md`。
- [INCIDENT_LOGGER.md](./INCIDENT_LOGGER.md)：Logger stdout 关闭事故的根因与修复。
- [INTERVIEW_GUIDE.md](./INTERVIEW_GUIDE.md)：面试材料，内容以当前代码为准。
- [CODE_INTELLIGENCE.md](./CODE_INTELLIGENCE.md)：Graphify/GitNexus 等代码导航工具。

`archive/` 保存历史 review、阶段性设计、旧对比报告和一次性排查记录。归档文档只用于追溯，不作为当前实现依据。

`refactor-run/` 是已完成的 Run/Session 重构实施记录、契约和后续维护边界。当前生产语义以 [`refactor-run/STATUS.md`](./refactor-run/STATUS.md) 及 `ARCHITECTURE.md` 为准；其中阶段文档保留实施时的检查点和回滚依据。`重构.md` 是更早的原始提案，仅用于追溯，不是当前设计基线。
