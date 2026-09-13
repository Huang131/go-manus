
---

# 增量审查 2026-09-12（流式接线 + service 层专项）

基线：`4a5f6a2`（P0 修复）、`4892225`（端到端 token streaming）、`b1f3827`（sandbox 长命令修复）之后。

## A. 流式链路复查（commit 4892225）

三颗雷验证：15s 流超时 ✅已修（openai_llm.go:366）；MergeDeltas tool_use ✅已修（normalize.go:94）；**Anthropic 流式 usage 漏 message_start ❌未修**（anthropic_llm.go:372，PromptTokens 恒 0、成本恒 0）。

接线后游标闭环验证成立：SSE `id:` ↔ lastEventId ↔ readTaskOutput 归一 + ValidStreamID ↔ XREAD 严格 after。

新 P1：
1. **整条流零超时**：拆掉 15s 后 http.Client 无 Timeout、task ctx 仅 WithCancel → 上游挂起任务永久卡 Running（openai_llm.go:64 / anthropic_llm.go:137）
2. **token 级写放大**：每 delta → flow 通道 → marshal → Redis XADD → DB AppendEvent 逐条落盘（task_runner.go:286），MAXLEN 10000 可能裁早期事件
3. **ctx 取消产生半截完整回复**：range 后不查 ctx.Err()，半截内容被当完整回复合并进记忆/DB（base.go:249）

P2：SSE 30min 滚动续期误报 stream_error；Summarize 原始 JSON 增量泄漏且与最终 message 不一致；流错误后孤儿增量（换 messageID 重发致前端重复半截）；路由无流式候选不回退 Invoke；Anthropic 建连错误文案按 hasTools 误标；空游标约定 `$` vs `0` 不一致。

## B. service 层专项（新发现）

P1（资源收口）：
1. AgentService.Shutdown 不调 mcpTool.Cleanup()/a2aTool.Cleanup()（全项目零调用）也不等 runner 退出 → MCP 子进程泄漏、停机序错乱（agent/service.go:216）
2. DeleteSession 软删不停活跃 task → 已删会话继续烧 LLM/沙箱，事件写入静默失败（session_service.go:107）
3. DeleteFile 先删 storage 后删 DB → 反向僵尸（对象没了记录在），下载必 500（file_service.go:118）

P2：UnsetDefault 不进 defaultMu；Delete 事务外读过期 IsDefault；Create 显式 IsDefault 撞唯一索引被误报；file_cleanup.Stop 无条件等 5 分钟；batchSize=0/负数零校验；UploadFile 不校验 session → 伪造 session 文件永不被清理；Chat 双写 latest_message；错误语义（500 应为 404/Conflict）；用户消息 AppendEvent 失败仅 Warn；死代码（ValidateExpireDuration、CleanExpiredFiles 便捷方法、SessionService.AppendEvent 接口方法、AgentService configRep/mcpConfig/a2aConfig 只写不读）；status_service 串行检查且未配置依赖与故障混级。

## C. 昨日 backlog 复核（HEAD b1f3827 仍开放）

- Anthropic 工厂分支不透传 RequestPolicy/CostPolicy/ToolCallTimeout（llm.go）
- routed plan() 不把 env fallback 追加进候选链
- 事务回滚复用业务 ctx（transaction.go:50）→ ctx 已取消时 Rollback 失败事务悬挂
- 流式 usage（见 A）

## D. 修复批次

- 批次 1（本日）：流式三硬伤、资源收口三 P1、昨日四项、错误语义快速批
- 批次 2（待办）：status 并行化、default 竞态全量、delta 合并 flush、P2 死代码清扫

## E. 批次 1 修复记录（2026-09-12，全部经全量测试验证）

### 流式三硬伤
1. **流整体截止**：双 adapter Stream 接入 `streamContext(ctx)`（`llmStreamOverallTimeout = 10min`，llm.go）。cancel 所有权归 reader goroutine（`readXStream(..., cancel)`），修复过程中踩过一次"Stream 返回即 defer cancel"导致流被秒断的坑，已由 TestXxxClient_StreamProducesDeltas 守护
2. **delta 写放大**：task_runner 对 `EventTypeMessageDelta` 只写 Redis、跳过 DB AppendEvent；权威内容仍由 MessageDoneEvent 落库
3. **半截回复**：`invokeLLMWithEmission` 在 range 结束后检查 `ctx.Err()`，取消/超时的半截内容不再被 MergeDeltas 当完整回复合并进记忆

### 资源收口
4. **AgentService.Shutdown**：补 `mcpTool.Cleanup()`/`a2aTool.Cleanup()`；Cancel 后按 `taskShutdownTimeout`(10s) 上限等待各任务退出，避免 stopHook 关库后 runner 仍在写
5. **删会话先停任务**：handler Delete 在软删前调用 `agent.StopSession`（无活跃任务仅告警）
6. **DeleteFile 调序**：先删 DB 记录再删存储对象（孤儿对象仅存储成本泄漏，优于反向僵尸 404）

### 昨日 backlog 四项
7. **Anthropic 工厂 policy**：defaultLLMClientFactory Anthropic 分支补透传 RequestPolicy/CostPolicy/ToolCallTimeout（与 OpenAI 分支对齐）
8. **事务回滚**：rollbackTx 改用 `context.WithoutCancel(ctx)`，业务 ctx 取消不再导致 Rollback 失败、事务悬挂
9. **model_id 路由**：routed plan() 中 ctx 指定 model_id 强制作为 primary（按 Profile.ID 匹配，不参与健康排序），"临时切模型"语义恢复
10. **流式 usage**：anthropic 流解析 message_start 的 input_tokens，message_delta 组装完整 usage（PromptTokens 不再恒 0）

### 错误语义快速批
11. Chat 会话不存在 → `apperr.NotFound`（404）；"task is stopping" → `apperr.Conflict`（409）；删除 Chat 里与 AppendEvent 重复的 UpdateLatestMessage 双写；UnsetDefault 补 defaultMu 锁

### 设计决策已定（2026-09-12，Auto 功能落地）
- env fallback 追加进 plan 尾部：与既有测试固化的语义冲突（fallback 仅在 catalog 为空时使用，且不跨协议回退），已回退。若要"DB 模型全挂后兜底到 env 配置"，需先在测试层面确认设计意图（建议：同协议且 capability 兼容时追加，或提供显式配置开关）。
- **已按业界语义（Cursor/Copilot 调研）落地 Auto 路由**：Auto（无 model_id）时 env fallback 追加 plan 末尾（仅配置过时）；指定 model_id 时粘性路由（单元素 plan、失败显式报错 ErrModelNotAvailable，Chat 层同步预检 404）；前端选择器新增「Auto · 跟随系统」且为默认选项。新增 4 个路由测试固化语义。


---

# 增量审查 2026-09-13（三服务全量复审）

三路并行复审 api/ui/sandbox 当前基线。**修复核验**：sandbox 上轮 20 项 17 项正确落地（超预期：`..` 拒绝、sudo rm 参数数组、流式读文件、路径锁+原子替换、49+7 测试）；ui 上轮 P0 三项确认彻底修复；api 14 项 backlog 修 7。

## 本轮新发现（已按批次修复）

**P0**：api `startShellWatch` 向已关闭事件通道发送（任务正常完成路径必触发，可致进程崩溃）→ `StopShellWatch()` + flow 收尾调用 + emitEvent recover 兜底。
**P1**：① shell_output 落库写放大（只进 Redis，同 delta）② ui shell_output 发现循环漏扫 step.tools（实时 console 主路径失效）③ ui 首页 handleSend 丢弃 modelId（payload 编码进 URL，详情页 init 链路透传）④ sandbox read-shell-output 全量重算无缓存（增加 prefix 增量清洗缓存）⑤ watcher 轮询触发 wait-process 400 → ERROR 日志洪水（AppException 按 status_code 分级日志）⑥ sudo 派生 root 子进程 killpg EPERM 被误判组已死（EPERM=存在）。
**P2 已修**：MCP client 串行化锁（nextID/共享 bufio 竞争）、ErrSessionNotFound → 404（哨兵移入 apperr）、GetVNCURL 校验会话存在、upload filename None 守卫、sudo search 正则移出事件循环、ui Auto 静默失败 toast、重命名失败保留弹窗、Auto 重试不清草稿。

## 待办 backlog（第三批，按优先级）

### api
- [ ] DSN 用 url.UserPassword 构造（特殊字符密码连接失败）
- [ ] app_config MCP/A2A/LLM 合并事务化（读改写并发丢失）
- [ ] persistHealth 异步化/去抖（每次调用同步写 DB）
- [ ] GetAllSessions 加 LIMIT（SSE 每 5s 全量拉取）
- [ ] BrowserClient JS 转义（LLM 可控字符串拼进模板）
- [x] 内容级 sha256 去重 ✓2026-09-13（migration 005 加列+(session_id,sha256) 唯一索引封死 TOCTOU；流式 TeeReader 哈希；004 占位模型移除，env 种子正式生效）
- [ ] DownloadFile io.ReadAll 加大小上限
- [ ] RoutedLLM 共享 fallback Health 数据竞争（applyStoredHealth 写共享 cfg）

### ui
- [ ] manus-settings.tsx 拆分（895 行，含 97 行死代码 LLMSetting + A2A 文案复制自 MCP）
- [ ] timeline O(n²) 增量化（message_delta/console 高频事件下全量重算）
- [ ] fetch.ts 截断残包不再当完整事件解析（避免假 stream_error）
- [ ] FileInfo/SessionFile 类型合并；死代码清扫（createSSEConnection/getSession/clearUnreadMessageCount）
- [ ] refresh() 不 bump streamRetrySignal；prevToolCountRef 切会话重置
- [ ] appendEvent step 状态读嵌套 step.status（重连空流状态机不翻转）
- [x] 持久化 model id 不在列表时回落 Auto ✓2026-09-13（chat-input 校验）
- [ ] tool-preview-panel 关闭图标 Maximize2 → X

### sandbox
- [ ] 运行中会话不回收（TTL 仅覆盖已退出进程）；console_records 总预算
- [ ] sudo search 正则 ReDoS 限制
- [ ] supervisor stop-all 仍停止 app 自身；restart 无总超时
- [ ] read-shell-output 增量 delta 协议（替代全量快照传输）
- [ ] endpoint 测试扩展（wait/read/kill/会话复用/读取器竞态）


---

# 增量修复 2026-09-13（内容级去重 + 测试验证）

- migration 005：files.sha256 列 + (session_id, sha256) 部分唯一索引（TOCTOU 收口）
- UploadFile 流式 TeeReader 计算 sha256；同会话同内容复用既有记录（含并发唯一索引兜底）
- 004 占位 GPT-4 INSERT 移除：env 种子（seedDefaultModelFromEnv）成为初始模型唯一来源，开发库已验证生效
- sandbox 测试套件容器内跑通：56 passed（运行方式见上）


---

# 增量修复 2026-09-13（第四批小项）

- fetch.ts：外部 signal 与超时 controller 合并（AbortSignal 联动，外部中止不再被覆盖）
- fetch.ts：skipErrorHandler 只返回信封 data 字段（修类型陷阱，调用方拿到的才是 T）
- session-events.ts：assistant message 按 message_id 去重（重连重放/delta+message 并发不再渲染两条）
- sandbox：删除无锁死代码 get_console_records（调用方均走 _unlocked 版本）

仍开放（维持原优先级）：timeline O(n²) 增量化、manus-settings 多文件拆分、SSE_STREAM_END 魔法字符串类型化、sandbox 增量 delta 协议、tests/ 打进镜像。

# 增量修复 2026-09-13（第五批：终审发现项）

- ui P1：page.tsx initialModelId 透传补齐（首页模型选择此前静默失效）
- ui P2：A2A/MCP 两区描述文案互换归位；脏 model id 校验（chat-input 回落 Auto）
- sandbox P1：清洗缓存 ANSI 跨界残留修复（_clean_pending 待并缓冲 + _TRAILING_PARTIAL_ESC 边界检测），56 测试复跑通过
- sandbox：restart 8s 总超时重新落地（重构中丢失）；console 预算改按 UTF-8 字节计；drain 超时 500→4xx
- sandbox 测试运行方式：uv pip install --python /venv/bin/python pytest（生产镜像未含 pytest/tests，已记入待办）
