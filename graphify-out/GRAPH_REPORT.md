# Graph Report - go-manus  (2026-09-10)

## Corpus Check
- 285 files · ~873,053 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3378 nodes · 7887 edges · 193 communities (170 shown, 23 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 585 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `bd3b8737`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- createSessionForTest
- cn
- LLMModel
- NewAppConfigService
- index.ts
- index.tsx
- FileRepository
- manus-settings.tsx
- Message
- testing.T
- session-header.tsx
- ToolResult
- RoutedLLM
- PostgresSessionRepository
- mooc-manus vs go-manus 功能差异与缺失分析
- go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter
- .loadOne
- 二、待修复问题详情
- github.com/gin-gonic/gin.Context
- session-detail-view.tsx
- NewSessionHandler
- message_queue_test.go
- Response
- LLMResponse
- 2. 根因链：11 个 Bug 拆解
- logger 包 os.Stdout 误关闭 Bug 深度解析
- AnthropicClient
- external_test.go
- SupervisorService
- compilerOptions
- OpenAIClient
- endpoints/file.py
- Config
- endpoints/shell.py
- FileService
- AgentService
- MessageQueue
- tools_integration_test.go
- NewMockSessionRepository
- external/search.go
- go-manus 项目代码异味（Smell）分析与重构建议
- Err
- DefaultFileCleanupService
- service_dependencies.py
- logger_test.go
- tools_test.go
- PlannerReActFlow
- devDependencies
- 代码智能数据使用说明
- BaseAgent
- NewRoutedLLM
- newTransportClient
- fallback_test.go
- dependencies
- components.json
- getJSON
- Postgres
- RedisStreamTask
- json_parser_repair.go
- RedisConsumerGroup
- 部署指南
- app_test.go
- context.Context
- testLLMModelRepo
- Session
- ShellService
- StdioMCPClient
- NewRedisStreamTask
- AppConfigHandler
- Agent 核心架构
- App
- event.go
- logger.go
- agent/config.go
- llm.go
- Refactoring with GitNexus
- .initRoutes
- go-manus - 通用 AI Agent 系统
- 工具系统 (Tool System)
- OSS
- MergeDeltas
- json_benchmark_test.go
- MessageTool
- Info
- MCPTool
- app/page.tsx
- File
- 集成测试
- run
- .Clear
- status_service_test.go
- A2ATool
- Manus 沙箱服务
- Manus 前端 UI
- Factories
- Commands
- Python vs Go 版本文件对比报告
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- package.json
- 4.3 11 个 Bug 拆解
- logRollbackFailure
- go-manus 面试指南
- Plan
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- UnixStreamHTTPConnection
- 3. 内置工具
- DefaultConsumerGroupManager
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 七、详细模块对比
- response_test.go
- vnc-overlay.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- 3.3 Repository 层架构设计
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- 7.4 存储层详细对比
- 7.5 外部依赖详细对比
- SessionHandler
- 学习路径
- PlanSchemaJSON
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- 三、服务层
- 六、差异汇总
- 3.4 集成测试
- 4. 快速启动（nginx 网关模式，推荐）
- 2. 核心接口
- 7. 安全考虑
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 3. 核心知识点详解
- @novnc/novnc
- Error
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
- left-panel.tsx
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- Impact Analysis with GitNexus
- TaskStream
- NewFileService
- 核心概念
- test-env-down.sh
- test-env-up.sh
- github.com/Huang131/go-manus/api
- Tool
- 八、改进建议
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- next-themes
- @radix-ui/react-avatar
- BaseModel
- FileHandler
- @radix-ui/react-slot
- inMemoryMessageQueue
- @radix-ui/react-switch
- react-dom
- remark-gfm
- tailwind-merge
- SearchResults
- task_runner_test.go
- VNCProxy
- Sandbox
- NewSimpleMemory

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 100 edges
3. `Err()` - 55 edges
4. `File` - 54 edges
5. `Info()` - 46 edges
6. `RedisStreamTask` - 37 edges
7. `Session` - 36 edges
8. `createSessionForTest()` - 36 edges
9. `NewRedisStreamTask()` - 32 edges
10. `App` - 32 edges

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

## Communities (193 total, 23 thin omitted)

### Community 0 - "createSessionForTest"
Cohesion: 0.13
Nodes (43): deleteSessionDirect(), testFileRepo(), TestFileRepo_CountExpiredFiles(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound() (+35 more)

### Community 1 - "cn"
Cohesion: 0.06
Nodes (54): DropdownMenu(), DropdownMenuCheckboxItem(), DropdownMenuContent(), DropdownMenuItem(), DropdownMenuLabel(), DropdownMenuRadioItem(), DropdownMenuSeparator(), DropdownMenuShortcut() (+46 more)

### Community 2 - "LLMModel"
Cohesion: 0.06
Nodes (54): DefaultCapabilities(), CostPolicy, LLMModel, RuntimeHealth, ModelCapabilities, RequestPolicy, MergeDefaultCapabilities(), pgx.Row (+46 more)

### Community 3 - "NewAppConfigService"
Cohesion: 0.06
Nodes (41): MCPServer, A2AConfig, AgentConfig, AppConfig, HealthStatus, LLMConfig, MCPConfig, AppConfigType (+33 more)

### Community 4 - "index.ts"
Cohesion: 0.09
Nodes (52): UseSessionDetailResult, configApi, API_CONFIG, ApiError, createSSEConnection(), createSSEStream(), del(), fetchWithTimeout() (+44 more)

### Community 5 - "index.tsx"
Cohesion: 0.06
Nodes (42): PageProps, findLatestTool(), SessionDetailView(), A2APreview(), BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent() (+34 more)

### Community 6 - "FileRepository"
Cohesion: 0.12
Nodes (9): pgx.Row, pgx.Rows, pgx.Tx, FileRepository, NewFileRepository(), NewFileRepositoryWithTx(), filePoolQueryContext, fileTxQueryContext (+1 more)

### Community 7 - "manus-settings.tsx"
Cohesion: 0.07
Nodes (39): DeleteSessionDialogProps, GlobalHeader(), A2ASettingProps, CommonSettingProps, LLMSettingProps, ManusSettings(), MCPSettingProps, SETTING_MENUS (+31 more)

### Community 9 - "testing.T"
Cohesion: 0.06
Nodes (53): TestFlowStatus_Values(), TestPlannerReActFlow_InvokeContext(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters(), NewMCPClientManager(), TestMCPClientManager_Close() (+45 more)

### Community 10 - "session-header.tsx"
Cohesion: 0.13
Nodes (29): FilePreviewPanel(), isSupportedFileType(), SessionHeader(), SessionItem(), SessionItemProps, Avatar(), AvatarBadge(), AvatarFallback() (+21 more)

### Community 11 - "ToolResult"
Cohesion: 0.06
Nodes (10): TestBrowser_Navigate(), SandboxClient, sandboxToolResult(), ToolResult, NewToolError(), NewToolResultWithMessage(), BrowserClient, MockBrowser (+2 more)

### Community 12 - "RoutedLLM"
Cohesion: 0.12
Nodes (23): defaultLLMClientFactory(), betterHealth(), canFallbackAfterToolUse(), configKey(), RoutedLLM, LLMRequest, healthRank(), NewRoutedLLMFromSingleProvider() (+15 more)

### Community 13 - "PostgresSessionRepository"
Cohesion: 0.12
Nodes (11): pgx.Row, pgx.Rows, pgx.Tx, SessionRepository, marshalSessionEvents(), NewSessionRepository(), NewSessionRepositoryWithTx(), poolQueryContext (+3 more)

### Community 14 - "mooc-manus vs go-manus 功能差异与缺失分析"
Cohesion: 0.05
Nodes (40): 2.1 整体路由映射, 2.2 **仍然缺失的关键路由**, 3.1 Planner 阶段: JSON 强制输出 ✅ Go 更优, 3.2 **空响应重试注入 (重要差异)**, 3.3 LLM 错误重试策略, 3.4 ReAct 阶段的 JSON 约束, 3.5 流式事件分发, 3.6 Agent 任务执行架构 (+32 more)

### Community 15 - "go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter"
Cohesion: 0.05
Nodes (40): Adapter 接口, AnthropicAdapter, CustomGatewayAdapter, go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter, LLM 控制面, ModelHint, OpenAICompatibleAdapter, Provider Adapter (+32 more)

### Community 16 - ".loadOne"
Cohesion: 0.10
Nodes (28): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+20 more)

### Community 17 - "二、待修复问题详情"
Cohesion: 0.05
Nodes (36): 1. 对齐目标, 1. 工具注册 ✅, 2. JSON 解析器 ✅, 2. 关键设计决策, 3.1 TaskRunner 接口, 3.2 RedisStreamTask 架构, 3.3 执行流程, 3. MCP 工具 ✅ (+28 more)

### Community 18 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.19
Nodes (10): LLMModelHandler, NewLLMModelHandler(), NewLLMModelResponse(), LLMModelService, FromError(), TotalResponse, Success(), SuccessWithMsg() (+2 more)

### Community 19 - "session-detail-view.tsx"
Cohesion: 0.08
Nodes (35): AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock(), ToolRow(), FilePreviewPanelProps (+27 more)

### Community 20 - "NewSessionHandler"
Cohesion: 0.38
Nodes (14): NewSessionHandler(), NewMockSessionServiceForHandler(), setupRouter(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), TestSessionHandler_Chat(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create(), TestSessionHandler_Delete() (+6 more)

### Community 21 - "message_queue_test.go"
Cohesion: 0.10
Nodes (19): Message, TestContextTimeout(), TestGetBlocking_Basic(), TestGetBlocking_ContextCancel(), TestGetBlocking_DefaultTimeout(), TestGetBlocking_WithStartID(), TestMessageQueue_DataTypes(), TestMessageQueue_Interface() (+11 more)

### Community 22 - "Response"
Cohesion: 0.15
Nodes (27): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+19 more)

### Community 23 - "LLMResponse"
Cohesion: 0.09
Nodes (19): mockLLM, mockLLMForTest, LLMRequest, LLMRequest, LLMResponse, ResponseFormat, ToolCall, ToolSpec (+11 more)

### Community 24 - "2. 根因链：11 个 Bug 拆解"
Cohesion: 0.06
Nodes (30): 1. 现象回顾, 2.1 模型能力对比（当前项目已测模型）, 2. 根因链：11 个 Bug 拆解, 3.1 简单问答, 3.2 带工具调用的完整任务（回归 Bug 9 场景）, 3. 修复后验证（端到端）, 4. 原项目 mooc-manus (Python) 是否存在这些问题？, 5.1 协议层的"两次包装" (+22 more)

### Community 25 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 26 - "AnthropicClient"
Cohesion: 0.13
Nodes (18): LLMRequest, mustMarshalMap(), NewAnthropicClient(), parseJSONMap(), newAnthropicTestClient(), TestAnthropicClient_ResponseBlocks(), TestAnthropicClient_TextOnlyWire(), TestAnthropicClient_ToolUseRequestWire() (+10 more)

### Community 27 - "external_test.go"
Cohesion: 0.12
Nodes (16): TestBrowser_Screenshot(), TestBrowserInterface(), TestLLM_Invoke(), TestLLM_Invoke_WithError(), TestLLM_ModelInfo(), TestLLMInterface(), TestSandbox_ExecCommand(), TestSandbox_ExecCommand_WithError() (+8 more)

### Community 28 - "SupervisorService"
Cohesion: 0.12
Nodes (10): Exception, AppException, Any, 使用python的xml-rpc客户端连接一个本地sock文件文件实现连接rpc服务, 获取当前supervisor管理的所有进程信息, 传递指定分钟，并激活定时销毁任务同时关闭自动保活, 传递指定的时长，延长超时销毁的时间，单默认延长3分钟, 构造函数，完成supervisor服务链接 (+2 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "OpenAIClient"
Cohesion: 0.11
Nodes (22): estimateCostUSD(), LLMRequest, NewOpenAIClient(), reasoningTokens(), newTestClient(), toOpenAIMessages(), toOpenAITools(), truncateBody() (+14 more)

### Community 31 - "endpoints/file.py"
Cohesion: 0.16
Nodes (33): FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get, post (+25 more)

### Community 32 - "Config"
Cohesion: 0.08
Nodes (26): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, RedisConfig, SandboxConfig (+18 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.19
Nodes (24): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+16 more)

### Community 34 - "FileService"
Cohesion: 0.13
Nodes (8): NotFoundException, Any, FileService, UploadFile, 根据传递的文件路径+内容向指定文件写入内容, 根据传递的文件路径+匹配规则查询文件内符合的内容, 根据传递的文件夹路径+glob规则查询文件列表, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容

### Community 35 - "AgentService"
Cohesion: 0.10
Nodes (13): BrowserTool, SearchTool, A2AConfig, AgentService, AgentConfig, COSFileStorage, MCPConfig, NewAgentService() (+5 more)

### Community 36 - "MessageQueue"
Cohesion: 0.12
Nodes (10): GetPendingMessages(), NewConsumerGroupManager(), MessageQueue, ConsumerGroupInfo, ConsumerGroupManager, ConsumerInfo, DefaultConsumerGroup, GroupPendingInfo (+2 more)

### Community 37 - "tools_integration_test.go"
Cohesion: 0.19
Nodes (17): NewMCPTool(), TestMCPToolInvokeAllowsMissingParams(), TestMCPToolInvokeValidatesParameters(), NewShellTool(), TestMCPTool(), TestMCPTool_WithConfig(), TestShellTool_Description(), TestShellTool_Invoke_Error() (+9 more)

### Community 38 - "NewMockSessionRepository"
Cohesion: 0.19
Nodes (22): SessionService, NewSessionService(), NewSessionServiceWithSandbox(), newMockFileRepo(), NewMockSessionRepository(), TestSessionService_AppendEvent(), TestSessionService_ClearUnreadCount(), TestSessionService_CreateSession() (+14 more)

### Community 39 - "external/search.go"
Cohesion: 0.33
Nodes (9): NewBingSearchClient(), NewGoogleSearchClient(), NewSearchEngine(), BingSearchClient, bingSearchResponse, GoogleSearchClient, googleSearchResponse, SearchConfig (+1 more)

### Community 40 - "go-manus 项目代码异味（Smell）分析与重构建议"
Cohesion: 0.09
Nodes (22): 4.1 🔴 C1~C5：`json_parser_repair.go` 的 Bug 与 Dead Code, 4.2 🟡 W1：`AgentService` God Object, 4.3 🟡 W6/W7：`PlannerReActFlow` 与 `TaskRunner` 的 Long Method + Switch, 4.4 🟡 W9：`SessionRepository` Fat Interface, 🔴 Critical（必须立刻处理）, go-manus 项目代码异味（Smell）分析与重构建议, Immediate（本周，1~2 天）, Long-Term（1 个月+） (+14 more)

### Community 41 - "Err"
Cohesion: 0.14
Nodes (10): deliverMessage(), RedisStreamMessageQueue, Message, redis.Client, NewRedisStreamMessageQueue(), TestRedisMessageQueueDefaultTimeoutsUseDurations(), TestDeliverMessage_StopsWhenContextCanceled(), TestDeliverMessage_WaitsWhenBufferIsFull() (+2 more)

### Community 42 - "DefaultFileCleanupService"
Cohesion: 0.11
Nodes (13): COSFileStorage, DefaultFileService, FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupStatsAccumulateAcrossCleanupTypes(), TestFileCleanupStatsReadableDuringCleanup() (+5 more)

### Community 43 - "service_dependencies.py"
Cohesion: 0.13
Nodes (16): APIRouter, BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, create_api_routes() (+8 more)

### Community 44 - "logger_test.go"
Cohesion: 0.22
Nodes (19): Init(), InitWithConfig(), stdoutWriteOK(), TestCallerInBusinessCode(), testCallerLocation(), TestDebug(), TestGet(), TestInfo() (+11 more)

### Community 45 - "tools_test.go"
Cohesion: 0.14
Nodes (18): mockTool, NewA2ATool(), TestA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description(), TestA2ATool_Initialize(), TestA2ATool_Invoke_CallAgentWithoutAgentID() (+10 more)

### Community 46 - "PlannerReActFlow"
Cohesion: 0.23
Nodes (7): FlowStatus, PlannerAgent, PlannerReActFlow, AgentConfig, NewPlannerAgent(), AgentConfig, NewPlannerReActFlow()

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.11
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "BaseAgent"
Cohesion: 0.14
Nodes (13): BaseAgent, InvokeResult, ToolCallResult, AgentConfig, NewBaseAgent(), NewBaseAgentWithParser(), DefaultAgentConfig(), TestBaseAgent_InvokeWithEmptyRetry() (+5 more)

### Community 50 - "NewRoutedLLM"
Cohesion: 0.14
Nodes (28): RoutedLLM, modelNames(), newMockRuntimeHealthStore(), openAITextProfile(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_DoNotFallbackAcrossProtocol(), TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect(), TestRoutedLLM_FallbackOnRateLimit() (+20 more)

### Community 51 - "newTransportClient"
Cohesion: 0.22
Nodes (19): newTransportClient(), rawOK(), responseJSON(), TestOpenAIClient_401_Auth(), TestOpenAIClient_429_RateLimit(), TestOpenAIClient_5xx_Server(), TestOpenAIClient_ContentAndReasoning_Separated(), TestOpenAIClient_ContentOnly() (+11 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.12
Nodes (25): CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), ContextFits(), EstimateContextTokens(), Message, ModelCapabilities, fullCaps() (+17 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "getJSON"
Cohesion: 0.12
Nodes (48): Response, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_EmptyUpdate(), TestAppConfigAPI_InvalidJSON(), TestAppConfigAPI_LLMConfig_Lifecycle(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle() (+40 more)

### Community 56 - "Postgres"
Cohesion: 0.17
Nodes (8): Postgres, NewPostgres(), Redis, redis.Client, NewRedis(), StatusService, NewStatusService(), DefaultStatusService

### Community 57 - "RedisStreamTask"
Cohesion: 0.14
Nodes (6): DefaultTaskRegistry, RedisStreamTask, TaskRegistryInterface, TaskRunner, context.CancelFunc, sync.RWMutex

### Community 58 - "json_parser_repair.go"
Cohesion: 0.17
Nodes (12): NewDefaultJSONParser(), extractJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), removeTrailingCommas(), TestDefaultJSONParser(), TestExtractJSON(), TestRemoveMarkdownCodeBlocks() (+4 more)

### Community 59 - "RedisConsumerGroup"
Cohesion: 0.12
Nodes (7): Message, redis.Client, MessageToJSON(), NewRedisConsumerGroup(), NewRedisConsumerGroupManager(), RedisConsumerGroup, RedisConsumerGroupManager

### Community 60 - "部署指南"
Cohesion: 0.12
Nodes (17): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查, 1. 部署方式, 2. 架构总览, 3. 前置要求, 5.1 `.env` 模板（`api/.env.example`） (+9 more)

### Community 61 - "app_test.go"
Cohesion: 0.12
Nodes (24): Build(), newLifecycleManager(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce(), TestBuildInitializeErrorWrapsSentinel(), TestBuildRejectsNilConfigSentinel() (+16 more)

### Community 62 - "context.Context"
Cohesion: 0.05
Nodes (12): attachmentStorage, mockMQWrapper, mockSandbox, Message, DefaultFileService, mockStorage, mockMessageQueue, context.Context (+4 more)

### Community 63 - "testLLMModelRepo"
Cohesion: 0.29
Nodes (19): cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError(), TestLLMModelRepo_GetByID_NotFoundReturnsNilNil(), TestLLMModelRepo_GetDefault_NoDefaultReturnsNilNil() (+11 more)

### Community 64 - "Session"
Cohesion: 0.05
Nodes (9): mockSessionRepo, Event, Session, SessionStatus, unixPointer(), MockSessionServiceForHandler, stubVNCService, DefaultSessionService (+1 more)

### Community 65 - "ShellService"
Cohesion: 0.17
Nodes (8): ConsoleRecord, Process, 根据传递的会话id+是否输出控制台记录获取Shell命令结果, 传递会话id+执行目录+命令在沙箱中执行后返回, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程, 启动协程以连续读取进程输出并将其存储到会话中, ShellService

### Community 66 - "StdioMCPClient"
Cohesion: 0.12
Nodes (17): MCPClientManager, MCPToolInfo, NewStdioMCPClient(), MCPClient, MCPConfig, MCPConfigServer, MCPContent, MCPError (+9 more)

### Community 67 - "NewRedisStreamTask"
Cohesion: 0.32
Nodes (14): NewDefaultTaskRegistry(), NewRedisStreamTask(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister(), TestDefaultTaskRegistry_List() (+6 more)

### Community 68 - "AppConfigHandler"
Cohesion: 0.24
Nodes (6): getConfig(), AppConfigHandler, T, NewAppConfigHandler(), updateConfig(), AppConfigService

### Community 69 - "Agent 核心架构"
Cohesion: 0.13
Nodes (15): 1. 整体架构, 2.1 BaseAgent, 2.2 PlannerAgent, 2.3 ReActAgent, 2.4 Flow 编排, 2. 核心组件, 3. Memory 系统, 4. 会话管理 (+7 more)

### Community 70 - "App"
Cohesion: 0.24
Nodes (4): BuildWithFactories(), App, lifecycleManager, Options

### Community 71 - "event.go"
Cohesion: 0.07
Nodes (30): clonePlanSteps(), cloneStrings(), NewDoneEvent(), NewErrorEvent(), NewMessageEvent(), NewPlanEvent(), NewStepEvent(), NewTitleEvent() (+22 more)

### Community 72 - "logger.go"
Cohesion: 0.10
Nodes (30): Bool(), DebugContext(), Dur(), Error(), ErrorContext(), Fatal(), Float64(), Get() (+22 more)

### Community 73 - "agent/config.go"
Cohesion: 0.20
Nodes (9): A2AAgent, AgentConfig, MCPServer, A2AAgent, A2AConfig, MCPConfig, MCPServer, newA2AConfig() (+1 more)

### Community 74 - "llm.go"
Cohesion: 0.14
Nodes (11): LLMRequest, TestDynamicLLM_UsesFactory(), TestRoutedLLMFromSingleProvider(), ModelIDFromContext(), NewDynamicLLM(), NewDynamicLLMWithFactory(), WithModelID(), DynamicLLM (+3 more)

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - ".initRoutes"
Cohesion: 0.20
Nodes (17): defaultTrustedProxies(), CORS(), generateRequestID(), Logger(), Recovery(), RequestID(), setupTestEngine(), TestCORS_AllowedHeader() (+9 more)

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.15
Nodes (9): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 健康检查, 常用命令, 本地开发, 核心代码指引, 许可证 (+1 more)

### Community 78 - "工具系统 (Tool System)"
Cohesion: 0.15
Nodes (13): 1. 概述, 4.1 MCP 客户端, 4.2 工具注册, 4. MCP 集成, 5.1 转换为 LLM 格式, 5.2 解析 LLM 响应, 5. LLM 工具调用, 6.1 步骤 1：定义工具 (+5 more)

### Community 79 - "OSS"
Cohesion: 0.13
Nodes (15): ObjectStorageConfig, generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL(), TestNewOSS_WithoutCredentials() (+7 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.23
Nodes (11): MergeDeltas(), NormalizeResponse(), repeat(), TestEstimateContextTokens(), TestMergeDeltas_ContentOnly(), TestMergeDeltas_MultipleToolCalls(), TestMergeDeltas_ReasoningAndContent(), TestMergeDeltas_ReasoningOnly() (+3 more)

### Community 81 - "json_benchmark_test.go"
Cohesion: 0.22
Nodes (15): Benchmark_Sonic_Marshal_Large(), Benchmark_Sonic_Marshal_Small(), Benchmark_Sonic_Roundtrip_Large(), Benchmark_Sonic_Roundtrip_Small(), Benchmark_Sonic_Unmarshal_Large(), Benchmark_Sonic_Unmarshal_Small(), Benchmark_StdLib_Marshal_Large(), Benchmark_StdLib_Marshal_Small() (+7 more)

### Community 83 - "Info"
Cohesion: 0.26
Nodes (4): AgentTaskRunner, COSFileStorage, Info(), Warn()

### Community 84 - "MCPTool"
Cohesion: 0.20
Nodes (3): MCPTool, MCPConfig, parseMCPInvokeParams()

### Community 85 - "app/page.tsx"
Cohesion: 0.24
Nodes (7): ChatInput, ChatInputProps, ChatInputRef, SuggestedQuestions(), SuggestedQuestionsProps, suggestedQuestions, FileInfo

### Community 86 - "File"
Cohesion: 0.06
Nodes (6): attachmentFileRepository, File, cleanupRepoStub, emptyFileRepo, mockFileRepo, stubFileRepo

### Community 87 - "集成测试"
Cohesion: 0.14
Nodes (13): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+5 more)

### Community 88 - "run"
Cohesion: 0.31
Nodes (7): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), net/http.Server, Config

### Community 89 - ".Clear"
Cohesion: 0.18
Nodes (11): mockTaskRunner, TestRedisStreamTask_Cancel(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_ID(), TestRedisStreamTask_Invoke(), TestRedisStreamTask_PutInput_GetOutput(), TestTaskRegistry_Clear(), TestTaskRegistry_Get() (+3 more)

### Community 90 - "status_service_test.go"
Cohesion: 0.14
Nodes (7): TestStatusService_GetHealthStatus_SkippedServicesDegraded(), TestStatusService_GetHealthStatus_WrapsHealthErrors(), TestStatusServiceUsesInjectedClock(), TestToInternal(), fakeOSS, fakePostgres, fakeRedis

### Community 91 - "A2ATool"
Cohesion: 0.10
Nodes (16): A2ATool, A2AConfig, A2AClientManager, NewA2AClientManager(), A2AAgentCapabilities, A2AAgentCard, A2AAgentSkill, A2AClientManagerConfig (+8 more)

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.20
Nodes (9): API 接口, Docker 部署, Manus 沙箱服务, 使用开发容器, 启动服务, 技术栈, 本地开发, 架构 (+1 more)

### Community 93 - "Manus 前端 UI"
Cohesion: 0.20
Nodes (9): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 环境准备 (+1 more)

### Community 94 - "Factories"
Cohesion: 0.33
Nodes (8): defaultFactories(), DefaultOptions(), newRepositories(), normalizeFactories(), TestNormalizeFactoriesFullInjection(), TestNormalizeFactoriesPartialInjection(), Factories, repositories

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

### Community 100 - "logRollbackFailure"
Cohesion: 0.44
Nodes (4): pgx.Tx, joinRollbackError(), logRollbackFailure(), rollbackTx()

### Community 101 - "go-manus 面试指南"
Cohesion: 0.22
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "Plan"
Cohesion: 0.10
Nodes (13): ReActAgent, TaskInput, BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext(), AgentConfig (+5 more)

### Community 103 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 104 - "6. 代码评审与回归测试：测试不是覆盖，而是契约"
Cohesion: 0.22
Nodes (9): 6.1 Why：测试是行为的契约, 6.2 历史 Bug 的回归测试, 6.3 测试分类, 6.4 Go 测试模式, 6.5 面试要点, 6. 代码评审与回归测试：测试不是覆盖，而是契约, 子测试 + t.Helper, 模糊测试（Go 1.18+） (+1 more)

### Community 105 - "UnixStreamHTTPConnection"
Cohesion: 0.25
Nodes (4): HTTPConnection, 重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接, UnixStreamHTTPConnection, UnixStreamTransport

### Community 106 - "3. 内置工具"
Cohesion: 0.29
Nodes (7): 3.1 FileTool - 文件操作, 3.2 ShellTool - Shell 命令执行, 3.3 BrowserTool - 浏览器自动化, 3.4 SearchTool - 搜索功能, 3.5 MessageTool - 消息通知, 3.6 A2ATool - Agent 间通信, 3. 内置工具

### Community 107 - "DefaultConsumerGroupManager"
Cohesion: 0.33
Nodes (3): ConsumerGroup, ConsumerGroupConfig, DefaultConsumerGroupManager

### Community 108 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 109 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 110 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 111 - "response_test.go"
Cohesion: 0.29
Nodes (5): TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure(), TestSuccess_HttpResponse(), TestTotalResponse_Structure()

### Community 112 - "vnc-overlay.tsx"
Cohesion: 0.31
Nodes (6): buildVNCUrl(), VNCOverlay(), VNCOverlayProps, VNCViewer, VNCStatus, VNCViewerProps

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

### Community 120 - "SessionHandler"
Cohesion: 0.23
Nodes (8): SessionHandler, mergeEventMetadata(), newSSEContext(), parseChatRequest(), setSSEHeaders(), TestMergeEventMetadata(), TestSetSSEHeaders(), chatRequest

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

### Community 128 - "2. 核心接口"
Cohesion: 0.50
Nodes (4): 2.1 Tool 接口, 2.2 工具定义, 2.3 执行结果, 2. 核心接口

### Community 129 - "7. 安全考虑"
Cohesion: 0.50
Nodes (4): 7.1 沙箱隔离, 7.2 权限控制, 7.3 输入验证, 7. 安全考虑

### Community 130 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 131 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 133 - "Error"
Cohesion: 0.19
Nodes (13): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Internal(), New(), NotFound(), ToInternal() (+5 more)

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

### Community 157 - "left-panel.tsx"
Cohesion: 0.14
Nodes (14): metadata, DeleteSessionDialog(), LeftPanel(), SessionList(), ItemGroup(), Kbd(), KbdGroup(), Toaster() (+6 more)

### Community 158 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 159 - "Impact Analysis with GitNexus"
Cohesion: 0.22
Nodes (8): Checklist, Example: "What breaks if I change validateUser?", Impact Analysis with GitNexus, Risk Assessment, Tools, Understanding Output, When to Use, Workflow

### Community 160 - "TaskStream"
Cohesion: 0.16
Nodes (5): Stream, Task, TaskStream, NewTaskStream(), sync.Mutex

### Community 161 - "NewFileService"
Cohesion: 0.23
Nodes (14): NewFileService(), TestFileServiceDeleteFileDeletesStorageAndRepo(), TestFileServiceDeleteFileRepoError(), TestFileServiceDeleteFileStorageError(), TestFileServiceDownloadFileNilStorageReturnsNilReader(), TestFileServiceDownloadFileStorageError(), TestFileServiceDownloadFileWithStorage(), TestFileServiceGetFileInfoFound() (+6 more)

### Community 162 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

### Community 171 - "Tool"
Cohesion: 0.17
Nodes (9): AgentTaskRunnerConfig, COSFileStorage, MultiFunctionTool, readerWrapper, Tool, ToolRegistry, AgentConfig, NewAgentTaskRunner() (+1 more)

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

### Community 181 - "FileHandler"
Cohesion: 0.25
Nodes (8): FileHandler, NewFileHandler(), StatusHandler, NewStatusHandler(), SetupRoutes(), FileService, github.com/gin-gonic/gin.Engine, Handlers

### Community 183 - "inMemoryMessageQueue"
Cohesion: 0.10
Nodes (9): inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, mockFailingTool, newInMemoryMessageQueue(), parseStreamSeq(), TestToolCallingEvents_SSEStream() (+1 more)

### Community 188 - "SearchResults"
Cohesion: 0.67
Nodes (3): SearchEvent, SearchResultItem, SearchResults

### Community 195 - "task_runner_test.go"
Cohesion: 0.22
Nodes (7): attachmentSandbox, TestAgentService_ResolveMessageAttachments(), TestAgentTaskRunner_SyncUserAttachmentsToSandbox(), TestMessageTool_NotifyUserSchema(), TestPopRetryConfig(), NewMessageTool(), TestMessageTool()

### Community 207 - "VNCProxy"
Cohesion: 0.27
Nodes (8): dialSandboxVNC(), newLocalTestServer(), TestVNCProxy_ProxyEcho(), TestVNCProxy_ServiceError(), VNCProxy(), github.com/gorilla/websocket.Conn, net/http.Handler, net/http/httptest.Server

### Community 220 - "Sandbox"
Cohesion: 0.12
Nodes (8): FileTool, ShellTool, NewFileTool(), TestFileTool(), Sandbox, NewBrowserClient(), sandboxErrorResponse, sandboxRequest

### Community 233 - "NewSimpleMemory"
Cohesion: 0.36
Nodes (6): Memory, NewSimpleMemory(), TestSimpleMemory_Add(), TestSimpleMemory_Clear(), TestSimpleMemory_Compact(), TestSimpleMemory_GetMessages()

## Knowledge Gaps
- **610 isolated node(s):** `LLMConfig`, `SearchConfig`, `MCPServer`, `A2AAgent`, `github.com/Huang131/go-manus/api` (+605 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **23 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Err()` connect `Err` to `Error`, `ToolResult`, `.loadOne`, `github.com/gin-gonic/gin.Context`, `DefaultFileCleanupService`, `BaseAgent`, `RedisStreamTask`, `RedisConsumerGroup`, `context.Context`, `StdioMCPClient`, `App`, `event.go`, `logger.go`, `.initRoutes`, `VNCProxy`, `OSS`, `Info`, `MCPTool`, `run`, `A2ATool`, `logRollbackFailure`, `Plan`, `SessionHandler`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Why does `Info()` connect `Info` to `StdioMCPClient`, `AgentService`, `NewRedisStreamTask`, `Plan`, `event.go`, `logger.go`, `DefaultFileCleanupService`, `ToolResult`, `Tool`, `logger_test.go`, `OSS`, `.loadOne`, `BaseAgent`, `MCPTool`, `run`, `RedisStreamTask`, `Postgres`, `A2ATool`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Why does `AgentService` connect `AgentService` to `NewAppConfigService`, `MessageQueue`, `FileRepository`, `App`, `logger.go`, `Tool`, `PostgresSessionRepository`, `BaseAgent`, `Info`, `MCPTool`, `NewSessionHandler`, `SessionHandler`, `RedisStreamTask`, `A2ATool`, `Sandbox`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **What connects `LLMConfig`, `SearchConfig`, `MCPServer` to the rest of the system?**
  _610 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `createSessionForTest` be split into smaller, more focused modules?**
  _Cohesion score 0.12753623188405797 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.05711849957374254 - nodes in this community are weakly interconnected._
- **Should `LLMModel` be split into smaller, more focused modules?**
  _Cohesion score 0.056802244039270686 - nodes in this community are weakly interconnected._