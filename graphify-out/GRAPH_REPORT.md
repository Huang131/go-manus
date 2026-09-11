# Graph Report - go-manus  (2026-09-11)

## Corpus Check
- 294 files · ~879,333 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3451 nodes · 8184 edges · 206 communities (178 shown, 28 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 650 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4a5f6a21`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- createSessionForTest
- cn
- NewLLMModelService
- NewAppConfigService
- index.ts
- app_test.go
- queryer
- manus-settings.tsx
- go-manus 前后端端到端 Streaming Review 与重构记录
- testing.T
- chat-input.tsx
- ToolResult
- RoutedLLM
- PostgresSessionRepository
- mooc-manus vs go-manus 功能差异与缺失分析
- go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter
- .loadOne
- Config
- logger_test.go
- lib/utils.ts
- .initRoutes
- TestMessageQueueMockOperations
- Response
- Message
- 2. 根因链：11 个 Bug 拆解
- logger 包 os.Stdout 误关闭 Bug 深度解析
- AnthropicClient
- TestLLM_ModelInfo
- SupervisorService
- compilerOptions
- OpenAIClient
- endpoints/file.py
- App
- endpoints/shell.py
- NewOpenAIClient
- AgentService
- session-detail-view.tsx
- tools_integration_test.go
- NewMockSessionRepository
- external/search.go
- go-manus 项目代码异味（Smell）分析与重构建议
- Err
- Factories
- service_dependencies.py
- run
- tools_test.go
- PlannerReActFlow
- devDependencies
- 代码智能数据使用说明
- BaseAgent
- fallback_llm_test.go
- openai_llm_test.go
- NewProviderError
- dependencies
- components.json
- getJSON
- Postgres
- RedisStreamTask
- json_parser_repair.go
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- 部署指南
- index.tsx
- context.Context
- testLLMModelRepo
- Session
- ShellService
- StdioMCPClient
- LLMModel
- Detailed Findings
- Agent 核心架构
- app_config.go
- event.go
- logger.go
- NewBaseAgent
- Info
- Refactoring with GitNexus
- stubLLM
- go-manus - 通用 AI Agent 系统
- inMemoryMessageQueue
- OSS
- MergeDeltas
- json_benchmark_test.go
- Error
- llm_model.go
- anthropic_llm_test.go
- github.com/gin-gonic/gin.Context
- File
- 集成测试
- task_redis.go
- NewRedisStreamTask
- FileService
- NewDefaultTaskRegistry
- Manus 沙箱服务
- Manus 前端 UI
- DefaultLLMModelService
- Commands
- Python vs Go 版本文件对比报告
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- package.json
- 4.3 11 个 Bug 拆解
- NotFound
- go-manus 面试指南
- DefaultFileCleanupService
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- UnixStreamHTTPConnection
- MCPTool
- 二、Findings（问题清单）
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 七、详细模块对比
- SessionHandler
- button.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- 3.3 Repository 层架构设计
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- 7.4 存储层详细对比
- 7.5 外部依赖详细对比
- LLMResponse
- 学习路径
- PlanSchemaJSON
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- 三、服务层
- 六、差异汇总
- 3.4 集成测试
- 4. 快速启动（nginx 网关模式，推荐）
- mockMQWrapper
- .ExecCommand
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 3. 核心知识点详解
- @novnc/novnc
- 端到端 Streaming Review 计划
- @radix-ui/react-label
- @radix-ui/react-scroll-area
- 5. 注意事项和踩坑点
- 快速部署
- 3.2 PostgreSQL INTERVAL 与参数化查询
- react-markdown
- 3.5 Session 与 Agent 架构
- sonner
- 4. 可能的面试追问
- eslint.config.mjs
- next.config.ts
- postcss.config.mjs
- validConfig
- sandbox
- 2. 核心组件
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- Impact Analysis with GitNexus
- Tool
- NewFileService
- 核心概念
- test-env-down.sh
- test-env-up.sh
- github.com/Huang131/go-manus/api
- status_service_test.go
- 八、改进建议
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- next-themes
- @radix-ui/react-avatar
- BaseModel
- NewSessionHandler
- @radix-ui/react-slot
- VNCProxy
- @radix-ui/react-switch
- react-dom
- remark-gfm
- tailwind-merge
- SearchResults
- app.go
- 10. 故障排查
- Review 发现
- progress.md
- AppConfigHandler
- session_test.go
- runInTx
- fakeRedis
- BrowserTool
- FileTool
- BuildAttachmentContextSection
- SearchTool
- ShellTool
- shutdownCurrent
- NewMessageTool
- NewToolError
- NewSimpleMemory

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 102 edges
3. `File` - 57 edges
4. `Err()` - 57 edges
5. `RedisStreamTask` - 46 edges
6. `NewRedisStreamTask()` - 44 edges
7. `queryer` - 38 edges
8. `Info()` - 38 edges
9. `Session` - 37 edges
10. `createSessionForTest()` - 36 edges

## Surprising Connections (you probably didn't know these)
- `TestShouldInlineAndTruncate()` --calls--> `ShouldRAG()`  [INFERRED]
  api/internal/agent/attachment/loader_test.go → api/internal/agent/attachment/policy.go
- `BaseAgent` --references--> `Memory`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewToolRegistry()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/tools.go
- `NewPlannerAgent()` --calls--> `NewBaseAgent()`  [INFERRED]
  api/internal/agent/planner_agent.go → api/internal/agent/base.go

## Import Cycles
- None detected.

## Communities (206 total, 28 thin omitted)

### Community 0 - "createSessionForTest"
Cohesion: 0.16
Nodes (27): deleteSessionDirect(), testFileRepo(), TestFileRepo_CountExpiredFiles(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound() (+19 more)

### Community 1 - "cn"
Cohesion: 0.05
Nodes (59): metadata, GlobalHeader(), LeftPanel(), ManusSettings(), DropdownMenuCheckboxItem(), DropdownMenuItem(), DropdownMenuLabel(), DropdownMenuRadioItem() (+51 more)

### Community 2 - "NewLLMModelService"
Cohesion: 0.19
Nodes (23): NewLLMModelService(), NewMockLLMModelRepository(), TestLLMModelService_Create(), TestLLMModelService_Create_MapsUniqueViolationToConflict(), TestLLMModelService_Create_PreservesDefaultLookupError(), TestLLMModelService_Create_PreservesPartialCapabilities(), TestLLMModelService_Create_PreservesRepositoryError(), TestLLMModelService_Create_Validation() (+15 more)

### Community 3 - "NewAppConfigService"
Cohesion: 0.13
Nodes (23): AppConfig, AppConfigType, pgx.Tx, AppConfigRepository, marshalConfigValue(), NewAppConfigRepository(), NewAppConfigRepositoryWithTx(), NewAppConfigService() (+15 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (57): useSessionDetail(), UseSessionDetailResult, configApi, API_CONFIG, ApiError, createSSEConnection(), createSSEStream(), del() (+49 more)

### Community 5 - "app_test.go"
Cohesion: 0.09
Nodes (28): Build(), newLifecycleManager(), normalizeFactories(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce(), TestBuildInitializeErrorWrapsSentinel() (+20 more)

### Community 6 - "queryer"
Cohesion: 0.13
Nodes (11): pgx.Tx, scanFile(), collectRows(), pgx.Tx, T, newQueryer(), scanSessionSummary(), pgx.Rows (+3 more)

### Community 7 - "manus-settings.tsx"
Cohesion: 0.07
Nodes (39): DeleteSessionDialogProps, A2ASettingProps, CommonSettingProps, LLMSettingProps, MCPSettingProps, SETTING_MENUS, SettingTab, emptyDraft (+31 more)

### Community 8 - "go-manus 前后端端到端 Streaming Review 与重构记录"
Cohesion: 0.06
Nodes (33): 1. 结论, 2.1 后端任务事件链路, 2.2 前端续读链路, 2.3 Provider streaming 链路, 2. 当前链路核对, 3. 已确认问题, 4. 已处理的 P2 问题, 5. 仍待处理的维护性建议 (+25 more)

### Community 9 - "testing.T"
Cohesion: 0.04
Nodes (76): TestFlowStatus_Values(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters(), TestAgentService_GetActiveTaskIDClearsCompletedTask(), TestAgentService_ResolveMessageAttachments() (+68 more)

### Community 10 - "chat-input.tsx"
Cohesion: 0.08
Nodes (37): ChatInput, ChatInputProps, ChatInputRef, DeleteSessionDialog(), SessionItem(), SessionItemProps, SessionList(), Avatar() (+29 more)

### Community 11 - "ToolResult"
Cohesion: 0.06
Nodes (8): attachmentSandbox, MessageTool, requiredToolString(), ToolResult, NewToolResult(), NewToolResultWithMessage(), MockBrowser, MockSandbox

### Community 12 - "RoutedLLM"
Cohesion: 0.10
Nodes (23): defaultLLMClientFactory(), LLM, NewDynamicLLM(), NewDynamicLLMWithFactory(), WithModelID(), betterHealth(), canFallbackAfterToolUse(), configKey() (+15 more)

### Community 13 - "PostgresSessionRepository"
Cohesion: 0.10
Nodes (11): stopSessionRepository, successfulStopSessionRepository, TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits(), TestAgentService_StopSessionReturnsStatusUpdateError(), pgx.Tx, SessionRepository, marshalSessionEvents(), NewSessionRepository() (+3 more)

### Community 14 - "mooc-manus vs go-manus 功能差异与缺失分析"
Cohesion: 0.05
Nodes (40): 2.1 整体路由映射, 2.2 **仍然缺失的关键路由**, 3.1 Planner 阶段: JSON 强制输出 ✅ Go 更优, 3.2 **空响应重试注入 (重要差异)**, 3.3 LLM 错误重试策略, 3.4 ReAct 阶段的 JSON 约束, 3.5 流式事件分发, 3.6 Agent 任务执行架构 (+32 more)

### Community 15 - "go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter"
Cohesion: 0.05
Nodes (40): Adapter 接口, AnthropicAdapter, CustomGatewayAdapter, go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter, LLM 控制面, ModelHint, OpenAICompatibleAdapter, Provider Adapter (+32 more)

### Community 16 - ".loadOne"
Cohesion: 0.09
Nodes (29): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+21 more)

### Community 17 - "Config"
Cohesion: 0.09
Nodes (23): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, RedisConfig, SandboxConfig (+15 more)

### Community 18 - "logger_test.go"
Cohesion: 0.16
Nodes (22): GetLevel(), Init(), InitWithConfig(), SetLevel(), stdoutWriteOK(), TestCallerInBusinessCode(), testCallerLocation(), TestGetLevel_DefaultWhenNotInitialized() (+14 more)

### Community 19 - "lib/utils.ts"
Cohesion: 0.10
Nodes (26): AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessageProps, StepBlock(), ToolRow(), FilePreviewPanel(), FilePreviewPanelProps (+18 more)

### Community 20 - ".initRoutes"
Cohesion: 0.20
Nodes (17): defaultTrustedProxies(), CORS(), generateRequestID(), Logger(), Recovery(), RequestID(), setupTestEngine(), TestCORS_AllowedHeader() (+9 more)

### Community 21 - "TestMessageQueueMockOperations"
Cohesion: 0.27
Nodes (5): TestMessageQueueDefaultTimeouts(), TestMessageQueueGetBlockingHonorsContext(), TestMessageQueueInterface(), TestMessageQueueMockOperations(), mockMQ

### Community 22 - "Response"
Cohesion: 0.15
Nodes (27): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+19 more)

### Community 23 - "Message"
Cohesion: 0.10
Nodes (15): SimpleMemory, Message, ResponseFormat, ToolCall, ToolSpec, sync.RWMutex, AudioURL, ContentPart (+7 more)

### Community 24 - "2. 根因链：11 个 Bug 拆解"
Cohesion: 0.06
Nodes (30): 1. 现象回顾, 2.1 模型能力对比（当前项目已测模型）, 2. 根因链：11 个 Bug 拆解, 3.1 简单问答, 3.2 带工具调用的完整任务（回归 Bug 9 场景）, 3. 修复后验证（端到端）, 4. 原项目 mooc-manus (Python) 是否存在这些问题？, 5.1 协议层的"两次包装" (+22 more)

### Community 25 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 26 - "AnthropicClient"
Cohesion: 0.16
Nodes (13): LLMRequest, mustMarshalMap(), parseJSONMap(), textOf(), userContent(), AnthropicClient, AnthropicContent, AnthropicImageSource (+5 more)

### Community 28 - "SupervisorService"
Cohesion: 0.12
Nodes (10): Exception, AppException, Any, 使用python的xml-rpc客户端连接一个本地sock文件文件实现连接rpc服务, 获取当前supervisor管理的所有进程信息, 传递指定分钟，并激活定时销毁任务同时关闭自动保活, 传递指定的时长，延长超时销毁的时间，单默认延长3分钟, 构造函数，完成supervisor服务链接 (+2 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "OpenAIClient"
Cohesion: 0.13
Nodes (19): estimateCostUSD(), LLMRequest, reasoningTokens(), sendStreamDelta(), toOpenAIMessages(), toOpenAITools(), truncateBody(), LLMDelta (+11 more)

### Community 31 - "endpoints/file.py"
Cohesion: 0.16
Nodes (33): FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get, post (+25 more)

### Community 32 - "App"
Cohesion: 0.18
Nodes (9): BuildWithFactories(), ModelIDFromContext(), NewBrowserClient(), Warn(), App, lifecycleManager, Options, sync.Mutex (+1 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.19
Nodes (24): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+16 more)

### Community 34 - "NewOpenAIClient"
Cohesion: 0.13
Nodes (19): NewOpenAIClient(), newTestClient(), TestNewOpenAIClient_DoesNotApplyGlobalTimeout(), BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), runtimeConfigToOpenAIClientConfig(), runtimeHealthFromModel() (+11 more)

### Community 35 - "AgentService"
Cohesion: 0.19
Nodes (9): A2AConfig, AgentService, AgentConfig, COSFileStorage, MCPConfig, NewAgentService(), nonNilEvents(), Sandbox (+1 more)

### Community 36 - "session-detail-view.tsx"
Cohesion: 0.11
Nodes (30): PageProps, ChatMessage(), findLatestTool(), SessionDetailView(), SessionDetailViewProps, A2APreview(), BrowserPreview(), ConsoleRecord (+22 more)

### Community 37 - "tools_integration_test.go"
Cohesion: 0.30
Nodes (11): NewShellTool(), TestShellTool_Description(), TestShellTool_Invoke_Error(), TestShellTool_Invoke_Exec(), TestShellTool_Invoke_Kill(), TestShellTool_Invoke_Read(), TestShellTool_Invoke_UnknownAction(), TestShellTool_Invoke_Wait() (+3 more)

### Community 38 - "NewMockSessionRepository"
Cohesion: 0.19
Nodes (22): SessionService, NewSessionService(), NewSessionServiceWithSandbox(), newMockFileRepo(), NewMockSessionRepository(), TestSessionService_AppendEvent(), TestSessionService_ClearUnreadCount(), TestSessionService_CreateSession() (+14 more)

### Community 39 - "external/search.go"
Cohesion: 0.24
Nodes (14): SearchEngine, NewBingSearchClient(), NewBingSearchClientWithTimeout(), NewGoogleSearchClient(), NewGoogleSearchClientWithTimeout(), NewSearchEngine(), TestSearchClients_InvalidTimeoutUsesDefault(), TestSearchClients_UseConfiguredTimeout() (+6 more)

### Community 40 - "go-manus 项目代码异味（Smell）分析与重构建议"
Cohesion: 0.09
Nodes (22): 4.1 🔴 C1~C5：`json_parser_repair.go` 的 Bug 与 Dead Code, 4.2 🟡 W1：`AgentService` God Object, 4.3 🟡 W6/W7：`PlannerReActFlow` 与 `TaskRunner` 的 Long Method + Switch, 4.4 🟡 W9：`SessionRepository` Fat Interface, 🔴 Critical（必须立刻处理）, go-manus 项目代码异味（Smell）分析与重构建议, Immediate（本周，1~2 天）, Long-Term（1 个月+） (+14 more)

### Community 41 - "Err"
Cohesion: 0.11
Nodes (14): AgentTaskRunner, AgentTaskRunnerConfig, COSFileStorage, readerWrapper, AgentConfig, COSFileStorage, NewAgentTaskRunner(), RedisStreamMessageQueue (+6 more)

### Community 42 - "Factories"
Cohesion: 0.20
Nodes (13): defaultFactories(), COSFileStorage, FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(), TestFileCleanupSchedulerStartIsIdempotent() (+5 more)

### Community 43 - "service_dependencies.py"
Cohesion: 0.13
Nodes (16): APIRouter, BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, create_api_routes() (+8 more)

### Community 44 - "run"
Cohesion: 0.31
Nodes (7): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), net/http.Server, Config

### Community 45 - "tools_test.go"
Cohesion: 0.06
Nodes (37): A2ATool, mockTool, A2AConfig, marshalResponseText(), NewA2ATool(), TestA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup() (+29 more)

### Community 46 - "PlannerReActFlow"
Cohesion: 0.24
Nodes (8): FlowStatus, PlannerReActFlow, TaskInput, BaseEvent, NewErrorEvent(), NewMessageEvent(), ErrorContext(), InfoContext()

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.11
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "BaseAgent"
Cohesion: 0.14
Nodes (9): BaseAgent, InvokeResult, ToolCallResult, AgentConfig, NewBaseAgentWithParser(), shouldPublishDeltas(), JSONParser, NewDefaultJSONParser() (+1 more)

### Community 50 - "fallback_llm_test.go"
Cohesion: 0.15
Nodes (26): RoutedLLM, LLMRequest, modelNames(), newMockRuntimeHealthStore(), openAITextProfile(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_DoNotFallbackAcrossProtocol(), TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect() (+18 more)

### Community 51 - "openai_llm_test.go"
Cohesion: 0.17
Nodes (20): newTransportClient(), rawOK(), responseJSON(), TestOpenAIClient_401_Auth(), TestOpenAIClient_429_RateLimit(), TestOpenAIClient_5xx_Server(), TestOpenAIClient_ContentAndReasoning_Separated(), TestOpenAIClient_ContentOnly() (+12 more)

### Community 52 - "NewProviderError"
Cohesion: 0.10
Nodes (29): ErrorKind, isFallbackable(), isRetryable(), NewProviderError(), CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), ContextFits() (+21 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "getJSON"
Cohesion: 0.12
Nodes (54): Response, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_EmptyUpdate(), TestAppConfigAPI_InvalidJSON(), TestAppConfigAPI_LLMConfig_Lifecycle(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle() (+46 more)

### Community 56 - "Postgres"
Cohesion: 0.13
Nodes (12): StatusHandler, NewStatusHandler(), Postgres, NewPostgres(), Redis, redis.Client, NewRedis(), NewLLMModelRepository() (+4 more)

### Community 57 - "RedisStreamTask"
Cohesion: 0.07
Nodes (8): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask, Stream, Task, CompletedStreamRetention(), Debug()

### Community 58 - "json_parser_repair.go"
Cohesion: 0.23
Nodes (10): extractJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), removeTrailingCommas(), TestDefaultJSONParser(), TestExtractJSON(), TestRemoveMarkdownCodeBlocks(), TestRemoveTrailingCommas() (+2 more)

### Community 59 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 60 - "部署指南"
Cohesion: 0.15
Nodes (13): 1. 部署方式, 2. 架构总览, 3. 前置要求, 5.1 `.env` 模板（`api/.env.example`）, 5.2 必须在 `.env` 中补充的变量, 5. 环境变量配置, 6.1 通过 Makefile, 6.2 直接使用 docker-compose (+5 more)

### Community 61 - "index.tsx"
Cohesion: 0.13
Nodes (19): A2aTool(), A2aToolProps, BashTool(), BashToolProps, BrowserTool(), BrowserToolProps, DefaultTool(), DefaultToolProps (+11 more)

### Community 62 - "context.Context"
Cohesion: 0.05
Nodes (12): attachmentStorage, chatContextSessionRepository, mockSandbox, TestAgentServiceChatUsesDetachedContextForMessagePersistence(), DefaultFileService, mockMessageQueue, context.Context, io.Reader (+4 more)

### Community 63 - "testLLMModelRepo"
Cohesion: 0.29
Nodes (19): cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError(), TestLLMModelRepo_GetByID_NotFoundReturnsNilNil(), TestLLMModelRepo_GetDefault_NoDefaultReturnsNilNil() (+11 more)

### Community 64 - "Session"
Cohesion: 0.06
Nodes (9): mockSessionRepo, Event, Session, SessionStatus, unixPointer(), time.Time, MockSessionServiceForHandler, DefaultSessionService (+1 more)

### Community 65 - "ShellService"
Cohesion: 0.17
Nodes (8): ConsoleRecord, Process, 根据传递的会话id+是否输出控制台记录获取Shell命令结果, 传递会话id+执行目录+命令在沙箱中执行后返回, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程, 启动协程以连续读取进程输出并将其存储到会话中, ShellService

### Community 66 - "StdioMCPClient"
Cohesion: 0.12
Nodes (17): MCPClientManager, MCPToolInfo, NewStdioMCPClient(), MCPClient, MCPConfig, MCPConfigServer, MCPContent, MCPError (+9 more)

### Community 67 - "LLMModel"
Cohesion: 0.14
Nodes (11): LLMModel, pgx.Tx, LLMModelRepository, modelJSON(), modelJSONFields(), NewLLMModelRepositoryWithTx(), runtimeHealthToJSON(), scanLLMModel() (+3 more)

### Community 68 - "Detailed Findings"
Cohesion: 0.10
Nodes (20): Architecture Smell Report, Detailed Findings, Executive Summary, Historical Findings Status, Immediate Actions, Long Term, Module Health Scorecard, P0: 当前工作树 logger 测试回归 (+12 more)

### Community 69 - "Agent 核心架构"
Cohesion: 0.20
Nodes (10): 1. 整体架构, 3. Memory 系统, 4. 会话管理, 5. 事件驱动, 6. 代码结构, 7.1 添加新工具, 7.2 自定义 Agent, 7.3 集成新 LLM (+2 more)

### Community 70 - "app_config.go"
Cohesion: 0.10
Nodes (20): MCPServer, A2AConfig, AgentConfig, HealthStatus, LLMConfig, MCPConfig, NewLLMConfigResponse(), TestLLMConfigRequest_MarshalJSON() (+12 more)

### Community 71 - "event.go"
Cohesion: 0.06
Nodes (33): TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), clonePlanSteps(), cloneStrings(), PlanStep, NewDoneEvent(), NewPlanEvent(), NewStepEvent(), NewTitleEvent() (+25 more)

### Community 72 - "logger.go"
Cohesion: 0.16
Nodes (21): Any(), Bool(), Dur(), Error(), Fatal(), Float64(), Get(), Int() (+13 more)

### Community 73 - "NewBaseAgent"
Cohesion: 0.17
Nodes (13): A2AAgent, AgentConfig, MCPServer, ReActAgent, NewBaseAgent(), TestBaseAgentInvokePublishesTextDeltas(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(), DefaultAgentConfig() (+5 more)

### Community 74 - "Info"
Cohesion: 0.24
Nodes (5): newSSEContext(), Info(), context.CancelFunc, sync.WaitGroup, FileCleanupScheduler

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "stubLLM"
Cohesion: 0.31
Nodes (4): LLMRequest, TestDynamicLLM_UsesFactory(), TestRoutedLLMFromSingleProvider(), stubLLM

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.17
Nodes (9): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 健康检查, 常用命令, 本地开发, 核心代码指引, 许可证 (+1 more)

### Community 78 - "inMemoryMessageQueue"
Cohesion: 0.12
Nodes (9): inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, mockFailingTool, newInMemoryMessageQueue(), parseStreamSeq(), TestToolCallingEvents_SSEStream() (+1 more)

### Community 79 - "OSS"
Cohesion: 0.19
Nodes (11): ObjectStorageConfig, TestBuildWithFactoriesInjectsFileCleanupScheduler(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), github.com/aws/aws-sdk-go-v2/feature/s3/manager.Uploader (+3 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (14): isToolCallFinishReason(), MergeDeltas(), NormalizeResponse(), repeat(), TestEstimateContextTokens(), TestMergeDeltas_AnthropicToolUseRetainsToolCall(), TestMergeDeltas_ContentOnly(), TestMergeDeltas_MultipleToolCalls() (+6 more)

### Community 81 - "json_benchmark_test.go"
Cohesion: 0.22
Nodes (15): Benchmark_Sonic_Marshal_Large(), Benchmark_Sonic_Marshal_Small(), Benchmark_Sonic_Roundtrip_Large(), Benchmark_Sonic_Roundtrip_Small(), Benchmark_Sonic_Unmarshal_Large(), Benchmark_Sonic_Unmarshal_Small(), Benchmark_StdLib_Marshal_Large(), Benchmark_StdLib_Marshal_Small() (+7 more)

### Community 82 - "Error"
Cohesion: 0.20
Nodes (12): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Internal(), New(), ToInternal(), Unauthorized() (+4 more)

### Community 83 - "llm_model.go"
Cohesion: 0.16
Nodes (14): DefaultCapabilities(), CostPolicy, RuntimeHealth, ModelCapabilities, RequestPolicy, MergeDefaultCapabilities(), encoding/json.RawMessage, CostPolicy (+6 more)

### Community 84 - "anthropic_llm_test.go"
Cohesion: 0.27
Nodes (18): NewAnthropicClient(), anthropicErrorResponse(), errKind(), newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_NetworkError(), TestAnthropicClient_ProtocolError_NotFallbackable() (+10 more)

### Community 85 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.17
Nodes (10): LLMModelHandler, NewLLMModelHandler(), NewLLMModelResponse(), LLMModelService, FromError(), TotalResponse, Success(), SuccessWithMsg() (+2 more)

### Community 86 - "File"
Cohesion: 0.05
Nodes (10): attachmentFileRepository, generatedFileRepository, File, FileRepository, NewFileRepository(), NewFileRepositoryWithTx(), cleanupRepoStub, emptyFileRepo (+2 more)

### Community 87 - "集成测试"
Cohesion: 0.14
Nodes (13): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+5 more)

### Community 88 - "task_redis.go"
Cohesion: 0.17
Nodes (14): TaskRegistryInterface, TaskRunner, TaskStream, NewTaskStream(), parseTaskOutputMessages(), ReadTaskOutput(), readTaskOutput(), streamDataString() (+6 more)

### Community 89 - "NewRedisStreamTask"
Cohesion: 0.16
Nodes (21): mockTaskRunner, NewRedisStreamTask(), TestReadTaskOutputAfterTaskUnregistered(), TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_FinishDestroysRunner(), TestRedisStreamTask_FinishedChangesAfterRunnerExit() (+13 more)

### Community 90 - "FileService"
Cohesion: 0.13
Nodes (8): NotFoundException, Any, FileService, UploadFile, 根据传递的文件路径+内容向指定文件写入内容, 根据传递的文件路径+匹配规则查询文件内符合的内容, 根据传递的文件夹路径+glob规则查询文件列表, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容

### Community 91 - "NewDefaultTaskRegistry"
Cohesion: 0.24
Nodes (14): NewDefaultTaskRegistry(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_CleanupCompletedWaitsForFinished(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister(), TestDefaultTaskRegistry_List() (+6 more)

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.20
Nodes (9): API 接口, Docker 部署, Manus 沙箱服务, 使用开发容器, 启动服务, 技术栈, 本地开发, 架构 (+1 more)

### Community 93 - "Manus 前端 UI"
Cohesion: 0.20
Nodes (9): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 环境准备 (+1 more)

### Community 94 - "DefaultLLMModelService"
Cohesion: 0.33
Nodes (3): isUniqueViolation(), validateRequired(), DefaultLLMModelService

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 97 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 98 - "package.json"
Cohesion: 0.22
Nodes (8): name, private, scripts, build, dev, lint, start, version

### Community 99 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 100 - "NotFound"
Cohesion: 0.13
Nodes (12): NotFound(), FileHandler, NewFileHandler(), DefaultFileService, FileService, TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure() (+4 more)

### Community 101 - "go-manus 面试指南"
Cohesion: 0.22
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 103 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 104 - "6. 代码评审与回归测试：测试不是覆盖，而是契约"
Cohesion: 0.22
Nodes (9): 6.1 Why：测试是行为的契约, 6.2 历史 Bug 的回归测试, 6.3 测试分类, 6.4 Go 测试模式, 6.5 面试要点, 6. 代码评审与回归测试：测试不是覆盖，而是契约, 子测试 + t.Helper, 模糊测试（Go 1.18+） (+1 more)

### Community 105 - "UnixStreamHTTPConnection"
Cohesion: 0.25
Nodes (4): HTTPConnection, 重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接, UnixStreamHTTPConnection, UnixStreamTransport

### Community 106 - "MCPTool"
Cohesion: 0.14
Nodes (8): MCPTool, MCPConfig, NewMCPTool(), parseMCPInvokeParams(), TestMCPToolInvokeAllowsMissingParams(), TestMCPToolInvokeValidatesParameters(), TestMCPTool(), TestMCPTool_WithConfig()

### Community 107 - "二、Findings（问题清单）"
Cohesion: 0.18
Nodes (10): 🔴 A. 正确性, 🟡 B. 重复代码, 🟡 C. 长方法 / 复杂度, 🔵 D. 分层 / 架构, 🔵 E. 其他, go-manus `api/` 代码 Review 与重构报告, 一、Executive Summary, 三、Refactoring 执行记录（本次已落地） (+2 more)

### Community 108 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 109 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 110 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 111 - "SessionHandler"
Cohesion: 0.22
Nodes (10): SessionHandler, mergeEventMetadata(), parseChatRequest(), setSSEHeaders(), TestMergeEventMetadata(), TestMergeEventMetadata_NullPayloadReturnsOriginalData(), TestSetSSEHeaders(), TestWriteSSEEventIncludesStreamCursor() (+2 more)

### Community 112 - "button.tsx"
Cohesion: 0.14
Nodes (14): PlanPanel(), PlanPanelProps, SuggestedQuestions(), SuggestedQuestionsProps, Button(), buttonVariants, buildVNCUrl(), VNCOverlay() (+6 more)

### Community 113 - "go-manus 实战经验与面试考点沉淀（2026-09-04）"
Cohesion: 0.25
Nodes (7): 7.1 资深 Go 后端, 7.2 系统设计, 7.3 工程实践, 7. 综合面试题, 8. 参考资料, go-manus 实战经验与面试考点沉淀（2026-09-04）, 目录

### Community 114 - "3. Docker Desktop 替换：macOS 容器运行时选型"
Cohesion: 0.25
Nodes (8): 3.1 现象, 3.2 What, 3.3 主流方案对比, 3.4 推荐方案：Colima, 3.5 迁移关键点, 3.6 资源分配建议（16GB MacBook）, 3.7 面试要点, 3. Docker Desktop 替换：macOS 容器运行时选型

### Community 115 - "3.3 Repository 层架构设计"
Cohesion: 0.29
Nodes (7): 3.3 Repository 层架构设计, WithTx 事务封装, 事务支持：QueryContext 接口, 接口设计, 构造函数设计：普通版 vs 事务版, 软删除模式, 面试追问

### Community 116 - "五、外部依赖"
Cohesion: 0.40
Nodes (5): 5.1 LLM 调用, 5.2 消息队列, 5.3 Sandbox 沙箱, 5.4 MCP 客户端, 五、外部依赖

### Community 117 - "7.1 Memory 记忆模块详细对比"
Cohesion: 0.40
Nodes (5): 7.1.1 接口设计对比, 7.1.2 核心差异分析, 7.1.3 压缩逻辑深度对比, 7.1.4 面试分析点, 7.1 Memory 记忆模块详细对比

### Community 118 - "7.4 存储层详细对比"
Cohesion: 0.40
Nodes (5): 7.4.1 Repository 接口设计, 7.4.2 数据库 Schema 对比, 7.4.3 事务处理对比, 7.4.4 面试分析点, 7.4 存储层详细对比

### Community 119 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 120 - "LLMResponse"
Cohesion: 0.18
Nodes (8): mockLLM, streamingAgentLLM, LLMRequest, LLMRequest, LLMResponse, NewMessageDeltaEvent(), NewMessageDoneEvent(), TestMessageStreamingEvents()

### Community 121 - "学习路径"
Cohesion: 0.40
Nodes (5): 学习路径, 第一阶段：理解 Agent 基础, 第三阶段：理解外部集成, 第二阶段：掌握工具系统, 第四阶段：部署和扩展

### Community 123 - "5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异"
Cohesion: 0.29
Nodes (7): 5.1 现象, 5.2 What, 5.3 sonic vs encoding/json 差异, 5.4 sonic 正确使用姿势, 5.5 替换 JSON 库的检查清单, 5.6 面试要点, 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### Community 124 - "三、服务层"
Cohesion: 0.50
Nodes (4): 3.1 AgentService, 3.2 SessionService, 3.3 FileService, 三、服务层

### Community 125 - "六、差异汇总"
Cohesion: 0.50
Nodes (4): 6.1 功能缺失列表, 6.2 实现差异列表, 6.3 架构差异, 六、差异汇总

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "4. 快速启动（nginx 网关模式，推荐）"
Cohesion: 0.50
Nodes (4): 4.1 准备环境变量, 4.2 一键启动, 4.3 访问入口（统一走 nginx）, 4. 快速启动（nginx 网关模式，推荐）

### Community 128 - "mockMQWrapper"
Cohesion: 0.15
Nodes (6): batchTaskOutputMQ, mockMQWrapper, retentionCall, StreamMessage, BatchMessageQueue, MessageQueue

### Community 130 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 131 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 133 - "端到端 Streaming Review 计划"
Cohesion: 0.33
Nodes (5): 目标, 端到端 Streaming Review 计划, 约束, 结果, 阶段

### Community 136 - "5. 注意事项和踩坑点"
Cohesion: 0.33
Nodes (6): 5.1 Go 错误处理, 5.2 PostgreSQL 参数化, 5.3 Repository 设计, 5.4 软删除查询, 5.5 文件清理, 5. 注意事项和踩坑点

### Community 137 - "快速部署"
Cohesion: 0.33
Nodes (6): 一键部署（推荐）, 前置要求, 容器列表, 快速部署, 服务架构（Nginx 网关模式）, 本地开发

### Community 138 - "3.2 PostgreSQL INTERVAL 与参数化查询"
Cohesion: 0.40
Nodes (5): 3.2 PostgreSQL INTERVAL 与参数化查询, Why：为什么 INTERVAL 不能参数化？, 问题背景, 面试追问, 项目中的正确实现

### Community 140 - "3.5 Session 与 Agent 架构"
Cohesion: 0.40
Nodes (5): 3.5 Session 与 Agent 架构, Agent Memory 机制, AppendEvent 业务逻辑, MessageEvent 结构, 面试追问

### Community 142 - "4. 可能的面试追问"
Cohesion: 0.40
Nodes (5): 4.1 Go 语言层面, 4.2 数据库层面, 4.3 架构设计层面, 4.4 测试层面, 4. 可能的面试追问

### Community 148 - "validConfig"
Cohesion: 0.53
Nodes (5): TestConfigApplyDefaultsUsesDomainDefaults(), TestValidate_OK(), TestValidate_RequiredFields(), validConfig(), Config

### Community 157 - "2. 核心组件"
Cohesion: 0.40
Nodes (5): 2.1 BaseAgent, 2.2 PlannerAgent, 2.3 ReActAgent, 2.4 Flow 编排, 2. 核心组件

### Community 158 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 159 - "Impact Analysis with GitNexus"
Cohesion: 0.22
Nodes (8): Checklist, Example: "What breaks if I change validateUser?", Impact Analysis with GitNexus, Risk Assessment, Tools, Understanding Output, When to Use, Workflow

### Community 160 - "Tool"
Cohesion: 0.12
Nodes (11): mockLLMForTest, MultiFunctionTool, PlannerAgent, Tool, ToolRegistry, AgentConfig, NewPlannerAgent(), AgentConfig (+3 more)

### Community 161 - "NewFileService"
Cohesion: 0.26
Nodes (13): NewFileService(), TestFileServiceDeleteFileDeletesStorageAndRepo(), TestFileServiceDeleteFileRepoError(), TestFileServiceDeleteFileStorageError(), TestFileServiceDownloadFileNilStorageReturnsNilReader(), TestFileServiceDownloadFileStorageError(), TestFileServiceDownloadFileWithStorage(), TestFileServiceGetFileInfoFound() (+5 more)

### Community 162 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

### Community 171 - "status_service_test.go"
Cohesion: 0.25
Nodes (5): TestStatusService_GetHealthStatus_SkippedServicesDegraded(), TestStatusService_GetHealthStatus_WrapsHealthErrors(), TestStatusServiceUsesInjectedClock(), TestToInternal(), fakePostgres

### Community 172 - "八、改进建议"
Cohesion: 0.50
Nodes (4): 中优先级, 低优先级, 八、改进建议, 高优先级

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

### Community 181 - "NewSessionHandler"
Cohesion: 0.38
Nodes (14): NewSessionHandler(), NewMockSessionServiceForHandler(), setupRouter(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), TestSessionHandler_Chat(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create(), TestSessionHandler_Delete() (+6 more)

### Community 183 - "VNCProxy"
Cohesion: 0.17
Nodes (13): dialSandboxVNC(), sameOrigin(), newLocalTestServer(), TestVNCProxy_AllowsSameOriginAndNonBrowserClient(), TestVNCProxy_ProxyEcho(), TestVNCProxy_RejectsCrossOrigin(), TestVNCProxy_ServiceError(), VNCProxy() (+5 more)

### Community 188 - "SearchResults"
Cohesion: 0.67
Nodes (3): SearchEvent, SearchResultItem, SearchResults

### Community 189 - "app.go"
Cohesion: 0.19
Nodes (11): A2AAgent, A2AConfig, MCPConfig, MCPServer, DefaultOptions(), newA2AConfig(), newMCPConfig(), newRepositories() (+3 more)

### Community 190 - "10. 故障排查"
Cohesion: 0.50
Nodes (4): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查

### Community 191 - "Review 发现"
Cohesion: 0.50
Nodes (3): Review 发现, 初步证据（2026-09-11）, 进一步核验

### Community 193 - "AppConfigHandler"
Cohesion: 0.27
Nodes (6): getConfig(), AppConfigHandler, T, NewAppConfigHandler(), updateConfig(), AppConfigService

### Community 194 - "session_test.go"
Cohesion: 0.31
Nodes (10): parseResponseDataAsArray(), parseTotalResponse(), createSessionsForTest(), TestSessionAPI_CreateAndList(), TestSessionAPI_DeleteNotFound(), TestSessionAPI_GetAllSessions(), TestSessionAPI_GetInvalidID(), TestSessionAPI_GetNotFound() (+2 more)

### Community 195 - "runInTx"
Cohesion: 0.33
Nodes (6): pgx.Tx, T, joinRollbackError(), logRollbackFailure(), rollbackTx(), runInTx()

### Community 197 - "BrowserTool"
Cohesion: 0.29
Nodes (3): BrowserTool, NewBrowserTool(), TestToolValidationRejectsMissingRequiredParameters()

### Community 198 - "FileTool"
Cohesion: 0.29
Nodes (3): FileTool, NewFileTool(), TestFileTool()

### Community 199 - "BuildAttachmentContextSection"
Cohesion: 0.32
Nodes (5): BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext()

### Community 202 - "shutdownCurrent"
Cohesion: 0.33
Nodes (4): shutdownCurrent(), Sync(), go.uber.org/zap/zapcore.WriteSyncer, neverCloseSyncer

### Community 203 - "NewMessageTool"
Cohesion: 0.50
Nodes (3): TestMessageTool_NotifyUserSchema(), NewMessageTool(), TestMessageTool()

### Community 220 - "NewToolError"
Cohesion: 0.25
Nodes (7): SandboxClient, NewSandboxClient(), sandboxToolResult(), NewToolError(), sandboxErrorResponse, sandboxRequest, sandboxResponse

### Community 233 - "NewSimpleMemory"
Cohesion: 0.36
Nodes (6): Memory, NewSimpleMemory(), TestSimpleMemory_Add(), TestSimpleMemory_Clear(), TestSimpleMemory_Compact(), TestSimpleMemory_GetMessages()

## Knowledge Gaps
- **643 isolated node(s):** `LLMConfig`, `SearchConfig`, `MCPServer`, `A2AAgent`, `github.com/Huang131/go-manus/api` (+638 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Event` connect `Session` to `AgentService`, `event.go`, `PostgresSessionRepository`, `SessionHandler`, `llm_model.go`, `task_redis.go`, `RedisStreamTask`, `context.Context`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Why does `Err()` connect `Err` to `.ExecCommand`, `ToolResult`, `RoutedLLM`, `.loadOne`, `.initRoutes`, `App`, `AgentService`, `run`, `tools_test.go`, `PlannerReActFlow`, `BaseAgent`, `VNCProxy`, `RedisStreamTask`, `context.Context`, `StdioMCPClient`, `runInTx`, `event.go`, `logger.go`, `Info`, `OSS`, `github.com/gin-gonic/gin.Context`, `DefaultFileCleanupService`, `MCPTool`, `SessionHandler`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Why does `EventType` connect `event.go` to `Session`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **What connects `LLMConfig`, `SearchConfig`, `MCPServer` to the rest of the system?**
  _643 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.05126582278481013 - nodes in this community are weakly interconnected._
- **Should `NewAppConfigService` be split into smaller, more focused modules?**
  _Cohesion score 0.13213213213213212 - nodes in this community are weakly interconnected._
- **Should `index.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.07925407925407925 - nodes in this community are weakly interconnected._