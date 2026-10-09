# Graph Report - go-manus  (2026-10-09)

## Corpus Check
- 366 files · ~909,286 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 4122 nodes · 10129 edges · 236 communities (211 shown, 25 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 942 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `cc23606f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- index.tsx
- cn
- Session
- BrowserService
- index.ts
- client.go
- queryer
- MCPSetting.tsx
- BuildRuntimeConfigFromModel
- testing.T
- 重构.md
- tool-preview-panel.tsx
- PostgresSessionRepository
- Response
- session-item.tsx
- Docker 构建优化实践笔记
- StdioMCPClient
- openai_llm.go
- NewMockLLMModelRepository
- File
- app_test.go
- newSandboxClient
- refactor-run/README.md
- SessionHandler
- logger 包 os.Stdout 误关闭 Bug 深度解析
- tools_test.go
- lib/utils.ts
- A2ATool
- endpoints/file.py
- compilerOptions
- tool_event_flow_test.go
- CODE_REVIEW_2026-09-11.md
- App
- endpoints/shell.py
- session-detail-view.tsx
- 阶段 5：唯一 Run API、UI 与 SSE 契约
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- Event
- LLMDelta
- github.com/gin-gonic/gin.Context
- AppConfigService
- llm.go
- MCPConfig
- AppException
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
- getJSON
- Sandbox
- 阶段 2：Engine、Outcome 与 ContextPolicy
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
- RedisStreamMessageQueue
- go-manus - 通用 AI Agent 系统
- Info
- OSS
- MergeDeltas
- BaseAgent
- chat_lifecycle_test.go
- Tool
- PromptCatalog
- 6. 分阶段实施
- NewToolProvider
- 集成测试
- AppConfigHandler
- NewProviderError
- ServiceRegressionTests
- ApiEndpointTests
- Manus 沙箱服务
- Manus 前端 UI
- Plan
- Commands
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- 阶段 3：Run 领域与存储（未接生产）
- AppConfig
- logger.go
- context.Context
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
- createSessionForTest
- 阶段 4：Run 执行装配与唯一后端切换
- LLMModelService
- 3.3 Repository 层架构设计
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- json_parser_repair.go
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 七、详细模块对比
- RuntimeA2AConfig
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- NewToolError
- 技术方案文档编写计划
- 3.4 集成测试
- LoadIntegrationEnv
- A2AClientManager
- NewSessionRuntime
- BrowserCLIContractTests
- ParseWithContext
- FileHandler
- API 当前事实基线（2026-10）
- UnixStreamHTTPConnection
- DefaultLLMModelService
- NewToolResult
- contextBlockingReader
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
- a2a_test.go
- test-env-down.sh
- test-env-up.sh
- service_dependencies.py
- protocol.go
- DefaultAppConfigService
- github.com/Huang131/go-manus/api
- ._iter_file_lines
- llm_model.go
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
- LLMModel
- sandbox_external_test.go
- next-themes
- Run/Session 架构重构规格
- 三、服务层
- sandbox/package.json
- 六、差异汇总
- 八、改进建议
- @radix-ui/react-avatar
- archive/README.md
- Postgres
- LLMResponse
- anthropic_llm.go
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
- Factories
- LLMModelRepository
- .uploadFile
- encoding/json.RawMessage
- Warn
- HealthStatus
- openai_llm_test.go
- task_redis_test.go
- A2AAgentCard
- runInTx
- shutdownCurrent
- sonner
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
2. `ToolResult` - 94 edges
3. `Err()` - 59 edges
4. `ServiceRegressionTests` - 58 edges
5. `ShellService` - 53 edges
6. `File` - 49 edges
7. `RedisStreamTask` - 46 edges
8. `FileService` - 45 edges
9. `Response` - 41 edges
10. `FromError()` - 40 edges

## Surprising Connections (you probably didn't know these)
- `BaseAgent` --references--> `Memory`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `BaseAgent` --references--> `JSONParser`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/jsonx/json_parser.go
- `NewBaseAgent()` --calls--> `NewContextBuilder()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/context_builder.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `TestToolProviderOmitsEmptyDynamicTools()` --calls--> `defaultAgentSettings()`  [INFERRED]
  api/internal/agent/config_test.go → api/internal/agent/settings_test.go

## Import Cycles
- None detected.

## Communities (236 total, 25 thin omitted)

### Community 0 - "index.tsx"
Cohesion: 0.13
Nodes (19): A2aTool(), A2aToolProps, BashTool(), BashToolProps, BrowserTool(), BrowserToolProps, DefaultTool(), DefaultToolProps (+11 more)

### Community 1 - "cn"
Cohesion: 0.06
Nodes (55): metadata, GlobalHeader(), LeftPanel(), Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup(), AvatarGroupCount() (+47 more)

### Community 2 - "Session"
Cohesion: 0.05
Nodes (12): mockSessionRepo, stopSessionRepository, successfulStopSessionRepository, TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits(), TestAgentService_StopSessionReturnsStatusUpdateError(), Session, SessionStatus, unixPointer() (+4 more)

### Community 3 - "BrowserService"
Cohesion: 0.07
Nodes (55): _budget_ms(), click(), console_exec(), console_view(), input_text(), navigate(), press_key(), post (+47 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (50): SessionHeaderProps, configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout(), get() (+42 more)

### Community 5 - "client.go"
Cohesion: 0.19
Nodes (16): A2AArtifact, A2AClient, A2AFilePart, A2AJSONRPCRequest, A2AMessage, A2APart, A2ARequestParams, A2AResult (+8 more)

### Community 6 - "queryer"
Cohesion: 0.09
Nodes (16): pgx.Tx, scanFile(), encodeModelJSONColumns(), pgx.Tx, modelJSON(), scanLLMModel(), collectRows(), pgx.Tx (+8 more)

### Community 7 - "MCPSetting.tsx"
Cohesion: 0.09
Nodes (52): DeleteSessionDialogProps, ManusSettings(), SETTING_MENUS, SettingTab, emptyDraft, Mode, ModelConfigManager(), RenameSessionDialogProps (+44 more)

### Community 8 - "BuildRuntimeConfigFromModel"
Cohesion: 0.31
Nodes (6): BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), TestBuildRuntimeConfigFromModel(), TestProtocolFromProvider(), ProviderProtocol

### Community 9 - "testing.T"
Cohesion: 0.06
Nodes (56): TestFlowStatus_ToSessionStatus(), TestFlowStatus_Values(), TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_InvokeContext(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions() (+48 more)

### Community 10 - "重构.md"
Cohesion: 0.08
Nodes (24): messages, outbox, Plan 和 Step, Prompt 配置, Run, run_events, run_plans, runs (+16 more)

### Community 11 - "tool-preview-panel.tsx"
Cohesion: 0.14
Nodes (22): A2APreview(), ArtifactRef, BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent(), getToolDescription(), MCPPreview() (+14 more)

### Community 12 - "PostgresSessionRepository"
Cohesion: 0.15
Nodes (8): extractSessionMessage(), pgx.Tx, marshalSessionEvents(), scanSessionSummary(), TestExtractSessionMessageProjectsCompletedAssistantMessage(), TestMarshalSessionEventsRejectsInvalidJSON(), TestSessionRepository_JSONSerialization(), PostgresSessionRepository

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
Cohesion: 0.09
Nodes (21): MCPClientManager, MCPToolInfo, MCPToolResult, NewStdioMCPClient(), parseToolInfos(), parseToolResult(), TestNewStdioMCPClient(), TestParseToolInfos() (+13 more)

### Community 17 - "openai_llm.go"
Cohesion: 0.13
Nodes (17): classifyHTTPError(), LLMRequest, normalizeOpenAIFinishReason(), reasoningTokens(), TestNormalizeOpenAIFinishReason(), toOpenAIMessages(), truncateBody(), openAIChatRequest (+9 more)

### Community 18 - "NewMockLLMModelRepository"
Cohesion: 0.20
Nodes (23): NewMockLLMModelRepository(), NewLLMModelService(), newHealthService(), TestLLMModelService_Create(), TestLLMModelService_Create_MapsUniqueViolationToConflict(), TestLLMModelService_Create_PreservesPartialCapabilities(), TestLLMModelService_Create_Validation(), TestLLMModelService_Delete_Default() (+15 more)

### Community 19 - "File"
Cohesion: 0.06
Nodes (15): attachmentFileRepository, attachmentStorage, generatedFileRepository, File, FileRepository, NewFileRepository(), FileStorage, mockStorage (+7 more)

### Community 20 - "app_test.go"
Cohesion: 0.08
Nodes (32): Build(), newLifecycleManager(), normalizeFactories(), resolveAgentSettings(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce() (+24 more)

### Community 21 - "newSandboxClient"
Cohesion: 0.23
Nodes (14): NewSandboxClient(), SandboxClient, newSandboxClient(), TestExecCommandDecodesEnvelope(), TestFindFilesSendsGlobPattern(), TestHealthCheckReportsFailure(), TestPostJSONErrorMapping(), TestRequireAddress() (+6 more)

### Community 22 - "refactor-run/README.md"
Cohesion: 0.15
Nodes (7): 历史文档指针, Run/Session 重构方案集, 中断恢复, 每个阶段的完成标准, 目标, 禁止双轨长期存在, 阶段总览

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

### Community 27 - "A2ATool"
Cohesion: 0.18
Nodes (9): cleanupToolResources(), newToolSetState(), toolNames(), a2aResource, A2ATool, mcpResource, toolCleanup, ToolProvider (+1 more)

### Community 28 - "endpoints/file.py"
Cohesion: 0.13
Nodes (35): api_route, FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get (+27 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "tool_event_flow_test.go"
Cohesion: 0.12
Nodes (12): artifactTool, inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, TestReadTaskOutputAfterTaskUnregistered(), TestRedisStreamTask_GetOutputReadsBufferedEventsWhenStartIDEmpty(), newInMemoryMessageQueue() (+4 more)

### Community 31 - "CODE_REVIEW_2026-09-11.md"
Cohesion: 0.09
Nodes (22): A. 流式链路复查（commit 4892225）, api, B. service 层专项（新发现）, C. 昨日 backlog 复核（HEAD b1f3827 仍开放）, D. 修复批次, E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）, sandbox, ui (+14 more)

### Community 32 - "App"
Cohesion: 0.19
Nodes (10): BuildWithFactories(), DefaultOptions(), App, loadRuntimeToolConfigs(), newA2AConfig(), newMCPConfig(), externalClients, lifecycleManager (+2 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.15
Nodes (26): APIRouter, create_api_routes(), 创建API路由，涵盖整个沙箱项目的所有API, exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据 (+18 more)

### Community 34 - "session-detail-view.tsx"
Cohesion: 0.07
Nodes (41): PageProps, ChatMessage(), ChatMessageProps, StepBlock(), ToolRow(), ManusIcon(), components, headingClasses (+33 more)

### Community 35 - "阶段 5：唯一 Run API、UI 与 SSE 契约"
Cohesion: 0.18
Nodes (11): 5A：Run API/SSE, 5B：UI 切换, 5C：删除旧路由, API 契约, SSE 契约, UI 切换, 代码范围, 回滚边界 (+3 more)

### Community 36 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 37 - "Event"
Cohesion: 0.10
Nodes (19): Stream, Task, TaskStream, nonNilEvents(), NewTaskStream(), parseTaskOutputMessages(), ReadTaskOutput(), readTaskOutput() (+11 more)

### Community 38 - "LLMDelta"
Cohesion: 0.17
Nodes (11): LLMRequest, mustMarshalString(), normalizeAnthropicStopReason(), TestNormalizeAnthropicStopReason(), streamContext(), estimateCostUSD(), sendStreamDelta(), LLMDelta (+3 more)

### Community 39 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.18
Nodes (9): reloadOrFail(), LLMModelHandler, NewLLMModelResponse(), FromError(), TotalResponse, Success(), SuccessWithMsg(), SuccessWithTotal() (+1 more)

### Community 40 - "AppConfigService"
Cohesion: 0.39
Nodes (6): NewAppConfigHandler(), TestGetAgentSettingsReturnsDefaultsWhenRecordIsMissing(), TestUpdateAgentSettingsReloadsSearchLimit(), TestUpdateAgentSettingsReturnsBadRequestForInvalidSettings(), AppConfigService, appConfigServiceStub

### Community 41 - "llm.go"
Cohesion: 0.14
Nodes (12): defaultLLMClientFactory(), TestDynamicLLM_UsesFactory(), LLMClientFactory, NewDynamicLLMWithFactory(), NewLLMClient(), NewOpenAIClient(), runtimeConfigToOpenAIClientConfig(), DynamicLLM (+4 more)

### Community 42 - "MCPConfig"
Cohesion: 0.18
Nodes (9): A2AConfig, A2AServer, MCPConfig, MCPServer, mergeA2AServers(), mergeMCPServers(), validateA2AConfig(), validateMCPConfig() (+1 more)

### Community 43 - "AppException"
Cohesion: 0.06
Nodes (25): Exception, FastAPI, register_exception_handlers(), AppException, BadRequestException, BusyException, Any, 沙箱繁忙：浏览器动作全局串行，锁等待耗尽预算时触发，可稍后重试。 与 504「动作超时」的语义区分： - 503… (+17 more)

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
Cohesion: 0.11
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "Message"
Cohesion: 0.14
Nodes (15): ContextBuilder, ContextPolicy, Memory, SimpleMemory, completeConversationGroups(), estimateMessage(), estimateMessages(), NewContextBuilder() (+7 more)

### Community 50 - "阶段 1：Settings 快照与 Prompt 可追踪性"
Cohesion: 0.18
Nodes (10): Prompt, Settings, 不做, 实施状态, 校验行为变更, 检查点, 目标, 范围 (+2 more)

### Community 51 - "NewSessionService"
Cohesion: 0.25
Nodes (21): NewSessionService(), NewMockSessionRepository(), requireAppErrKind(), TestSessionService_ClearUnreadCount_NotFound(), TestSessionService_CreateSession(), TestSessionService_CreateSession_RepositoryError(), TestSessionService_DeleteSession_NotFound(), TestSessionService_DeleteSession_RepositoryError() (+13 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.18
Nodes (19): CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), fullCaps(), fullProfile(), TestCanFallbackAfterToolUse_DeclaredWriteToolBlocks(), TestCanFallbackAfterToolUse_ExecutedButToolsEmpty(), TestCanFallbackAfterToolUse_ExecutedWithoutName() (+11 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "getJSON"
Cohesion: 0.11
Nodes (49): Response, TestAgentSettingsMigrationNormalizesInvalidRecordIdempotently(), TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_AgentConfigRejectsInvalidValues(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle(), TestFileAPI_Delete_NotFound() (+41 more)

### Community 56 - "Sandbox"
Cohesion: 0.10
Nodes (16): TestMessageTool_NotifyUserSchema(), NewFileTool(), NewMessageTool(), NewShellTool(), TestFileToolFindUsesGlobPattern(), TestMCPTool_InitializeWithoutConfig(), TestMessageTool_Invoke_PassesTextThrough(), TestMessageToolAskUserPreservesInteractionFields() (+8 more)

### Community 57 - "阶段 2：Engine、Outcome 与 ContextPolicy"
Cohesion: 0.25
Nodes (8): ContextPolicy, Engine 输出, 回滚, 工具边界前置治理, 当前事实, 检查点, 目标, 阶段 2：Engine、Outcome 与 ContextPolicy

### Community 58 - "browser_cli.js"
Cohesion: 0.20
Nodes (27): ActionError, attachLogCollector(), domHelpers(), elementExpr(), fail(), fs, INTERACTIVE_SELECTOR, loadPlaywright() (+19 more)

### Community 59 - "Config"
Cohesion: 0.12
Nodes (18): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, ObjectStorageConfig, RedisConfig (+10 more)

### Community 60 - "部署指南"
Cohesion: 0.10
Nodes (21): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查, 1. 部署方式, 2. 架构总览, 3. 前置要求, 4.1 准备环境变量 (+13 more)

### Community 61 - "阶段 6：删除旧执行基础设施"
Cohesion: 0.20
Nodes (9): Agent 旧 Task 链, Session 执行字段, 删除前置条件, 删除清单, 完成定义, 实施提交, 数据库迁移, 模型包边界 (+1 more)

### Community 62 - "routed_llm_test.go"
Cohesion: 0.12
Nodes (22): LLMRequest, healthOfModel(), healthRecorder(), routedCatalogModel(), routedTestProfile(), TestRoutedLLM_ConfigErrorsDoNotPolluteHealth(), TestRoutedLLM_FallbackWithWrappedProviderError(), TestRoutedLLM_FallbackWorksWhenPlanHeadIsNil() (+14 more)

### Community 63 - "testLLMModelRepo"
Cohesion: 0.21
Nodes (25): NewTestContext(), cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_Create_SecondDefaultViolatesUniqueIndex(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError() (+17 more)

### Community 64 - "NewRoutedLLM"
Cohesion: 0.21
Nodes (27): modelNames(), openAIModelWithID(), recordAs(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_AutoAppendsConfiguredEnvFallback(), TestRoutedLLM_AutoSkipsCrossProtocolEnvFallback(), TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback(), TestRoutedLLM_DegradedModelIsSortedLower() (+19 more)

### Community 65 - "ShellService"
Cohesion: 0.07
Nodes (27): ConsoleRecord, NotFoundException, Lock, Process, 启动并持有输出读取任务，保证一个会话只消费当前进程的输出。, 取消并等待输出读取器，避免旧进程向新命令的记录写入输出。, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程 (+19 more)

### Community 66 - "types.go"
Cohesion: 0.09
Nodes (21): browserConsoleExecRequest, browserConsoleViewRequest, browserInputRequest, browserNavigateRequest, browserPressKeyRequest, browserScreenshotData, browserScreenshotRequest, browserScrollRequest (+13 more)

### Community 67 - "defaultAgentSettings"
Cohesion: 0.10
Nodes (22): PlannerAgent, waitingOutcomeTool, NewBaseAgent(), TestBaseAgentStopShellWatchWaitsForWatcherExit(), TestBaseAgentInvokePublishesTextDeltas(), TestBaseAgentInvokeStreamingStopsWhenProviderLeavesStreamOpen(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(), TestBaseAgent_InvokeWithEmptyRetry() (+14 more)

### Community 68 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 69 - "anthropic_llm_test.go"
Cohesion: 0.18
Nodes (18): NewAnthropicClient(), errKind(), roundTripperFunc, newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_ParentDeadlineNotMisattributedToToolCalling(), TestAnthropicClient_ProtocolError_NotFallbackable() (+10 more)

### Community 70 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 71 - "event.go"
Cohesion: 0.05
Nodes (40): TestConversationMessagesRestoresPersistedUserAndAssistantMessages(), clonePlanSteps(), cloneStrings(), EventType, MessageRole, NewDoneEvent(), NewMessageDeltaEvent(), NewMessageDoneEvent() (+32 more)

### Community 72 - "logger_test.go"
Cohesion: 0.16
Nodes (22): GetLevel(), Init(), InitWithConfig(), SetLevel(), stdoutWriteOK(), TestCallerInBusinessCode(), testCallerLocation(), TestGetLevel_DefaultWhenNotInitialized() (+14 more)

### Community 73 - "LLMRuntimeConfig"
Cohesion: 0.16
Nodes (14): ModelIDFromContext(), betterHealth(), configKey(), containsModel(), LLMRequest, hasCapabilityProfile(), healthRank(), isConfigError() (+6 more)

### Community 74 - "PlannerReActFlow"
Cohesion: 0.24
Nodes (7): FlowStatus, PlannerReActFlow, TaskInput, BaseEvent, NewErrorEvent(), NewPlanEvent(), InfoContext()

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "RedisStreamMessageQueue"
Cohesion: 0.16
Nodes (9): batchTaskOutputMQ, StreamMessage, decodeStreamData(), RedisStreamMessageQueue, redis.Client, NewRedisStreamMessageQueue(), resolveBlockTimeout(), BatchMessageQueue (+1 more)

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.08
Nodes (25): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, API 开发, API 文档, go-manus - 通用 AI Agent 系统, 一键部署（推荐） (+17 more)

### Community 78 - "Info"
Cohesion: 0.20
Nodes (5): blockingWatchSandbox, Info(), sync.Once, sync.WaitGroup, FileCleanupScheduler

### Community 79 - "OSS"
Cohesion: 0.13
Nodes (16): TestBuildWithFactoriesInjectsFileCleanupScheduler(), ensureBucket(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL() (+8 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (15): EstimateContextTokens(), isToolCallFinishReason(), MergeDeltas(), repeat(), TestEstimateContextTokens_ASCII(), TestEstimateContextTokens_Mixed(), TestEstimateContextTokens_NonASCII(), TestMergeDeltas_CanonicalToolCallsRetainsToolCall() (+7 more)

### Community 81 - "BaseAgent"
Cohesion: 0.16
Nodes (11): BaseAgent, InvokeResult, OutcomeKind, StepOutcome, ToolCallResult, WaitingInput, shouldPublishDeltas(), toolResultWaitingForUser() (+3 more)

### Community 82 - "chat_lifecycle_test.go"
Cohesion: 0.31
Nodes (14): appendTaskEvent(), eventIndex(), newChatTestApp(), parseSSEEvents(), parseStreamID(), readSSEEvent(), TestChatEndpoint_LastEventIDResumesWithoutDuplicatesOrGaps(), TestChatEndpoint_SuccessStreamsOrderedEvents() (+6 more)

### Community 83 - "Tool"
Cohesion: 0.29
Nodes (6): descriptorFromSchema(), MultiFunctionTool, Tool, ToolDescriptor, ToolRegistry, ToolSet

### Community 84 - "PromptCatalog"
Cohesion: 0.27
Nodes (8): PromptCatalog, PromptDefinition, PromptName, mustNewPromptCatalog(), NewPromptCatalog(), TestDefaultPromptCatalogExposesVersionedDefinitions(), TestNewPromptCatalogRejectsIncompleteDefinition(), TestPromptCatalogCopiesInputAndHashesDeterministically()

### Community 85 - "6. 分阶段实施"
Cohesion: 0.11
Nodes (19): 1. 当前判断, 2. 测试分层, 3. 环境与命令, 4. 高收益测试契约, 5. 已完成的测试精简, 6. 分阶段实施, 7. 完成标准与阶段归属, API 测试方案 (+11 more)

### Community 86 - "NewToolProvider"
Cohesion: 0.12
Nodes (12): mcpResultText(), NewMCPTool(), TestMCPToolCleanupDropsLoadedTools(), TestMCPToolInvokeWithNameRejectsUnknownFunction(), TestMCPToolRegistersDynamicFunctions(), A2AConfig, NewToolProvider(), TestToolProviderDefersRetiredStateCleanupUntilRelease() (+4 more)

### Community 87 - "集成测试"
Cohesion: 0.13
Nodes (14): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+6 more)

### Community 88 - "AppConfigHandler"
Cohesion: 0.24
Nodes (6): getConfig(), AppConfigHandler, T, updateConfig(), configRuntimeReloader, Handlers

### Community 89 - "NewProviderError"
Cohesion: 0.35
Nodes (8): ErrorKind, IsDeterministicKind(), isFallbackable(), isRetryable(), NewProviderError(), TestIsDeterministicKind(), TestProviderErrorFallbackableConsistency(), ProviderError

### Community 90 - "ServiceRegressionTests"
Cohesion: 0.04
Nodes (11): FileService, UploadFile, 文件操作只接受普通文件，避免目录被当作文件读取或删除。, 目录遍历只接受目录路径，尽早返回明确的 404。, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容, 按固定块读取 sudo 输出，避免超长单行触发 readline 缓冲上限。, 根据传递的文件夹路径+glob规则查询文件列表, getAllProcessInfo 返回拆开的 name/group，RPC 需要拼好的 namespec。 (+3 more)

### Community 91 - "ApiEndpointTests"
Cohesion: 0.29
Nodes (3): ApiEndpointTests, Any, 通过 ASGI 协议直接调用应用，避免测试依赖额外 HTTP 客户端。

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.22
Nodes (8): API 接口, Docker 部署, Manus 沙箱服务, 信任模型与安全边界, 技术栈, 本地开发, 架构, 端口说明

### Community 93 - "Manus 前端 UI"
Cohesion: 0.18
Nodes (10): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 模型选择（Auto） (+2 more)

### Community 94 - "Plan"
Cohesion: 0.13
Nodes (12): ReActAgent, BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext(), planHasFailedStep(), DefaultPromptCatalog() (+4 more)

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 97 - "阶段 3：Run 领域与存储（未接生产）"
Cohesion: 0.07
Nodes (26): 1. 适用范围, 2. 核心边界, 3. 目标调用链, 4. 状态原则, 5. 等待输入恢复原则, 6. 数据所有权, 7. 演进边界, Engine (+18 more)

### Community 98 - "AppConfig"
Cohesion: 0.22
Nodes (7): AppConfig, AppConfigType, pgx.Tx, scanAppConfig(), newAppConfig(), PostgresAppConfigRepository, MockAppConfigRepository

### Community 99 - "logger.go"
Cohesion: 0.15
Nodes (22): Bool(), Debug(), DebugContext(), Dur(), Error(), Fatal(), Float64(), Get() (+14 more)

### Community 100 - "context.Context"
Cohesion: 0.06
Nodes (9): ToolResult, SandboxClient, NewBrowserClient(), SandboxClient, mockMessageQueue, context.Context, stubVNCService, BrowserClient (+1 more)

### Community 101 - "go-manus 面试指南"
Cohesion: 0.25
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "Run/Session 重构实施状态"
Cohesion: 0.22
Nodes (9): Run/Session 重构实施状态, 决策记录, 当前状态, 当前生产语义, 恢复流程, 最近完成：阶段 1 Settings 与 Prompt, 最近验证基线, 检查点 (+1 more)

### Community 103 - "docs/README.md"
Cohesion: 0.10
Nodes (15): Agent 架构（兼容入口）, API 测试规则入口, 已完成, 已知限制, 维护状态, 维护规则, 验证基线, 文档导航 (+7 more)

### Community 104 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 105 - ".initRoutes"
Cohesion: 0.20
Nodes (17): defaultTrustedProxies(), CORS(), generateRequestID(), Logger(), Recovery(), RequestID(), setupTestEngine(), TestCORS_AllowedHeader() (+9 more)

### Community 106 - "NewBrowserTool"
Cohesion: 0.24
Nodes (5): NewBrowserTool(), TestBrowserToolRejectsInvalidParams(), TestBrowserToolRoutesActionsToBrowser(), TestBrowserToolScreenshotKeepsImageOutOfLLMData(), BrowserTool

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

### Community 112 - "createSessionForTest"
Cohesion: 0.13
Nodes (40): deleteSessionDirect(), testFileRepo(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound(), TestFileRepo_GetBySessionAndID_EnforcesOwnership() (+32 more)

### Community 113 - "阶段 4：Run 执行装配与唯一后端切换"
Cohesion: 0.17
Nodes (12): 4A：装配但不接生产, 4B：原子切换生产语义, 代码范围, 切换前置条件, 取消竞争测试矩阵, 可观测性, 回滚边界, 完成条件 (+4 more)

### Community 114 - "LLMModelService"
Cohesion: 0.40
Nodes (3): NewLLMModelHandler(), TestLLMModelHandler_CreateRejectsMalformedJSON(), LLMModelService

### Community 115 - "3.3 Repository 层架构设计"
Cohesion: 0.29
Nodes (7): 3.3 Repository 层架构设计, WithTx 事务封装, 事务支持：QueryContext 接口, 接口设计, 构造函数设计：普通版 vs 事务版, 软删除模式, 面试追问

### Community 116 - "5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异"
Cohesion: 0.29
Nodes (7): 5.1 现象, 5.2 What, 5.3 sonic vs encoding/json 差异, 5.4 sonic 正确使用姿势, 5.5 替换 JSON 库的检查清单, 5.6 面试要点, 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### Community 117 - "json_parser_repair.go"
Cohesion: 0.13
Nodes (17): formatRepairTypes(), JSONParser, NewDefaultJSONParser(), completeTruncatedJSON(), Repair, isTruncatedJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks() (+9 more)

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
Cohesion: 0.08
Nodes (16): mockFailingTool, browserTargetFromParams(), optionalToolBool(), optionalToolFloat(), optionalToolInt(), optionalToolString(), requiredToolString(), ToolArtifact (+8 more)

### Community 125 - "技术方案文档编写计划"
Cohesion: 0.33
Nodes (5): 已知环境, 技术方案文档编写计划, 目标, 硬约束, 阶段

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "LoadIntegrationEnv"
Cohesion: 0.24
Nodes (13): applyPostgresEnv(), applyRedisEnv(), applyStorageEnv(), isTestResource(), LoadIntegrationEnv(), clearIntegrationOverrides(), TestLoadIntegrationEnvAppliesOverrides(), TestLoadIntegrationEnvRejectsInvalidOverrides() (+5 more)

### Community 128 - "A2AClientManager"
Cohesion: 0.18
Nodes (11): A2AClientManagerConfig, A2ARemoteAgent, A2AServerConfig, cloneAgentCard(), cloneJSONMap(), cloneJSONValue(), A2AClientManager, isInputRequiredA2AState() (+3 more)

### Community 129 - "NewSessionRuntime"
Cohesion: 0.12
Nodes (12): attachmentSandbox, chatContextSessionRepository, NewSessionRuntime(), TestAgentService_GetActiveTaskIDClearsCompletedTask(), TestAgentService_ResolveMessageAttachments(), TestAgentService_ResolveMessageAttachmentsRejectsOtherSession(), TestAgentServiceChatUsesDetachedContextForMessagePersistence(), TestPopRetryConfig() (+4 more)

### Community 131 - "ParseWithContext"
Cohesion: 0.20
Nodes (15): Parse(), ParseWithContext(), TestParse_BasicFunctionality(), TestParse_CommentLines(), TestParse_DataFieldWithEmptyContent(), TestParse_EventTypePreserved(), TestParse_RetryAndIdFields(), TestParseWithContext_ConcurrentDifferentData() (+7 more)

### Community 132 - "FileHandler"
Cohesion: 0.27
Nodes (4): FileHandler, NewFileHandler(), FileService, SessionService

### Community 133 - "API 当前事实基线（2026-10）"
Cohesion: 0.20
Nodes (10): Agent 配置与搜索 limit, API 当前事实基线（2026-10）, Engine 与上下文, 仍在使用、禁止提前删除, 工具系统, 已完成能力, 当前生产链, 核验基线 (+2 more)

### Community 134 - "UnixStreamHTTPConnection"
Cohesion: 0.33
Nodes (3): HTTPConnection, 重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接, UnixStreamHTTPConnection

### Community 135 - "DefaultLLMModelService"
Cohesion: 0.23
Nodes (7): isUniqueViolation(), normalizeDefaults(), truncateModelTestContent(), validateRequired(), DefaultLLMModelService, HealthInvalidator, RuntimeHealthReader

### Community 136 - "NewToolResult"
Cohesion: 0.09
Nodes (12): NewToolResult(), TestNewToolError(), TestNewToolResult(), TestToolResult_ArtifactsStayOutOfJSON(), TestToolResult_FromSandbox(), TestToolResult_LLMJSONOnNil(), TestToolResult_WithDisplayIsChainable(), BrowserInput (+4 more)

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
Cohesion: 0.23
Nodes (7): StatusHandler, NewStatusHandler(), Redis, redis.Client, NewRedis(), StatusService, NewStatusService()

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
Cohesion: 0.27
Nodes (6): SessionRuntime, A2AConfig, logRollbackFailure(), Err(), ErrorContext(), WarnContext()

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

### Community 162 - "a2a_test.go"
Cohesion: 0.20
Nodes (18): roundTripperFunc, jsonResponse(), TestA2AAgentCapabilities_CanStream(), TestA2AAgentCard_ResolveEndpoint(), TestA2AClientManagerAgentCardsAreIsolatedSnapshots(), TestA2AClientManagerAuthRequiredDoesNotPoll(), TestA2AClientManagerCancelRejectsMissingEndpoint(), TestA2AClientManagerCleanupInvalidatesInFlightInitialize() (+10 more)

### Community 166 - "service_dependencies.py"
Cohesion: 0.20
Nodes (14): BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, get_shell_service(), get_supervisor_service() (+6 more)

### Community 168 - "protocol.go"
Cohesion: 0.17
Nodes (15): toOpenAITools(), ExtraParam, RequestPolicy, ResponseFormat, ToolCall, ToolSpec, AnthropicClientConfig, AudioURL (+7 more)

### Community 169 - "DefaultAppConfigService"
Cohesion: 0.22
Nodes (3): getConfig(), T, DefaultAppConfigService

### Community 171 - "._iter_file_lines"
Cohesion: 0.25
Nodes (6): _LineChunk, 逐行收集范围内内容，任何上限命中后立即停止读取。, 按固定块解码文件，超长无换行内容达到上限即截断。, 文件流的一段逻辑行；truncated 表示单行超过内存上限。, 按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。, _truncate_utf8()

### Community 172 - "llm_model.go"
Cohesion: 0.23
Nodes (12): DefaultCapabilities(), CostPolicy, LLMModelTestResponse, ModelCapabilities, ReasoningMode, RuntimeHealth, RequestPolicy, MergeDefaultCapabilities() (+4 more)

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
Cohesion: 0.19
Nodes (7): NewSearchTool(), TestSearchToolForwardsConfiguredLimit(), TestSearchToolRejectsEmptyQuery(), TestToolProviderReloadSearchLimitAffectsNewTools(), SearchEngine, searchEngineStub, SearchTool

### Community 181 - "AgentSettings"
Cohesion: 0.08
Nodes (17): AgentTaskRunner, AgentTaskRunnerConfig, Capabilities, Repositories, conversationMessages(), TestNewAgentTaskRunnerLoadsInitialConversation(), NewPlannerReActFlow(), A2AConfig (+9 more)

### Community 182 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 183 - ".loadOne"
Cohesion: 0.10
Nodes (28): Loader, headRunes(), NewLoader(), normalizeText(), tailRunes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+20 more)

### Community 184 - "LLMModel"
Cohesion: 0.19
Nodes (4): LLMModel, time.Time, MCPServerStatus, MockLLMModelRepository

### Community 185 - "sandbox_external_test.go"
Cohesion: 0.39
Nodes (14): assertContentLimitedTo(), containsPath(), requireSandbox(), sandboxContext(), sandboxData(), TestSandbox_BrowserScreenshotReturnsPNG(), TestSandbox_BusinessErrorMapsToAPIError(), TestSandbox_FileLifecycle() (+6 more)

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

### Community 194 - "Postgres"
Cohesion: 0.22
Nodes (9): newRepositories(), Postgres, NewPostgres(), AppConfigRepository, NewAppConfigRepository(), NewLLMModelRepository(), NewSessionRepository(), repositories (+1 more)

### Community 195 - "LLMResponse"
Cohesion: 0.06
Nodes (12): blockingStreamingAgentLLM, mockLLM, mockLLMForTest, streamingAgentLLM, LLMRequest, LLMRequest, LLMResponse, streamingStubLLM (+4 more)

### Community 196 - "anthropic_llm.go"
Cohesion: 0.22
Nodes (12): parseJSONMap(), TestAnthropicClient_UserContentTextNotDuplicated(), TestAnthropicClient_UserContentTextOnlyUsesContentText(), textOf(), userContent(), AnthropicContent, AnthropicImageSource, AnthropicMessage (+4 more)

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
Cohesion: 0.17
Nodes (24): NewBochaSearchClientWithTimeout(), NewFallbackSearchClient(), NewSearchEngine(), NewTavilySearchClientWithTimeout(), resolveSearchTimeout(), loadSearchKeys(), requireBochaKey(), requireExternalSearchTests() (+16 more)

### Community 215 - "NewAppConfigService"
Cohesion: 0.36
Nodes (12): NewAppConfigService(), NewMockAppConfigRepository(), TestAppConfigService_DeleteAndUpdateA2AServer(), TestAppConfigService_DeleteMCPServer(), TestAppConfigService_DeleteMCPServer_NotFoundWithoutConfig(), TestAppConfigService_GetConfig_RepositoryError(), TestAppConfigService_UpdateA2AConfigRejectsInvalidURL(), TestAppConfigService_UpdateMCPConfig() (+4 more)

### Community 216 - "Factories"
Cohesion: 0.27
Nodes (10): defaultFactories(), FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(), TestFileCleanupSchedulerStartIsIdempotent(), TestFileCleanupSchedulerStopCancelsCleanupContext() (+2 more)

### Community 217 - "LLMModelRepository"
Cohesion: 0.29
Nodes (8): LLMModelRepository, NewLLMModelServiceWithLLMFactory(), connectionTestModel(), TestLLMModelService_Test_MapsProviderAuthError(), TestLLMModelService_Test_RequiresAPIKey(), TestLLMModelService_Test_ReturnsResponseMetadata(), TestLLMModelService_Test_SendsMinimalRequest(), TestMapModelTestError()

### Community 218 - ".uploadFile"
Cohesion: 0.36
Nodes (4): SandboxClient, newAPIError(), toolResultErr(), net/http.Header

### Community 219 - "encoding/json.RawMessage"
Cohesion: 0.33
Nodes (7): A2AJSONRPCError, A2AJSONRPCResponse, decodeData(), decodeMapQuietly(), SandboxErrorStatus(), encoding/json.RawMessage, envelope

### Community 220 - "Warn"
Cohesion: 0.25
Nodes (6): dialSandboxVNC(), VNCProxy(), SetupRoutes(), Warn(), github.com/gin-gonic/gin.Engine, github.com/gorilla/websocket.Conn

### Community 221 - "HealthStatus"
Cohesion: 0.42
Nodes (5): HealthStatus, HealthState, ServiceName, ServiceStatus, DefaultStatusService

### Community 222 - "openai_llm_test.go"
Cohesion: 0.17
Nodes (27): anthropicErrorResponse(), blockingTransport(), roundTripperFunc, newTransportClient(), newWireCaptureClient(), okChatResponse(), TestOpenAIClient_200WithErrorBodySurfacesUpstreamMessage(), TestOpenAIClient_401_Auth() (+19 more)

### Community 223 - "task_redis_test.go"
Cohesion: 0.13
Nodes (14): mockMQWrapper, mockTaskRunner, retentionCall, TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_FinishDestroysRunner(), TestRedisStreamTask_FinishedChangesAfterRunnerExit() (+6 more)

### Community 224 - "A2AAgentCard"
Cohesion: 0.43
Nodes (4): A2AAgentCapabilities, A2AAgentSkill, A2AInterface, A2AAgentCard

### Community 225 - "runInTx"
Cohesion: 0.53
Nodes (5): pgx.Tx, T, joinRollbackError(), rollbackTx(), runInTx()

### Community 226 - "shutdownCurrent"
Cohesion: 0.33
Nodes (4): shutdownCurrent(), Sync(), go.uber.org/zap/zapcore.WriteSyncer, neverCloseSyncer

### Community 253 - "RedisStreamTask"
Cohesion: 0.09
Nodes (7): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask, TaskRegistryInterface, TaskRunner, Any()

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
- **694 isolated node(s):** `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest`, `A2ATaskParams`, `Task` (+689 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `toolResultErr()` connect `.uploadFile` to `NewToolError`, `types.go`, `context.Context`?**
  _High betweenness centrality (0.021) - this node is a cross-community bridge._
- **Why does `sandboxContext()` connect `sandbox_external_test.go` to `testing.T`, `context.Context`?**
  _High betweenness centrality (0.020) - this node is a cross-community bridge._
- **Why does `Err()` connect `Err` to `FileHandler`, `BuildRuntimeConfigFromModel`, `testing.T`, `StdioMCPClient`, `File`, `SessionHandler`, `A2ATool`, `tool_event_flow_test.go`, `App`, `Event`, `github.com/gin-gonic/gin.Context`, `BadRequest`, `AgentSettings`, `.loadOne`, `event.go`, `LLMRuntimeConfig`, `PlannerReActFlow`, `RedisStreamMessageQueue`, `Info`, `OSS`, `BaseAgent`, `NewToolProvider`, `Warn`, `Plan`, `logger.go`, `.initRoutes`, `run`, `NewToolError`, `RedisStreamTask`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **What connects `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest` to the rest of the system?**
  _694 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `index.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.12561576354679804 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.057902973395931145 - nodes in this community are weakly interconnected._
- **Should `Session` be split into smaller, more focused modules?**
  _Cohesion score 0.047086247086247084 - nodes in this community are weakly interconnected._