# Graph Report - go-manus  (2026-09-14)

## Corpus Check
- 312 files · ~894,252 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3809 nodes · 9177 edges · 208 communities (182 shown, 26 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 699 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a33b4c86`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- index.tsx
- cn
- BuildAttachmentContextSection
- AppException
- index.ts
- NewToolError
- queryer
- MCPSetting.tsx
- mooc-manus vs go-manus 功能差异与缺失分析
- testing.T
- app_test.go
- session-detail-view.tsx
- PostgresSessionRepository
- SupervisorService
- go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter
- Docker 构建优化实践笔记
- StdioMCPClient
- AnthropicClient
- NewDefaultTaskRegistry
- markdown-content.tsx
- llm_model.go
- TestMessageQueueMockOperations
- go-manus 前后端端到端 Streaming Review 与重构记录
- batchTaskOutputMQ
- logger 包 os.Stdout 误关闭 Bug 深度解析
- 2. 根因链：11 个 Bug 拆解
- chat-input.tsx
- fallback_llm_test.go
- endpoints/file.py
- compilerOptions
- OpenAIClient
- CODE_REVIEW_2026-09-11.md
- Config
- endpoints/shell.py
- Tool
- LLMModel
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- FileHandler
- .initRoutes
- Detailed Findings
- Session
- RedisStreamTask
- MessageTool
- test_api_endpoints.py
- NewAppConfigService
- BadRequest
- PlannerReActFlow
- devDependencies
- 代码智能数据使用说明
- BaseAgent
- a2a_client.go
- NewMockSessionRepository
- fallback_test.go
- dependencies
- components.json
- CleanupSession
- tools_integration_test.go
- vnc-overlay.tsx
- json_parser_repair.go
- external/search.go
- 部署指南
- 快速部署
- .browserExec
- testLLMModelRepo
- context.Context
- ShellService
- anthropic_llm_test.go
- NewBaseAgent
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- go-manus 项目代码异味（Smell）分析与重构建议
- 4.3 11 个 Bug 拆解
- event.go
- logger.go
- LLMRuntimeConfig
- Err
- Refactoring with GitNexus
- routes.py
- go-manus - 通用 AI Agent 系统
- Event
- OSS
- MergeDeltas
- json_benchmark_test.go
- SessionHandler
- A2ATool
- Message
- github.com/gin-gonic/gin.Context
- File
- 集成测试
- inMemoryMessageQueue
- NewRedisStreamTask
- ServiceRegressionTests
- ApiEndpointTests
- Manus 沙箱服务
- Manus 前端 UI
- 二、Findings（问题清单）
- Commands
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- BrowserTool
- DefaultAppConfigService
- AppConfigHandler
- status_service_test.go
- go-manus 面试指南
- stubLLM
- docs/README.md
- Python vs Go 版本文件对比报告
- ._iter_file_lines
- RuntimeHealth
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- NewSimpleMemory
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- next-themes
- session-item.tsx
- LLM
- FileRepository
- 3.3 Repository 层架构设计
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- 当前架构总览
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- Factories
- 七、详细模块对比
- PlanSchemaJSON
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- Logger stdout 关闭事故复盘
- LLM 集成与 Planner 契约
- 3.4 集成测试
- app.go
- AgentService
- LLMResponse
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 3. 核心知识点详解
- @radix-ui/react-avatar
- @radix-ui/react-scroll-area
- @radix-ui/react-slot
- NewLLMModelService
- 5. 注意事项和踩坑点
- DefaultFileService
- 3.2 PostgreSQL INTERVAL 与参数化查询
- @radix-ui/react-switch
- 3.5 Session 与 Agent 架构
- react-dom
- 4. 可能的面试追问
- react-markdown
- createSessionForTest
- postcss.config.mjs
- validConfig
- sandbox
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- Impact Analysis with GitNexus
- 7.4 存储层详细对比
- remark-gfm
- 核心概念
- test-env-down.sh
- test-env-up.sh
- github.com/Huang131/go-manus/api
- tools_test.go
- mockFailingTool
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- SearchResults
- next.config.ts
- BaseModel
- sensenova_integration_test.go
- 7.5 外部依赖详细对比
- .loadOne
- Sandbox 审查发现
- Sandbox 问题修复计划
- 维护状态
- App
- 三、服务层
- external_test.go
- 六、差异汇总
- 八、改进建议
- Sandbox 修复进度
- archive/README.md
- NewOpenAIClient
- vnc_handler_test.go
- NewProviderError
- DefaultFileCleanupService
- response_test.go
- ToolResult
- openai_llm_test.go
- sonner
- package.json
- Postgres
- eslint.config.mjs
- @novnc/novnc
- @radix-ui/react-label
- tailwind-merge

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 104 edges
3. `File` - 79 edges
4. `Err()` - 62 edges
5. `ServiceRegressionTests` - 56 edges
6. `ShellService` - 53 edges
7. `RedisStreamTask` - 46 edges
8. `FileService` - 45 edges
9. `NewRedisStreamTask()` - 44 edges
10. `FromError()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `TestShouldInlineAndTruncate()` --calls--> `ShouldRAG()`  [INFERRED]
  api/internal/agent/attachment/loader_test.go → api/internal/agent/attachment/policy.go
- `BaseAgent` --references--> `Memory`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewToolRegistry()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/tools.go
- `NewBaseAgentWithParser()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go

## Import Cycles
- None detected.

## Communities (208 total, 26 thin omitted)

### Community 0 - "index.tsx"
Cohesion: 0.07
Nodes (39): A2APreview(), BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent(), getToolDescription(), MCPPreview(), renderToolIcon() (+31 more)

### Community 1 - "cn"
Cohesion: 0.06
Nodes (58): metadata, GlobalHeader(), LeftPanel(), DialogOverlay(), Field(), FieldContent(), FieldError(), FieldSeparator() (+50 more)

### Community 2 - "BuildAttachmentContextSection"
Cohesion: 0.32
Nodes (5): BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext()

### Community 3 - "AppException"
Cohesion: 0.09
Nodes (17): Exception, FastAPI, register_exception_handlers(), AppException, BadRequestException, NotFoundException, Any, 文件操作只接受普通文件，避免目录被当作文件读取或删除。 (+9 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (53): SessionHeaderProps, UseSessionDetailResult, configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout() (+45 more)

### Community 5 - "NewToolError"
Cohesion: 0.29
Nodes (4): SandboxClient, sandboxToolResult(), NewToolError(), sandboxResponse

### Community 6 - "queryer"
Cohesion: 0.12
Nodes (8): scanFile(), collectRows(), pgx.Tx, T, newQueryer(), pgx.Rows, PostgresFileRepository, queryer

### Community 7 - "MCPSetting.tsx"
Cohesion: 0.11
Nodes (41): DeleteSessionDialogProps, ManusSettings(), SETTING_MENUS, SettingTab, emptyDraft, Mode, ModelConfigManager(), RenameSessionDialogProps (+33 more)

### Community 8 - "mooc-manus vs go-manus 功能差异与缺失分析"
Cohesion: 0.05
Nodes (40): 2.1 整体路由映射, 2.2 **仍然缺失的关键路由**, 3.1 Planner 阶段: JSON 强制输出 ✅ Go 更优, 3.2 **空响应重试注入 (重要差异)**, 3.3 LLM 错误重试策略, 3.4 ReAct 阶段的 JSON 约束, 3.5 流式事件分发, 3.6 Agent 任务执行架构 (+32 more)

### Community 9 - "testing.T"
Cohesion: 0.05
Nodes (73): TestFlowStatus_Values(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters(), TestAgentService_GetActiveTaskIDClearsCompletedTask(), TestAgentService_ResolveMessageAttachments() (+65 more)

### Community 10 - "app_test.go"
Cohesion: 0.11
Nodes (28): Build(), BuildWithFactories(), newLifecycleManager(), normalizeFactories(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce() (+20 more)

### Community 11 - "session-detail-view.tsx"
Cohesion: 0.08
Nodes (40): PageProps, AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock(), ToolRow() (+32 more)

### Community 12 - "PostgresSessionRepository"
Cohesion: 0.09
Nodes (13): stopSessionRepository, successfulStopSessionRepository, TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits(), TestAgentService_StopSessionReturnsStatusUpdateError(), pgx.Tx, SessionRepository, marshalSessionEvents(), NewSessionRepository() (+5 more)

### Community 13 - "SupervisorService"
Cohesion: 0.09
Nodes (31): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+23 more)

### Community 14 - "go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter"
Cohesion: 0.05
Nodes (40): Adapter 接口, AnthropicAdapter, CustomGatewayAdapter, go-manus 多大模型支持技术方案：强类型核心协议 + 控制面 + Provider Adapter, LLM 控制面, ModelHint, OpenAICompatibleAdapter, Provider Adapter (+32 more)

### Community 15 - "Docker 构建优化实践笔记"
Cohesion: 0.05
Nodes (38): 1.1 问题分析, 1.2 解决方案, 1.3 验证结果, 2.1 问题分析, 2.2 解决方案, 2.3 验证结果, 3.1 问题分析, 3.2.1 使用 uv 替代 pip (+30 more)

### Community 16 - "StdioMCPClient"
Cohesion: 0.11
Nodes (18): MCPClientManager, MCPToolInfo, NewStdioMCPClient(), MCPClient, MCPConfig, MCPConfigServer, MCPContent, MCPError (+10 more)

### Community 17 - "AnthropicClient"
Cohesion: 0.14
Nodes (15): LLMRequest, mustMarshalMap(), parseJSONMap(), textOf(), userContent(), streamContext(), AnthropicClient, AnthropicContent (+7 more)

### Community 18 - "NewDefaultTaskRegistry"
Cohesion: 0.24
Nodes (14): NewDefaultTaskRegistry(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_CleanupCompletedWaitsForFinished(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister(), TestDefaultTaskRegistry_List() (+6 more)

### Community 19 - "markdown-content.tsx"
Cohesion: 0.33
Nodes (6): components, headingClasses, MarkdownContent(), MarkdownContentProps, normalizeAutolinks(), URL_FOLLOWED_BY_CJK

### Community 20 - "llm_model.go"
Cohesion: 0.18
Nodes (13): DefaultCapabilities(), CostPolicy, ModelCapabilities, RequestPolicy, MergeDefaultCapabilities(), encoding/json.RawMessage, CostPolicy, LLMModelListResponse (+5 more)

### Community 21 - "TestMessageQueueMockOperations"
Cohesion: 0.27
Nodes (5): TestMessageQueueDefaultTimeouts(), TestMessageQueueGetBlockingHonorsContext(), TestMessageQueueInterface(), TestMessageQueueMockOperations(), mockMQ

### Community 22 - "go-manus 前后端端到端 Streaming Review 与重构记录"
Cohesion: 0.06
Nodes (33): 1. 结论, 2.1 后端任务事件链路, 2.2 前端续读链路, 2.3 Provider streaming 链路, 2. 当前链路核对, 3. 已确认问题, 4. 已处理的 P2 问题, 5. 仍待处理的维护性建议 (+25 more)

### Community 23 - "batchTaskOutputMQ"
Cohesion: 0.29
Nodes (4): batchTaskOutputMQ, StreamMessage, BatchMessageQueue, MessageQueue

### Community 24 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 25 - "2. 根因链：11 个 Bug 拆解"
Cohesion: 0.06
Nodes (30): 1. 现象回顾, 2.1 模型能力对比（当前项目已测模型）, 2. 根因链：11 个 Bug 拆解, 3.1 简单问答, 3.2 带工具调用的完整任务（回归 Bug 9 场景）, 3. 修复后验证（端到端）, 4. 原项目 mooc-manus (Python) 是否存在这些问题？, 5.1 协议层的"两次包装" (+22 more)

### Community 26 - "chat-input.tsx"
Cohesion: 0.11
Nodes (22): ChatInput, ChatInputProps, ChatInputRef, SuggestedQuestions(), SuggestedQuestionsProps, Item(), ItemActions(), ItemFooter() (+14 more)

### Community 27 - "fallback_llm_test.go"
Cohesion: 0.18
Nodes (28): RoutedLLM, modelNames(), newMockRuntimeHealthStore(), openAIModelWithID(), openAITextProfile(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_AutoAppendsConfiguredEnvFallback(), TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback() (+20 more)

### Community 28 - "endpoints/file.py"
Cohesion: 0.13
Nodes (35): api_route, FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get (+27 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "OpenAIClient"
Cohesion: 0.12
Nodes (21): LLMRequest, estimateCostUSD(), LLMRequest, reasoningTokens(), sendStreamDelta(), toOpenAIMessages(), toOpenAITools(), truncateBody() (+13 more)

### Community 31 - "CODE_REVIEW_2026-09-11.md"
Cohesion: 0.09
Nodes (22): A. 流式链路复查（commit 4892225）, api, B. service 层专项（新发现）, C. 昨日 backlog 复核（HEAD b1f3827 仍开放）, D. 修复批次, E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）, sandbox, ui (+14 more)

### Community 32 - "Config"
Cohesion: 0.08
Nodes (24): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, RedisConfig, MCPConfig (+16 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.20
Nodes (23): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+15 more)

### Community 34 - "Tool"
Cohesion: 0.13
Nodes (12): AgentTaskRunnerConfig, COSFileStorage, MultiFunctionTool, readerWrapper, Tool, ToolRegistry, AgentConfig, NewPlannerReActFlow() (+4 more)

### Community 35 - "LLMModel"
Cohesion: 0.10
Nodes (17): LLMModel, LLMModelTestResponse, pgx.Tx, LLMModelRepository, modelJSON(), modelJSONFields(), NewLLMModelRepositoryWithTx(), runtimeHealthToJSON() (+9 more)

### Community 36 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 37 - "FileHandler"
Cohesion: 0.24
Nodes (5): FileHandler, NewFileHandler(), FileService, SessionService, Handlers

### Community 38 - ".initRoutes"
Cohesion: 0.11
Nodes (36): defaultTrustedProxies(), NewSessionHandler(), NewMockSessionServiceForHandler(), setupRouter(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), TestSessionHandler_Chat(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create() (+28 more)

### Community 39 - "Detailed Findings"
Cohesion: 0.10
Nodes (20): Architecture Smell Report, Detailed Findings, Executive Summary, Historical Findings Status, Immediate Actions, Long Term, Module Health Scorecard, P0: 当前工作树 logger 测试回归 (+12 more)

### Community 40 - "Session"
Cohesion: 0.07
Nodes (7): mockSessionRepo, Session, SessionStatus, unixPointer(), time.Time, DefaultSessionService, MockSessionRepository

### Community 41 - "RedisStreamTask"
Cohesion: 0.07
Nodes (14): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask, TaskRegistryInterface, TaskRunner, CompletedStreamRetention(), TestRedisMessageQueueDefaultTimeoutsUseDurations() (+6 more)

### Community 42 - "MessageTool"
Cohesion: 0.14
Nodes (5): MessageTool, TestMessageTool_NotifyUserSchema(), NewMessageTool(), TestMessageTool(), NewToolResultWithMessage()

### Community 43 - "test_api_endpoints.py"
Cohesion: 0.07
Nodes (30): BaseSettings, HTTPConnection, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, get_file_service() (+22 more)

### Community 44 - "NewAppConfigService"
Cohesion: 0.13
Nodes (25): AppConfig, AppConfigType, pgx.Tx, AppConfigRepository, marshalConfigValue(), NewAppConfigRepository(), NewAppConfigRepositoryWithTx(), NewAppConfigService() (+17 more)

### Community 45 - "BadRequest"
Cohesion: 0.18
Nodes (15): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Internal(), New(), NotFound(), ToInternal() (+7 more)

### Community 46 - "PlannerReActFlow"
Cohesion: 0.25
Nodes (7): FlowStatus, PlannerReActFlow, TaskInput, BaseEvent, NewErrorEvent(), NewMessageEvent(), InfoContext()

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.11
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "BaseAgent"
Cohesion: 0.12
Nodes (10): BaseAgent, InvokeResult, ToolCallResult, AgentConfig, NewBaseAgentWithParser(), shouldPublishDeltas(), JSONParser, NewDefaultJSONParser() (+2 more)

### Community 50 - "a2a_client.go"
Cohesion: 0.16
Nodes (14): A2AClientManager, NewA2AClientManager(), A2AAgentCapabilities, A2AAgentCard, A2AAgentSkill, A2AClientManagerConfig, A2AInvokeResult, A2AJSONRPCError (+6 more)

### Community 51 - "NewMockSessionRepository"
Cohesion: 0.22
Nodes (21): NewSessionService(), NewSessionServiceWithSandbox(), newMockFileRepo(), NewMockSessionRepository(), TestSessionService_AppendEvent(), TestSessionService_ClearUnreadCount(), TestSessionService_CreateSession(), TestSessionService_CreateSession_RepositoryError() (+13 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.16
Nodes (19): CanFallbackAfterToolUse(), CanFallbackTo(), fullCaps(), fullProfile(), ModelCapabilities, TestCanFallbackAfterToolUse_AllReadOnly(), TestCanFallbackAfterToolUse_HasWriteTool(), TestCanFallbackAfterToolUse_NoTools() (+11 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "CleanupSession"
Cohesion: 0.10
Nodes (67): Response, TotalResponse, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_EmptyUpdate(), TestAppConfigAPI_InvalidJSON(), TestAppConfigAPI_LLMConfig_Lifecycle(), TestAppConfigAPI_MCPConfig_Delete() (+59 more)

### Community 56 - "tools_integration_test.go"
Cohesion: 0.14
Nodes (15): attachmentSandbox, ShellTool, NewShellTool(), TestShellTool_Description(), TestShellTool_Invoke_Error(), TestShellTool_Invoke_Exec(), TestShellTool_Invoke_Kill(), TestShellTool_Invoke_Read() (+7 more)

### Community 57 - "vnc-overlay.tsx"
Cohesion: 0.31
Nodes (6): buildVNCUrl(), VNCOverlay(), VNCOverlayProps, VNCViewer, VNCStatus, VNCViewerProps

### Community 58 - "json_parser_repair.go"
Cohesion: 0.26
Nodes (9): extractJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), removeTrailingCommas(), TestExtractJSON(), TestRemoveMarkdownCodeBlocks(), TestRemoveTrailingCommas(), TestRepairJSONParser_Parse() (+1 more)

### Community 59 - "external/search.go"
Cohesion: 0.13
Nodes (16): SearchTool, NewSearchTool(), SearchEngine, NewBingSearchClient(), NewBingSearchClientWithTimeout(), NewGoogleSearchClient(), NewGoogleSearchClientWithTimeout(), NewSearchEngine() (+8 more)

### Community 60 - "部署指南"
Cohesion: 0.10
Nodes (21): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查, 1. 部署方式, 2. 架构总览, 3. 前置要求, 4.1 准备环境变量 (+13 more)

### Community 61 - "快速部署"
Cohesion: 0.33
Nodes (6): 一键部署（推荐）, 前置要求, 容器列表, 快速部署, 服务架构（Nginx 网关模式）, 本地开发

### Community 62 - ".browserExec"
Cohesion: 0.18
Nodes (8): browserScript(), jsString(), SandboxErrorStatus(), shellQuote(), BrowserClient, SandboxAPIError, sandboxErrorResponse, sandboxRequest

### Community 63 - "testLLMModelRepo"
Cohesion: 0.29
Nodes (19): cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError(), TestLLMModelRepo_GetByID_NotFoundReturnsNilNil(), TestLLMModelRepo_GetDefault_NoDefaultReturnsNilNil() (+11 more)

### Community 64 - "context.Context"
Cohesion: 0.06
Nodes (8): chatContextSessionRepository, mockSandbox, TestAgentServiceChatUsesDetachedContextForMessagePersistence(), DefaultFileService, mockMessageQueue, context.Context, MockSessionServiceForHandler, stubVNCService

### Community 65 - "ShellService"
Cohesion: 0.06
Nodes (25): ConsoleRecord, Process, Lock, 启动并持有输出读取任务，保证一个会话只消费当前进程的输出。, 取消并等待输出读取器，避免旧进程向新命令的记录写入输出。, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程, 追加当前命令输出并维护字节预算，避免每个输出块重复扫描历史记录。 (+17 more)

### Community 66 - "anthropic_llm_test.go"
Cohesion: 0.27
Nodes (18): NewAnthropicClient(), anthropicErrorResponse(), errKind(), newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_NetworkError(), TestAnthropicClient_ProtocolError_NotFallbackable() (+10 more)

### Community 67 - "NewBaseAgent"
Cohesion: 0.12
Nodes (18): A2AAgent, AgentConfig, MCPServer, PlannerAgent, ReActAgent, NewBaseAgent(), TestBaseAgentInvokePublishesTextDeltas(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted() (+10 more)

### Community 68 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 69 - "go-manus 项目代码异味（Smell）分析与重构建议"
Cohesion: 0.09
Nodes (22): 4.1 🔴 C1~C5：`json_parser_repair.go` 的 Bug 与 Dead Code, 4.2 🟡 W1：`AgentService` God Object, 4.3 🟡 W6/W7：`PlannerReActFlow` 与 `TaskRunner` 的 Long Method + Switch, 4.4 🟡 W9：`SessionRepository` Fat Interface, 🔴 Critical（必须立刻处理）, go-manus 项目代码异味（Smell）分析与重构建议, Immediate（本周，1~2 天）, Long-Term（1 个月+） (+14 more)

### Community 70 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 71 - "event.go"
Cohesion: 0.05
Nodes (38): TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), clonePlanSteps(), cloneStrings(), Plan, PlanStep, NewDoneEvent(), NewMessageDeltaEvent(), NewMessageDoneEvent() (+30 more)

### Community 72 - "logger.go"
Cohesion: 0.06
Nodes (53): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), Any(), Bool(), Dur() (+45 more)

### Community 73 - "LLMRuntimeConfig"
Cohesion: 0.19
Nodes (12): betterHealth(), canFallbackAfterToolUse(), configKey(), RoutedLLM, LLMRequest, hasCapabilityProfile(), healthRank(), updateLatencyMS() (+4 more)

### Community 74 - "Err"
Cohesion: 0.12
Nodes (11): AgentTaskRunner, COSFileStorage, RedisStreamMessageQueue, redis.Client, NewRedisStreamMessageQueue(), logRollbackFailure(), DebugContext(), Err() (+3 more)

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "routes.py"
Cohesion: 0.40
Nodes (3): APIRouter, create_api_routes(), 创建API路由，涵盖整个沙箱项目的所有API

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.14
Nodes (14): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 健康检查, 学习路径, 常用命令, 本地开发, 核心代码指引 (+6 more)

### Community 78 - "Event"
Cohesion: 0.11
Nodes (15): Stream, Task, TaskStream, nonNilEvents(), NewTaskStream(), parseTaskOutputMessages(), ReadTaskOutput(), readTaskOutput() (+7 more)

### Community 79 - "OSS"
Cohesion: 0.14
Nodes (16): ObjectStorageConfig, ensureBucket(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL() (+8 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (14): isToolCallFinishReason(), MergeDeltas(), NormalizeResponse(), repeat(), TestEstimateContextTokens(), TestMergeDeltas_AnthropicToolUseRetainsToolCall(), TestMergeDeltas_ContentOnly(), TestMergeDeltas_MultipleToolCalls() (+6 more)

### Community 81 - "json_benchmark_test.go"
Cohesion: 0.22
Nodes (15): Benchmark_Sonic_Marshal_Large(), Benchmark_Sonic_Marshal_Small(), Benchmark_Sonic_Roundtrip_Large(), Benchmark_Sonic_Roundtrip_Small(), Benchmark_Sonic_Unmarshal_Large(), Benchmark_Sonic_Unmarshal_Small(), Benchmark_StdLib_Marshal_Large(), Benchmark_StdLib_Marshal_Small() (+7 more)

### Community 82 - "SessionHandler"
Cohesion: 0.21
Nodes (11): SessionHandler, mergeEventMetadata(), newSSEContext(), parseChatRequest(), setSSEHeaders(), TestMergeEventMetadata(), TestMergeEventMetadata_NullPayloadReturnsOriginalData(), TestSetSSEHeaders() (+3 more)

### Community 83 - "A2ATool"
Cohesion: 0.19
Nodes (4): A2ATool, A2AConfig, marshalResponseText(), TestA2ATool_MarshalResponseTextHandlesUnsupportedData()

### Community 84 - "Message"
Cohesion: 0.10
Nodes (17): SimpleMemory, capabilitiesEquivalent(), ContextFits(), EstimateContextTokens(), ModelCapabilities, Message, ResponseFormat, ToolCall (+9 more)

### Community 85 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.18
Nodes (10): updateConfig(), LLMModelHandler, NewLLMModelHandler(), NewLLMModelResponse(), LLMModelService, FromError(), Success(), SuccessWithMsg() (+2 more)

### Community 86 - "File"
Cohesion: 0.05
Nodes (7): attachmentFileRepository, generatedFileRepository, File, cleanupRepoStub, emptyFileRepo, mockFileRepo, stubFileRepo

### Community 87 - "集成测试"
Cohesion: 0.14
Nodes (13): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+5 more)

### Community 88 - "inMemoryMessageQueue"
Cohesion: 0.15
Nodes (8): inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, newInMemoryMessageQueue(), parseStreamSeq(), TestToolCallingEvents_SSEStream(), TestToolCallingEvents_SSEStream_Failure()

### Community 89 - "NewRedisStreamTask"
Cohesion: 0.12
Nodes (23): mockMQWrapper, mockTaskRunner, retentionCall, NewRedisStreamTask(), TestReadTaskOutputAfterTaskUnregistered(), TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan() (+15 more)

### Community 90 - "ServiceRegressionTests"
Cohesion: 0.06
Nodes (7): FileService, UploadFile, 目录遍历只接受目录路径，尽早返回明确的 404。, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容, 按固定块读取 sudo 输出，避免超长单行触发 readline 缓冲上限。, 根据传递的文件夹路径+glob规则查询文件列表, ServiceRegressionTests

### Community 91 - "ApiEndpointTests"
Cohesion: 0.29
Nodes (3): ApiEndpointTests, Any, 通过 ASGI 协议直接调用应用，避免测试依赖额外 HTTP 客户端。

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.22
Nodes (8): API 接口, Docker 部署, Manus 沙箱服务, 信任模型与安全边界, 技术栈, 本地开发, 架构, 端口说明

### Community 93 - "Manus 前端 UI"
Cohesion: 0.18
Nodes (10): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 模型选择（Auto） (+2 more)

### Community 94 - "二、Findings（问题清单）"
Cohesion: 0.18
Nodes (10): 🔴 A. 正确性, 🟡 B. 重复代码, 🟡 C. 长方法 / 复杂度, 🔵 D. 分层 / 架构, 🔵 E. 其他, go-manus `api/` 代码 Review 与重构报告, 一、Executive Summary, 三、Refactoring 执行记录（本次已落地） (+2 more)

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 97 - "BrowserTool"
Cohesion: 0.11
Nodes (5): BrowserTool, FileTool, NewFileTool(), requiredToolString(), TestFileTool()

### Community 98 - "DefaultAppConfigService"
Cohesion: 0.10
Nodes (16): MCPServer, A2AConfig, AgentConfig, LLMConfig, MCPConfig, NewLLMConfigResponse(), TestLLMConfigRequest_MarshalJSON(), TestLLMConfigRequest_PreservesAPIKeyForInternalUpdate() (+8 more)

### Community 99 - "AppConfigHandler"
Cohesion: 0.22
Nodes (7): Unavailable(), getConfig(), AppConfigHandler, T, NewAppConfigHandler(), AppConfigService, configRuntimeReloader

### Community 100 - "status_service_test.go"
Cohesion: 0.14
Nodes (7): TestStatusService_GetHealthStatus_SkippedServicesDegraded(), TestStatusService_GetHealthStatus_WrapsHealthErrors(), TestStatusServiceUsesInjectedClock(), TestToInternal(), fakeOSS, fakePostgres, fakeRedis

### Community 101 - "go-manus 面试指南"
Cohesion: 0.25
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "stubLLM"
Cohesion: 0.31
Nodes (4): LLMRequest, TestDynamicLLM_UsesFactory(), TestRoutedLLMFromSingleProvider(), stubLLM

### Community 104 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 105 - "._iter_file_lines"
Cohesion: 0.25
Nodes (6): _LineChunk, 逐行收集范围内内容，任何上限命中后立即停止读取。, 按固定块解码文件，超长无换行内容达到上限即截断。, 文件流的一段逻辑行；truncated 表示单行超过内存上限。, 按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。, _truncate_utf8()

### Community 106 - "RuntimeHealth"
Cohesion: 0.14
Nodes (17): ModelIDFromContext(), BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), runtimeHealthFromModel(), runtimeHealthToModel(), TestBuildRuntimeConfigFromModel(), TestProtocolFromProvider() (+9 more)

### Community 107 - "6. 代码评审与回归测试：测试不是覆盖，而是契约"
Cohesion: 0.22
Nodes (9): 6.1 Why：测试是行为的契约, 6.2 历史 Bug 的回归测试, 6.3 测试分类, 6.4 Go 测试模式, 6.5 面试要点, 6. 代码评审与回归测试：测试不是覆盖，而是契约, 子测试 + t.Helper, 模糊测试（Go 1.18+） (+1 more)

### Community 108 - "NewSimpleMemory"
Cohesion: 0.36
Nodes (6): Memory, NewSimpleMemory(), TestSimpleMemory_Add(), TestSimpleMemory_Clear(), TestSimpleMemory_Compact(), TestSimpleMemory_GetMessages()

### Community 109 - "go-manus 实战经验与面试考点沉淀（2026-09-04）"
Cohesion: 0.25
Nodes (7): 7.1 资深 Go 后端, 7.2 系统设计, 7.3 工程实践, 7. 综合面试题, 8. 参考资料, go-manus 实战经验与面试考点沉淀（2026-09-04）, 目录

### Community 110 - "3. Docker Desktop 替换：macOS 容器运行时选型"
Cohesion: 0.25
Nodes (8): 3.1 现象, 3.2 What, 3.3 主流方案对比, 3.4 推荐方案：Colima, 3.5 迁移关键点, 3.6 资源分配建议（16GB MacBook）, 3.7 面试要点, 3. Docker Desktop 替换：macOS 容器运行时选型

### Community 112 - "session-item.tsx"
Cohesion: 0.07
Nodes (30): DeleteSessionDialog(), RenameSessionDialog(), SessionItem(), SessionItemProps, SessionList(), Avatar(), AvatarBadge(), AvatarFallback() (+22 more)

### Community 113 - "LLM"
Cohesion: 0.20
Nodes (12): defaultLLMClientFactory(), LLM, LLMClientFactory, NewDynamicLLM(), NewDynamicLLMWithFactory(), NewLLMClient(), NewRoutedLLMFromSingleProvider(), runtimeConfigToOpenAIClientConfig() (+4 more)

### Community 114 - "FileRepository"
Cohesion: 0.15
Nodes (9): pgx.Tx, FileRepository, NewFileRepository(), NewFileRepositoryWithTx(), pgx.Tx, T, joinRollbackError(), rollbackTx() (+1 more)

### Community 115 - "3.3 Repository 层架构设计"
Cohesion: 0.29
Nodes (7): 3.3 Repository 层架构设计, WithTx 事务封装, 事务支持：QueryContext 接口, 接口设计, 构造函数设计：普通版 vs 事务版, 软删除模式, 面试追问

### Community 116 - "5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异"
Cohesion: 0.29
Nodes (7): 5.1 现象, 5.2 What, 5.3 sonic vs encoding/json 差异, 5.4 sonic 正确使用姿势, 5.5 替换 JSON 库的检查清单, 5.6 面试要点, 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### Community 117 - "当前架构总览"
Cohesion: 0.33
Nodes (6): Planner 与 ReAct 边界, 当前架构总览, 当前限制, 流式事件, 请求链路, 重要代码入口

### Community 118 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 119 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 120 - "Factories"
Cohesion: 0.20
Nodes (13): defaultFactories(), COSFileStorage, FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(), TestFileCleanupSchedulerStartIsIdempotent() (+5 more)

### Community 121 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 123 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 124 - "Logger stdout 关闭事故复盘"
Cohesion: 0.33
Nodes (5): Logger stdout 关闭事故复盘, 修复, 根因, 现象, 验证

### Community 125 - "LLM 集成与 Planner 契约"
Cohesion: 0.33
Nodes (6): LLM 集成与 Planner 契约, Planner 业务契约, SenseNova, 安全与验证, 常见失败, 统一协议

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "app.go"
Cohesion: 0.17
Nodes (12): SandboxConfig, A2AAgent, A2AConfig, MCPConfig, MCPServer, DefaultOptions(), newA2AConfig(), newMCPConfig() (+4 more)

### Community 128 - "AgentService"
Cohesion: 0.10
Nodes (15): MCPTool, A2AConfig, AgentService, AgentConfig, COSFileStorage, MCPConfig, NewAgentService(), MCPConfig (+7 more)

### Community 129 - "LLMResponse"
Cohesion: 0.14
Nodes (6): mockLLM, mockLLMForTest, streamingAgentLLM, LLMRequest, LLMResponse, connectionTestLLM

### Community 130 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 131 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 135 - "NewLLMModelService"
Cohesion: 0.16
Nodes (27): NewLLMModelService(), NewLLMModelServiceWithLLMFactory(), NewMockLLMModelRepository(), TestLLMModelService_Create(), TestLLMModelService_Create_MapsUniqueViolationToConflict(), TestLLMModelService_Create_PreservesDefaultLookupError(), TestLLMModelService_Create_PreservesPartialCapabilities(), TestLLMModelService_Create_PreservesRepositoryError() (+19 more)

### Community 136 - "5. 注意事项和踩坑点"
Cohesion: 0.33
Nodes (6): 5.1 Go 错误处理, 5.2 PostgreSQL 参数化, 5.3 Repository 设计, 5.4 软删除查询, 5.5 文件清理, 5. 注意事项和踩坑点

### Community 137 - "DefaultFileService"
Cohesion: 0.09
Nodes (8): attachmentStorage, DefaultFileService, mockStorage, io.ReadCloser, io.Reader, cleanupStorageStub, COSFileStorage, stubStorage

### Community 138 - "3.2 PostgreSQL INTERVAL 与参数化查询"
Cohesion: 0.40
Nodes (5): 3.2 PostgreSQL INTERVAL 与参数化查询, Why：为什么 INTERVAL 不能参数化？, 问题背景, 面试追问, 项目中的正确实现

### Community 140 - "3.5 Session 与 Agent 架构"
Cohesion: 0.40
Nodes (5): 3.5 Session 与 Agent 架构, Agent Memory 机制, AppendEvent 业务逻辑, MessageEvent 结构, 面试追问

### Community 142 - "4. 可能的面试追问"
Cohesion: 0.40
Nodes (5): 4.1 Go 语言层面, 4.2 数据库层面, 4.3 架构设计层面, 4.4 测试层面, 4. 可能的面试追问

### Community 144 - "createSessionForTest"
Cohesion: 0.16
Nodes (28): deleteSessionDirect(), testFileRepo(), TestFileRepo_CountExpiredFiles(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound() (+20 more)

### Community 148 - "validConfig"
Cohesion: 0.53
Nodes (5): TestConfigApplyDefaultsUsesDomainDefaults(), TestValidate_OK(), TestValidate_RequiredFields(), validConfig(), Config

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

### Community 162 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

### Community 171 - "tools_test.go"
Cohesion: 0.13
Nodes (21): mockTool, NewA2ATool(), NewBrowserTool(), TestA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description(), TestA2ATool_Initialize() (+13 more)

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

### Community 178 - "SearchResults"
Cohesion: 0.67
Nodes (3): SearchEvent, SearchResultItem, SearchResults

### Community 181 - "sensenova_integration_test.go"
Cohesion: 0.44
Nodes (8): requireSenseNovaKey(), senseNovaReasoningEffort(), senseNovaRequest(), TestSenseNova_ChatModelsAreOpenAICompatible(), TestSenseNova_ImageModels(), TestSenseNova_ListModels(), TestSenseNova_StructuredJSONForPlannerCompatibleModels(), senseNovaChatResponse

### Community 182 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 183 - ".loadOne"
Cohesion: 0.10
Nodes (28): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+20 more)

### Community 184 - "Sandbox 审查发现"
Cohesion: 0.40
Nodes (4): Sandbox 审查发现, 已确认问题, 待覆盖测试, 本轮复核结论

### Community 185 - "Sandbox 问题修复计划"
Cohesion: 0.40
Nodes (4): Sandbox 问题修复计划, 目标, 约束, 阶段

### Community 186 - "维护状态"
Cohesion: 0.40
Nodes (5): 已完成, 已知限制, 维护状态, 维护规则, 验证基线

### Community 187 - "App"
Cohesion: 0.23
Nodes (4): App, lifecycleManager, sync.Mutex, sync.Once

### Community 188 - "三、服务层"
Cohesion: 0.50
Nodes (4): 3.1 AgentService, 3.2 SessionService, 3.3 FileService, 三、服务层

### Community 189 - "external_test.go"
Cohesion: 0.09
Nodes (20): LLMRequest, TestBrowser_Screenshot(), TestBrowserClientScreenshotExtractsSandboxOutput(), TestBrowserInterface(), TestBrowserScriptUsesNodeAndCDP(), TestLLM_Invoke(), TestLLM_Invoke_WithError(), TestLLM_ModelInfo() (+12 more)

### Community 190 - "六、差异汇总"
Cohesion: 0.50
Nodes (4): 6.1 功能缺失列表, 6.2 实现差异列表, 6.3 架构差异, 六、差异汇总

### Community 191 - "八、改进建议"
Cohesion: 0.50
Nodes (4): 中优先级, 低优先级, 八、改进建议, 高优先级

### Community 194 - "NewOpenAIClient"
Cohesion: 0.24
Nodes (10): NewOpenAIClient(), newTestClient(), TestNewOpenAIClient_DoesNotApplyGlobalTimeout(), CostPolicy, ExtraParam, RequestPolicy, AnthropicClientConfig, OpenAIClientConfig (+2 more)

### Community 195 - "vnc_handler_test.go"
Cohesion: 0.28
Nodes (8): sameOrigin(), newLocalTestServer(), TestVNCProxy_AllowsSameOriginAndNonBrowserClient(), TestVNCProxy_ProxyEcho(), TestVNCProxy_RejectsCrossOrigin(), TestVNCProxy_ServiceError(), net/http.Handler, net/http/httptest.Server

### Community 196 - "NewProviderError"
Cohesion: 0.50
Nodes (5): ErrorKind, isFallbackable(), isRetryable(), NewProviderError(), ProviderError

### Community 198 - "response_test.go"
Cohesion: 0.29
Nodes (5): TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure(), TestSuccess_HttpResponse(), TestTotalResponse_Structure()

### Community 220 - "ToolResult"
Cohesion: 0.11
Nodes (4): TestBrowser_Navigate(), ToolResult, MockBrowser, MockSandbox

### Community 222 - "openai_llm_test.go"
Cohesion: 0.17
Nodes (20): newTransportClient(), rawOK(), responseJSON(), TestOpenAIClient_401_Auth(), TestOpenAIClient_429_RateLimit(), TestOpenAIClient_5xx_Server(), TestOpenAIClient_ContentAndReasoning_Separated(), TestOpenAIClient_ContentOnly() (+12 more)

### Community 298 - "package.json"
Cohesion: 0.22
Nodes (8): name, private, scripts, build, dev, lint, start, version

### Community 303 - "Postgres"
Cohesion: 0.13
Nodes (13): newRepositories(), StatusHandler, NewStatusHandler(), Postgres, NewPostgres(), Redis, redis.Client, NewRedis() (+5 more)

## Knowledge Gaps
- **704 isolated node(s):** `LLMConfig`, `SearchConfig`, `MCPServer`, `A2AAgent`, `github.com/Huang131/go-manus/api` (+699 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Err()` connect `Err` to `AgentService`, `DefaultFileService`, `StdioMCPClient`, `FileHandler`, `.initRoutes`, `RedisStreamTask`, `BadRequest`, `PlannerReActFlow`, `BaseAgent`, `.loadOne`, `App`, `DefaultFileCleanupService`, `event.go`, `logger.go`, `LLMRuntimeConfig`, `Event`, `OSS`, `SessionHandler`, `A2ATool`, `github.com/gin-gonic/gin.Context`, `RuntimeHealth`, `app.go`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `toJSON()` connect `event.go` to `RedisStreamTask`, `Err`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **Why does `BrowserTool` connect `BrowserTool` to `tools_test.go`, `app.go`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **What connects `LLMConfig`, `SearchConfig`, `MCPServer` to the rest of the system?**
  _704 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `index.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.07329462989840348 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.05775803144224197 - nodes in this community are weakly interconnected._
- **Should `AppException` be split into smaller, more focused modules?**
  _Cohesion score 0.09243697478991597 - nodes in this community are weakly interconnected._