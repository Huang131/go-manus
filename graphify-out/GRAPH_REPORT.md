# Graph Report - go-manus  (2026-10-09)

## Corpus Check
- 373 files · ~915,725 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 4231 nodes · 10428 edges · 238 communities (210 shown, 28 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 968 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `154819b8`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- index.tsx
- cn
- context.Context
- BrowserService
- index.ts
- tool-preview-panel.tsx
- queryer
- MCPSetting.tsx
- json_parser_repair.go
- testing.T
- 重构.md
- mockMQWrapper
- PostgresSessionRepository
- Response
- session-item.tsx
- Docker 构建优化实践笔记
- StdioMCPClient
- openai_llm.go
- LLMModel
- File
- task_redis.go
- newSandboxClient
- NewToolProvider
- SessionHandler
- logger 包 os.Stdout 误关闭 Bug 深度解析
- tools_test.go
- lib/utils.ts
- Warn
- endpoints/file.py
- compilerOptions
- tool_event_flow_test.go
- CODE_REVIEW_2026-09-11.md
- App
- BadRequestException
- session-detail-view.tsx
- 阶段 5：唯一 Run API、UI 与 SSE 契约
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- ModelProfile
- AnthropicClient
- github.com/gin-gonic/gin.Context
- NewAppConfigHandler
- llm.go
- AgentService
- RunMessage
- NewSessionHandler
- BadRequest
- NewRedisStreamTask
- devDependencies
- 代码智能数据使用说明
- Message
- 阶段 1：Settings 快照与 Prompt 可追踪性
- NewSessionService
- fallback_test.go
- dependencies
- components.json
- createSessionForTest
- Sandbox
- refactor-run/README.md
- browser_cli.js
- Config
- 部署指南
- 阶段 6：删除旧执行基础设施
- routed_llm_test.go
- testLLMModelRepo
- NewRoutedLLM
- ShellService
- types.go
- defaultAgentSettings
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- anthropic_llm_test.go
- 4.3 11 个 Bug 拆解
- event.go
- logger_test.go
- LLMRuntimeConfig
- PlannerReActFlow
- Refactoring with GitNexus
- mcp_client_test.go
- go-manus - 通用 AI Agent 系统
- sync.Once
- OSS
- MergeDeltas
- .handleToolCall
- chat_lifecycle_test.go
- BaseAgent
- PromptCatalog
- API 测试方案
- MCPTool
- 集成测试
- AppConfigHandler
- NewProviderError
- ServiceRegressionTests
- ApiEndpointTests
- Manus 沙箱服务
- Manus 前端 UI
- 阶段 3：Run 领域与存储（未接生产）
- Commands
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- 目标架构：Session、Run 与进程内 Engine
- AppConfig
- logger.go
- ToolResult
- go-manus 面试指南
- Run/Session 重构实施状态
- docs/README.md
- Python vs Go 版本文件对比报告
- .initRoutes
- NewBrowserTool
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- vnc-overlay.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- run
- testSessionRepo
- 阶段 4：Run 执行装配与唯一后端切换
- testRunRepository
- 3.3 Repository 层架构设计
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- stubLLM
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 七、详细模块对比
- RuntimeA2AConfig
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- NewToolError
- 技术方案文档编写计划
- 3.4 集成测试
- integration_env_test.go
- a2a_test.go
- NewSessionRuntime
- 6. 分阶段实施
- ParseWithContext
- FileHandler
- API 当前事实基线（2026-10）
- VNCProxy
- sandbox_external_test.go
- dynamicTestTool
- Postgres
- sensenova_integration_test.go
- 3. 核心知识点详解
- 5. 注意事项和踩坑点
- Redis
- 3.2 PostgreSQL INTERVAL 与参数化查询
- 3.5 Session 与 Agent 架构
- 4. 可能的面试追问
- postcss.config.mjs
- Err
- sandbox
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- Impact Analysis with GitNexus
- 7.4 存储层详细对比
- remark-gfm
- 阶段 2：Engine、Outcome 与 ContextPolicy
- test-env-down.sh
- test-env-up.sh
- test_api_endpoints.py
- ._iter_file_lines
- MCPConfig
- github.com/Huang131/go-manus/api
- API 测试规则
- 六、持久化模型
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- SearchEngine
- next.config.ts
- BaseModel
- AgentSettings
- 7.5 外部依赖详细对比
- .loadOne
- testFileRepo
- UnixStreamHTTPConnection
- next-themes
- Run/Session 架构重构规格
- 三、服务层
- sandbox/package.json
- 六、差异汇总
- 八、改进建议
- @radix-ui/react-avatar
- archive/README.md
- mcp_client.go
- LLMRequest
- .uploadFile
- response_test.go
- Run/Session 架构重构验收清单
- NewMockFileRepository
- tasks.md
- @radix-ui/react-scroll-area
- @radix-ui/react-slot
- @radix-ui/react-switch
- a2a_config.go
- time.Duration
- react-dom
- react-markdown
- NewAppConfigService
- MCPClientManager
- 维护状态
- 核心概念
- 学习路径
- Plan
- mockLLMForTest
- openai_llm_test.go
- task_redis_test.go
- MCPToolInfo
- shutdownCurrent
- mockFailingTool
- sonner
- 10. 故障排查
- stubMCPClient
- RedisStreamTask
- 当前架构总览
- validConfig
- package.json
- eslint.config.mjs
- @novnc/novnc
- @radix-ui/react-label
- tailwind-merge

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 95 edges
3. `ServiceRegressionTests` - 58 edges
4. `Err()` - 56 edges
5. `ShellService` - 53 edges
6. `File` - 49 edges
7. `queryer` - 48 edges
8. `RedisStreamTask` - 46 edges
9. `FileService` - 45 edges
10. `createSessionForTest()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `BaseAgent` --references--> `Memory`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `BaseAgent` --references--> `JSONParser`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/jsonx/json_parser.go
- `NewBaseAgent()` --calls--> `NewContextBuilder()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/context_builder.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewPlannerAgent()` --calls--> `NewBaseAgent()`  [INFERRED]
  api/internal/agent/planner_agent.go → api/internal/agent/base.go

## Import Cycles
- None detected.

## Communities (238 total, 28 thin omitted)

### Community 0 - "index.tsx"
Cohesion: 0.13
Nodes (19): A2aTool(), A2aToolProps, BashTool(), BashToolProps, BrowserTool(), BrowserToolProps, DefaultTool(), DefaultToolProps (+11 more)

### Community 1 - "cn"
Cohesion: 0.06
Nodes (55): metadata, GlobalHeader(), LeftPanel(), Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup(), AvatarGroupCount() (+47 more)

### Community 2 - "context.Context"
Cohesion: 0.06
Nodes (13): chatContextSessionRepository, mockSessionRepo, successfulStopSessionRepository, Event, Session, SessionStatus, unixPointer(), mockMessageQueue (+5 more)

### Community 3 - "BrowserService"
Cohesion: 0.05
Nodes (62): _budget_ms(), click(), console_exec(), console_view(), input_text(), navigate(), press_key(), post (+54 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (50): SessionHeaderProps, configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout(), get() (+42 more)

### Community 5 - "tool-preview-panel.tsx"
Cohesion: 0.14
Nodes (22): A2APreview(), ArtifactRef, BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent(), getToolDescription(), MCPPreview() (+14 more)

### Community 6 - "queryer"
Cohesion: 0.08
Nodes (16): pgx.Tx, NewFileRepository(), scanFile(), encodeModelJSONColumns(), pgx.Tx, modelJSON(), scanLLMModel(), collectRows() (+8 more)

### Community 7 - "MCPSetting.tsx"
Cohesion: 0.09
Nodes (52): DeleteSessionDialogProps, ManusSettings(), SETTING_MENUS, SettingTab, emptyDraft, Mode, ModelConfigManager(), RenameSessionDialogProps (+44 more)

### Community 8 - "json_parser_repair.go"
Cohesion: 0.13
Nodes (17): formatRepairTypes(), JSONParser, NewDefaultJSONParser(), completeTruncatedJSON(), Repair, isTruncatedJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks() (+9 more)

### Community 9 - "testing.T"
Cohesion: 0.04
Nodes (75): TestFlowStatus_ToSessionStatus(), TestFlowStatus_Values(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_InvokeContext(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters() (+67 more)

### Community 10 - "重构.md"
Cohesion: 0.11
Nodes (17): Plan 和 Step, Prompt 配置, Run, Session, 一、总体架构, 七、事件契约, 三、Agent Engine, 九、API 契约 (+9 more)

### Community 11 - "mockMQWrapper"
Cohesion: 0.17
Nodes (5): batchTaskOutputMQ, mockMQWrapper, StreamMessage, BatchMessageQueue, MessageQueue

### Community 12 - "PostgresSessionRepository"
Cohesion: 0.13
Nodes (8): stopSessionRepository, extractSessionMessage(), pgx.Tx, SessionRepository, marshalSessionEvents(), NewSessionRepository(), TestExtractSessionMessageProjectsCompletedAssistantMessage(), PostgresSessionRepository

### Community 13 - "Response"
Cohesion: 0.07
Nodes (41): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+33 more)

### Community 14 - "session-item.tsx"
Cohesion: 0.07
Nodes (27): DeleteSessionDialog(), RenameSessionDialog(), SessionItem(), SessionItemProps, SessionList(), DropdownMenu(), DropdownMenuCheckboxItem(), DropdownMenuContent() (+19 more)

### Community 15 - "Docker 构建优化实践笔记"
Cohesion: 0.05
Nodes (38): 1.1 问题分析, 1.2 解决方案, 1.3 验证结果, 2.1 问题分析, 2.2 解决方案, 2.3 验证结果, 3.1 问题分析, 3.2.1 使用 uv 替代 pip (+30 more)

### Community 16 - "StdioMCPClient"
Cohesion: 0.19
Nodes (8): bufio.Reader, io.WriteCloser, os/exec.Cmd, sync/atomic.Bool, MCPError, MCPRequest, MCPResponse, StdioMCPClient

### Community 17 - "openai_llm.go"
Cohesion: 0.11
Nodes (24): streamContext(), classifyHTTPError(), estimateCostUSD(), LLMRequest, normalizeOpenAIFinishReason(), reasoningTokens(), sendStreamDelta(), toOpenAIMessages() (+16 more)

### Community 18 - "LLMModel"
Cohesion: 0.05
Nodes (54): NewLLMModelHandler(), TestLLMModelHandler_CreateRejectsMalformedJSON(), DefaultCapabilities(), CostPolicy, LLMModel, LLMModelTestResponse, ModelCapabilities, ReasoningMode (+46 more)

### Community 19 - "File"
Cohesion: 0.08
Nodes (10): attachmentFileRepository, generatedFileRepository, File, FileRepository, FileService, FileStorage, io.ReadCloser, DefaultFileService (+2 more)

### Community 20 - "task_redis.go"
Cohesion: 0.09
Nodes (20): Stream, Task, TaskRegistryInterface, TaskRunner, TaskStream, NewTaskStream(), parseTaskOutputMessages(), ReadTaskOutput() (+12 more)

### Community 21 - "newSandboxClient"
Cohesion: 0.20
Nodes (16): SandboxClient, NewBrowserClient(), NewSandboxClient(), SandboxClient, newSandboxClient(), TestExecCommandDecodesEnvelope(), TestFindFilesSendsGlobPattern(), TestHealthCheckReportsFailure() (+8 more)

### Community 22 - "NewToolProvider"
Cohesion: 0.21
Nodes (14): NewMCPTool(), initializedMCPTool(), TestMCPToolCleanupDropsLoadedTools(), TestMCPToolInitializeCleansUpManagerFailure(), TestMCPToolInitializeRejectsDuplicateFunctionNames(), TestMCPToolInvokeWithNameRejectsUnknownFunction(), TestMCPToolInvokeWithNameSeparatesTransportAndRemoteErrors(), TestMCPToolRegistersDynamicFunctions() (+6 more)

### Community 23 - "SessionHandler"
Cohesion: 0.21
Nodes (11): SessionHandler, mergeEventMetadata(), newSSEContext(), parseChatRequest(), setSSEHeaders(), TestMergeEventMetadata(), TestMergeEventMetadata_NullPayloadReturnsOriginalData(), TestSetSSEHeaders() (+3 more)

### Community 24 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 25 - "tools_test.go"
Cohesion: 0.12
Nodes (25): NewA2ATool(), sortedAgentIDs(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description(), TestA2ATool_Initialize(), TestA2ATool_InitializeSkipsDuplicateAgentNames(), TestA2ATool_Invoke_CallAgentRejectsInvalidRequiredTypes() (+17 more)

### Community 26 - "lib/utils.ts"
Cohesion: 0.11
Nodes (23): AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatInput, ChatInputProps, ChatInputRef, FilePreviewPanel(), FilePreviewPanelProps (+15 more)

### Community 27 - "Warn"
Cohesion: 0.20
Nodes (9): cleanupToolResources(), newToolSetState(), Warn(), a2aResource, A2ATool, mcpResource, toolCleanup, ToolProvider (+1 more)

### Community 28 - "endpoints/file.py"
Cohesion: 0.13
Nodes (35): api_route, FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get (+27 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "tool_event_flow_test.go"
Cohesion: 0.13
Nodes (11): artifactTool, inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, TestRedisStreamTask_GetOutputReadsBufferedEventsWhenStartIDEmpty(), newInMemoryMessageQueue(), parseStreamSeq() (+3 more)

### Community 31 - "CODE_REVIEW_2026-09-11.md"
Cohesion: 0.09
Nodes (22): A. 流式链路复查（commit 4892225）, api, B. service 层专项（新发现）, C. 昨日 backlog 复核（HEAD b1f3827 仍开放）, D. 修复批次, E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）, sandbox, ui (+14 more)

### Community 32 - "App"
Cohesion: 0.17
Nodes (14): Build(), BuildWithFactories(), defaultFactories(), DefaultOptions(), App, loadRuntimeToolConfigs(), newA2AConfig(), newMCPConfig() (+6 more)

### Community 33 - "BadRequestException"
Cohesion: 0.20
Nodes (24): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+16 more)

### Community 34 - "session-detail-view.tsx"
Cohesion: 0.07
Nodes (41): PageProps, ChatMessage(), ChatMessageProps, StepBlock(), ToolRow(), ManusIcon(), components, headingClasses (+33 more)

### Community 35 - "阶段 5：唯一 Run API、UI 与 SSE 契约"
Cohesion: 0.17
Nodes (11): 5A：Run API/SSE, 5B：UI 切换, 5C：删除旧路由, API 契约, SSE 契约, UI 切换, 代码范围, 回滚边界 (+3 more)

### Community 36 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 37 - "ModelProfile"
Cohesion: 0.17
Nodes (13): withProfileID(), BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), TestBuildRuntimeConfigFromModel(), TestProtocolFromProvider(), ExtraParam, ModelProfile (+5 more)

### Community 38 - "AnthropicClient"
Cohesion: 0.16
Nodes (13): LLMRequest, mustMarshalString(), normalizeAnthropicStopReason(), parseJSONMap(), textOf(), AnthropicClient, AnthropicContent, AnthropicImageSource (+5 more)

### Community 39 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.22
Nodes (7): LLMModelHandler, NewLLMModelResponse(), FromError(), Success(), SuccessWithMsg(), SuccessWithTotal(), github.com/gin-gonic/gin.Context

### Community 40 - "NewAppConfigHandler"
Cohesion: 0.43
Nodes (5): NewAppConfigHandler(), TestGetAgentSettingsReturnsDefaultsWhenRecordIsMissing(), TestUpdateAgentSettingsReloadsSearchLimit(), TestUpdateAgentSettingsReturnsBadRequestForInvalidSettings(), appConfigServiceStub

### Community 41 - "llm.go"
Cohesion: 0.15
Nodes (12): defaultLLMClientFactory(), TestDynamicLLM_UsesFactory(), LLMClientFactory, NewDynamicLLMWithFactory(), NewLLMClient(), NewOpenAIClient(), runtimeConfigToOpenAIClientConfig(), DynamicLLM (+4 more)

### Community 42 - "AgentService"
Cohesion: 0.20
Nodes (6): Capabilities, Repositories, A2AConfig, AgentService, NewAgentService(), nonNilEvents()

### Community 43 - "RunMessage"
Cohesion: 0.10
Nodes (24): Run, RunExecutionSnapshot, RunMessage, RunStatus, pgx.Tx, RunRepository, isUniqueConstraint(), NewRunRepository() (+16 more)

### Community 44 - "NewSessionHandler"
Cohesion: 0.36
Nodes (11): NewSessionHandler(), decodeHandlerResponse(), setupRouter(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create(), TestSessionHandler_Delete(), TestSessionHandler_Get() (+3 more)

### Community 45 - "BadRequest"
Cohesion: 0.15
Nodes (20): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Kind, Internal(), New(), NotFound() (+12 more)

### Community 46 - "NewRedisStreamTask"
Cohesion: 0.31
Nodes (15): NewDefaultTaskRegistry(), NewRedisStreamTask(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_CleanupCompletedWaitsForFinished(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister() (+7 more)

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.12
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "Message"
Cohesion: 0.08
Nodes (32): approximateTokenEstimator, ContextBuilder, ContextPolicy, ContextRequest, fixedTokenEstimator, Memory, SimpleMemory, TokenEstimator (+24 more)

### Community 50 - "阶段 1：Settings 快照与 Prompt 可追踪性"
Cohesion: 0.18
Nodes (10): Prompt, Settings, 不做, 实施状态, 校验行为变更, 检查点, 目标, 范围 (+2 more)

### Community 51 - "NewSessionService"
Cohesion: 0.25
Nodes (21): NewSessionService(), NewMockSessionRepository(), requireAppErrKind(), TestSessionService_ClearUnreadCount_NotFound(), TestSessionService_CreateSession(), TestSessionService_CreateSession_RepositoryError(), TestSessionService_DeleteSession_NotFound(), TestSessionService_DeleteSession_RepositoryError() (+13 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.19
Nodes (18): CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), fullCaps(), fullProfile(), TestCanFallbackAfterToolUse_DeclaredWriteToolBlocks(), TestCanFallbackAfterToolUse_ExecutedButToolsEmpty(), TestCanFallbackAfterToolUse_ExecutedWithoutName() (+10 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "createSessionForTest"
Cohesion: 0.10
Nodes (67): Response, TotalResponse, TestAgentSettingsMigrationNormalizesInvalidRecordIdempotently(), TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_AgentConfigRejectsInvalidValues(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle() (+59 more)

### Community 56 - "Sandbox"
Cohesion: 0.10
Nodes (17): TestMessageTool_NotifyUserSchema(), NewFileTool(), NewMessageTool(), toolNames(), NewShellTool(), TestFileToolFindUsesGlobPattern(), TestMCPTool_InitializeWithoutConfig(), TestMessageTool_Invoke_PassesTextThrough() (+9 more)

### Community 57 - "refactor-run/README.md"
Cohesion: 0.17
Nodes (7): 历史文档指针, Run/Session 重构方案集, 中断恢复, 每个阶段的完成标准, 目标, 禁止双轨长期存在, 阶段总览

### Community 58 - "browser_cli.js"
Cohesion: 0.20
Nodes (27): ActionError, attachLogCollector(), domHelpers(), elementExpr(), fail(), fs, INTERACTIVE_SELECTOR, loadPlaywright() (+19 more)

### Community 59 - "Config"
Cohesion: 0.10
Nodes (26): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, ObjectStorageConfig, RedisConfig (+18 more)

### Community 60 - "部署指南"
Cohesion: 0.12
Nodes (17): 1. 部署方式, 2. 架构总览, 3. 前置要求, 4.1 准备环境变量, 4.2 一键启动, 4.3 访问入口（统一走 nginx）, 4. 快速启动（nginx 网关模式，推荐）, 5.1 `.env` 模板（`api/.env.example`） (+9 more)

### Community 61 - "阶段 6：删除旧执行基础设施"
Cohesion: 0.20
Nodes (9): Agent 旧 Task 链, Session 执行字段, 删除前置条件, 删除清单, 完成定义, 实施提交, 数据库迁移, 模型包边界 (+1 more)

### Community 62 - "routed_llm_test.go"
Cohesion: 0.13
Nodes (19): LLMRequest, healthOfModel(), healthRecorder(), routedCatalogModel(), routedTestProfile(), TestRoutedLLM_ConfigErrorsDoNotPolluteHealth(), TestRoutedLLM_FallbackWithWrappedProviderError(), TestRoutedLLM_FallbackWorksWhenPlanHeadIsNil() (+11 more)

### Community 63 - "testLLMModelRepo"
Cohesion: 0.21
Nodes (25): NewTestContext(), cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_Create_SecondDefaultViolatesUniqueIndex(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError() (+17 more)

### Community 64 - "NewRoutedLLM"
Cohesion: 0.21
Nodes (27): modelNames(), openAIModelWithID(), recordAs(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_AutoAppendsConfiguredEnvFallback(), TestRoutedLLM_AutoSkipsCrossProtocolEnvFallback(), TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback(), TestRoutedLLM_DegradedModelIsSortedLower() (+19 more)

### Community 65 - "ShellService"
Cohesion: 0.05
Nodes (34): ConsoleRecord, Exception, FastAPI, register_exception_handlers(), AppException, BusyException, NotFoundException, Any (+26 more)

### Community 66 - "types.go"
Cohesion: 0.09
Nodes (21): browserConsoleExecRequest, browserConsoleViewRequest, browserInputRequest, browserNavigateRequest, browserPressKeyRequest, browserScreenshotData, browserScreenshotRequest, browserScrollRequest (+13 more)

### Community 67 - "defaultAgentSettings"
Cohesion: 0.10
Nodes (21): contextPairTool, waitingOutcomeTool, NewBaseAgent(), TestBaseAgentStopShellWatchWaitsForWatcherExit(), TestBaseAgentInvokePublishesTextDeltas(), TestBaseAgentInvokeStreamingStopsWhenProviderLeavesStreamOpen(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(), TestBaseAgent_InvokeWithEmptyRetry() (+13 more)

### Community 68 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 69 - "anthropic_llm_test.go"
Cohesion: 0.14
Nodes (22): NewAnthropicClient(), errKind(), roundTripperFunc, newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_ParentDeadlineNotMisattributedToToolCalling(), TestAnthropicClient_ProtocolError_NotFallbackable() (+14 more)

### Community 70 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 71 - "event.go"
Cohesion: 0.05
Nodes (42): TestConversationMessagesRestoresPersistedUserAndAssistantMessages(), TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), clonePlanSteps(), cloneStrings(), EventType, MessageRole, NewDoneEvent(), NewMessageDeltaEvent() (+34 more)

### Community 72 - "logger_test.go"
Cohesion: 0.16
Nodes (22): GetLevel(), Init(), InitWithConfig(), SetLevel(), stdoutWriteOK(), TestCallerInBusinessCode(), testCallerLocation(), TestGetLevel_DefaultWhenNotInitialized() (+14 more)

### Community 73 - "LLMRuntimeConfig"
Cohesion: 0.16
Nodes (14): ModelIDFromContext(), betterHealth(), configKey(), containsModel(), LLMRequest, hasCapabilityProfile(), healthRank(), isConfigError() (+6 more)

### Community 74 - "PlannerReActFlow"
Cohesion: 0.25
Nodes (6): FlowStatus, PlannerReActFlow, TaskInput, BaseEvent, NewErrorEvent(), InfoContext()

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "mcp_client_test.go"
Cohesion: 0.15
Nodes (16): NewMCPClientManager(), TestErrClientClosed(), TestErrClientNotInitialized(), TestErrInvalidResponse(), TestMCPClientManager_Close(), TestMCPClientManager_GetClient(), TestMCPClientManager_ListAllTools_Empty(), TestMCPClientManager_New() (+8 more)

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.13
Nodes (15): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 一键部署（推荐）, 健康检查, 前置要求, 容器列表, 常用命令 (+7 more)

### Community 78 - "sync.Once"
Cohesion: 0.33
Nodes (3): blockingWatchSandbox, sync.Once, ToolSet

### Community 79 - "OSS"
Cohesion: 0.12
Nodes (17): TestBuildWithFactoriesInjectsFileCleanupScheduler(), ensureBucket(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL() (+9 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (15): EstimateContextTokens(), isToolCallFinishReason(), MergeDeltas(), repeat(), TestEstimateContextTokens_ASCII(), TestEstimateContextTokens_Mixed(), TestEstimateContextTokens_NonASCII(), TestMergeDeltas_CanonicalToolCallsRetainsToolCall() (+7 more)

### Community 81 - ".handleToolCall"
Cohesion: 0.19
Nodes (9): InvokeResult, OutcomeKind, StepOutcome, ToolCallResult, WaitingInput, shouldPublishDeltas(), toolResultWaitingForUser(), waitingInputAttachments() (+1 more)

### Community 82 - "chat_lifecycle_test.go"
Cohesion: 0.31
Nodes (14): appendTaskEvent(), eventIndex(), newChatTestApp(), parseSSEEvents(), parseStreamID(), readSSEEvent(), TestChatEndpoint_LastEventIDResumesWithoutDuplicatesOrGaps(), TestChatEndpoint_SuccessStreamsOrderedEvents() (+6 more)

### Community 83 - "BaseAgent"
Cohesion: 0.17
Nodes (6): BaseAgent, descriptorFromSchema(), MultiFunctionTool, Tool, ToolDescriptor, ToolRegistry

### Community 84 - "PromptCatalog"
Cohesion: 0.27
Nodes (8): PromptCatalog, PromptDefinition, PromptName, mustNewPromptCatalog(), NewPromptCatalog(), TestDefaultPromptCatalogExposesVersionedDefinitions(), TestNewPromptCatalogRejectsIncompleteDefinition(), TestPromptCatalogCopiesInputAndHashesDeterministically()

### Community 85 - "API 测试方案"
Cohesion: 0.18
Nodes (11): 1. 当前判断, 2. 测试分层, 3. 环境与命令, 4. 高收益测试契约, 5. 已完成的测试精简, 7. 完成标准与阶段归属, API 测试方案, HTTP integration (+3 more)

### Community 86 - "MCPTool"
Cohesion: 0.23
Nodes (4): buildMCPFunctionIndex(), mcpFunction, MCPTool, mcpToolManager

### Community 87 - "集成测试"
Cohesion: 0.13
Nodes (14): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+6 more)

### Community 88 - "AppConfigHandler"
Cohesion: 0.21
Nodes (6): getConfig(), AppConfigHandler, T, reloadOrFail(), updateConfig(), configRuntimeReloader

### Community 89 - "NewProviderError"
Cohesion: 0.35
Nodes (8): ErrorKind, IsDeterministicKind(), isFallbackable(), isRetryable(), NewProviderError(), TestIsDeterministicKind(), TestProviderErrorFallbackableConsistency(), ProviderError

### Community 90 - "ServiceRegressionTests"
Cohesion: 0.04
Nodes (19): FileService, UploadFile, 文件操作只接受普通文件，避免目录被当作文件读取或删除。, 目录遍历只接受目录路径，尽早返回明确的 404。, 校验全局读取上限；具体文件按流式读取，超限时返回截断结果。, 停止达到读取上限的子进程，避免 terminate 后永久等待。, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容, 按固定块读取 sudo 输出，避免超长单行触发 readline 缓冲上限。 (+11 more)

### Community 91 - "ApiEndpointTests"
Cohesion: 0.29
Nodes (3): ApiEndpointTests, Any, 通过 ASGI 协议直接调用应用，避免测试依赖额外 HTTP 客户端。

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.22
Nodes (8): API 接口, Docker 部署, Manus 沙箱服务, 信任模型与安全边界, 技术栈, 本地开发, 架构, 端口说明

### Community 93 - "Manus 前端 UI"
Cohesion: 0.18
Nodes (10): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 模型选择（Auto） (+2 more)

### Community 94 - "阶段 3：Run 领域与存储（未接生产）"
Cohesion: 0.14
Nodes (14): 3A 领域检查点（已完成）, Messages 契约, Redis 与恢复边界, Run 领域模型, 历史数据策略, 原子事务, 完成条件与回滚, 实施顺序 (+6 more)

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 97 - "目标架构：Session、Run 与进程内 Engine"
Cohesion: 0.12
Nodes (14): 1. 适用范围, 2. 核心边界, 3. 目标调用链, 4. 状态原则, 5. 等待输入恢复原则, 6. 数据所有权, 7. 演进边界, Engine (+6 more)

### Community 98 - "AppConfig"
Cohesion: 0.22
Nodes (7): AppConfig, AppConfigType, pgx.Tx, scanAppConfig(), newAppConfig(), PostgresAppConfigRepository, MockAppConfigRepository

### Community 99 - "logger.go"
Cohesion: 0.16
Nodes (20): Any(), Bool(), Dur(), Error(), Fatal(), Float64(), Get(), Int() (+12 more)

### Community 100 - "ToolResult"
Cohesion: 0.05
Nodes (10): ToolArtifact, ToolResult, NewToolResult(), BrowserInput, BrowserTarget, SandboxClient, BrowserClient, browserTargetRequest (+2 more)

### Community 101 - "go-manus 面试指南"
Cohesion: 0.25
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "Run/Session 重构实施状态"
Cohesion: 0.14
Nodes (14): Run/Session 重构实施状态, 决策记录, 已完成：检查点 2A StepOutcome, 已完成：检查点 2B ContextPolicy, 当前状态, 当前生产语义, 恢复流程, 最近完成：检查点 2C MCP 动态路由、初始化与错误契约 (+6 more)

### Community 103 - "docs/README.md"
Cohesion: 0.20
Nodes (3): Agent 架构（兼容入口）, API 测试规则入口, 文档导航

### Community 104 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 105 - ".initRoutes"
Cohesion: 0.20
Nodes (17): defaultTrustedProxies(), CORS(), generateRequestID(), Logger(), Recovery(), RequestID(), setupTestEngine(), TestCORS_AllowedHeader() (+9 more)

### Community 106 - "NewBrowserTool"
Cohesion: 0.23
Nodes (6): NewBrowserTool(), TestBrowserToolRejectsInvalidParams(), TestBrowserToolRoutesActionsToBrowser(), TestBrowserToolScreenshotKeepsImageOutOfLLMData(), Browser, BrowserTool

### Community 107 - "6. 代码评审与回归测试：测试不是覆盖，而是契约"
Cohesion: 0.22
Nodes (9): 6.1 Why：测试是行为的契约, 6.2 历史 Bug 的回归测试, 6.3 测试分类, 6.4 Go 测试模式, 6.5 面试要点, 6. 代码评审与回归测试：测试不是覆盖，而是契约, 子测试 + t.Helper, 模糊测试（Go 1.18+） (+1 more)

### Community 108 - "vnc-overlay.tsx"
Cohesion: 0.31
Nodes (6): buildVNCUrl(), VNCOverlay(), VNCOverlayProps, VNCViewer, VNCStatus, VNCViewerProps

### Community 109 - "go-manus 实战经验与面试考点沉淀（2026-09-04）"
Cohesion: 0.25
Nodes (7): 7.1 资深 Go 后端, 7.2 系统设计, 7.3 工程实践, 7. 综合面试题, 8. 参考资料, go-manus 实战经验与面试考点沉淀（2026-09-04）, 目录

### Community 110 - "3. Docker Desktop 替换：macOS 容器运行时选型"
Cohesion: 0.25
Nodes (8): 3.1 现象, 3.2 What, 3.3 主流方案对比, 3.4 推荐方案：Colima, 3.5 迁移关键点, 3.6 资源分配建议（16GB MacBook）, 3.7 面试要点, 3. Docker Desktop 替换：macOS 容器运行时选型

### Community 111 - "run"
Cohesion: 0.31
Nodes (7): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), net/http.Server, Config

### Community 112 - "testSessionRepo"
Cohesion: 0.32
Nodes (11): testSessionRepo(), TestSessionRepo_AppendEvent_AtomicUpdate(), TestSessionRepo_AppendEvent_UpdatesLatestMessage(), TestSessionRepo_AppendEvent_UserMessageNotUnread(), TestSessionRepo_CreateAndGetByID(), TestSessionRepo_GetAll_ExcludesSoftDeleted(), TestSessionRepo_GetByID_NotFoundReturnsNilNil(), TestSessionRepo_List_Pagination() (+3 more)

### Community 113 - "阶段 4：Run 执行装配与唯一后端切换"
Cohesion: 0.17
Nodes (12): 4A：装配但不接生产, 4B：原子切换生产语义, 代码范围, 切换前置条件, 取消竞争测试矩阵, 可观测性, 回滚边界, 完成条件 (+4 more)

### Community 114 - "testRunRepository"
Cohesion: 0.64
Nodes (8): initialRunMessage(), newRunForTest(), TestRunRepo_CreateWithInitialMessage_EnforcesSingleActiveRun(), TestRunRepo_CreateWithInitialMessage_PersistsOneAggregate(), TestRunRepo_CreateWithInitialMessage_ReusesOriginalAggregateForSameClientRequest(), TestRunRepo_EnterWaitingInput_RollsBackWhenQuestionBelongsToAnotherSession(), TestRunRepo_WaitAndResume_UsesSnapshotRevisionAndInputIdempotency(), testRunRepository()

### Community 115 - "3.3 Repository 层架构设计"
Cohesion: 0.29
Nodes (7): 3.3 Repository 层架构设计, WithTx 事务封装, 事务支持：QueryContext 接口, 接口设计, 构造函数设计：普通版 vs 事务版, 软删除模式, 面试追问

### Community 116 - "5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异"
Cohesion: 0.29
Nodes (7): 5.1 现象, 5.2 What, 5.3 sonic vs encoding/json 差异, 5.4 sonic 正确使用姿势, 5.5 替换 JSON 库的检查清单, 5.6 面试要点, 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### Community 117 - "stubLLM"
Cohesion: 0.31
Nodes (3): LLMRequest, streamingStubLLM, stubLLM

### Community 118 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 119 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 120 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 121 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 122 - "RuntimeA2AConfig"
Cohesion: 0.28
Nodes (7): cloneStringMap(), A2AConfig, RuntimeA2AConfig(), RuntimeMCPConfig(), TestRuntimeA2AConfigFiltersDisabledServers(), TestRuntimeMCPConfigFiltersDisabledServersAndCopiesValues(), TestToolProviderOmitsEmptyDynamicTools()

### Community 123 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 124 - "NewToolError"
Cohesion: 0.11
Nodes (10): browserTargetFromParams(), optionalToolBool(), optionalToolFloat(), optionalToolInt(), optionalToolString(), requiredToolString(), NewToolError(), NewToolResultWithMessage() (+2 more)

### Community 125 - "技术方案文档编写计划"
Cohesion: 0.33
Nodes (5): 已知环境, 技术方案文档编写计划, 目标, 硬约束, 阶段

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "integration_env_test.go"
Cohesion: 0.73
Nodes (5): clearIntegrationOverrides(), TestLoadIntegrationEnvAppliesOverrides(), TestLoadIntegrationEnvRejectsInvalidOverrides(), TestLoadIntegrationEnvRejectsUnsafeResources(), writeConfig()

### Community 128 - "a2a_test.go"
Cohesion: 0.05
Nodes (56): A2AAgentCapabilities, A2AAgentSkill, A2AArtifact, A2AClient, A2AClientManagerConfig, A2AFilePart, A2AInterface, A2AJSONRPCError (+48 more)

### Community 129 - "NewSessionRuntime"
Cohesion: 0.11
Nodes (12): attachmentSandbox, attachmentStorage, NewSessionRuntime(), TestAgentService_GetActiveTaskIDClearsCompletedTask(), TestAgentService_ResolveMessageAttachments(), TestAgentService_ResolveMessageAttachmentsRejectsOtherSession(), TestAgentServiceChatUsesDetachedContextForMessagePersistence(), TestPopRetryConfig() (+4 more)

### Community 130 - "6. 分阶段实施"
Cohesion: 0.25
Nodes (8): 6. 分阶段实施, Future R：Run 重构后的测试, Phase 0：基线, Phase 1：环境入口, Phase 2：边界归位, Phase 3：并发稳定性, Phase 4：当前 API HTTP 契约, Phase 5：当前低收益测试清理

### Community 131 - "ParseWithContext"
Cohesion: 0.19
Nodes (16): Parse(), ParseWithContext(), TestParse_BasicFunctionality(), TestParse_CommentLines(), TestParse_DataFieldWithEmptyContent(), TestParse_EventTypePreserved(), TestParse_RetryAndIdFields(), TestParseWithContext_ConcurrentDifferentData() (+8 more)

### Community 132 - "FileHandler"
Cohesion: 0.25
Nodes (4): FileHandler, NewFileHandler(), SessionService, Handlers

### Community 133 - "API 当前事实基线（2026-10）"
Cohesion: 0.20
Nodes (10): Agent 配置与搜索 limit, API 当前事实基线（2026-10）, Engine 与上下文, 仍在使用、禁止提前删除, 工具系统, 已完成能力, 当前生产链, 核验基线 (+2 more)

### Community 134 - "VNCProxy"
Cohesion: 0.18
Nodes (12): dialSandboxVNC(), sameOrigin(), newLocalTestServer(), TestVNCProxy_AllowsSameOriginAndNonBrowserClient(), TestVNCProxy_ProxyEcho(), TestVNCProxy_RejectsCrossOrigin(), TestVNCProxy_ServiceError(), VNCProxy() (+4 more)

### Community 135 - "sandbox_external_test.go"
Cohesion: 0.39
Nodes (14): assertContentLimitedTo(), containsPath(), requireSandbox(), sandboxContext(), sandboxData(), TestSandbox_BrowserScreenshotReturnsPNG(), TestSandbox_BusinessErrorMapsToAPIError(), TestSandbox_FileLifecycle() (+6 more)

### Community 137 - "Postgres"
Cohesion: 0.13
Nodes (15): newRepositories(), Postgres, NewPostgres(), AppConfigRepository, NewAppConfigRepository(), LLMModelRepository, NewLLMModelRepository(), pgx.Tx (+7 more)

### Community 138 - "sensenova_integration_test.go"
Cohesion: 0.44
Nodes (8): requireSenseNovaKey(), senseNovaReasoningEffort(), senseNovaRequest(), TestSenseNova_ChatModelsAreOpenAICompatible(), TestSenseNova_ImageModels(), TestSenseNova_ListModels(), TestSenseNova_StructuredJSONForPlannerCompatibleModels(), senseNovaChatResponse

### Community 139 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 140 - "5. 注意事项和踩坑点"
Cohesion: 0.33
Nodes (6): 5.1 Go 错误处理, 5.2 PostgreSQL 参数化, 5.3 Repository 设计, 5.4 软删除查询, 5.5 文件清理, 5. 注意事项和踩坑点

### Community 141 - "Redis"
Cohesion: 0.11
Nodes (17): StatusHandler, NewStatusHandler(), Redis, redis.Client, NewRedis(), HealthStatus, HealthState, ServiceName (+9 more)

### Community 142 - "3.2 PostgreSQL INTERVAL 与参数化查询"
Cohesion: 0.40
Nodes (5): 3.2 PostgreSQL INTERVAL 与参数化查询, Why：为什么 INTERVAL 不能参数化？, 问题背景, 面试追问, 项目中的正确实现

### Community 143 - "3.5 Session 与 Agent 架构"
Cohesion: 0.40
Nodes (5): 3.5 Session 与 Agent 架构, Agent Memory 机制, AppendEvent 业务逻辑, MessageEvent 结构, 面试追问

### Community 144 - "4. 可能的面试追问"
Cohesion: 0.40
Nodes (5): 4.1 Go 语言层面, 4.2 数据库层面, 4.3 架构设计层面, 4.4 测试层面, 4. 可能的面试追问

### Community 148 - "Err"
Cohesion: 0.14
Nodes (12): SessionRuntime, A2AConfig, decodeStreamData(), RedisStreamMessageQueue, redis.Client, NewRedisStreamMessageQueue(), resolveBlockTimeout(), DebugContext() (+4 more)

### Community 157 - "五、外部依赖"
Cohesion: 0.40
Nodes (5): 5.1 LLM 调用, 5.2 消息队列, 5.3 Sandbox 沙箱, 5.4 MCP 客户端, 五、外部依赖

### Community 158 - "7.1 Memory 记忆模块详细对比"
Cohesion: 0.40
Nodes (5): 7.1.1 接口设计对比, 7.1.2 核心差异分析, 7.1.3 压缩逻辑深度对比, 7.1.4 面试分析点, 7.1 Memory 记忆模块详细对比

### Community 159 - "Impact Analysis with GitNexus"
Cohesion: 0.22
Nodes (8): Checklist, Example: "What breaks if I change validateUser?", Impact Analysis with GitNexus, Risk Assessment, Tools, Understanding Output, When to Use, Workflow

### Community 160 - "7.4 存储层详细对比"
Cohesion: 0.40
Nodes (5): 7.4.1 Repository 接口设计, 7.4.2 数据库 Schema 对比, 7.4.3 事务处理对比, 7.4.4 面试分析点, 7.4 存储层详细对比

### Community 162 - "阶段 2：Engine、Outcome 与 ContextPolicy"
Cohesion: 0.25
Nodes (8): ContextPolicy, Engine 输出, 回滚, 工具边界前置治理, 当前事实, 检查点, 目标, 阶段 2：Engine、Outcome 与 ContextPolicy

### Community 166 - "test_api_endpoints.py"
Cohesion: 0.10
Nodes (21): APIRouter, BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, create_api_routes() (+13 more)

### Community 168 - "._iter_file_lines"
Cohesion: 0.25
Nodes (6): _LineChunk, 逐行收集范围内内容，任何上限命中后立即停止读取。, 按固定块解码文件，超长无换行内容达到上限即截断。, 文件流的一段逻辑行；truncated 表示单行超过内存上限。, 按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。, _truncate_utf8()

### Community 169 - "MCPConfig"
Cohesion: 0.11
Nodes (13): A2AConfig, A2AServer, MCPConfig, MCPServer, getConfig(), AppConfigService, T, mergeA2AServers() (+5 more)

### Community 171 - "API 测试规则"
Cohesion: 0.29
Nodes (7): API 测试规则, 分层边界, 变更要求, 外部环境, 测试先绑定契约, 测试替身, 衰减风险评审

### Community 172 - "六、持久化模型"
Cohesion: 0.29
Nodes (7): messages, outbox, run_events, run_plans, runs, sessions, 六、持久化模型

### Community 173 - "Debugging with GitNexus"
Cohesion: 0.25
Nodes (7): Checklist, Debugging Patterns, Debugging with GitNexus, Example: "Payment endpoint returns 500 intermittently", Tools, When to Use, Workflow

### Community 174 - "Exploring Codebases with GitNexus"
Cohesion: 0.25
Nodes (7): Checklist, Example: "How does payment processing work?", Exploring Codebases with GitNexus, Resources, Tools, When to Use, Workflow

### Community 175 - "GitNexus Guide"
Cohesion: 0.29
Nodes (6): Always Start Here, GitNexus Guide, Graph Schema, Resources Reference, Skills, Tools Reference

### Community 176 - "GitNexus — Code Intelligence"
Cohesion: 0.33
Nodes (5): Always Do, CLI, GitNexus — Code Intelligence, Never Do, Resources

### Community 177 - "GitNexus — Code Intelligence"
Cohesion: 0.33
Nodes (5): Always Do, CLI, GitNexus — Code Intelligence, Never Do, Resources

### Community 178 - "SearchEngine"
Cohesion: 0.21
Nodes (7): NewSearchTool(), TestSearchToolForwardsConfiguredLimit(), TestSearchToolRejectsEmptyQuery(), TestToolProviderReloadSearchLimitAffectsNewTools(), SearchEngine, searchEngineStub, SearchTool

### Community 181 - "AgentSettings"
Cohesion: 0.12
Nodes (13): AgentTaskRunner, AgentTaskRunnerConfig, conversationMessages(), TestNewAgentTaskRunnerLoadsInitialConversation(), TestPlanner_CreatePlanRejectsNonJSONContent(), TestPlanner_CreatePlanUsesStructuredRequestWithoutTools(), NewPlannerAgent(), NewPlannerReActFlow() (+5 more)

### Community 182 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 183 - ".loadOne"
Cohesion: 0.09
Nodes (29): Loader, headRunes(), NewLoader(), normalizeText(), tailRunes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+21 more)

### Community 184 - "testFileRepo"
Cohesion: 0.29
Nodes (12): deleteSessionDirect(), testFileRepo(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound(), TestFileRepo_GetBySessionAndID_EnforcesOwnership() (+4 more)

### Community 185 - "UnixStreamHTTPConnection"
Cohesion: 0.33
Nodes (3): HTTPConnection, 重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接, UnixStreamHTTPConnection

### Community 187 - "Run/Session 架构重构规格"
Cohesion: 0.40
Nodes (4): Run/Session 架构重构规格, 实施依据, 核心约束, 目标

### Community 188 - "三、服务层"
Cohesion: 0.50
Nodes (4): 3.1 AgentService, 3.2 SessionService, 3.3 FileService, 三、服务层

### Community 189 - "sandbox/package.json"
Cohesion: 0.25
Nodes (7): playwright-core, dependencies, playwright-core, description, name, private, version

### Community 190 - "六、差异汇总"
Cohesion: 0.50
Nodes (4): 6.1 功能缺失列表, 6.2 实现差异列表, 6.3 架构差异, 六、差异汇总

### Community 191 - "八、改进建议"
Cohesion: 0.50
Nodes (4): 中优先级, 低优先级, 八、改进建议, 高优先级

### Community 194 - "mcp_client.go"
Cohesion: 0.15
Nodes (9): mcpResultText(), MCPToolResult, NewStdioMCPClient(), parseToolResult(), TestNewStdioMCPClient(), MCPContent, MCPServerError, StdioMCPClientConfig (+1 more)

### Community 195 - "LLMRequest"
Cohesion: 0.08
Nodes (8): blockingStreamingAgentLLM, mockLLM, streamingAgentLLM, LLMRequest, LLMResponse, connectionTestLLM, blockingLLM, scriptedLLM

### Community 196 - ".uploadFile"
Cohesion: 0.36
Nodes (4): SandboxClient, newAPIError(), toolResultErr(), net/http.Header

### Community 197 - "response_test.go"
Cohesion: 0.29
Nodes (5): TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure(), TestSuccess_HttpResponse(), TestTotalResponse_Structure()

### Community 205 - "NewMockFileRepository"
Cohesion: 0.36
Nodes (13): NewMockFileRepository(), NewFileService(), TestFileServiceDeleteFileDeletesStorageAndRepo(), TestFileServiceDeleteFileStorageError(), TestFileServiceDownloadFileNilStorageReturnsStorageUnavailable(), TestFileServiceDownloadFileWithStorage(), TestFileServiceGetFileInfoFound(), TestFileServiceMissingFileReturnsNotFound() (+5 more)

### Community 210 - "a2a_config.go"
Cohesion: 0.50
Nodes (3): A2AAgent, A2AAgent, A2AConfig

### Community 211 - "time.Duration"
Cohesion: 0.08
Nodes (38): bochaFreshness(), NewBochaSearchClientWithTimeout(), NewFallbackSearchClient(), NewSearchEngine(), NewTavilySearchClientWithTimeout(), resolveSearchTimeout(), tavilyTimeRange(), loadSearchKeys() (+30 more)

### Community 215 - "NewAppConfigService"
Cohesion: 0.40
Nodes (12): NewAppConfigService(), NewMockAppConfigRepository(), TestAppConfigService_DeleteAndUpdateA2AServer(), TestAppConfigService_DeleteMCPServer(), TestAppConfigService_DeleteMCPServer_NotFoundWithoutConfig(), TestAppConfigService_GetConfig_RepositoryError(), TestAppConfigService_UpdateA2AConfigRejectsInvalidURL(), TestAppConfigService_UpdateMCPConfig() (+4 more)

### Community 217 - "维护状态"
Cohesion: 0.40
Nodes (5): 已完成, 已知限制, 维护状态, 维护规则, 验证基线

### Community 218 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

### Community 219 - "学习路径"
Cohesion: 0.40
Nodes (5): 学习路径, 第一阶段：理解 Agent 基础, 第三阶段：理解外部集成, 第二阶段：掌握工具系统, 第四阶段：部署和扩展

### Community 220 - "Plan"
Cohesion: 0.11
Nodes (13): PlannerAgent, ReActAgent, BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext(), planHasFailedStep() (+5 more)

### Community 222 - "openai_llm_test.go"
Cohesion: 0.13
Nodes (30): anthropicErrorResponse(), blockingTransport(), roundTripperFunc, newTransportClient(), newWireCaptureClient(), okChatResponse(), TestNormalizeOpenAIFinishReason(), TestOpenAIClient_200WithErrorBodySurfacesUpstreamMessage() (+22 more)

### Community 223 - "task_redis_test.go"
Cohesion: 0.16
Nodes (13): mockTaskRunner, retentionCall, TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_FinishDestroysRunner(), TestRedisStreamTask_FinishedChangesAfterRunnerExit(), TestRedisStreamTask_FinishSetsStreamRetention() (+5 more)

### Community 224 - "MCPToolInfo"
Cohesion: 0.36
Nodes (4): MCPToolInfo, parseToolInfos(), TestParseToolInfos(), fakeMCPManager

### Community 225 - "shutdownCurrent"
Cohesion: 0.33
Nodes (4): shutdownCurrent(), Sync(), go.uber.org/zap/zapcore.WriteSyncer, neverCloseSyncer

### Community 228 - "10. 故障排查"
Cohesion: 0.50
Nodes (4): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查

### Community 253 - "RedisStreamTask"
Cohesion: 0.10
Nodes (5): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask, Info()

### Community 270 - "当前架构总览"
Cohesion: 0.33
Nodes (6): Planner 与 ReAct 边界, 当前架构总览, 当前限制, 流式事件, 请求链路, 重要代码入口

### Community 285 - "validConfig"
Cohesion: 0.53
Nodes (5): TestConfigApplyDefaultsUsesDomainDefaults(), TestValidate_OK(), TestValidate_RequiredFields(), validConfig(), Config

### Community 298 - "package.json"
Cohesion: 0.22
Nodes (8): name, private, scripts, build, dev, lint, start, version

## Knowledge Gaps
- **700 isolated node(s):** `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest`, `A2ATaskParams`, `Task` (+695 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `sandboxContext()` connect `sandbox_external_test.go` to `testing.T`, `context.Context`?**
  _High betweenness centrality (0.030) - this node is a cross-community bridge._
- **Why does `NewToolProvider()` connect `NewToolProvider` to `context.Context`, `MCPConfig`, `AgentService`, `NewBrowserTool`, `SearchEngine`, `Err`, `Sandbox`, `tools_test.go`, `RuntimeA2AConfig`, `Warn`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Are the 6 inferred relationships involving `ServiceRegressionTests` (e.g. with `AppException` and `BadRequestException`) actually correct?**
  _`ServiceRegressionTests` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest` to the rest of the system?**
  _700 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `index.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.12561576354679804 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.057902973395931145 - nodes in this community are weakly interconnected._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.05583308845136644 - nodes in this community are weakly interconnected._