# Run/Session 重构实施状态

该文件是跨会话恢复入口。实施模型必须在每个阶段提交后更新它。

## 当前状态

| 阶段 | 状态 | 提交 SHA | 验证结果 |
|---|---|---|---|
| 1. Settings 与 Prompt | in_progress | `1d9d5cd`, `236972f` | 配置加载和搜索 limit 已接通；PromptCatalog、SettingsManager 未完成 |
| 2. Engine 与 Context | in_progress | `fa64e78` | 基础 ContextBuilder 已接入；模型预算来源、StepOutcome 未完成 |
| 3. Run 领域与存储 | pending | - | - |
| 4. 后端执行切换 | pending | - | - |
| 5. API/UI/SSE 切换 | pending | - | - |
| 6. 遗留删除 | pending | - | - |

允许的状态只有：`pending`、`in_progress`、`complete`、`blocked`。

## 当前入口

- 下一入口：阶段 1 检查点 1A（PromptCatalog）
- 方案文档：[02-settings-and-prompts.md](./02-settings-and-prompts.md)
- 状态核对基线：`77496cc`
- 生产业务语义：现有 Session/RedisStreamTask；Run 模型、表、Repository、API 和 UI 均未开始切换

## 阶段内检查点

| 检查点 | 状态 | 提交 SHA | 独立验证 |
|---|---|---|---|
| 1A. PromptCatalog 切换 | pending | - | - |
| 1B. SettingsManager 与搜索 limit 切换 | in_progress | `1d9d5cd`, `236972f` | 搜索 limit、启动加载和无效字段清理已完成；统一 SettingsManager、任务快照和 `max_plan_steps` 尚未完成 |
| 2A. ContextBuilder 切换 | in_progress | `fa64e78` | Builder 已进入生产 LLM 请求路径，伪记忆容量已删除；模型画像预算、输出预留和完整裁剪策略尚未完成 |
| 2B. StepOutcome 切换 | pending | - | - |
| 3A. Run 模型与 migration | pending | - | - |
| 3B. Run/Message Repository | pending | - | - |
| 4A. 新执行组件（不接生产） | pending | - | - |
| 4B. 生产装配原子切换 | pending | - | - |
| 4C. 生命周期并发加固 | pending | - | - |
| 5A. Run API 与 SSE | pending | - | - |
| 5B1. UI Run client 与 Hook（未启用） | pending | - | - |
| 5B2. UI 切换与旧路由删除 | pending | - | - |
| 6A. 删除 Task 基础设施 | pending | - | - |
| 6B. 删除 Session 执行字段 | pending | - | - |
| 6C. 收紧 model 依赖 | pending | - | - |

检查点允许的状态同样只有 `pending`、`in_progress`、`complete`、`blocked`。每个检查点提交时必须保持 API 可编译、相关单元测试通过，并可单独回滚；阶段门禁通过前阶段状态不得改为 complete。

## 最近一次验证

- `GOCACHE=/private/tmp/go-manus-gocache make test-api`：通过，包含 Chat 成功、SSE 顺序、MCP 配置和取消终态契约。
- Run/Session 方案检查：生产源码中无 `RunStatus`、`Run` 表、Run Repository 或 Run API。
- 当前配置治理和基础 ContextBuilder 的提交均已在当前分支，阶段 1/2 的未完成项仍可按各自检查点独立实施。

## 决策偏差

- 本方案形成前，配置加载/搜索 limit（`1d9d5cd`、`236972f`）和基础 ContextBuilder（`fa64e78`）已独立落地。当前 Builder 仍使用固定默认预算，未完成阶段 2A 设计中的模型画像预算与输出预留；状态表按实际能力回填，不把这些提交追认成完整阶段交付。
- 当前取消生命周期已由 `2535c3c`、`e45c48c` 加固，并由 `a0f1965` 覆盖 HTTP 契约；这仍属于旧 Session/RedisStreamTask 语义，不等同于检查点 4C 的 Run 生命周期加固。

## 阻塞记录

尚无。

## 更新模板

完成阶段后追加：

```text
阶段：
状态：complete
提交：<sha>
验证：<命令及结果>
生产语义：<当前唯一写路径>
删除内容：<已删除接口/字段>
决策偏差：<无或具体说明>
下一入口：<下一阶段文档和首个任务>
```
