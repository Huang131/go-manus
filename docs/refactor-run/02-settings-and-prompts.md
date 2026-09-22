# 阶段 1：Settings 与 Prompt 单一来源

## 目标

完成 Agent 运行时配置和 Prompt 的治理，但保持现有 Session/RedisStreamTask 生产语义与 HTTP 契约不变。本阶段结束后，所有新建任务读取同一个不可变的 `AgentSettings` 快照，搜索结果数量真正由配置传入上游，Prompt 有稳定的文件来源和 hash。

## 非目标

- 不新增 Run 表或 Run API。
- 不改变任务创建、等待、取消和 SSE 路由。
- 不删除 `RedisStreamTask`、`Session.Status` 或旧配置路由。
- 不引入外部 Prompt 覆盖、配置中心或分布式热更新。

## 前置条件与恢复基线

- 先读取 `STATUS.md`，确认当前基线和阶段 1 已落地的前置治理；不得依赖本文创建时的历史提交号。
- 阅读 `README.md`、`01-architecture.md` 和 `STATUS.md`。
- 进入 `api/` 后先运行 `GOCACHE=/private/tmp/go-manus-gocache go test ./...`，记录基线结果。

## 文件清单

创建：

- `api/internal/agent/settings.go`：唯一运行时配置类型、默认值、校验、深拷贝。
- `api/internal/service/agent_settings_manager.go`：启动加载、持久化、原子读取和更新实现。
- `api/internal/agent/prompts/catalog.go`：`go:embed` PromptCatalog、名称校验和 SHA-256。
- `api/internal/agent/prompts/*.tmpl`：从现有 `prompts.go` 原样迁移的七个模板。
- `api/internal/agent/settings_test.go`、`api/internal/service/agent_settings_manager_test.go`、`api/internal/agent/prompts/catalog_test.go`。

修改：

- `api/internal/model/app_config.go`：删除 `model.AgentConfig`；AppConfigService 的 Agent settings 方法迁到 `agent.SettingsManager`，不在 model 保留迁移 DTO。
- `api/internal/agent/config.go`：删除 `AgentConfig` 和 Prompt 字段，保留 MCP/A2A 配置类型，避免出现第二份默认值。
- `api/internal/agent/base.go`、`planner_agent.go`、`react_agent.go`、`planner_react_flow.go`、`task_runner.go`、`service.go`：参数统一改为 `*AgentSettings`，Prompt 从 Catalog 获取。
- `api/internal/agent/tool_search.go`、`api/internal/external/search.go`：Search 接口增加显式 `limit` 参数；Tavily/Bocha/Google 请求使用该值。
- `api/internal/service/app_config_service.go`、`api/internal/handler/app_config_handler.go`：Agent 配置读写改为 SettingsManager 的唯一入口，更新后原子替换。
- `api/internal/bootstrap/app.go`：构造 SettingsManager，启动时加载默认值/数据库值，并把同一实例注入 AgentService 与配置 Handler。
- `api/internal/agent/*_test.go`：将旧 `DefaultAgentConfig` 引用替换为 `DefaultAgentSettings`。

删除（仅在引用已迁移后）：

- `api/internal/agent/prompts.go` 中七个 Prompt 常量。
- `api/internal/agent/config.go` 中 `AgentConfig` 及 `DefaultAgentConfig`。

## 关键契约

```go
type AgentSettings struct {
    MaxIterations    int `json:"max_iterations"`
    MaxRetries       int `json:"max_retries"`
    MaxPlanSteps     int `json:"max_plan_steps"`
    MaxSearchResults int `json:"max_search_results"`
}

// 位于 agent 包，由消费方声明；不依赖 repository。
type SettingsProvider interface {
    Current() AgentSettings
}

// 位于 service 包，具体实现依赖 AppConfigRepository。
type AgentSettingsManager interface {
    agent.SettingsProvider
    Load(context.Context) error
    Update(context.Context, AgentSettings) (AgentSettings, error)
}

type PromptName string
type PromptCatalog interface {
    Get(PromptName) (Prompt, error)
    Hash(PromptName) (string, error)
}

type SearchEngine interface {
    Invoke(context.Context, string, *string, int) (*model.ToolResult, error)
}
```

默认值必须完整且集中定义：`MaxIterations=10`、`MaxRetries=3`、`MaxPlanSteps=50`、`MaxSearchResults=10`。其中 10/3/50 分别继承当前 `agent.DefaultAgentConfig()` 的运行时默认值；搜索默认值 10 继承 Tavily `MaxResults: 10` 和 Bocha `Count: 10` 的现有行为，而不是来自旧配置类型的默认函数。`MaxPlanSteps` 必须真正用于规划结果校验，超过上限的计划返回可识别错误，不能继续保留一个不生效的配置字段。

所有整数配置都拒绝零值和负值，并设置显式上限。`MaxSearchResults` 统一限制为 1–10，因为 Google Custom Search 的 `num` 合法区间为 1–10；这样同一 Run 快照切换搜索 provider 时仍具有一致含义。配置更新先校验再持久化，持久化成功后才替换内存快照。

`AgentService` 只依赖 agent 包内的 `SettingsProvider`，创建任务时复制 `Current()` 返回值，后续配置更新不得改变已经创建的任务。Handler 依赖 service 包的 `AgentSettingsManager` 更新配置。Settings API 返回脱敏后的当前值，不能返回 LLM/MCP/A2A 密钥。

Prompt 文件只承载现有七个模板：system、planner system、create plan、update plan、ReAct system、execution、summary。文件内容迁移必须保持字节级语义；Catalog 计算原始文件内容的 SHA-256，Run 阶段会保存七个名称到 hash 的完整映射。

## 数据库与 API 变更

本阶段不新增表。继续使用 `app_configs(config_type='agent', config_key='default')`，service 包的 AgentSettingsManager 直接通过现有 AppConfigRepository 读写 JSON，并统一解码为 `agent.AgentSettings`。

当前数据库中的 `model.AgentConfig` 只有 `max_iterations`、`max_retries`、`max_search_results` 三个字段：启动时先建立完整默认值，再用旧 JSON 中存在且为正数的字段覆盖，最后补齐新增的 `max_plan_steps=50`；负数或超过上限的旧值直接报配置错误，不能静默修正。这样也能处理旧 Handler 保存部分字段后形成的零值。`agent.AgentConfig` 中的 `max_steps`、记忆容量和 Prompt 字段从未持久化，不能把它们描述成旧 DB JSON 的迁移字段。首次成功加载后立即回写规范化的完整 Settings，使数据库只保留新契约。

保留现有 `/api/app-config/agent` 路由作为阶段 5 前的适配入口，响应结构不变。阶段 1 不新增公开路由，避免同时出现两套 Settings API。

## 实施顺序与检查点

1. 检查点 1A：新增 PromptCatalog 和七个模板文件，切换所有生产引用并删除旧 Prompt 常量。运行 Agent 单元测试，提交 `refactor(api): move agent prompts to embedded catalog`。
2. 检查点 1B：新增 Settings、默认值、校验和 Manager；一次性切换 Agent/AppConfig/bootstrap 并删除旧 AgentConfig；同时扩展 SearchEngine limit 并补测试。运行阶段全量门禁，提交 `refactor(api): unify agent settings and search limits`。

1A、1B 均必须单独编译和通过相关单元测试。1B 未完成前运行时仍只使用旧 AgentConfig；1B 提交内完成唯一配置语义切换，不提交同时可用的两套生产配置。

## 针对性测试

- `settings_test.go`：默认值完整、边界校验、返回值修改不会污染 Manager。
- `settings_manager_test.go`：并发 `Current/Update` 无数据竞争；仓储失败时内存值不变。
- `prompts/catalog_test.go`：所有名称可读、hash 稳定、未知名称返回明确错误、模板占位符完整。
- `search_test.go`：Google 的 `num`、Tavily 的 `max_results`、Bocha 的 `count` 均等于传入 limit；1 和 10 可接受，0 和 11 被校验拒绝且不会发起上游请求。
- Agent 配置服务测试：旧 DB JSON 的正数字段保持原值，缺失/零值字段使用默认值，负数/越界值导致加载失败，`max_plan_steps` 补 50；规范化回写后只包含四个新字段；更新后新任务读取新快照。

## 阶段验证命令

```bash
cd api
GOCACHE=/private/tmp/go-manus-gocache go test ./internal/agent ./internal/external ./internal/service ./internal/handler ./internal/model -count=1
GOCACHE=/private/tmp/go-manus-gocache go test ./...
GOCACHE=/private/tmp/go-manus-gocache go test -race ./internal/agent ./internal/service ./internal/repository -count=1
GOCACHE=/private/tmp/go-manus-gocache go vet ./...
cd ..
git diff --check
```

## 提交与回滚

按 1B、1A 的逆序执行 `git revert <sha>`；若尚未提交，只恢复本阶段列出的文件，不得重置用户已有的 `llmcore` 修改。回滚任一检查点后必须重新运行相关测试。

## 完成定义

- 生产代码只存在一个 AgentSettings 类型和一个 Prompt 来源。
- 搜索 provider 请求体不再出现无法配置的固定 `10`。
- 配置更新具有校验、持久化后原子替换和任务级复制语义。
- 现有 Session/Task API 行为未改变，`go test ./...`、race、vet 均通过。
- `STATUS.md` 记录提交 SHA、验证结果和下一阶段入口 `03-engine-and-context.md`。

## 失败或中断恢复

从最近阶段提交恢复后，先运行：

```bash
git status --short --branch
git show --stat --oneline HEAD
GOCACHE=/private/tmp/go-manus-gocache go test ./...
```

若失败发生在删除旧类型之后，优先恢复最后一个通过测试的提交，不要临时保留两套配置类型；修复应继续沿用本阶段契约。
