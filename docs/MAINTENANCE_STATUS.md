# 维护状态

## 已完成

- 多模型配置支持草稿测试和已保存配置测试。
- OpenAI/Anthropic 适配器统一到 `llmcore` 协议。
- Planner 与 ReAct 的结构化响应、工具调用和文本流式边界已明确。
- Logger 不再关闭进程级 stdout/stderr；Sandbox 资源清理、文件权限和取消路径已有回归覆盖。
- API 测试已按 unit、component、HTTP integration 和 external smoke 分层；Chat 成功、SSE 顺序和取消终态进入 `make test-api` 门禁。
- MCP 配置 JSON 字段已统一为 `name`，后端 integration fixture 和前端消费者使用同一契约。

## 验证基线

- Go 定向单元测试和 `go vet` 应通过。
- UI `npm run build` 应通过；lint 中已有的未使用变量和 Next Image 警告不作为阻断错误。
- `make test-component` 和 `make test-api` 会启动并清理独立 PostgreSQL、Redis、MinIO 环境；Sandbox 测试使用单独的 `make test-sandbox`。

## 已知限制

- Planner 依赖模型结构化输出；模型不遵守 JSON 契约时应返回明确错误并保留可诊断信息。
- 当前 HTTP 集成测试尚未端到端覆盖携带 `Last-Event-ID` 的断线续读；Redis exclusive cursor 已有真实组件测试。
- Run/Session 重构仍处于方案阶段；当前生产语义仍是 Session/RedisStreamTask。
- 历史 review 中的性能数字、接口示例和 TODO 不代表当前实现，修改前必须以代码和测试为准。

## 维护规则

新增问题记录应补充复现、根因、修复、验证四项；完成后更新本文档，不再新增无日期的临时计划文件。
