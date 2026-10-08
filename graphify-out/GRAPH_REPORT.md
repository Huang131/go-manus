# Graph Report - go-manus  (2026-10-08)

## Corpus Check
- 354 files · ~906,529 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 4054 nodes · 9869 edges · 235 communities (210 shown, 25 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 884 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9457a54a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- tool-preview-panel.tsx
- cn
- context.Context
- BrowserService
- index.ts
- client.go
- queryer
- MCPSetting.tsx
- app_test.go
- testing.T
- 重构.md
- Plan
- PostgresSessionRepository
- SupervisorService
- session-header.tsx
- Docker 构建优化实践笔记
- StdioMCPClient
- openai_llm.go
- Run/Session 重构目标架构
- File
- NewMockLLMModelRepository
- newSandboxClient
- 阶段 4：后端生产执行语义切换
- session-detail-view.tsx
- logger 包 os.Stdout 误关闭 Bug 深度解析
- tools_test.go
- app/page.tsx
- AgentTaskRunner
- Response
- compilerOptions
- AnthropicClient
- CODE_REVIEW_2026-09-11.md
- Config
- BadRequestException
- Tool
- 阶段 5：Run API、UI 与 SSE 契约切换
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- Event
- .initRoutes
- github.com/gin-gonic/gin.Context
- NewSessionRuntime
- NewFileCleanupScheduler
- LLMModel
- AppException
- NewSessionHandler
- BadRequest
- 阶段 3：Run 领域模型与 PostgreSQL 存储
- devDependencies
- 代码智能数据使用说明
- BaseAgent
- 阶段 1：Settings 与 Prompt 单一来源
- NewSessionService
- fallback_test.go
- dependencies
- components.json
- createSessionForTest
- Sandbox
- 阶段 2：Engine 结果契约与 ContextBuilder
- browser_cli.js
- time.Duration
- 部署指南
- 阶段 6：遗留执行基础设施与 Session 执行字段清理
- routed_llm_test.go
- testLLMModelRepo
- NewRoutedLLM
- ShellService
- types.go
- DefaultAgentConfig
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- run
- 4.3 11 个 Bug 拆解
- event.go
- logger.go
- LLMRuntimeConfig
- PlannerReActFlow
- Refactoring with GitNexus
- RedisStreamMessageQueue
- go-manus - 通用 AI Agent 系统
- RedisStreamTask
- OSS
- MergeDeltas
- SessionHandler
- chat_lifecycle_test.go
- MCPTool
- agent/config.go
- API 测试方案
- anthropic_llm_test.go
- 集成测试
- tool_event_flow_test.go
- NewRedisStreamTask
- ServiceRegressionTests
- ApiEndpointTests
- Manus 沙箱服务
- Manus 前端 UI
- Run/Session 重构方案集
- Commands
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- ToolResult
- MCPConfig
- protocol.go
- NewToolResult
- go-manus 面试指南
- Run/Session 重构实施状态
- docs/README.md
- Python vs Go 版本文件对比报告
- llm_model.go
- NewBrowserTool
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- vnc-overlay.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- logger_test.go
- Err
- AppConfig
- sync.Once
- 3.3 Repository 层架构设计
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- json_parser_repair.go
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 3. 核心知识点详解
- 七、详细模块对比
- layout.tsx
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- NewToolResultWithMessage
- 技术方案文档编写计划
- 3.4 集成测试
- LoadIntegrationEnv
- AgentService
- Message
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- ParseWithContext
- FileHandler
- sandbox_external_test.go
- TaskStream
- DefaultLLMModelService
- NewToolError
- mockFailingTool
- stubLLM
- Postgres
- next-themes
- Redis
- @radix-ui/react-avatar
- @radix-ui/react-scroll-area
- testFileRepo
- postcss.config.mjs
- NewProviderError
- sandbox
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- Impact Analysis with GitNexus
- 7.4 存储层详细对比
- remark-gfm
- @radix-ui/react-slot
- test-env-down.sh
- test-env-up.sh
- API 当前实现审查（2026-09）
- BuildAttachmentContextSection
- @radix-ui/react-switch
- github.com/Huang131/go-manus/api
- ._iter_file_lines
- react-dom
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- react-markdown
- next.config.ts
- BaseModel
- NewToolProvider
- 7.5 外部依赖详细对比
- NewLoader
- API 测试规则
- AppConfigHandler
- validConfig
- App
- 三、服务层
- sandbox/package.json
- 六、差异汇总
- 八、改进建议
- 5. 注意事项和踩坑点
- archive/README.md
- mockLLMForTest
- LLMResponse
- BrowserCLIContractTests
- response_test.go
- UnixStreamHTTPConnection
- NewMockFileRepository
- .loadOne
- 3.2 PostgreSQL INTERVAL 与参数化查询
- 3.5 Session 与 Agent 架构
- 4. 可能的面试追问
- a2a_config.go
- .uploadFile
- NewAppConfigService
- 主要问题与重构建议
- 6. 分阶段实施
- NewMCPTool
- 六、持久化模型
- NewSimpleMemory
- 当前架构总览
- blockingStreamingAgentLLM
- openai_llm_test.go
- task_redis_test.go
- LLMModelService
- 维护状态
- 核心概念
- sonner
- 学习路径
- 10. 故障排查
- package.json
- eslint.config.mjs
- @novnc/novnc
- @radix-ui/react-label
- tailwind-merge

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 87 edges
3. `Err()` - 60 edges
4. `ServiceRegressionTests` - 56 edges
5. `ShellService` - 53 edges
6. `File` - 49 edges
7. `RedisStreamTask` - 46 edges
8. `FileService` - 45 edges
9. `Response` - 41 edges
10. `NewRedisStreamTask()` - 39 edges

## Surprising Connections (you probably didn't know these)
- `TestParseA2AResult()` --calls--> `parseA2AResult()`  [INFERRED]
  api/internal/a2a/a2a_test.go → api/internal/a2a/client.go
- `TestShouldInlineAndTruncate()` --calls--> `ShouldRAG()`  [INFERRED]
  api/internal/agent/attachment/loader_test.go → api/internal/agent/attachment/policy.go
- `BaseAgent` --references--> `Memory`  [EXTRACTED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewContextBuilder()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/context_builder.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go

## Import Cycles
- None detected.

## Communities (235 total, 25 thin omitted)

### Community 0 - "tool-preview-panel.tsx"
Cohesion: 0.07
Nodes (42): A2APreview(), ArtifactRef, BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent(), getToolDescription(), MCPPreview() (+34 more)

### Community 1 - "cn"
Cohesion: 0.05
Nodes (58): GlobalHeader(), DialogOverlay(), DropdownMenu(), DropdownMenuCheckboxItem(), DropdownMenuContent(), DropdownMenuItem(), DropdownMenuLabel(), DropdownMenuRadioItem() (+50 more)

### Community 2 - "context.Context"
Cohesion: 0.06
Nodes (8): mockSessionRepo, Session, mockMessageQueue, context.Context, sessionServiceStub, stubVNCService, DefaultSessionService, MockSessionRepository

### Community 3 - "BrowserService"
Cohesion: 0.07
Nodes (55): _budget_ms(), click(), console_exec(), console_view(), input_text(), navigate(), press_key(), post (+47 more)

### Community 4 - "index.ts"
Cohesion: 0.07
Nodes (61): useSessionDetail(), UseSessionDetailResult, configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout() (+53 more)

### Community 5 - "client.go"
Cohesion: 0.07
Nodes (33): A2AAgentCapabilities, A2AAgentCard, A2AAgentSkill, A2AArtifact, A2AClient, A2AClientManagerConfig, A2AFilePart, A2AInterface (+25 more)

### Community 6 - "queryer"
Cohesion: 0.12
Nodes (11): pgx.Tx, scanFile(), collectRows(), pgx.Tx, T, newQueryer(), scanSessionSummary(), pgx.Rows (+3 more)

### Community 7 - "MCPSetting.tsx"
Cohesion: 0.10
Nodes (45): DeleteSessionDialogProps, ManusSettings(), SETTING_MENUS, SettingTab, emptyDraft, Mode, ModelConfigManager(), RenameSessionDialogProps (+37 more)

### Community 8 - "app_test.go"
Cohesion: 0.08
Nodes (30): Build(), newLifecycleManager(), normalizeFactories(), resolveAgentConfig(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce() (+22 more)

### Community 9 - "testing.T"
Cohesion: 0.06
Nodes (56): TestA2AAgentCapabilities_CanStream(), TestA2AAgentCard_ResolveEndpoint(), TestA2AJSONRPCError_CodeIsInt(), TestParseA2AResult(), TestFlowStatus_ToSessionStatus(), TestFlowStatus_Values(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_PlanCreation() (+48 more)

### Community 10 - "重构.md"
Cohesion: 0.11
Nodes (17): Plan 和 Step, Prompt 配置, Run, Session, 一、总体架构, 七、事件契约, 三、Agent Engine, 九、API 契约 (+9 more)

### Community 11 - "Plan"
Cohesion: 0.08
Nodes (20): PlannerAgent, ReActAgent, TaskInput, AgentConfig, NewReActAgent(), clonePlanSteps(), cloneStrings(), Plan (+12 more)

### Community 12 - "PostgresSessionRepository"
Cohesion: 0.10
Nodes (14): stopSessionRepository, successfulStopSessionRepository, TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits(), TestAgentService_StopSessionReturnsStatusUpdateError(), SessionStatus, extractSessionMessage(), pgx.Tx, SessionRepository (+6 more)

### Community 13 - "SupervisorService"
Cohesion: 0.05
Nodes (53): APIRouter, BaseSettings, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, create_api_routes() (+45 more)

### Community 14 - "session-header.tsx"
Cohesion: 0.14
Nodes (25): SessionHeader(), SessionItem(), SessionItemProps, Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup(), AvatarGroupCount() (+17 more)

### Community 15 - "Docker 构建优化实践笔记"
Cohesion: 0.05
Nodes (38): 1.1 问题分析, 1.2 解决方案, 1.3 验证结果, 2.1 问题分析, 2.2 解决方案, 2.3 验证结果, 3.1 问题分析, 3.2.1 使用 uv 替代 pip (+30 more)

### Community 16 - "StdioMCPClient"
Cohesion: 0.11
Nodes (16): NewStdioMCPClient(), parseToolResult(), TestNewStdioMCPClient(), TestParseToolResult(), bufio.Reader, io.WriteCloser, os/exec.Cmd, sync/atomic.Bool (+8 more)

### Community 17 - "openai_llm.go"
Cohesion: 0.10
Nodes (26): streamContext(), classifyHTTPError(), estimateCostUSD(), LLMRequest, NewOpenAIClient(), normalizeOpenAIFinishReason(), reasoningTokens(), sendStreamDelta() (+18 more)

### Community 18 - "Run/Session 重构目标架构"
Cohesion: 0.08
Nodes (26): 10. API 目标, 11. 包依赖约束, 12. 最终删除项, 13. 架构验收, 1. 背景, 2. 范围, 3.1 Session, 3.2 Run (+18 more)

### Community 19 - "File"
Cohesion: 0.07
Nodes (12): attachmentFileRepository, attachmentStorage, generatedFileRepository, File, FileRepository, NewFileRepository(), FileStorage, io.ReadCloser (+4 more)

### Community 20 - "NewMockLLMModelRepository"
Cohesion: 0.15
Nodes (30): NewMockLLMModelRepository(), NewLLMModelService(), NewLLMModelServiceWithLLMFactory(), newHealthService(), TestLLMModelService_Create(), TestLLMModelService_Create_MapsUniqueViolationToConflict(), TestLLMModelService_Create_PreservesPartialCapabilities(), TestLLMModelService_Create_Validation() (+22 more)

### Community 21 - "newSandboxClient"
Cohesion: 0.23
Nodes (14): NewSandboxClient(), SandboxClient, newSandboxClient(), TestExecCommandDecodesEnvelope(), TestFindFilesSendsGlobPattern(), TestHealthCheckReportsFailure(), TestPostJSONErrorMapping(), TestRequireAddress() (+6 more)

### Community 22 - "阶段 4：后端生产执行语义切换"
Cohesion: 0.10
Nodes (19): A. 新执行组件（尚未接生产）, B. 原子切换生产装配, C. 并发、重启与回归加固, Redis Run Event Stream, 创建、继续和状态提交, 前置条件, 取消优先与并发, 回滚 (+11 more)

### Community 23 - "session-detail-view.tsx"
Cohesion: 0.08
Nodes (35): PageProps, AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock(), ToolRow() (+27 more)

### Community 24 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 25 - "tools_test.go"
Cohesion: 0.14
Nodes (19): NewA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description(), TestA2ATool_Initialize(), TestA2ATool_InitializeSkipsDuplicateAgentNames(), TestA2ATool_Invoke_CallAgentRejectsInvalidRequiredTypes(), TestA2ATool_Invoke_CallAgentWithoutAgentID() (+11 more)

### Community 26 - "app/page.tsx"
Cohesion: 0.24
Nodes (7): ChatInput, ChatInputProps, ChatInputRef, SuggestedQuestions(), SuggestedQuestionsProps, suggestedQuestions, FileInfo

### Community 27 - "AgentTaskRunner"
Cohesion: 0.31
Nodes (3): AgentTaskRunner, AgentTaskRunnerConfig, AgentConfig

### Community 28 - "Response"
Cohesion: 0.11
Nodes (41): api_route, FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get (+33 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "AnthropicClient"
Cohesion: 0.13
Nodes (17): LLMRequest, mustMarshalString(), normalizeAnthropicStopReason(), parseJSONMap(), TestAnthropicClient_UserContentTextNotDuplicated(), TestAnthropicClient_UserContentTextOnlyUsesContentText(), TestNormalizeAnthropicStopReason(), textOf() (+9 more)

### Community 31 - "CODE_REVIEW_2026-09-11.md"
Cohesion: 0.09
Nodes (22): A. 流式链路复查（commit 4892225）, api, B. service 层专项（新发现）, C. 昨日 backlog 复核（HEAD b1f3827 仍开放）, D. 修复批次, E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）, sandbox, ui (+14 more)

### Community 32 - "Config"
Cohesion: 0.12
Nodes (18): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, ObjectStorageConfig, RedisConfig (+10 more)

### Community 33 - "BadRequestException"
Cohesion: 0.26
Nodes (16): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+8 more)

### Community 34 - "Tool"
Cohesion: 0.39
Nodes (4): isReadOnlyToolSchema(), MultiFunctionTool, Tool, ToolRegistry

### Community 35 - "阶段 5：Run API、UI 与 SSE 契约切换"
Cohesion: 0.12
Nodes (16): A. 新 API 和 SSE（旧 UI 仍可运行）, B. UI 切换并删除旧路由, SSE 契约, UI 状态规则, 前置条件与唯一语义, 回滚, 失败或中断恢复, 完成定义 (+8 more)

### Community 36 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 37 - "Event"
Cohesion: 0.17
Nodes (15): TaskRegistryInterface, TaskRunner, nonNilEvents(), parseTaskOutputMessages(), ReadTaskOutput(), readTaskOutput(), streamDataString(), taskOutputStreamName() (+7 more)

### Community 38 - ".initRoutes"
Cohesion: 0.16
Nodes (20): defaultTrustedProxies(), SetupRoutes(), CORS(), generateRequestID(), Logger(), Recovery(), RequestID(), setupTestEngine() (+12 more)

### Community 39 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.20
Nodes (8): LLMModelHandler, NewLLMModelResponse(), FromError(), TotalResponse, Success(), SuccessWithMsg(), SuccessWithTotal(), github.com/gin-gonic/gin.Context

### Community 40 - "NewSessionRuntime"
Cohesion: 0.15
Nodes (11): chatContextSessionRepository, NewSessionRuntime(), TestAgentService_GetActiveTaskIDClearsCompletedTask(), TestAgentService_ResolveMessageAttachments(), TestAgentService_ResolveMessageAttachmentsRejectsOtherSession(), TestAgentServiceChatUsesDetachedContextForMessagePersistence(), TestPopRetryConfig(), TestSessionRuntime_StoreToolArtifactsDropsWithoutStorage() (+3 more)

### Community 41 - "NewFileCleanupScheduler"
Cohesion: 0.27
Nodes (9): FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(), TestFileCleanupSchedulerStartIsIdempotent(), TestFileCleanupSchedulerStopCancelsCleanupContext(), TestFileCleanupStopsWhenWholeBatchStorageDeleteFails() (+1 more)

### Community 42 - "LLMModel"
Cohesion: 0.11
Nodes (10): LLMModel, encodeModelJSONColumns(), pgx.Tx, LLMModelRepository, modelJSON(), NewLLMModelRepository(), scanLLMModel(), modelJSONColumns (+2 more)

### Community 43 - "AppException"
Cohesion: 0.13
Nodes (12): Exception, FastAPI, register_exception_handlers(), AppException, BusyException, Any, 沙箱繁忙：浏览器动作全局串行，锁等待耗尽预算时触发，可稍后重试。 与 504「动作超时」的语义区分： - 503…, Any (+4 more)

### Community 44 - "NewSessionHandler"
Cohesion: 0.49
Nodes (9): NewSessionHandler(), decodeHandlerResponse(), setupRouter(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create(), TestSessionHandler_Delete(), TestSessionHandler_Get(), TestSessionHandler_GetFiles() (+1 more)

### Community 45 - "BadRequest"
Cohesion: 0.15
Nodes (20): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Kind, Internal(), New(), NotFound() (+12 more)

### Community 46 - "阶段 3：Run 领域模型与 PostgreSQL 存储"
Cohesion: 0.14
Nodes (14): Repository 契约, 前置条件与唯一语义, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库设计, 文件清单 (+6 more)

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.12
Nodes (17): Claude Code 配置, Clone 后的首次使用, GitNexus, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify (+9 more)

### Community 49 - "BaseAgent"
Cohesion: 0.15
Nodes (8): BaseAgent, InvokeResult, ToolCallResult, formatRepairTypes(), AgentConfig, NewBaseAgentWithParser(), shouldPublishDeltas(), JSONParser

### Community 50 - "阶段 1：Settings 与 Prompt 单一来源"
Cohesion: 0.15
Nodes (13): 关键契约, 前置条件与恢复基线, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库与 API 变更, 文件清单 (+5 more)

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

### Community 55 - "createSessionForTest"
Cohesion: 0.09
Nodes (75): Response, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle(), TestFileAPI_Delete_NotFound(), TestFileAPI_Download_Lifecycle(), TestFileAPI_FileTableConsistency() (+67 more)

### Community 56 - "Sandbox"
Cohesion: 0.11
Nodes (14): TestMessageTool_NotifyUserSchema(), NewFileTool(), NewMessageTool(), NewShellTool(), TestMCPTool_InitializeWithoutConfig(), TestMessageTool_Invoke_PassesTextThrough(), TestShellTool_Invoke_PassesParamsToSandbox(), TestShellTool_Invoke_PropagatesSandboxError() (+6 more)

### Community 57 - "阶段 2：Engine 结果契约与 ContextBuilder"
Cohesion: 0.15
Nodes (13): 关键契约, 前置条件与唯一语义, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库与 API 变更, 文件清单 (+5 more)

### Community 58 - "browser_cli.js"
Cohesion: 0.20
Nodes (27): ActionError, attachLogCollector(), domHelpers(), elementExpr(), fail(), fs, INTERACTIVE_SELECTOR, loadPlaywright() (+19 more)

### Community 59 - "time.Duration"
Cohesion: 0.13
Nodes (27): bochaFreshness(), NewBochaSearchClientWithTimeout(), NewFallbackSearchClient(), NewSearchEngine(), NewTavilySearchClientWithTimeout(), normalizeSearchLimit(), resolveSearchTimeout(), tavilyTimeRange() (+19 more)

### Community 60 - "部署指南"
Cohesion: 0.12
Nodes (17): 1. 部署方式, 2. 架构总览, 3. 前置要求, 4.1 准备环境变量, 4.2 一键启动, 4.3 访问入口（统一走 nginx）, 4. 快速启动（nginx 网关模式，推荐）, 5.1 `.env` 模板（`api/.env.example`） (+9 more)

### Community 61 - "阶段 6：遗留执行基础设施与 Session 执行字段清理"
Cohesion: 0.14
Nodes (13): 删除清单, 前置条件与恢复基线, 包边界收紧, 回滚, 失败或中断恢复, 完成定义, 实施顺序与提交检查点, 数据库破坏式收敛 (+5 more)

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
Cohesion: 0.06
Nodes (35): ConsoleRecord, NotFoundException, ConsoleRecord, BaseModel, Shell, ShellExecuteResult, ShellKillResult, ShellReadResult (+27 more)

### Community 66 - "types.go"
Cohesion: 0.09
Nodes (21): browserConsoleExecRequest, browserConsoleViewRequest, browserInputRequest, browserNavigateRequest, browserPressKeyRequest, browserScreenshotData, browserScreenshotRequest, browserScrollRequest (+13 more)

### Community 67 - "DefaultAgentConfig"
Cohesion: 0.18
Nodes (16): NewBaseAgent(), TestBaseAgentStopShellWatchWaitsForWatcherExit(), TestBaseAgentInvokePublishesTextDeltas(), TestBaseAgentInvokeStreamingStopsWhenProviderLeavesStreamOpen(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(), DefaultAgentConfig(), TestNewAgentTaskRunnerLoadsInitialConversation(), TestBaseAgent_InvokeWithEmptyRetry() (+8 more)

### Community 68 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 69 - "run"
Cohesion: 0.31
Nodes (7): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), net/http.Server, Config

### Community 70 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 71 - "event.go"
Cohesion: 0.05
Nodes (35): conversationMessages(), TestConversationMessagesRestoresPersistedUserAndAssistantMessages(), TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), EventType, MessageRole, NewDoneEvent(), NewMessageDeltaEvent(), NewMessageDoneEvent() (+27 more)

### Community 72 - "logger.go"
Cohesion: 0.12
Nodes (25): Any(), Bool(), Dur(), Error(), Fatal(), Float64(), Get(), Int() (+17 more)

### Community 73 - "LLMRuntimeConfig"
Cohesion: 0.09
Nodes (24): defaultLLMClientFactory(), TestDynamicLLM_UsesFactory(), LLMClientFactory, ModelIDFromContext(), NewDynamicLLMWithFactory(), NewLLMClient(), betterHealth(), configKey() (+16 more)

### Community 74 - "PlannerReActFlow"
Cohesion: 0.17
Nodes (11): FlowStatus, PlannerReActFlow, AgentConfig, NewPlannerReActFlow(), planHasFailedStep(), TestPlannerReActFlow_InvokeContext(), BaseEvent, NewErrorEvent() (+3 more)

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "RedisStreamMessageQueue"
Cohesion: 0.11
Nodes (11): batchTaskOutputMQ, mockMQWrapper, StreamMessage, CompletedStreamRetention(), decodeStreamData(), RedisStreamMessageQueue, redis.Client, NewRedisStreamMessageQueue() (+3 more)

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.13
Nodes (15): API 开发, API 文档, go-manus - 通用 AI Agent 系统, 一键部署（推荐）, 健康检查, 前置要求, 容器列表, 常用命令 (+7 more)

### Community 78 - "RedisStreamTask"
Cohesion: 0.10
Nodes (4): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask

### Community 79 - "OSS"
Cohesion: 0.18
Nodes (12): TestBuildWithFactoriesInjectsFileCleanupScheduler(), ensureBucket(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL() (+4 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (15): EstimateContextTokens(), isToolCallFinishReason(), MergeDeltas(), repeat(), TestEstimateContextTokens_ASCII(), TestEstimateContextTokens_Mixed(), TestEstimateContextTokens_NonASCII(), TestMergeDeltas_CanonicalToolCallsRetainsToolCall() (+7 more)

### Community 81 - "SessionHandler"
Cohesion: 0.18
Nodes (13): SessionHandler, mergeEventMetadata(), newSSEContext(), parseChatRequest(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), setSSEHeaders(), TestMergeEventMetadata(), TestMergeEventMetadata_NullPayloadReturnsOriginalData() (+5 more)

### Community 82 - "chat_lifecycle_test.go"
Cohesion: 0.17
Nodes (21): sameOrigin(), newLocalTestServer(), TestVNCProxy_AllowsSameOriginAndNonBrowserClient(), TestVNCProxy_ProxyEcho(), TestVNCProxy_RejectsCrossOrigin(), TestVNCProxy_ServiceError(), appendTaskEvent(), eventIndex() (+13 more)

### Community 83 - "MCPTool"
Cohesion: 0.16
Nodes (7): MCPClientManager, MCPToolInfo, parseToolInfos(), TestParseToolInfos(), sync.RWMutex, MCPClient, MCPTool

### Community 84 - "agent/config.go"
Cohesion: 0.14
Nodes (11): cloneStringMap(), A2AConfig, AgentConfig, RuntimeA2AConfig(), RuntimeMCPConfig(), TestRuntimeA2AConfigFiltersDisabledServers(), TestRuntimeMCPConfigFiltersDisabledServersAndCopiesValues(), TestToolProviderOmitsEmptyDynamicTools() (+3 more)

### Community 85 - "API 测试方案"
Cohesion: 0.18
Nodes (11): 1. 当前判断, 2. 测试分层, 3. 环境与命令, 4. 高收益测试契约, 5. 已完成的测试精简, 7. 完成标准与阶段归属, API 测试方案, HTTP integration (+3 more)

### Community 86 - "anthropic_llm_test.go"
Cohesion: 0.20
Nodes (17): NewAnthropicClient(), errKind(), newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_ParentDeadlineNotMisattributedToToolCalling(), TestAnthropicClient_ProtocolError_NotFallbackable(), TestAnthropicClient_ResponseBlocks() (+9 more)

### Community 87 - "集成测试"
Cohesion: 0.13
Nodes (14): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+6 more)

### Community 88 - "tool_event_flow_test.go"
Cohesion: 0.12
Nodes (13): artifactTool, inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, TestReadTaskOutputAfterTaskUnregistered(), TestRedisStreamTask_GetOutputReadsBufferedEventsWhenStartIDEmpty(), NewAgentTaskRunner() (+5 more)

### Community 89 - "NewRedisStreamTask"
Cohesion: 0.31
Nodes (15): NewDefaultTaskRegistry(), NewRedisStreamTask(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_CleanupCompletedWaitsForFinished(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister() (+7 more)

### Community 90 - "ServiceRegressionTests"
Cohesion: 0.05
Nodes (16): FileService, UploadFile, 文件操作只接受普通文件，避免目录被当作文件读取或删除。, 目录遍历只接受目录路径，尽早返回明确的 404。, 校验全局读取上限；具体文件按流式读取，超限时返回截断结果。, 停止达到读取上限的子进程，避免 terminate 后永久等待。, 根据传递的文件路径+起始行号+权限+最大长度读取文件内容, 按固定块读取 sudo 输出，避免超长单行触发 readline 缓冲上限。 (+8 more)

### Community 91 - "ApiEndpointTests"
Cohesion: 0.16
Nodes (5): ApiEndpointTests, FakeShellService, FakeSupervisorService, Any, 通过 ASGI 协议直接调用应用，避免测试依赖额外 HTTP 客户端。

### Community 92 - "Manus 沙箱服务"
Cohesion: 0.22
Nodes (8): API 接口, Docker 部署, Manus 沙箱服务, 信任模型与安全边界, 技术栈, 本地开发, 架构, 端口说明

### Community 93 - "Manus 前端 UI"
Cohesion: 0.18
Nodes (10): API 调用, Docker 部署, Manus 前端 UI, 安装与启动, 技术栈, 本地开发, 构建, 模型选择（Auto） (+2 more)

### Community 94 - "Run/Session 重构方案集"
Cohesion: 0.29
Nodes (7): Run/Session 重构方案集, 不保留两套长期业务语义, 后续模型的恢复步骤, 工作包, 文档与代码的更新规则, 每个工作包的硬门禁, 目标

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 97 - "ToolResult"
Cohesion: 0.08
Nodes (4): ToolArtifact, ToolResult, SandboxClient, mockSandbox

### Community 98 - "MCPConfig"
Cohesion: 0.11
Nodes (15): A2AConfig, A2AServer, AgentConfig, MCPConfig, MCPServer, getConfig(), AppConfigService, T (+7 more)

### Community 99 - "protocol.go"
Cohesion: 0.26
Nodes (10): ResponseFormat, ToolCall, ToolSpec, AudioURL, ContentPart, ImageURL, LLMRequest, ToolCallDelta (+2 more)

### Community 100 - "NewToolResult"
Cohesion: 0.10
Nodes (9): attachmentSandbox, NewToolResult(), BrowserInput, BrowserTarget, SandboxClient, NewBrowserClient(), BrowserClient, browserTargetRequest (+1 more)

### Community 101 - "go-manus 面试指南"
Cohesion: 0.25
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "Run/Session 重构实施状态"
Cohesion: 0.25
Nodes (8): Run/Session 重构实施状态, 决策偏差, 当前入口, 当前状态, 更新模板, 最近一次验证, 阶段内检查点, 阻塞记录

### Community 103 - "docs/README.md"
Cohesion: 0.20
Nodes (3): Agent 架构（兼容入口）, API 测试规则入口, 文档导航

### Community 104 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 105 - "llm_model.go"
Cohesion: 0.12
Nodes (21): BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), TestBuildRuntimeConfigFromModel(), TestProtocolFromProvider(), ExtraParam, ProviderProtocol, RequestPolicy (+13 more)

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

### Community 111 - "logger_test.go"
Cohesion: 0.16
Nodes (22): GetLevel(), Init(), InitWithConfig(), SetLevel(), stdoutWriteOK(), TestCallerInBusinessCode(), testCallerLocation(), TestGetLevel_DefaultWhenNotInitialized() (+14 more)

### Community 112 - "Err"
Cohesion: 0.10
Nodes (13): SessionRuntime, dialSandboxVNC(), VNCProxy(), Debug(), DebugContext(), Err(), Info(), Warn() (+5 more)

### Community 113 - "AppConfig"
Cohesion: 0.13
Nodes (11): AppConfig, AppConfigType, unixPointer(), pgx.Tx, AppConfigRepository, NewAppConfigRepository(), scanAppConfig(), time.Time (+3 more)

### Community 114 - "sync.Once"
Cohesion: 0.22
Nodes (3): blockingWatchSandbox, sync.Once, blockingLLM

### Community 115 - "3.3 Repository 层架构设计"
Cohesion: 0.29
Nodes (7): 3.3 Repository 层架构设计, WithTx 事务封装, 事务支持：QueryContext 接口, 接口设计, 构造函数设计：普通版 vs 事务版, 软删除模式, 面试追问

### Community 116 - "5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异"
Cohesion: 0.29
Nodes (7): 5.1 现象, 5.2 What, 5.3 sonic vs encoding/json 差异, 5.4 sonic 正确使用姿势, 5.5 替换 JSON 库的检查清单, 5.6 面试要点, 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异

### Community 117 - "json_parser_repair.go"
Cohesion: 0.15
Nodes (15): NewDefaultJSONParser(), completeTruncatedJSON(), Repair, isTruncatedJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), TestCompleteTruncatedJSON(), TestDefaultJSONParser() (+7 more)

### Community 118 - "一、核心 Agent 模块"
Cohesion: 0.33
Nodes (6): 1.1 Base Agent, 1.2 ReAct Agent, 1.3 Planner Agent, 1.4 Flow 执行流, 1.5 Memory 记忆, 一、核心 Agent 模块

### Community 119 - "7.2 工具系统详细对比"
Cohesion: 0.33
Nodes (6): 7.2.1 工具接口设计, 7.2.2 工具注册机制, 7.2.3 FileTool 详细对比, 7.2.4 MCP 工具集成, 7.2.5 面试分析点, 7.2 工具系统详细对比

### Community 120 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 121 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 122 - "layout.tsx"
Cohesion: 0.11
Nodes (16): metadata, DeleteSessionDialog(), LeftPanel(), RenameSessionDialog(), SessionList(), Toaster(), AUTO_MODEL_ID, ModelsContext (+8 more)

### Community 123 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 124 - "NewToolResultWithMessage"
Cohesion: 0.12
Nodes (10): NewToolResultWithMessage(), TestNewToolError(), TestNewToolResult(), TestNewToolResultWithMessage(), TestToolResult_ArtifactsStayOutOfJSON(), TestToolResult_FromSandbox(), TestToolResult_LLMJSONOnNil(), TestToolResult_LLMJSONStripsDisplay() (+2 more)

### Community 125 - "技术方案文档编写计划"
Cohesion: 0.33
Nodes (5): 已知环境, 技术方案文档编写计划, 目标, 硬约束, 阶段

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "LoadIntegrationEnv"
Cohesion: 0.24
Nodes (13): applyPostgresEnv(), applyRedisEnv(), applyStorageEnv(), isTestResource(), LoadIntegrationEnv(), clearIntegrationOverrides(), TestLoadIntegrationEnvAppliesOverrides(), TestLoadIntegrationEnvRejectsInvalidOverrides() (+5 more)

### Community 128 - "AgentService"
Cohesion: 0.26
Nodes (7): Capabilities, Repositories, NormalizeAgentConfig(), A2AConfig, AgentService, AgentConfig, NewAgentService()

### Community 129 - "Message"
Cohesion: 0.19
Nodes (11): ContextBuilder, ContextPolicy, SimpleMemory, completeConversationGroups(), estimateMessage(), estimateMessages(), NewContextBuilder(), TestContextBuilderDropsIncompleteToolGroup() (+3 more)

### Community 130 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 131 - "ParseWithContext"
Cohesion: 0.20
Nodes (15): Parse(), ParseWithContext(), TestParse_BasicFunctionality(), TestParse_CommentLines(), TestParse_DataFieldWithEmptyContent(), TestParse_EventTypePreserved(), TestParse_RetryAndIdFields(), TestParseWithContext_ConcurrentDifferentData() (+7 more)

### Community 132 - "FileHandler"
Cohesion: 0.27
Nodes (4): FileHandler, NewFileHandler(), FileService, SessionService

### Community 133 - "sandbox_external_test.go"
Cohesion: 0.39
Nodes (14): assertContentLimitedTo(), containsPath(), requireSandbox(), sandboxContext(), sandboxData(), TestSandbox_BrowserScreenshotReturnsPNG(), TestSandbox_BusinessErrorMapsToAPIError(), TestSandbox_FileLifecycle() (+6 more)

### Community 134 - "TaskStream"
Cohesion: 0.16
Nodes (5): Stream, Task, TaskStream, NewTaskStream(), taskInputStreamName()

### Community 135 - "DefaultLLMModelService"
Cohesion: 0.21
Nodes (8): LLMModelTestResponse, isUniqueViolation(), normalizeDefaults(), truncateModelTestContent(), validateRequired(), DefaultLLMModelService, HealthInvalidator, RuntimeHealthReader

### Community 136 - "NewToolError"
Cohesion: 0.17
Nodes (9): A2AConfig, browserTargetFromParams(), optionalToolBool(), optionalToolFloat(), optionalToolInt(), optionalToolString(), requiredToolString(), NewToolError() (+1 more)

### Community 138 - "stubLLM"
Cohesion: 0.31
Nodes (3): LLMRequest, streamingStubLLM, stubLLM

### Community 139 - "Postgres"
Cohesion: 0.23
Nodes (9): Postgres, NewPostgres(), pgx.Tx, T, joinRollbackError(), logRollbackFailure(), rollbackTx(), runInTx() (+1 more)

### Community 141 - "Redis"
Cohesion: 0.15
Nodes (12): StatusHandler, NewStatusHandler(), Redis, redis.Client, NewRedis(), HealthStatus, HealthState, ServiceName (+4 more)

### Community 144 - "testFileRepo"
Cohesion: 0.29
Nodes (12): deleteSessionDirect(), testFileRepo(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound(), TestFileRepo_GetBySessionAndID_EnforcesOwnership() (+4 more)

### Community 148 - "NewProviderError"
Cohesion: 0.35
Nodes (8): ErrorKind, IsDeterministicKind(), isFallbackable(), isRetryable(), NewProviderError(), TestIsDeterministicKind(), TestProviderErrorFallbackableConsistency(), ProviderError

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

### Community 166 - "API 当前实现审查（2026-09）"
Cohesion: 0.17
Nodes (12): Agent 配置与搜索 limit 已接通，但还没有统一 Settings 模型, API 当前实现审查（2026-09）, ContextBuilder 已接入，但仍是保守估算, LLM 配置仍在使用, PlannerReActFlow.Invoke 已经拆分, 不应提前删除的候选, 后续审查入口, 审查范围与验证 (+4 more)

### Community 168 - "BuildAttachmentContextSection"
Cohesion: 0.32
Nodes (5): BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext()

### Community 171 - "._iter_file_lines"
Cohesion: 0.25
Nodes (6): _LineChunk, 逐行收集范围内内容，任何上限命中后立即停止读取。, 按固定块解码文件，超长无换行内容达到上限即截断。, 文件流的一段逻辑行；truncated 表示单行超过内存上限。, 按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。, _truncate_utf8()

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

### Community 181 - "NewToolProvider"
Cohesion: 0.14
Nodes (11): A2AConfig, NewToolProvider(), toolNames(), NewSearchTool(), TestSearchToolForwardsConfiguredLimit(), TestToolProviderReloadSearchLimitAffectsNewTools(), Browser, SearchEngine (+3 more)

### Community 182 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 183 - "NewLoader"
Cohesion: 0.11
Nodes (22): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+14 more)

### Community 184 - "API 测试规则"
Cohesion: 0.29
Nodes (7): API 测试规则, 分层边界, 变更要求, 外部环境, 测试先绑定契约, 测试替身, 衰减风险评审

### Community 185 - "AppConfigHandler"
Cohesion: 0.21
Nodes (7): getConfig(), AppConfigHandler, T, NewAppConfigHandler(), reloadOrFail(), updateConfig(), configRuntimeReloader

### Community 186 - "validConfig"
Cohesion: 0.53
Nodes (5): TestConfigApplyDefaultsUsesDomainDefaults(), TestValidate_OK(), TestValidate_RequiredFields(), validConfig(), Config

### Community 187 - "App"
Cohesion: 0.16
Nodes (15): BuildWithFactories(), defaultFactories(), DefaultOptions(), App, loadRuntimeToolConfigs(), newA2AConfig(), newMCPConfig(), newRepositories() (+7 more)

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

### Community 192 - "5. 注意事项和踩坑点"
Cohesion: 0.33
Nodes (6): 5.1 Go 错误处理, 5.2 PostgreSQL 参数化, 5.3 Repository 设计, 5.4 软删除查询, 5.5 文件清理, 5. 注意事项和踩坑点

### Community 195 - "LLMResponse"
Cohesion: 0.12
Nodes (7): mockLLM, streamingAgentLLM, LLMRequest, LLMResponse, sync.Mutex, connectionTestLLM, scriptedLLM

### Community 197 - "response_test.go"
Cohesion: 0.29
Nodes (5): TestFromError_GenericError(), TestFromError_MappedBusinessError(), TestResponse_Structure(), TestSuccess_HttpResponse(), TestTotalResponse_Structure()

### Community 198 - "UnixStreamHTTPConnection"
Cohesion: 0.33
Nodes (3): HTTPConnection, 重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接, UnixStreamHTTPConnection

### Community 205 - "NewMockFileRepository"
Cohesion: 0.36
Nodes (13): NewMockFileRepository(), NewFileService(), TestFileServiceDeleteFileDeletesStorageAndRepo(), TestFileServiceDeleteFileStorageError(), TestFileServiceDownloadFileNilStorageReturnsStorageUnavailable(), TestFileServiceDownloadFileWithStorage(), TestFileServiceGetFileInfoFound(), TestFileServiceMissingFileReturnsNotFound() (+5 more)

### Community 206 - ".loadOne"
Cohesion: 0.31
Nodes (7): TestShouldInlineAndTruncate(), FileContext, IsLikelyBinary(), ShouldInline(), ShouldRAG(), ShouldTruncate(), LoadMode

### Community 207 - "3.2 PostgreSQL INTERVAL 与参数化查询"
Cohesion: 0.40
Nodes (5): 3.2 PostgreSQL INTERVAL 与参数化查询, Why：为什么 INTERVAL 不能参数化？, 问题背景, 面试追问, 项目中的正确实现

### Community 208 - "3.5 Session 与 Agent 架构"
Cohesion: 0.40
Nodes (5): 3.5 Session 与 Agent 架构, Agent Memory 机制, AppendEvent 业务逻辑, MessageEvent 结构, 面试追问

### Community 209 - "4. 可能的面试追问"
Cohesion: 0.40
Nodes (5): 4.1 Go 语言层面, 4.2 数据库层面, 4.3 架构设计层面, 4.4 测试层面, 4. 可能的面试追问

### Community 210 - "a2a_config.go"
Cohesion: 0.50
Nodes (3): A2AAgent, A2AAgent, A2AConfig

### Community 211 - ".uploadFile"
Cohesion: 0.36
Nodes (4): SandboxClient, newAPIError(), toolResultErr(), net/http.Header

### Community 213 - "NewAppConfigService"
Cohesion: 0.45
Nodes (10): NewAppConfigService(), NewMockAppConfigRepository(), TestAppConfigService_DeleteAndUpdateA2AServer(), TestAppConfigService_DeleteMCPServer(), TestAppConfigService_DeleteMCPServer_NotFoundWithoutConfig(), TestAppConfigService_GetConfig_RepositoryError(), TestAppConfigService_UpdateA2AConfigRejectsInvalidURL(), TestAppConfigService_UpdateMCPConfig() (+2 more)

### Community 214 - "主要问题与重构建议"
Cohesion: 0.22
Nodes (9): P0：Session 混合会话事实和执行事实, P0：任务生命周期依赖进程内两个可变映射, P1：AgentService 与 SessionRuntime 仍职责过宽, P1：三套状态模型并存, P1：消息/事件的持久化边界不清, P2：ContextBuilder 的策略输入不足, P2：包边界仍有泄漏, P2：配置与 Prompt 仍缺少可复现性 (+1 more)

### Community 215 - "6. 分阶段实施"
Cohesion: 0.25
Nodes (8): 6. 分阶段实施, Future R：Run 重构后的测试, Phase 0：基线, Phase 1：环境入口, Phase 2：边界归位, Phase 3：并发稳定性, Phase 4：当前 API HTTP 契约, Phase 5：当前低收益测试清理

### Community 216 - "NewMCPTool"
Cohesion: 0.33
Nodes (4): NewMCPTool(), parseMCPInvokeParams(), TestMCPToolInvokeAllowsMissingParams(), TestMCPToolInvokeValidatesParameters()

### Community 218 - "六、持久化模型"
Cohesion: 0.29
Nodes (7): messages, outbox, run_events, run_plans, runs, sessions, 六、持久化模型

### Community 219 - "NewSimpleMemory"
Cohesion: 0.40
Nodes (4): Memory, NewSimpleMemory(), TestSimpleMemory_AppendAndMergePreserveOrder(), TestSimpleMemory_Clear()

### Community 220 - "当前架构总览"
Cohesion: 0.33
Nodes (6): Planner 与 ReAct 边界, 当前架构总览, 当前限制, 流式事件, 请求链路, 重要代码入口

### Community 222 - "openai_llm_test.go"
Cohesion: 0.15
Nodes (27): anthropicErrorResponse(), blockingTransport(), newTransportClient(), newWireCaptureClient(), okChatResponse(), TestOpenAIClient_200WithErrorBodySurfacesUpstreamMessage(), TestOpenAIClient_401_Auth(), TestOpenAIClient_429_RateLimit() (+19 more)

### Community 223 - "task_redis_test.go"
Cohesion: 0.16
Nodes (13): mockTaskRunner, retentionCall, TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_FinishDestroysRunner(), TestRedisStreamTask_FinishedChangesAfterRunnerExit(), TestRedisStreamTask_FinishSetsStreamRetention() (+5 more)

### Community 224 - "LLMModelService"
Cohesion: 0.40
Nodes (3): NewLLMModelHandler(), TestLLMModelHandler_CreateRejectsMalformedJSON(), LLMModelService

### Community 225 - "维护状态"
Cohesion: 0.40
Nodes (5): 已完成, 已知限制, 维护状态, 维护规则, 验证基线

### Community 226 - "核心概念"
Cohesion: 0.40
Nodes (5): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, 核心概念

### Community 228 - "学习路径"
Cohesion: 0.40
Nodes (5): 学习路径, 第一阶段：理解 Agent 基础, 第三阶段：理解外部集成, 第二阶段：掌握工具系统, 第四阶段：部署和扩展

### Community 229 - "10. 故障排查"
Cohesion: 0.50
Nodes (4): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查

### Community 298 - "package.json"
Cohesion: 0.22
Nodes (8): name, private, scripts, build, dev, lint, start, version

## Knowledge Gaps
- **736 isolated node(s):** `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest`, `A2ATaskParams`, `Task` (+731 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `toolResultErr()` connect `.uploadFile` to `NewToolError`, `ToolResult`, `types.go`?**
  _High betweenness centrality (0.020) - this node is a cross-community bridge._
- **Why does `sandboxContext()` connect `sandbox_external_test.go` to `testing.T`, `context.Context`?**
  _High betweenness centrality (0.020) - this node is a cross-community bridge._
- **Why does `ToolResult` connect `ToolResult` to `NewToolResult`, `sandbox_external_test.go`, `NewToolError`, `mockFailingTool`, `PlannerReActFlow`, `BadRequest`, `Err`, `BaseAgent`, `sync.Once`, `.uploadFile`, `NewToolProvider`, `tool_event_flow_test.go`, `tools_test.go`, `time.Duration`, `NewToolResultWithMessage`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **What connects `A2AAgent`, `github.com/Huang131/go-manus/api`, `A2AJSONRPCRequest` to the rest of the system?**
  _736 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `tool-preview-panel.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.07012987012987013 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.052982456140350874 - nodes in this community are weakly interconnected._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.06459627329192547 - nodes in this community are weakly interconnected._