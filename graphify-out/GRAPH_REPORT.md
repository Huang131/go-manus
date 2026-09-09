# Graph Report - go-manus  (2026-09-09)

## Corpus Check
- 277 files · ~858,231 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3082 nodes · 7103 edges · 178 communities (153 shown, 25 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 485 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `691898c9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- context.Context
- cn
- LLMModel
- NewAppConfigService
- index.ts
- index.tsx
- File
- manus-settings.tsx
- NewSimpleMemory
- testing.T
- session-header.tsx
- ToolResult
- RoutedLLM
- PostgresSessionRepository
- mooc-manus vs go-manus 功能差异与缺失分析
- go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter
- NewLoader
- 二、待修复问题详情
- FromError
- session-detail-view.tsx
- NewSessionHandler
- mockMQ
- Response
- Message
- 2. 根因链：11 个 Bug 拆解
- NewToolError
- AnthropicClient
- external_test.go
- SupervisorService
- compilerOptions
- OpenAIClient
- endpoints/file.py
- Config
- endpoints/shell.py
- services/file.py
- AgentService
- Sandbox
- tools_integration_test.go
- NewMockSessionRepository
- external/search.go
- go-manus 项目代码异味（Smell）分析与重构建议
- Err
- Factories
- service_dependencies.py
- ShellService
- A2ATool
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
- sync.RWMutex
- 部署指南
- AgentTaskRunner
- .ExecCommand
- Error
- app_test.go
- contains
- StdioMCPClient
- BuildRuntimeConfigFromModel
- MergeDeltas
- Agent 核心架构
- App
- event.go
- logger.go
- NewBaseAgent
- llm.go
- Refactoring with GitNexus
- .initRoutes
- go-manus - 通用 AI Agent 系统
- 工具系统 (Tool System)
- OSS
- LLMResponse
- json_benchmark_test.go
- MessageTool
- NewProviderError
- DefaultFileService
- app/page.tsx
- .loadOne
- 集成测试
- stubLLM
- Handlers
- status_service_test.go
- response.go
- Manus 沙箱服务
- Manus 前端 UI
- parseResponse
- Commands
- Python vs Go 版本文件对比报告
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- package.json
- 4.3 11 个 Bug 拆解
- MCPTool
- SessionHandler
- BuildAttachmentContextSection
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- UnixStreamHTTPConnection
- 3. 内置工具
- markdown-content.tsx
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 七、详细模块对比
- 快速部署
- vnc-overlay.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- response_test.go
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- 7.4 存储层详细对比
- 7.5 外部依赖详细对比
- 核心概念
- 学习路径
- PlanSchemaJSON
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- 三、服务层
- 六、差异汇总
- 八、改进建议
- 4. 快速启动（nginx 网关模式，推荐）
- 2. 核心接口
- 7. 安全考虑
- .fail
- next-themes
- @novnc/novnc
- @radix-ui/react-avatar
- @radix-ui/react-label
- @radix-ui/react-scroll-area
- @radix-ui/react-slot
- @radix-ui/react-switch
- react-dom
- react-markdown
- remark-gfm
- sonner
- tailwind-merge
- eslint.config.mjs
- next.config.ts
- postcss.config.mjs
- validConfig
- sandbox
- 常量按领域归属重构计划
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- Impact Analysis with GitNexus
- 重构发现
- FileHandler
- 进度日志
- test-env-down.sh
- test-env-up.sh
- github.com/Huang131/go-manus/api
- BrowserTool
- task_runner_test.go
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 99 edges
3. `Err()` - 52 edges
4. `Info()` - 50 edges
5. `FromError()` - 38 edges
6. `RedisStreamTask` - 36 edges
7. `Session` - 35 edges
8. `AgentService` - 34 edges
9. `Warn()` - 33 edges
10. `Success()` - 33 edges

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

## Communities (178 total, 25 thin omitted)

### Community 0 - "context.Context"
Cohesion: 0.05
Nodes (11): mockMQWrapper, Message, Event, Session, SessionStatus, mockMessageQueue, context.Context, MockSessionServiceForHandler (+3 more)

### Community 1 - "cn"
Cohesion: 0.05
Nodes (64): metadata, GlobalHeader(), LeftPanel(), ManusSettings(), DialogOverlay(), DropdownMenu(), DropdownMenuCheckboxItem(), DropdownMenuContent() (+56 more)

### Community 2 - "LLMModel"
Cohesion: 0.06
Nodes (43): DefaultCapabilities(), CostPolicy, CostPolicy, LLMModel, ModelCapabilities, RequestPolicy, RuntimeHealth, ModelCapabilities (+35 more)

### Community 3 - "NewAppConfigService"
Cohesion: 0.06
Nodes (41): MCPServer, A2AConfig, AgentConfig, AppConfig, HealthStatus, LLMConfig, MCPConfig, AppConfigType (+33 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (55): configApi, API_CONFIG, ApiError, createSSEConnection(), createSSEStream(), del(), fetchWithTimeout(), get() (+47 more)

### Community 5 - "index.tsx"
Cohesion: 0.08
Nodes (36): A2APreview(), BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent(), getToolDescription(), getToolIcon(), MCPPreview() (+28 more)

### Community 6 - "File"
Cohesion: 0.09
Nodes (12): attachmentFileRepository, File, pgx.Row, pgx.Rows, pgx.Tx, FileRepository, NewFileRepository(), NewFileRepositoryWithTx() (+4 more)

### Community 7 - "manus-settings.tsx"
Cohesion: 0.09
Nodes (35): DeleteSessionDialogProps, A2ASettingProps, CommonSettingProps, LLMSettingProps, MCPSettingProps, SETTING_MENUS, SettingTab, emptyDraft (+27 more)

### Community 8 - "NewSimpleMemory"
Cohesion: 0.36
Nodes (6): Memory, NewSimpleMemory(), TestSimpleMemory_Add(), TestSimpleMemory_Clear(), TestSimpleMemory_Compact(), TestSimpleMemory_GetMessages()

### Community 9 - "testing.T"
Cohesion: 0.05
Nodes (56): TestFlowStatus_Values(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters(), NewMCPClientManager(), TestMCPClientManager_Close(), TestMCPClientManager_GetClient() (+48 more)

### Community 10 - "session-header.tsx"
Cohesion: 0.12
Nodes (27): DeleteSessionDialog(), SessionItem(), SessionItemProps, SessionList(), Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup() (+19 more)

### Community 11 - "ToolResult"
Cohesion: 0.07
Nodes (5): mockSandbox, TestBrowser_Navigate(), ToolResult, MockBrowser, MockSandbox

### Community 12 - "RoutedLLM"
Cohesion: 0.19
Nodes (12): betterHealth(), canFallbackAfterToolUse(), configKey(), RoutedLLM, LLMRequest, healthRank(), NewRoutedLLMFromSingleProvider(), updateLatencyMS() (+4 more)

### Community 13 - "PostgresSessionRepository"
Cohesion: 0.12
Nodes (10): pgx.Row, pgx.Rows, pgx.Tx, SessionRepository, NewSessionRepository(), NewSessionRepositoryWithTx(), poolQueryContext, PostgresSessionRepository (+2 more)

### Community 14 - "mooc-manus vs go-manus 功能差异与缺失分析"
Cohesion: 0.05
Nodes (40): 2.1 整体路由映射, 2.2 **仍然缺失的关键路由**, 3.1 Planner 阶段: JSON 强制输出 ✅ Go 更优, 3.2 **空响应重试注入 (重要差异)**, 3.3 LLM 错误重试策略, 3.4 ReAct 阶段的 JSON 约束, 3.5 流式事件分发, 3.6 Agent 任务执行架构 (+32 more)

### Community 15 - "go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter"
Cohesion: 0.05
Nodes (40): Adapter 接口, AnthropicAdapter, CustomGatewayAdapter, go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter, LLM 控制面, ModelHint, OpenAICompatibleAdapter, Provider Adapter (+32 more)

### Community 16 - "NewLoader"
Cohesion: 0.12
Nodes (21): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+13 more)

### Community 17 - "二、待修复问题详情"
Cohesion: 0.05
Nodes (36): 1. 对齐目标, 1. 工具注册 ✅, 2. JSON 解析器 ✅, 2. 关键设计决策, 3.1 TaskRunner 接口, 3.2 RedisStreamTask 架构, 3.3 执行流程, 3. MCP 工具 ✅ (+28 more)

### Community 18 - "FromError"
Cohesion: 0.22
Nodes (5): AppConfigHandler, LLMModelHandler, FromError(), Success(), github.com/gin-gonic/gin.Context

### Community 19 - "session-detail-view.tsx"
Cohesion: 0.08
Nodes (39): PageProps, AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock(), ToolRow() (+31 more)

### Community 20 - "NewSessionHandler"
Cohesion: 0.48
Nodes (12): NewSessionHandler(), NewMockSessionServiceForHandler(), setupRouter(), TestSessionHandler_Chat(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create(), TestSessionHandler_Delete(), TestSessionHandler_Get() (+4 more)

### Community 21 - "mockMQ"
Cohesion: 0.24
Nodes (3): Message, TestMessageQueue_Mock(), mockMQ

### Community 22 - "Response"
Cohesion: 0.15
Nodes (24): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+16 more)

### Community 23 - "Message"
Cohesion: 0.13
Nodes (17): SimpleMemory, estimateCostUSD(), Message, ResponseFormat, ToolCall, ToolSpec, Usage, AudioURL (+9 more)

### Community 24 - "2. 根因链：11 个 Bug 拆解"
Cohesion: 0.06
Nodes (30): 1. 现象回顾, 2.1 模型能力对比（当前项目已测模型）, 2. 根因链：11 个 Bug 拆解, 3.1 简单问答, 3.2 带工具调用的完整任务（回归 Bug 9 场景）, 3. 修复后验证（端到端）, 4. 原项目 mooc-manus (Python) 是否存在这些问题？, 5.1 协议层的"两次包装" (+22 more)

### Community 25 - "NewToolError"
Cohesion: 0.25
Nodes (7): SandboxClient, NewSandboxClient(), sandboxToolResult(), NewToolError(), sandboxErrorResponse, sandboxRequest, sandboxResponse

### Community 26 - "AnthropicClient"
Cohesion: 0.12
Nodes (19): LLMRequest, mustMarshalMap(), NewAnthropicClient(), parseJSONMap(), textOf(), userContent(), CostPolicy, ExtraParam (+11 more)

### Community 27 - "external_test.go"
Cohesion: 0.10
Nodes (17): LLMRequest, TestBrowser_Screenshot(), TestBrowserInterface(), TestLLM_Invoke(), TestLLM_Invoke_WithError(), TestLLM_ModelInfo(), TestLLMInterface(), TestSandbox_ExecCommand() (+9 more)

### Community 28 - "SupervisorService"
Cohesion: 0.12
Nodes (12): Exception, AppException, BadRequestException, Any, Any, 使用python的xml-rpc客户端连接一个本地sock文件文件实现连接rpc服务, 获取当前supervisor管理的所有进程信息, 传递指定分钟，并激活定时销毁任务同时关闭自动保活 (+4 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "OpenAIClient"
Cohesion: 0.14
Nodes (16): LLMRequest, NewOpenAIClient(), reasoningTokens(), newTestClient(), toOpenAIMessages(), toOpenAITools(), truncateBody(), openAIChatRequest (+8 more)

### Community 31 - "endpoints/file.py"
Cohesion: 0.20
Nodes (25): FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get, post (+17 more)

### Community 32 - "Config"
Cohesion: 0.10
Nodes (23): formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, ObjectStorageConfig, RedisConfig, SandboxConfig (+15 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.21
Nodes (23): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+15 more)

### Community 34 - "services/file.py"
Cohesion: 0.15
Nodes (15): FileCheckResult, FileDeleteResult, FileFindResult, FileReadResult, FileReplaceResult, FileSearchResult, FileUploadResult, FileWriteResult (+7 more)

### Community 35 - "AgentService"
Cohesion: 0.14
Nodes (15): A2AAgent, A2AConfig, MCPConfig, MCPServer, A2AConfig, AgentService, AgentConfig, COSFileStorage (+7 more)

### Community 36 - "Sandbox"
Cohesion: 0.11
Nodes (7): attachmentSandbox, FileTool, ShellTool, NewFileTool(), TestFileTool(), Sandbox, NewBrowserClient()

### Community 37 - "tools_integration_test.go"
Cohesion: 0.20
Nodes (15): NewMCPTool(), NewShellTool(), TestMCPTool(), TestMCPTool_WithConfig(), TestShellTool_Description(), TestShellTool_Invoke_Error(), TestShellTool_Invoke_Exec(), TestShellTool_Invoke_Kill() (+7 more)

### Community 38 - "NewMockSessionRepository"
Cohesion: 0.12
Nodes (29): dialSandboxVNC(), newLocalTestServer(), TestVNCProxy_ProxyEcho(), TestVNCProxy_ServiceError(), VNCProxy(), SessionService, NewSessionService(), NewSessionServiceWithSandbox() (+21 more)

### Community 39 - "external/search.go"
Cohesion: 0.16
Nodes (12): SearchTool, NewSearchTool(), SearchEngine, NewBingSearchClient(), NewGoogleSearchClient(), NewSearchEngine(), BingSearchClient, bingSearchResponse (+4 more)

### Community 40 - "go-manus 项目代码异味（Smell）分析与重构建议"
Cohesion: 0.09
Nodes (22): 4.1 🔴 C1~C5：`json_parser_repair.go` 的 Bug 与 Dead Code, 4.2 🟡 W1：`AgentService` God Object, 4.3 🟡 W6/W7：`PlannerReActFlow` 与 `TaskRunner` 的 Long Method + Switch, 4.4 🟡 W9：`SessionRepository` Fat Interface, 🔴 Critical（必须立刻处理）, go-manus 项目代码异味（Smell）分析与重构建议, Immediate（本周，1~2 天）, Long-Term（1 个月+） (+14 more)

### Community 41 - "Err"
Cohesion: 0.08
Nodes (10): TaskInput, MCPConfig, RedisStreamMessageQueue, Message, redis.Client, NewRedisStreamMessageQueue(), Debug(), Err() (+2 more)

### Community 42 - "Factories"
Cohesion: 0.14
Nodes (13): defaultFactories(), COSFileStorage, DefaultFileService, FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), Factories (+5 more)

### Community 43 - "service_dependencies.py"
Cohesion: 0.13
Nodes (16): APIRouter, BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, create_api_routes() (+8 more)

### Community 44 - "ShellService"
Cohesion: 0.16
Nodes (9): ConsoleRecord, Process, NotFoundException, 根据传递的会话id+是否输出控制台记录获取Shell命令结果, 传递会话id+执行目录+命令在沙箱中执行后返回, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程, 启动协程以连续读取进程输出并将其存储到会话中 (+1 more)

### Community 45 - "A2ATool"
Cohesion: 0.06
Nodes (34): A2ATool, mockTool, A2AConfig, NewA2ATool(), TestA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description() (+26 more)

### Community 46 - "PlannerReActFlow"
Cohesion: 0.14
Nodes (11): FlowStatus, PlannerAgent, PlannerReActFlow, ReActAgent, AgentConfig, NewPlannerAgent(), AgentConfig, NewPlannerReActFlow() (+3 more)

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.11
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "BaseAgent"
Cohesion: 0.13
Nodes (10): BaseAgent, InvokeResult, ToolCallResult, AgentConfig, NewBaseAgentWithParser(), NewBaseAgentWithRepo(), NewBaseAgentWithRepoAndParser(), JSONParser (+2 more)

### Community 50 - "NewRoutedLLM"
Cohesion: 0.18
Nodes (23): RoutedLLM, modelNames(), newMockRuntimeHealthStore(), openAITextProfile(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_DoNotFallbackAcrossProtocol(), TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect(), TestRoutedLLM_FallbackOnRateLimit() (+15 more)

### Community 51 - "newTransportClient"
Cohesion: 0.17
Nodes (23): newAnthropicTestClient(), TestAnthropicClient_ResponseBlocks(), TestAnthropicClient_TextOnlyWire(), TestAnthropicClient_ToolUseRequestWire(), newTransportClient(), rawOK(), responseJSON(), TestOpenAIClient_401_Auth() (+15 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.14
Nodes (22): CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), ContextFits(), EstimateContextTokens(), Message, ModelCapabilities, fullCaps() (+14 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "getJSON"
Cohesion: 0.21
Nodes (29): TestFileAPI_Delete_Lifecycle(), TestFileAPI_Download_Lifecycle(), TestFileAPI_FileTableConsistency(), TestFileAPI_GetInfo_NotFound(), TestFileAPI_GetSessionFiles_Lifecycle(), TestFileAPI_Rename_Lifecycle(), TestFileAPI_Upload_Concurrent(), TestFileAPI_Upload_LargeFile() (+21 more)

### Community 56 - "Postgres"
Cohesion: 0.17
Nodes (8): Postgres, NewPostgres(), Redis, redis.Client, NewRedis(), StatusService, NewStatusService(), DefaultStatusService

### Community 57 - "RedisStreamTask"
Cohesion: 0.05
Nodes (42): DefaultTaskRegistry, MemoryStream, mockTaskRunner, RedisStreamTask, SimpleTask, Stream, streamMessage, Task (+34 more)

### Community 58 - "json_parser_repair.go"
Cohesion: 0.23
Nodes (10): extractJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), removeTrailingCommas(), TestDefaultJSONParser(), TestExtractJSON(), TestRemoveMarkdownCodeBlocks(), TestRemoveTrailingCommas() (+2 more)

### Community 59 - "sync.RWMutex"
Cohesion: 0.06
Nodes (20): GetPendingMessages(), NewConsumerGroupManager(), Message, redis.Client, MessageToJSON(), NewRedisConsumerGroup(), NewRedisConsumerGroupManager(), ConsumerGroup (+12 more)

### Community 60 - "部署指南"
Cohesion: 0.12
Nodes (17): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查, 1. 部署方式, 2. 架构总览, 3. 前置要求, 5.1 `.env` 模板（`api/.env.example`） (+9 more)

### Community 61 - "AgentTaskRunner"
Cohesion: 0.13
Nodes (11): AgentTaskRunner, AgentTaskRunnerConfig, COSFileStorage, MultiFunctionTool, readerWrapper, Tool, ToolRegistry, AgentConfig (+3 more)

### Community 63 - "Error"
Cohesion: 0.18
Nodes (13): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Internal(), New(), NotFound(), ToInternal() (+5 more)

### Community 64 - "app_test.go"
Cohesion: 0.10
Nodes (29): Build(), newLifecycleManager(), normalizeFactories(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce(), TestBuildInitializeErrorWrapsSentinel() (+21 more)

### Community 65 - "contains"
Cohesion: 0.21
Nodes (9): TestEvent_Data_JSONRoundtrip(), TestEvent_Data_Nil(), TestEvent_Data_NotBase64(), TestEvent_Data_PreservesStructure(), TestRequestPolicy_Extra_JSONRoundtrip(), TestRequestPolicy_Extra_NestedObject(), TestRequestPolicy_Extra_Omitempty(), contains() (+1 more)

### Community 66 - "StdioMCPClient"
Cohesion: 0.12
Nodes (16): MCPClientManager, MCPToolInfo, NewStdioMCPClient(), MCPClient, MCPConfig, MCPConfigServer, MCPContent, MCPError (+8 more)

### Community 67 - "BuildRuntimeConfigFromModel"
Cohesion: 0.26
Nodes (10): defaultLLMClientFactory(), BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), runtimeConfigToOpenAIClientConfig(), runtimeHealthFromModel(), runtimeHealthToModel(), TestBuildRuntimeConfigFromModel() (+2 more)

### Community 68 - "MergeDeltas"
Cohesion: 0.29
Nodes (10): MergeDeltas(), repeat(), TestEstimateContextTokens(), TestMergeDeltas_ContentOnly(), TestMergeDeltas_MultipleToolCalls(), TestMergeDeltas_ReasoningAndContent(), TestMergeDeltas_ReasoningOnly(), TestMergeDeltas_ToolCall() (+2 more)

### Community 69 - "Agent 核心架构"
Cohesion: 0.13
Nodes (15): 1. 整体架构, 2.1 BaseAgent, 2.2 PlannerAgent, 2.3 ReActAgent, 2.4 Flow 编排, 2. 核心组件, 3. Memory 系统, 4. 会话管理 (+7 more)

### Community 70 - "App"
Cohesion: 0.21
Nodes (8): BuildWithFactories(), DefaultOptions(), newRepositories(), App, lifecycleManager, Options, repositories, sync.Mutex

### Community 71 - "event.go"
Cohesion: 0.06
Nodes (35): BaseEvent, Plan, PlanStep, NewDoneEvent(), NewErrorEvent(), NewMessageEvent(), NewPlanEvent(), NewStepEvent() (+27 more)

### Community 72 - "logger.go"
Cohesion: 0.09
Nodes (38): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), Any(), Bool(), Dur() (+30 more)

### Community 73 - "NewBaseAgent"
Cohesion: 0.31
Nodes (8): A2AAgent, AgentConfig, MCPServer, NewBaseAgent(), DefaultAgentConfig(), TestBaseAgent_InvokeWithEmptyRetry(), TestBaseAgent_InvokeWithEmptyRetry_AllEmpty(), TestBaseAgent_InvokeWithEmptyRetry_FirstSuccess()

### Community 74 - "llm.go"
Cohesion: 0.24
Nodes (8): ModelIDFromContext(), NewDynamicLLM(), NewDynamicLLMWithFactory(), WithModelID(), DynamicLLM, LLMClientFactory, LLMConfigProvider, llmModelIDKey

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - ".initRoutes"
Cohesion: 0.18
Nodes (11): defaultTrustedProxies(), NewAppConfigHandler(), NewLLMModelHandler(), AppConfigService, LLMModelService, CORS(), generateRequestID(), Logger() (+3 more)

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.15
Nodes (9): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 健康检查, 常用命令, 本地开发, 核心代码指引, 许可证 (+1 more)

### Community 78 - "工具系统 (Tool System)"
Cohesion: 0.15
Nodes (13): 1. 概述, 4.1 MCP 客户端, 4.2 工具注册, 4. MCP 集成, 5.1 转换为 LLM 格式, 5.2 解析 LLM 响应, 5. LLM 工具调用, 6.1 步骤 1：定义工具 (+5 more)

### Community 79 - "OSS"
Cohesion: 0.24
Nodes (9): generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), github.com/aws/aws-sdk-go-v2/feature/s3/manager.Uploader, github.com/aws/aws-sdk-go-v2/service/s3.Client, Provider (+1 more)

### Community 80 - "LLMResponse"
Cohesion: 0.15
Nodes (6): mockLLM, mockLLMForTest, LLMRequest, NormalizeDelta(), NormalizeResponse(), LLMResponse

### Community 81 - "json_benchmark_test.go"
Cohesion: 0.22
Nodes (15): Benchmark_Sonic_Marshal_Large(), Benchmark_Sonic_Marshal_Small(), Benchmark_Sonic_Roundtrip_Large(), Benchmark_Sonic_Roundtrip_Small(), Benchmark_Sonic_Unmarshal_Large(), Benchmark_Sonic_Unmarshal_Small(), Benchmark_StdLib_Marshal_Large(), Benchmark_StdLib_Marshal_Small() (+7 more)

### Community 82 - "MessageTool"
Cohesion: 0.19
Nodes (4): MessageTool, TestMessageTool_NotifyUserSchema(), NewMessageTool(), TestMessageTool()

### Community 83 - "NewProviderError"
Cohesion: 0.31
Nodes (8): isFallbackable(), isRetryable(), NewProviderError(), TestIsKind(), TestProviderError_AuthNotFallbackable(), TestProviderError_Kind(), ErrorKind, ProviderError

### Community 84 - "DefaultFileService"
Cohesion: 0.14
Nodes (5): attachmentStorage, DefaultFileService, mockStorage, io.ReadCloser, io.Reader

### Community 85 - "app/page.tsx"
Cohesion: 0.24
Nodes (7): ChatInput, ChatInputProps, ChatInputRef, SuggestedQuestions(), SuggestedQuestionsProps, suggestedQuestions, FileInfo

### Community 86 - ".loadOne"
Cohesion: 0.36
Nodes (7): TestShouldInlineAndTruncate(), FileContext, IsLikelyBinary(), ShouldInline(), ShouldRAG(), ShouldTruncate(), LoadMode

### Community 87 - "集成测试"
Cohesion: 0.14
Nodes (13): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+5 more)

### Community 88 - "stubLLM"
Cohesion: 0.31
Nodes (4): LLMRequest, TestDynamicLLM_UsesFactory(), TestRoutedLLMFromSingleProvider(), stubLLM

### Community 89 - "Handlers"
Cohesion: 0.38
Nodes (5): StatusHandler, NewStatusHandler(), SetupRoutes(), github.com/gin-gonic/gin.Engine, Handlers

### Community 90 - "status_service_test.go"
Cohesion: 0.15
Nodes (6): TestStatusService_GetHealthStatus_SkippedServicesDegraded(), TestStatusService_GetHealthStatus_WrapsHealthErrors(), TestToInternal(), fakeOSS, fakePostgres, fakeRedis

### Community 91 - "response.go"
Cohesion: 0.40
Nodes (3): TotalResponse, SuccessWithMsg(), SuccessWithTotal()

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.20
Nodes (9): API 接口, Docker 部署, Manus 沙箱服务, 使用开发容器, 启动服务, 技术栈, 本地开发, 架构 (+1 more)

### Community 93 - "Manus 前端 UI"
Cohesion: 0.20
Nodes (9): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 环境准备 (+1 more)

### Community 94 - "parseResponse"
Cohesion: 0.14
Nodes (36): Response, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_EmptyUpdate(), TestAppConfigAPI_InvalidJSON(), TestAppConfigAPI_LLMConfig_Lifecycle(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle() (+28 more)

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

### Community 101 - "SessionHandler"
Cohesion: 0.29
Nodes (4): SessionHandler, mergeEventMetadata(), setSSEHeaders(), TestSetSSEHeaders()

### Community 102 - "BuildAttachmentContextSection"
Cohesion: 0.32
Nodes (5): BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext()

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

### Community 107 - "markdown-content.tsx"
Cohesion: 0.33
Nodes (6): components, headingClasses, MarkdownContent(), MarkdownContentProps, normalizeAutolinks(), URL_FOLLOWED_BY_CJK

### Community 108 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 109 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 110 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 111 - "快速部署"
Cohesion: 0.33
Nodes (6): 一键部署（推荐）, 前置要求, 容器列表, 快速部署, 服务架构（Nginx 网关模式）, 本地开发

### Community 112 - "vnc-overlay.tsx"
Cohesion: 0.31
Nodes (6): buildVNCUrl(), VNCOverlay(), VNCOverlayProps, VNCViewer, VNCStatus, VNCViewerProps

### Community 113 - "go-manus 实战经验与面试考点沉淀（2026-09-04）"
Cohesion: 0.25
Nodes (7): 7.1 资深 Go 后端, 7.2 系统设计, 7.3 工程实践, 7. 综合面试题, 8. 参考资料, go-manus 实战经验与面试考点沉淀（2026-09-04）, 目录

### Community 114 - "3. Docker Desktop 替换：macOS 容器运行时选型"
Cohesion: 0.25
Nodes (8): 3.1 现象, 3.2 What, 3.3 主流方案对比, 3.4 推荐方案：Colima, 3.5 迁移关键点, 3.6 资源分配建议（16GB MacBook）, 3.7 面试要点, 3. Docker Desktop 替换：macOS 容器运行时选型

### Community 115 - "response_test.go"
Cohesion: 0.29
Nodes (5): TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure(), TestSuccess_HttpResponse(), TestTotalResponse_Structure()

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

### Community 120 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

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

### Community 126 - "八、改进建议"
Cohesion: 0.50
Nodes (4): 中优先级, 低优先级, 八、改进建议, 高优先级

### Community 127 - "4. 快速启动（nginx 网关模式，推荐）"
Cohesion: 0.50
Nodes (4): 4.1 准备环境变量, 4.2 一键启动, 4.3 访问入口（统一走 nginx）, 4. 快速启动（nginx 网关模式，推荐）

### Community 128 - "2. 核心接口"
Cohesion: 0.50
Nodes (4): 2.1 Tool 接口, 2.2 工具定义, 2.3 执行结果, 2. 核心接口

### Community 129 - "7. 安全考虑"
Cohesion: 0.50
Nodes (4): 7.1 沙箱隔离, 7.2 权限控制, 7.3 输入验证, 7. 安全考虑

### Community 148 - "validConfig"
Cohesion: 0.53
Nodes (5): TestConfigApplyDefaultsUsesDomainDefaults(), TestValidate_OK(), TestValidate_RequiredFields(), validConfig(), Config

### Community 157 - "常量按领域归属重构计划"
Cohesion: 0.33
Nodes (5): 常量按领域归属重构计划, 目标, 约束, 遇到的错误, 阶段

### Community 158 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 159 - "Impact Analysis with GitNexus"
Cohesion: 0.22
Nodes (8): Checklist, Example: "What breaks if I change validateUser?", Impact Analysis with GitNexus, Risk Assessment, Tools, Understanding Output, When to Use, Workflow

### Community 160 - "重构发现"
Cohesion: 0.40
Nodes (4): 实施结果, 当前发现, 明确不抽取, 重构发现

### Community 161 - "FileHandler"
Cohesion: 0.39
Nodes (5): FileHandler, NewFileHandler(), FileService, NewFileService(), COSFileStorage

### Community 172 - "task_runner_test.go"
Cohesion: 0.25
Nodes (7): mustMarshal(), safeMarshal(), TestAgentService_ResolveMessageAttachments(), TestAgentTaskRunner_SyncUserAttachmentsToSandbox(), TestMustMarshal_BackwardCompatibility(), TestPopRetryConfig(), TestSafeMarshal()

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

## Knowledge Gaps
- **548 isolated node(s):** `LLMConfig`, `SearchConfig`, `MCPServer`, `A2AAgent`, `github.com/Huang131/go-manus/api` (+543 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Info()` connect `Err` to `StdioMCPClient`, `AgentService`, `MCPTool`, `SessionHandler`, `logger.go`, `Factories`, `.initRoutes`, `A2ATool`, `OSS`, `.loadOne`, `Postgres`, `RedisStreamTask`, `AgentTaskRunner`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Why does `Err()` connect `Err` to `context.Context`, `StdioMCPClient`, `MCPTool`, `NewMockSessionRepository`, `event.go`, `logger.go`, `App`, `llm.go`, `Factories`, `.initRoutes`, `A2ATool`, `BaseAgent`, `DefaultFileService`, `.loadOne`, `sync.RWMutex`, `.ExecCommand`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `AgentService` connect `AgentService` to `context.Context`, `NewAppConfigService`, `MCPTool`, `Sandbox`, `File`, `external/search.go`, `SessionHandler`, `Err`, `App`, `A2ATool`, `PostgresSessionRepository`, `NewSessionHandler`, `RedisStreamTask`, `sync.RWMutex`, `AgentTaskRunner`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **What connects `LLMConfig`, `SearchConfig`, `MCPServer` to the rest of the system?**
  _548 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.04891740176423416 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.04701733764325595 - nodes in this community are weakly interconnected._
- **Should `LLMModel` be split into smaller, more focused modules?**
  _Cohesion score 0.06049382716049383 - nodes in this community are weakly interconnected._