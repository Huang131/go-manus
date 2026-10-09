# 阶段 1：Settings 快照与 Prompt 可追踪性

> 当前已存在：启动加载 Agent 配置、Handler 传递 `max_search_results`、搜索 provider 使用 limit。此阶段不重复实现这些链路。

## 目标

把 `model.AgentConfig` 和 `agent.AgentConfig` 收敛为一个运行时 Settings 类型，明确校验边界，并为未来 Run 保存创建时快照。Prompt 从散落常量收敛为可枚举、可 hash 的目录。

## 范围

### Settings

在 `internal/settings` 定义唯一 `AgentSettings`、默认值和纯校验逻辑。该包不依赖 Handler、Service、Repository 或 Agent；`bootstrap`、配置服务和 Agent 只依赖它。数据库 DTO 不再定义第二套运行时配置语义。

```go
type AgentSettings struct {
    MaxIterations    int `json:"max_iterations"`
    MaxRetries       int `json:"max_retries"`
    MaxSearchResults int `json:"max_search_results"`
}
```

不要加入当前没有消费方的 `MaxPlanSteps`。如果未来规划器确实需要上限，先补消费逻辑和行为测试，再增加字段。

默认值沿用当前行为：`10/3/10`。默认值只用于“配置记录不存在”，不能用于覆盖已经存在但非法的字段。`MaxSearchResults` 还要按 provider 约束校验，Google 等 provider 的上限不能超过其协议允许值。

SettingsManager 负责加载、校验、保存和原子替换；创建 Run 时复制值，不持有可变指针。配置更新不改变已经运行的 Run。

### 校验行为变更

当前 `NormalizeAgentConfig` 会把非正数静默替换为默认值。本阶段把它改为严格校验，这是有意的破坏性行为：

- Handler 更新：在写数据库前校验，非法字段返回明确 4xx，不允许先保存再 reload。
- 启动加载：记录不存在时使用完整默认值；记录存在但任一字段非法时启动失败，并输出不包含密钥的字段级错误。
- migration/开发数据：通过一次性数据库 migration 扫描并修正或删除非法 Agent 配置记录，使 migration 完成后的所有存量记录都能通过新校验。项目未上线，不提供人工脚本兜底，也不保留 clamp 兼容路径；migration 必须可在测试数据库独立执行和回滚。
- 运行中更新：只有完整配置通过校验后才能原子替换；失败不改变当前有效配置。

### Prompt

将 Prompt 组织为 `PromptCatalog`，每个模板有稳定名称、目录版本和 SHA-256 hash。首期可以继续使用 `go:embed` 或包内模板，不开放外部目录覆盖。Run 保存目录版本、hash 和应用构建版本，用于定位实际模板；不保存每次渲染后的大文本，也不宣称可以脱离当时消息与工具上下文完整重放。

## 不做

- 不修改 Session/Task API。
- 不把部署级 LLM 配置迁移到 Agent Settings；`cfg.LLM` 仍负责 fallback、seed、timeout 和开关。
- 不增加没有消费方的配置字段。

## 检查点

1. `internal/settings.AgentSettings` 替换两个旧配置类型，搜索 limit 的有效值行为不变。
2. Handler 拒绝非法更新；启动对缺失配置使用默认值，对已存在的非法配置 fail fast。
3. 一次性配置 migration 能识别并处理零值、负值和超过 provider 上限的存量配置；migration 前后数据与回滚行为有集成测试。
4. PromptCatalog 能返回模板和 hash；同一内容 hash 稳定。
5. 增加配置快照转换测试，但暂不写 Run 表。
6. 删除 `NormalizeAgentConfig` 的静默 clamp 路径，不保留第二套兼容语义。

建议提交：`refactor(agent): unify settings and prompt catalog`。

## 验证与回滚

运行 README 的通用门禁，并额外运行 Agent 配置、搜索 provider、PromptCatalog 行为测试。代码提交与数据 migration 分开：代码提交可以独立回滚；数据 migration 必须幂等，并在测试数据库验证重复执行。只有能够无损恢复原始配置值时才提供 down migration，否则回滚策略是先恢复合法配置数据，再回滚代码，不能重新启用静默 clamp 或第二份长期配置语义。
