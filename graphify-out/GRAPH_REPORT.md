# Graph Report - go-manus  (2026-09-18)

## Corpus Check
- 314 files · ~888,118 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3742 nodes · 9121 edges · 208 communities (182 shown, 26 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 718 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `2343e042`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- index.tsx
- cn
- BuildAttachmentContextSection
- AppException
- index.ts
- ToolResult
- queryer
- MCPSetting.tsx
- app_test.go
- testing.T
- 重构.md
- session-detail-view.tsx
- PostgresSessionRepository
- SupervisorService
- layout.tsx
- Docker 构建优化实践笔记
- StdioMCPClient
- NewProviderError
- Run/Session 重构目标架构
- LLMModel
- anthropic_llm_test.go
- TestMessageQueueMockOperations
- 阶段 4：后端生产执行语义切换
- batchTaskOutputMQ
- logger 包 os.Stdout 误关闭 Bug 深度解析
- ToolProvider
- app/page.tsx
- NewRoutedLLM
- endpoints/file.py
- compilerOptions
- openai_llm.go
- CODE_REVIEW_2026-09-11.md
- Config
- endpoints/shell.py
- Tool
- 阶段 5：Run API、UI 与 SSE 契约切换
- 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复
- DefaultLLMModelService
- .initRoutes
- RedisStreamMessageQueue
- context.Context
- DefaultFileCleanupService
- agent/config.go
- test_api_endpoints.py
- AppConfig
- github.com/gin-gonic/gin.Context
- 阶段 3：Run 领域模型与 PostgreSQL 存储
- devDependencies
- 代码智能数据使用说明
- BaseAgent
- 阶段 1：Settings 与 Prompt 单一来源
- NewMockSessionRepository
- fallback_test.go
- dependencies
- components.json
- getJSON
- NewToolResult
- 阶段 2：Engine 结果契约与 ContextBuilder
- json_parser_repair.go
- time.Duration
- 部署指南
- 阶段 6：遗留执行基础设施与 Session 执行字段清理
- a2a_client.go
- testLLMModelRepo
- MCPTool
- ShellService
- LLM
- DefaultAgentConfig
- 2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式
- run
- 4.3 11 个 Bug 拆解
- event.go
- Err
- LLMRuntimeConfig
- PlannerReActFlow
- Refactoring with GitNexus
- routes.py
- go-manus - 通用 AI Agent 系统
- RedisStreamTask
- OSS
- MergeDeltas
- testAppConfigRepo
- llm_model.go
- A2ATool
- protocol.go
- NewAppConfigService
- File
- 集成测试
- inMemoryMessageQueue
- NewDefaultTaskRegistry
- ServiceRegressionTests
- ApiEndpointTests
- Manus 沙箱服务
- Manus 前端 UI
- Commands
- 1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱
- MessageTool
- DefaultAppConfigService
- .loadOne
- status_service_test.go
- go-manus 面试指南
- Run/Session 重构实施状态
- docs/README.md
- Python vs Go 版本文件对比报告
- ._iter_file_lines
- NewOpenAIClient
- 6. 代码评审与回归测试：测试不是覆盖，而是契约
- vnc-overlay.tsx
- go-manus 实战经验与面试考点沉淀（2026-09-04）
- 3. Docker Desktop 替换：macOS 容器运行时选型
- logger_test.go
- session-header.tsx
- Run/Session 重构方案集
- runInTx
- 3.3 Repository 层架构设计
- 5. JSON 库替换踩坑：sonic 与 encoding/json 行为差异
- 当前架构总览
- 一、核心 Agent 模块
- 7.2 工具系统详细对比
- 3. 核心知识点详解
- 七、详细模块对比
- 5. 注意事项和踩坑点
- 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作
- 快速部署
- 技术方案文档编写计划
- 3.4 集成测试
- 3.2 PostgreSQL INTERVAL 与参数化查询
- AgentService
- stubLLM
- 3.1 Go 数据库错误处理：errors.Is() vs ==
- 3.5 Session 与 Agent 架构
- 4. 可能的面试追问
- 3. 业务概念
- 学习路径
- NewLLMModelService
- 4. 快速启动（nginx 网关模式，推荐）
- LLMModelService
- 文档编写进度
- findings.md
- next-themes
- Postgres
- @radix-ui/react-avatar
- @radix-ui/react-scroll-area
- createSessionForTest
- postcss.config.mjs
- validConfig
- sandbox
- 五、外部依赖
- 7.1 Memory 记忆模块详细对比
- Impact Analysis with GitNexus
- 7.4 存储层详细对比
- remark-gfm
- @radix-ui/react-slot
- test-env-down.sh
- test-env-up.sh
- @radix-ui/react-switch
- github.com/Huang131/go-manus/api
- tools_test.go
- react-dom
- Debugging with GitNexus
- Exploring Codebases with GitNexus
- GitNexus Guide
- GitNexus — Code Intelligence
- GitNexus — Code Intelligence
- react-markdown
- next.config.ts
- BaseModel
- sensenova_integration_test.go
- 7.5 外部依赖详细对比
- NewLoader
- mockMQWrapper
- markdown-content.tsx
- 维护状态
- App
- 三、服务层
- external_test.go
- 六、差异汇总
- 八、改进建议
- NewSimpleMemory
- archive/README.md
- HealthStatus
- mockLLMForTest
- SearchResults
- NewFileService
- SearchTool
- openai_llm_test.go
- NewRedisStreamTask
- sonner
- 提交策略
- package.json
- eslint.config.mjs
- @novnc/novnc
- @radix-ui/react-label
- tailwind-merge

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `ToolResult` - 107 edges
3. `File` - 80 edges
4. `Err()` - 59 edges
5. `ServiceRegressionTests` - 56 edges
6. `ShellService` - 53 edges
7. `RedisStreamTask` - 45 edges
8. `FileService` - 45 edges
9. `NewRedisStreamTask()` - 43 edges
10. `Info()` - 40 edges

## Surprising Connections (you probably didn't know these)
- `TestShouldInlineAndTruncate()` --calls--> `ShouldRAG()`  [INFERRED]
  api/internal/agent/attachment/loader_test.go → api/internal/agent/attachment/policy.go
- `NewBaseAgent()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgent()` --calls--> `NewToolRegistry()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/tools.go
- `NewBaseAgentWithParser()` --calls--> `NewSimpleMemory()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/memory.go
- `NewBaseAgentWithParser()` --calls--> `NewToolRegistry()`  [INFERRED]
  api/internal/agent/base.go → api/internal/agent/tools.go

## Import Cycles
- None detected.

## Communities (208 total, 26 thin omitted)

### Community 0 - "index.tsx"
Cohesion: 0.06
Nodes (42): PageProps, findLatestTool(), SessionDetailView(), A2APreview(), BrowserPreview(), ConsoleRecord, FileToolPreview(), getToolContent() (+34 more)

### Community 1 - "cn"
Cohesion: 0.05
Nodes (60): GlobalHeader(), LeftPanel(), DialogOverlay(), DropdownMenu(), DropdownMenuCheckboxItem(), DropdownMenuContent(), DropdownMenuItem(), DropdownMenuLabel() (+52 more)

### Community 2 - "BuildAttachmentContextSection"
Cohesion: 0.32
Nodes (5): BuildAttachmentContextSection(), TestBuildAttachmentContextSection_Empty(), TestBuildAttachmentContextSection_Typed(), TestBuildAttachmentContextSection_IncludesContent(), TestCreatePlanPrompt_UsesAttachmentContext()

### Community 3 - "AppException"
Cohesion: 0.09
Nodes (17): Exception, FastAPI, register_exception_handlers(), AppException, BadRequestException, NotFoundException, Any, 文件操作只接受普通文件，避免目录被当作文件读取或删除。 (+9 more)

### Community 4 - "index.ts"
Cohesion: 0.08
Nodes (50): configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout(), get(), handleErrorResponse() (+42 more)

### Community 5 - "ToolResult"
Cohesion: 0.06
Nodes (21): requiredToolString(), TestBrowser_Navigate(), TestSandbox_ReadFile(), browserScript(), SandboxClient, jsString(), NewSandboxClient(), sandboxToolResult() (+13 more)

### Community 6 - "queryer"
Cohesion: 0.13
Nodes (10): pgx.Tx, scanFile(), collectRows(), pgx.Tx, T, newQueryer(), pgx.Rows, PostgresFileRepository (+2 more)

### Community 7 - "MCPSetting.tsx"
Cohesion: 0.10
Nodes (45): DeleteSessionDialogProps, ManusSettings(), SETTING_MENUS, SettingTab, emptyDraft, Mode, ModelConfigManager(), RenameSessionDialogProps (+37 more)

### Community 8 - "app_test.go"
Cohesion: 0.08
Nodes (31): Build(), newLifecycleManager(), normalizeFactories(), resolveAgentConfig(), TestAppCloseHandlesNilReceiver(), TestAppCloseIsIdempotent(), TestAppShutdownClosesRegisteredResources(), TestAppStartRunsRegisteredHooksOnce() (+23 more)

### Community 9 - "testing.T"
Cohesion: 0.06
Nodes (52): TestFlowStatus_ToSessionStatus(), TestFlowStatus_Values(), TestPlannerReActFlow_EmitEventStopsWhenContextCanceled(), TestPlannerReActFlow_GetPlanReturnsSnapshot(), TestPlannerReActFlow_PlanCreation(), TestPlannerReActFlow_PlanStepStatus(), TestPlannerReActFlow_StateTransitions(), TestPlannerReActFlow_StatusGetters() (+44 more)

### Community 10 - "重构.md"
Cohesion: 0.08
Nodes (24): messages, outbox, Plan 和 Step, Prompt 配置, Run, run_events, run_plans, runs (+16 more)

### Community 11 - "session-detail-view.tsx"
Cohesion: 0.09
Nodes (38): AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock(), ToolRow(), FilePreviewPanel() (+30 more)

### Community 12 - "PostgresSessionRepository"
Cohesion: 0.21
Nodes (4): pgx.Tx, marshalSessionEvents(), TestMarshalSessionEventsRejectsInvalidJSON(), PostgresSessionRepository

### Community 13 - "SupervisorService"
Cohesion: 0.09
Nodes (31): activate_timeout(), cancel_timeout(), extend_timeout(), get_status(), get_timeout_status(), get, post, Response (+23 more)

### Community 14 - "layout.tsx"
Cohesion: 0.12
Nodes (16): metadata, DeleteSessionDialog(), RenameSessionDialog(), SessionList(), Toaster(), Session, AUTO_MODEL_ID, ModelsContext (+8 more)

### Community 15 - "Docker 构建优化实践笔记"
Cohesion: 0.05
Nodes (38): 1.1 问题分析, 1.2 解决方案, 1.3 验证结果, 2.1 问题分析, 2.2 解决方案, 2.3 验证结果, 3.1 问题分析, 3.2.1 使用 uv 替代 pip (+30 more)

### Community 16 - "StdioMCPClient"
Cohesion: 0.13
Nodes (16): MCPClientManager, NewStdioMCPClient(), MCPClient, MCPConfig, MCPConfigServer, MCPContent, MCPError, MCPRequest (+8 more)

### Community 17 - "NewProviderError"
Cohesion: 0.11
Nodes (21): LLMRequest, mustMarshalMap(), parseJSONMap(), textOf(), userContent(), streamContext(), ErrorKind, isFallbackable() (+13 more)

### Community 18 - "Run/Session 重构目标架构"
Cohesion: 0.09
Nodes (21): 10. API 目标, 11. 包依赖约束, 12. 最终删除项, 13. 架构验收, 1. 背景, 2. 范围, 4. 目标调用链, 5. 数据所有权 (+13 more)

### Community 19 - "LLMModel"
Cohesion: 0.15
Nodes (9): LLMModel, encodeModelJSONColumns(), pgx.Tx, LLMModelRepository, modelJSON(), NewLLMModelRepository(), scanLLMModel(), modelJSONColumns (+1 more)

### Community 20 - "anthropic_llm_test.go"
Cohesion: 0.27
Nodes (17): NewAnthropicClient(), anthropicErrorResponse(), errKind(), newAnthropicErrorClient(), newAnthropicTestClient(), TestAnthropicClient_HTTPErrorClassification(), TestAnthropicClient_NetworkError(), TestAnthropicClient_ProtocolError_NotFallbackable() (+9 more)

### Community 21 - "TestMessageQueueMockOperations"
Cohesion: 0.27
Nodes (5): TestMessageQueueDefaultTimeouts(), TestMessageQueueGetBlockingHonorsContext(), TestMessageQueueInterface(), TestMessageQueueMockOperations(), mockMQ

### Community 22 - "阶段 4：后端生产执行语义切换"
Cohesion: 0.11
Nodes (19): A. 新执行组件（尚未接生产）, B. 原子切换生产装配, C. 并发、重启与回归加固, Redis Run Event Stream, 创建、继续和状态提交, 前置条件, 取消优先与并发, 回滚 (+11 more)

### Community 23 - "batchTaskOutputMQ"
Cohesion: 0.18
Nodes (8): batchTaskOutputMQ, parseTaskOutputMessages(), streamDataString(), TestParseTaskOutputMessagesBatchSkipsNilData(), TestStreamDataStringMarshalsNonStringData(), StreamMessage, BatchMessageQueue, MessageQueue

### Community 24 - "logger 包 os.Stdout 误关闭 Bug 深度解析"
Cohesion: 0.06
Nodes (32): 1. 谁创建，谁负责关闭, 2. 进程级共享资源的保护契约, 3. 接口组合与意外暴露, 4. 空操作（no-op）包装模式, logger 包 os.Stdout 误关闭 Bug 深度解析, neverCloseSyncer 自身的并发安全, 一、问题背景, 七、面试话术 (+24 more)

### Community 25 - "ToolProvider"
Cohesion: 0.16
Nodes (8): BrowserTool, ToolProvider, NewBrowserTool(), A2AConfig, MCPConfig, NewToolProvider(), toolNames(), Browser

### Community 26 - "app/page.tsx"
Cohesion: 0.21
Nodes (8): ChatInput, ChatInputProps, ChatInputRef, SuggestedQuestions(), SuggestedQuestionsProps, suggestedQuestions, sessionApi, FileInfo

### Community 27 - "NewRoutedLLM"
Cohesion: 0.21
Nodes (24): modelNames(), openAIModelWithID(), openAITextProfile(), TestRoutedLLM_AllowFallbackBeforeToolExecution(), TestRoutedLLM_AutoAppendsConfiguredEnvFallback(), TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback(), TestRoutedLLM_DoNotFallbackAcrossProtocol(), TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect() (+16 more)

### Community 28 - "endpoints/file.py"
Cohesion: 0.13
Nodes (35): api_route, FileResponse, check_file_exists(), delete_file(), download_file(), find_files(), FileService, get (+27 more)

### Community 29 - "compilerOptions"
Cohesion: 0.07
Nodes (28): dom, dom.iterable, esnext, **/*.mts, .next/dev/types/**/*.ts, next-env.d.ts, .next/types/**/*.ts, node_modules (+20 more)

### Community 30 - "openai_llm.go"
Cohesion: 0.10
Nodes (26): normalizeAnthropicStopReason(), TestNormalizeAnthropicStopReason(), LLMRequest, classifyHTTPError(), estimateCostUSD(), LLMRequest, normalizeOpenAIFinishReason(), reasoningTokens() (+18 more)

### Community 31 - "CODE_REVIEW_2026-09-11.md"
Cohesion: 0.09
Nodes (22): A. 流式链路复查（commit 4892225）, api, B. service 层专项（新发现）, C. 昨日 backlog 复核（HEAD b1f3827 仍开放）, D. 修复批次, E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）, sandbox, ui (+14 more)

### Community 32 - "Config"
Cohesion: 0.09
Nodes (24): TestLoadAllConfigFiles(), formatFieldError(), A2AAgent, A2AConfig, Config, DatabaseConfig, ObjectStorageConfig, RedisConfig (+16 more)

### Community 33 - "endpoints/shell.py"
Cohesion: 0.20
Nodes (23): exec_command(), kill_process(), post, Response, 根据传递的会话+写入内容+按下回车标识向指定子进程写入数据, 根据传递的会话id+是否返回控制台标识获取Shell命令执行结果, read_shell_output(), wait_process() (+15 more)

### Community 34 - "Tool"
Cohesion: 0.39
Nodes (4): MultiFunctionTool, Tool, ToolRegistry, isReadOnlyToolSchema()

### Community 35 - "阶段 5：Run API、UI 与 SSE 契约切换"
Cohesion: 0.12
Nodes (16): A. 新 API 和 SSE（旧 UI 仍可运行）, B. UI 切换并删除旧路由, SSE 契约, UI 状态规则, 前置条件与唯一语义, 回滚, 失败或中断恢复, 完成定义 (+8 more)

### Community 36 - "记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复"
Cohesion: 0.09
Nodes (21): neverCloseSyncer 自身, 业界方案对比, 修复方案演进, 多个 goroutine 同时调用 Init, 并发安全性分析, 延伸思考：这个 Bug 的本质, 方案一：哨兵身份比较（不完整）, 方案三：不调用 Close，只调用 Sync (+13 more)

### Community 37 - "DefaultLLMModelService"
Cohesion: 0.18
Nodes (8): LLMModelTestResponse, isUniqueViolation(), normalizeDefaults(), truncateModelTestContent(), validateRequired(), DefaultLLMModelService, HealthInvalidator, RuntimeHealthReader

### Community 38 - ".initRoutes"
Cohesion: 0.08
Nodes (44): defaultTrustedProxies(), NewSessionHandler(), NewMockSessionServiceForHandler(), setupRouter(), TestNewSSEContext_PreservesRequestValuesWithoutCancellation(), TestSessionHandler_Chat(), TestSessionHandler_ClearUnread(), TestSessionHandler_Create() (+36 more)

### Community 39 - "RedisStreamMessageQueue"
Cohesion: 0.27
Nodes (5): decodeStreamData(), RedisStreamMessageQueue, redis.Client, NewRedisStreamMessageQueue(), resolveBlockTimeout()

### Community 40 - "context.Context"
Cohesion: 0.03
Nodes (29): attachmentStorage, chatContextSessionRepository, mockSandbox, mockSessionRepo, stopSessionRepository, successfulStopSessionRepository, nonNilEvents(), TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits() (+21 more)

### Community 41 - "DefaultFileCleanupService"
Cohesion: 0.13
Nodes (17): defaultFactories(), FileCleanupService, SchedulerRunner, NewFileCleanupScheduler(), NewFileCleanupService(), TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(), TestFileCleanupSchedulerStartIsIdempotent(), TestFileCleanupSchedulerStopCancelsCleanupContext() (+9 more)

### Community 42 - "agent/config.go"
Cohesion: 0.18
Nodes (10): A2AAgent, MCPServer, A2AAgent, A2AConfig, AgentConfig, MCPConfig, MCPServer, newA2AConfig() (+2 more)

### Community 43 - "test_api_endpoints.py"
Cohesion: 0.07
Nodes (30): BaseSettings, HTTPConnection, Request, get_settings(), Settings, auto_extend_timeout_middleware(), 使用中间件延长每次API请求是超时销毁时间, get_file_service() (+22 more)

### Community 44 - "AppConfig"
Cohesion: 0.19
Nodes (8): AppConfig, AppConfigType, pgx.Tx, AppConfigRepository, NewAppConfigRepository(), scanAppConfig(), PostgresAppConfigRepository, MockAppConfigRepository

### Community 45 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.05
Nodes (50): BadRequest(), Conflict(), FailedPrecondition(), Forbidden(), Internal(), New(), NotFound(), ToInternal() (+42 more)

### Community 46 - "阶段 3：Run 领域模型与 PostgreSQL 存储"
Cohesion: 0.14
Nodes (14): Repository 契约, 前置条件与唯一语义, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库设计, 文件清单 (+6 more)

### Community 47 - "devDependencies"
Cohesion: 0.10
Nodes (21): eslint, eslint-config-next, tailwindcss, @tailwindcss/postcss, tw-animate-css, @types/node, @types/novnc__novnc, @types/react (+13 more)

### Community 48 - "代码智能数据使用说明"
Cohesion: 0.15
Nodes (13): Clone 后的首次使用, GitNexus, GitNexus, GitNexus 显示仓库未索引, Graphify, Graphify, Graphify 文件仍被全局 Git ignore 忽略, 代码智能数据使用说明 (+5 more)

### Community 49 - "BaseAgent"
Cohesion: 0.08
Nodes (16): BaseAgent, InvokeResult, Memory, SimpleMemory, ToolCallResult, AgentConfig, NewBaseAgentWithParser(), shouldPublishDeltas() (+8 more)

### Community 50 - "阶段 1：Settings 与 Prompt 单一来源"
Cohesion: 0.15
Nodes (13): 关键契约, 前置条件与恢复基线, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库与 API 变更, 文件清单 (+5 more)

### Community 51 - "NewMockSessionRepository"
Cohesion: 0.19
Nodes (22): SessionService, NewSessionService(), NewSessionServiceWithSandbox(), newMockFileRepo(), NewMockSessionRepository(), TestSessionService_AppendEvent(), TestSessionService_ClearUnreadCount(), TestSessionService_CreateSession() (+14 more)

### Community 52 - "fallback_test.go"
Cohesion: 0.20
Nodes (16): CanFallbackAfterToolUse(), CanFallbackTo(), capabilitiesEquivalent(), fullProfile(), TestCanFallbackAfterToolUse_AllReadOnly(), TestCanFallbackAfterToolUse_HasWriteTool(), TestCanFallbackAfterToolUse_NoTools(), TestCanFallbackTo_CapabilityEquivalent() (+8 more)

### Community 53 - "dependencies"
Cohesion: 0.11
Nodes (19): class-variance-authority, clsx, lucide-react, next, @radix-ui/react-dialog, @radix-ui/react-dropdown-menu, @radix-ui/react-separator, @radix-ui/react-tooltip (+11 more)

### Community 54 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 55 - "getJSON"
Cohesion: 0.11
Nodes (49): Response, TestAppConfigAPI_A2AConfig_Lifecycle(), TestAppConfigAPI_AgentConfig_Lifecycle(), TestAppConfigAPI_EmptyUpdate(), TestAppConfigAPI_MCPConfig_Delete(), TestAppConfigAPI_MCPConfig_Lifecycle(), TestFileAPI_Delete_NotFound(), TestFileAPI_GetInfo_NotFound() (+41 more)

### Community 56 - "NewToolResult"
Cohesion: 0.10
Nodes (19): attachmentSandbox, FileTool, ShellTool, NewFileTool(), NewShellTool(), TestFileTool(), TestShellTool_Description(), TestShellTool_Invoke_Error() (+11 more)

### Community 57 - "阶段 2：Engine 结果契约与 ContextBuilder"
Cohesion: 0.15
Nodes (13): 关键契约, 前置条件与唯一语义, 失败或中断恢复, 完成定义, 实施顺序与检查点, 提交与回滚, 数据库与 API 变更, 文件清单 (+5 more)

### Community 58 - "json_parser_repair.go"
Cohesion: 0.17
Nodes (12): NewDefaultJSONParser(), extractJSON(), NewRepairJSONParser(), removeMarkdownCodeBlocks(), removeTrailingCommas(), TestDefaultJSONParser(), TestExtractJSON(), TestRemoveMarkdownCodeBlocks() (+4 more)

### Community 59 - "time.Duration"
Cohesion: 0.10
Nodes (34): CompletedStreamRetention(), TestRedisMessageQueueDefaultTimeoutsUseDurations(), TestRedisMessageQueueStreamRetentionPolicy(), bochaFreshness(), SearchEngine, NewBochaSearchClient(), NewBochaSearchClientWithTimeout(), NewGoogleSearchClient() (+26 more)

### Community 60 - "部署指南"
Cohesion: 0.12
Nodes (17): 10.1 常见问题, 10.2 日志查看, 10.3 完全重置, 10. 故障排查, 1. 部署方式, 2. 架构总览, 3. 前置要求, 5.1 `.env` 模板（`api/.env.example`） (+9 more)

### Community 61 - "阶段 6：遗留执行基础设施与 Session 执行字段清理"
Cohesion: 0.15
Nodes (13): 删除清单, 前置条件与恢复基线, 包边界收紧, 回滚, 失败或中断恢复, 完成定义, 实施顺序与提交检查点, 数据库破坏式收敛 (+5 more)

### Community 62 - "a2a_client.go"
Cohesion: 0.17
Nodes (14): A2AClientManager, NewA2AClientManager(), A2AAgentCapabilities, A2AAgentCard, A2AAgentSkill, A2AClientManagerConfig, A2AInvokeResult, A2AJSONRPCError (+6 more)

### Community 63 - "testLLMModelRepo"
Cohesion: 0.28
Nodes (19): NewTestContext(), cleanupLLMModel(), newTestModel(), testLLMModelRepo(), TestLLMModelRepo_CreateAndGetByID(), TestLLMModelRepo_Delete(), TestLLMModelRepo_Delete_NotFoundNoError(), TestLLMModelRepo_GetByID_NotFoundReturnsNilNil() (+11 more)

### Community 64 - "MCPTool"
Cohesion: 0.14
Nodes (8): MCPTool, MCPConfig, NewMCPTool(), parseMCPInvokeParams(), TestMCPToolInvokeAllowsMissingParams(), TestMCPToolInvokeValidatesParameters(), TestMCPTool(), TestMCPTool_WithConfig()

### Community 65 - "ShellService"
Cohesion: 0.06
Nodes (25): ConsoleRecord, Process, Lock, 启动并持有输出读取任务，保证一个会话只消费当前进程的输出。, 取消并等待输出读取器，避免旧进程向新命令的记录写入输出。, 格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $, 根据传递的执行目录+命令创建一个asyncio管理的子进程, 追加当前命令输出并维护字节预算，避免每个输出块重复扫描历史记录。 (+17 more)

### Community 66 - "LLM"
Cohesion: 0.25
Nodes (10): defaultLLMClientFactory(), LLM, LLMClientFactory, NewDynamicLLM(), NewDynamicLLMWithFactory(), NewLLMClient(), DynamicLLM, LLMConfigProvider (+2 more)

### Community 67 - "DefaultAgentConfig"
Cohesion: 0.13
Nodes (15): mockLLM, streamingAgentLLM, NewBaseAgent(), TestBaseAgentInvokePublishesTextDeltas(), TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(), DefaultAgentConfig(), TestBaseAgent_InvokeWithEmptyRetry(), TestBaseAgent_InvokeWithEmptyRetry_AllEmpty() (+7 more)

### Community 68 - "2. Go 死锁与 channel 关闭：一次性 ping-pong 协程模式"
Cohesion: 0.17
Nodes (12): 2.1 现象, 2.2 What, 2.3 How, 2.4 Why（设计原则）, 2.5 Alternatives, 2.6 Trade-offs, 2.7 面试要点, 2.8 SSE 实现最佳实践 (+4 more)

### Community 69 - "run"
Cohesion: 0.27
Nodes (8): classifyBootstrapError(), main(), run(), serve(), TestServeReportsUnexpectedServerError(), DefaultOptions(), net/http.Server, Config

### Community 70 - "4.3 11 个 Bug 拆解"
Cohesion: 0.17
Nodes (12): 4.3 11 个 Bug 拆解, Bug 10：client-side 未处理 abort, Bug 11：React 18 StrictMode 双调用, Bug 1：SSE `event:` 字段被钉死为 `message`, Bug 2：plan/step/title 事件被当作 message 渲染, Bug 3：done 事件未发出, Bug 4：连接复用时残留 state, Bug 5：心跳机制缺失 (+4 more)

### Community 71 - "event.go"
Cohesion: 0.06
Nodes (31): MessageRole, NewDoneEvent(), NewMessageDeltaEvent(), NewMessageDoneEvent(), NewMessageEvent(), NewShellOutputEvent(), NewTitleEvent(), NewToolCalledEvent() (+23 more)

### Community 72 - "Err"
Cohesion: 0.05
Nodes (39): AgentTaskRunner, AgentTaskRunnerConfig, readerWrapper, SessionRuntime, NewSessionRuntime(), AgentConfig, NewAgentTaskRunner(), ModelIDFromContext() (+31 more)

### Community 73 - "LLMRuntimeConfig"
Cohesion: 0.14
Nodes (15): betterHealth(), canFallbackAfterToolUse(), configKey(), LLMRequest, hasCapabilityProfile(), healthRank(), healthOfModel(), TestRoutedLLM_GetHealth() (+7 more)

### Community 74 - "PlannerReActFlow"
Cohesion: 0.11
Nodes (22): FlowStatus, PlannerAgent, PlannerReActFlow, ReActAgent, TaskInput, AgentConfig, NewPlannerReActFlow(), planHasFailedStep() (+14 more)

### Community 75 - "Refactoring with GitNexus"
Cohesion: 0.18
Nodes (10): Checklists, Example: Rename `validateUser` to `authenticateUser`, Extract Module, Refactoring with GitNexus, Rename Symbol, Risk Rules, Split Function/Service, Tools (+2 more)

### Community 76 - "routes.py"
Cohesion: 0.40
Nodes (3): APIRouter, create_api_routes(), 创建API路由，涵盖整个沙箱项目的所有API

### Community 77 - "go-manus - 通用 AI Agent 系统"
Cohesion: 0.14
Nodes (14): 1. Agent 架构, 2. A2A (Agent to Agent), 3. MCP (Model Context Protocol), 4. 沙箱环境, API 开发, API 文档, go-manus - 通用 AI Agent 系统, 健康检查 (+6 more)

### Community 78 - "RedisStreamTask"
Cohesion: 0.07
Nodes (14): blockingTaskRunner, DefaultTaskRegistry, panicTaskRunner, RedisStreamTask, Stream, Task, TaskRegistryInterface, TaskRunner (+6 more)

### Community 79 - "OSS"
Cohesion: 0.15
Nodes (16): TestBuildWithFactoriesInjectsFileCleanupScheduler(), ensureBucket(), generateURL(), OSS, NewOSS(), newUploader(), providerDefaultEndpoint(), TestGenerateURL() (+8 more)

### Community 80 - "MergeDeltas"
Cohesion: 0.20
Nodes (15): EstimateContextTokens(), isToolCallFinishReason(), MergeDeltas(), repeat(), TestEstimateContextTokens_ASCII(), TestEstimateContextTokens_Mixed(), TestEstimateContextTokens_NonASCII(), TestMergeDeltas_CanonicalToolCallsRetainsToolCall() (+7 more)

### Community 81 - "testAppConfigRepo"
Cohesion: 0.42
Nodes (8): testAppConfigRepo(), TestAppConfigRepo_DeleteConfig(), TestAppConfigRepo_GetConfig_NotFoundReturnsNilNil(), TestAppConfigRepo_ListConfigs(), TestAppConfigRepo_SaveAndGetConfig_RoundTrip(), TestAppConfigRepo_SaveConfig_UpdateExisting(), TestAppConfigRepo_WithTx_CommitsOnSuccess(), TestAppConfigRepo_WithTx_RollsBackOnError()

### Community 82 - "llm_model.go"
Cohesion: 0.25
Nodes (13): fullCaps(), DefaultCapabilities(), CostPolicy, ModelCapabilities, ReasoningMode, RuntimeHealth, RequestPolicy, MergeDefaultCapabilities() (+5 more)

### Community 83 - "A2ATool"
Cohesion: 0.20
Nodes (4): A2ATool, A2AConfig, marshalResponseText(), TestA2ATool_MarshalResponseTextHandlesUnsupportedData()

### Community 84 - "protocol.go"
Cohesion: 0.23
Nodes (11): ResponseFormat, ToolCall, ToolSpec, AudioURL, ContentPart, ImageURL, LLMRequest, RouteStrategy (+3 more)

### Community 85 - "NewAppConfigService"
Cohesion: 0.42
Nodes (11): NewAppConfigService(), NewMockAppConfigRepository(), TestAppConfigService_DeleteAndUpdateA2AServer(), TestAppConfigService_DeleteMCPServer(), TestAppConfigService_DeleteMCPServer_NotFoundWithoutConfig(), TestAppConfigService_GetA2AConfig(), TestAppConfigService_GetAgentConfig(), TestAppConfigService_GetConfig_RepositoryError() (+3 more)

### Community 86 - "File"
Cohesion: 0.04
Nodes (11): attachmentFileRepository, generatedFileRepository, File, FileRepository, NewFileRepository(), io.ReadCloser, cleanupRepoStub, DefaultFileService (+3 more)

### Community 87 - "集成测试"
Cohesion: 0.14
Nodes (13): CI 集成, Makefile 命令说明, Q: 如何只运行特定测试, Q: 测试环境需要重新初始化吗, Q: 测试连接失败, 前置条件, 常见问题, 快速开始 (+5 more)

### Community 88 - "inMemoryMessageQueue"
Cohesion: 0.10
Nodes (11): inMemoryMessage, inMemoryMessageQueue, inMemoryStream, mockEchoTool, mockFailingTool, TestReadTaskOutputAfterTaskUnregistered(), TestRedisStreamTask_GetOutputReadsBufferedEventsWhenStartIDEmpty(), newInMemoryMessageQueue() (+3 more)

### Community 89 - "NewDefaultTaskRegistry"
Cohesion: 0.24
Nodes (14): NewDefaultTaskRegistry(), TestDefaultTaskRegistry_CleanupCompleted(), TestDefaultTaskRegistry_CleanupCompletedWaitsForFinished(), TestDefaultTaskRegistry_Clear(), TestDefaultTaskRegistry_ConcurrentAccess(), TestDefaultTaskRegistry_Count(), TestDefaultTaskRegistry_DoubleUnregister(), TestDefaultTaskRegistry_List() (+6 more)

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

### Community 95 - "Commands"
Cohesion: 0.20
Nodes (9): After Indexing, analyze — Build or refresh the index, clean — Delete the index, Commands, GitNexus CLI Commands, list — Show all indexed repos, status — Check index freshness, Troubleshooting (+1 more)

### Community 96 - "1. JSON `[]byte` vs `json.RawMessage`：PostgreSQL JSONB base64 编码陷阱"
Cohesion: 0.20
Nodes (10): 1.1 现象, 1.2 What, 1.3 How, 1.4 Why（根因）, 1.5 Alternatives, 1.6 Trade-offs, 1.7 面试要点, 1.8 适用场景 (+2 more)

### Community 97 - "MessageTool"
Cohesion: 0.22
Nodes (4): MessageTool, TestMessageTool_NotifyUserSchema(), NewMessageTool(), TestMessageTool()

### Community 98 - "DefaultAppConfigService"
Cohesion: 0.14
Nodes (12): MCPServer, A2AConfig, A2AServer, AgentConfig, MCPConfig, MCPServer, getConfig(), T (+4 more)

### Community 99 - ".loadOne"
Cohesion: 0.31
Nodes (7): TestShouldInlineAndTruncate(), FileContext, IsLikelyBinary(), ShouldInline(), ShouldRAG(), ShouldTruncate(), LoadMode

### Community 100 - "status_service_test.go"
Cohesion: 0.14
Nodes (7): TestStatusService_GetHealthStatus_SkippedServicesDegraded(), TestStatusService_GetHealthStatus_WrapsHealthErrors(), TestStatusServiceUsesInjectedClock(), TestToInternal(), fakeOSS, fakePostgres, fakeRedis

### Community 101 - "go-manus 面试指南"
Cohesion: 0.25
Nodes (8): 1.1 项目定位, 1.2 核心模块, 1.3 数据库设计, 1. 项目概述, 2. 技术栈总结, go-manus 面试指南, 目录, 附录：关键代码位置

### Community 102 - "Run/Session 重构实施状态"
Cohesion: 0.25
Nodes (8): Run/Session 重构实施状态, 决策偏差, 当前入口, 当前状态, 更新模板, 最近一次验证, 阶段内检查点, 阻塞记录

### Community 104 - "Python vs Go 版本文件对比报告"
Cohesion: 0.22
Nodes (8): 2.1 工具注册, 2.2 工具调用方式, 4.1 Session Repository, 4.2 File Repository, Python vs Go 版本文件对比报告, 二、工具系统, 四、存储层, 目录

### Community 105 - "._iter_file_lines"
Cohesion: 0.25
Nodes (6): _LineChunk, 逐行收集范围内内容，任何上限命中后立即停止读取。, 按固定块解码文件，超长无换行内容达到上限即截断。, 文件流的一段逻辑行；truncated 表示单行超过内存上限。, 按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。, _truncate_utf8()

### Community 106 - "NewOpenAIClient"
Cohesion: 0.17
Nodes (13): NewOpenAIClient(), newTestClient(), TestNewOpenAIClient_DoesNotApplyGlobalTimeout(), BuildRuntimeConfigFromModel(), convertRequestPolicyExtra(), ProtocolFromProvider(), runtimeConfigToOpenAIClientConfig(), TestBuildRuntimeConfigFromModel() (+5 more)

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
Cohesion: 0.11
Nodes (26): GetLevel(), Init(), InitWithConfig(), SetLevel(), shutdownCurrent(), Sync(), stdoutWriteOK(), TestCallerInBusinessCode() (+18 more)

### Community 112 - "session-header.tsx"
Cohesion: 0.15
Nodes (23): SessionItem(), SessionItemProps, Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup(), AvatarGroupCount(), AvatarImage() (+15 more)

### Community 113 - "Run/Session 重构方案集"
Cohesion: 0.29
Nodes (7): Run/Session 重构方案集, 不保留两套长期业务语义, 后续模型的恢复步骤, 工作包, 文档与代码的更新规则, 每个工作包的硬门禁, 目标

### Community 114 - "runInTx"
Cohesion: 0.36
Nodes (5): pgx.Tx, T, joinRollbackError(), rollbackTx(), runInTx()

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

### Community 120 - "3. 核心知识点详解"
Cohesion: 0.33
Nodes (6): 3.6 文件清理服务, 3. 核心知识点详解, 业务规则：孤儿文件判定, 清理服务架构, 调度器设计, 面试追问

### Community 121 - "七、详细模块对比"
Cohesion: 0.33
Nodes (6): 7.3.1 SessionService 接口设计, 7.3.2 核心实现对比, 7.3.3 AgentService 任务管理, 7.3.4 面试分析点, 7.3 服务层详细对比, 七、详细模块对比

### Community 122 - "5. 注意事项和踩坑点"
Cohesion: 0.33
Nodes (6): 5.1 Go 错误处理, 5.2 PostgreSQL 参数化, 5.3 Repository 设计, 5.4 软删除查询, 5.5 文件清理, 5. 注意事项和踩坑点

### Community 123 - "4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作"
Cohesion: 0.33
Nodes (6): 4.1 现象, 4.2 What, 4.4 SSE 协议规范, 4.5 SSE vs WebSocket vs 长轮询, 4.6 面试要点, 4. SSE 流式响应：从"一直转圈"看 SSE 协议与前端协作

### Community 124 - "快速部署"
Cohesion: 0.33
Nodes (6): 一键部署（推荐）, 前置要求, 容器列表, 快速部署, 服务架构（Nginx 网关模式）, 本地开发

### Community 125 - "技术方案文档编写计划"
Cohesion: 0.33
Nodes (5): 已知环境, 技术方案文档编写计划, 目标, 硬约束, 阶段

### Community 126 - "3.4 集成测试"
Cohesion: 0.29
Nodes (7): 3.4 集成测试, docker-compose 测试环境, TestMain 基础设施, 构建标签, 测试辅助函数, 竞态检测, 面试追问

### Community 127 - "3.2 PostgreSQL INTERVAL 与参数化查询"
Cohesion: 0.40
Nodes (5): 3.2 PostgreSQL INTERVAL 与参数化查询, Why：为什么 INTERVAL 不能参数化？, 问题背景, 面试追问, 项目中的正确实现

### Community 128 - "AgentService"
Cohesion: 0.22
Nodes (8): Capabilities, Repositories, NormalizeAgentConfig(), A2AConfig, AgentService, AgentConfig, MCPConfig, NewAgentService()

### Community 129 - "stubLLM"
Cohesion: 0.31
Nodes (4): LLMRequest, TestDynamicLLM_UsesFactory(), TestRoutedLLMFromSingleProvider(), stubLLM

### Community 130 - "3.1 Go 数据库错误处理：errors.Is() vs =="
Cohesion: 0.33
Nodes (6): 3.1 Go 数据库错误处理：errors.Is() vs ==, errors.Is() 工作原理, Why：为什么不能直接用 == 比较？, 问题背景, 面试追问, 项目中的正确写法

### Community 131 - "3.5 Session 与 Agent 架构"
Cohesion: 0.40
Nodes (5): 3.5 Session 与 Agent 架构, Agent Memory 机制, AppendEvent 业务逻辑, MessageEvent 结构, 面试追问

### Community 132 - "4. 可能的面试追问"
Cohesion: 0.40
Nodes (5): 4.1 Go 语言层面, 4.2 数据库层面, 4.3 架构设计层面, 4.4 测试层面, 4. 可能的面试追问

### Community 133 - "3. 业务概念"
Cohesion: 0.40
Nodes (5): 3.1 Session, 3.2 Run, 3.3 Plan 与 Step, 3.4 基础设施作业, 3. 业务概念

### Community 134 - "学习路径"
Cohesion: 0.40
Nodes (5): 学习路径, 第一阶段：理解 Agent 基础, 第三阶段：理解外部集成, 第二阶段：掌握工具系统, 第四阶段：部署和扩展

### Community 135 - "NewLLMModelService"
Cohesion: 0.18
Nodes (25): NewLLMModelService(), NewMockLLMModelRepository(), TestLLMModelService_Create(), TestLLMModelService_Create_MapsUniqueViolationToConflict(), TestLLMModelService_Create_PreservesDefaultLookupError(), TestLLMModelService_Create_PreservesPartialCapabilities(), TestLLMModelService_Create_PreservesRepositoryError(), TestLLMModelService_Create_Validation() (+17 more)

### Community 136 - "4. 快速启动（nginx 网关模式，推荐）"
Cohesion: 0.50
Nodes (4): 4.1 准备环境变量, 4.2 一键启动, 4.3 访问入口（统一走 nginx）, 4. 快速启动（nginx 网关模式，推荐）

### Community 137 - "LLMModelService"
Cohesion: 0.29
Nodes (6): NewLLMModelHandler(), LLMModelService, NewLLMModelServiceWithLLMFactory(), TestLLMModelService_Test_MapsProviderAuthError(), TestLLMModelService_Test_RequiresAPIKey(), TestLLMModelService_Test_SendsMinimalRequestAndReturnsLatency()

### Community 141 - "Postgres"
Cohesion: 0.14
Nodes (11): StatusHandler, NewStatusHandler(), Postgres, NewPostgres(), Redis, redis.Client, NewRedis(), StatusService (+3 more)

### Community 144 - "createSessionForTest"
Cohesion: 0.15
Nodes (38): deleteSessionDirect(), testFileRepo(), TestFileRepo_CountExpiredFiles(), TestFileRepo_CreateAndGetByID(), TestFileRepo_Delete_PhysicalDelete(), TestFileRepo_GetByID_NotFoundReturnsNil(), TestFileRepo_GetBySessionAndFilepath(), TestFileRepo_GetBySessionAndFilepath_NotFound() (+30 more)

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

### Community 171 - "tools_test.go"
Cohesion: 0.15
Nodes (19): mockTool, NewA2ATool(), TestA2ATool(), NewToolRegistry(), TestA2ATool_Cleanup(), TestA2ATool_Description(), TestA2ATool_Initialize(), TestA2ATool_Invoke_CallAgentRejectsInvalidRequiredTypes() (+11 more)

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

### Community 181 - "sensenova_integration_test.go"
Cohesion: 0.44
Nodes (8): requireSenseNovaKey(), senseNovaReasoningEffort(), senseNovaRequest(), TestSenseNova_ChatModelsAreOpenAICompatible(), TestSenseNova_ImageModels(), TestSenseNova_ListModels(), TestSenseNova_StructuredJSONForPlannerCompatibleModels(), senseNovaChatResponse

### Community 182 - "7.5 外部依赖详细对比"
Cohesion: 0.40
Nodes (5): 7.5.1 LLM 调用库对比, 7.5.2 Redis Stream 任务队列对比, 7.5.3 依赖包对比, 7.5.4 面试分析点, 7.5 外部依赖详细对比

### Community 183 - "NewLoader"
Cohesion: 0.11
Nodes (22): Loader, headBytes(), NewLoader(), normalizeText(), tailBytes(), TestLoader_BinarySkipped(), TestLoader_DownloadError(), TestLoader_InlineBudget() (+14 more)

### Community 185 - "markdown-content.tsx"
Cohesion: 0.33
Nodes (6): components, headingClasses, MarkdownContent(), MarkdownContentProps, normalizeAutolinks(), URL_FOLLOWED_BY_CJK

### Community 186 - "维护状态"
Cohesion: 0.40
Nodes (5): 已完成, 已知限制, 维护状态, 维护规则, 验证基线

### Community 187 - "App"
Cohesion: 0.20
Nodes (8): BuildWithFactories(), newRepositories(), App, externalClients, lifecycleManager, Options, repositories, sync.Mutex

### Community 188 - "三、服务层"
Cohesion: 0.50
Nodes (4): 3.1 AgentService, 3.2 SessionService, 3.3 FileService, 三、服务层

### Community 189 - "external_test.go"
Cohesion: 0.10
Nodes (18): TestBrowser_Screenshot(), TestBrowserClientScreenshotExtractsSandboxOutput(), TestBrowserInterface(), TestBrowserScriptUsesNodeAndCDP(), TestLLM_Invoke(), TestLLM_Invoke_WithError(), TestLLM_ModelInfo(), TestLLMInterface() (+10 more)

### Community 190 - "六、差异汇总"
Cohesion: 0.50
Nodes (4): 6.1 功能缺失列表, 6.2 实现差异列表, 6.3 架构差异, 六、差异汇总

### Community 191 - "八、改进建议"
Cohesion: 0.50
Nodes (4): 中优先级, 低优先级, 八、改进建议, 高优先级

### Community 192 - "NewSimpleMemory"
Cohesion: 0.53
Nodes (5): NewSimpleMemory(), TestSimpleMemory_Add(), TestSimpleMemory_Clear(), TestSimpleMemory_Compact(), TestSimpleMemory_GetMessages()

### Community 194 - "HealthStatus"
Cohesion: 0.70
Nodes (4): HealthStatus, HealthState, ServiceName, ServiceStatus

### Community 196 - "SearchResults"
Cohesion: 0.67
Nodes (3): SearchEvent, SearchResultItem, SearchResults

### Community 205 - "NewFileService"
Cohesion: 0.22
Nodes (16): NewFileService(), TestFileServiceDeleteFileDeletesStorageAndRepo(), TestFileServiceDeleteFileRepoError(), TestFileServiceDeleteFileStorageError(), TestFileServiceDownloadFileNilStorageReturnsStorageUnavailable(), TestFileServiceDownloadFileStorageError(), TestFileServiceDownloadFileWithStorage(), TestFileServiceGetFileInfoFound() (+8 more)

### Community 220 - "SearchTool"
Cohesion: 0.19
Nodes (5): searchEngineStub, SearchTool, NewSearchTool(), TestSearchToolForwardsConfiguredLimit(), TestToolProviderReloadSearchLimitAffectsNewTools()

### Community 222 - "openai_llm_test.go"
Cohesion: 0.17
Nodes (21): newTransportClient(), rawOK(), responseJSON(), TestOpenAIClient_401_Auth(), TestOpenAIClient_429_RateLimit(), TestOpenAIClient_5xx_Server(), TestOpenAIClient_ContentAndReasoning_Separated(), TestOpenAIClient_ContentOnly() (+13 more)

### Community 223 - "NewRedisStreamTask"
Cohesion: 0.17
Nodes (19): mockTaskRunner, NewRedisStreamTask(), TestReadTaskOutputPrefersBatchQueue(), TestRedisStreamTask_Cancel(), TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(), TestRedisStreamTask_DoneChan(), TestRedisStreamTask_FinishDestroysRunner(), TestRedisStreamTask_FinishedChangesAfterRunnerExit() (+11 more)

### Community 235 - "提交策略"
Cohesion: 0.50
Nodes (4): Claude Code 配置, GitNexus, Graphify, 提交策略

### Community 298 - "package.json"
Cohesion: 0.22
Nodes (8): name, private, scripts, build, dev, lint, start, version

## Knowledge Gaps
- **667 isolated node(s):** `SearchConfig`, `MCPServer`, `A2AAgent`, `github.com/Huang131/go-manus/api`, `MCPServer` (+662 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Err()` connect `Err` to `.loadOne`, `run`, `ToolResult`, `.initRoutes`, `context.Context`, `event.go`, `PlannerReActFlow`, `RedisStreamMessageQueue`, `LLMRuntimeConfig`, `github.com/gin-gonic/gin.Context`, `OSS`, `BaseAgent`, `ToolProvider`, `App`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **Why does `RedisStreamTask` connect `RedisStreamTask` to `AgentService`, `Err`, `DefaultFileCleanupService`, `context.Context`, `StdioMCPClient`, `openai_llm.go`, `NewRedisStreamTask`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Why does `NewToolProvider()` connect `ToolProvider` to `AgentService`, `MCPTool`, `context.Context`, `Err`, `tools_test.go`, `SearchTool`?**
  _High betweenness centrality (0.008) - this node is a cross-community bridge._
- **What connects `SearchConfig`, `MCPServer`, `A2AAgent` to the rest of the system?**
  _667 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `index.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.0647307924984876 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.05030834144758196 - nodes in this community are weakly interconnected._
- **Should `AppException` be split into smaller, more focused modules?**
  _Cohesion score 0.09243697478991597 - nodes in this community are weakly interconnected._