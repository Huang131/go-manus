# 代码审查报告 2026-09-11

- 审查范围：`api/internal`（agent/task 链路、external/llmcore、repository/service/bootstrap/config、pkg）+ 全局死代码扫描
- 基线提交：`fe5f9b6`（refactor(api): 代码 review 重构）之后的工作区
- 方法：三路并行深度阅读（agent/task 链路、external/llmcore 适配层、repository/service/pkg），发现均带 file:line，修复时以当时代码为准
- 上一轮（2026-09-10）已覆盖 handler/model 包，本轮不重复

## 总体结论

架构分层（handler→service→repository、llmcore 协议层）健康，非流式主链路质量尚可；但：

1. **task 状态机有 2 个 P0 缺陷**，直接导致多轮对话（ask_user）不可用与 zombie task 堆积；
2. **上次引入的 token streaming 是无生产消费者的半成品**，且自带 3 处会在接线当天爆雷的缺陷；
3. 其余集中在"半收敛"状态（新抽象留下未接线的旧代码、假互斥）与一批可控 P2。

---

## P0（必须先修）

### P0-1 Zombie task：flow 完成后 runner 永不退出
- 位置：`internal/agent/task_runner.go`（外层循环只看 ctx/DoneChan）+ `internal/agent/planner_react_flow.go`（Completed 分支）
- 现象：flow 发出 Done 事件后 runner 不退出，也没有终止信号进 input stream → task 永远不算 Done，registry、taskBySession、24h TTL 的 Redis 流全部滞留
- 修复方向：flow 进入 Completed 后 runner return，触发 onDone → destroy → Unregister → 短 TTL 清理链

### P0-2 Waiting 状态形同虚设：ask_user 多轮对话全链路失效
- 位置：`internal/agent/planner_react_flow.go` handleWaiting / ErrWaitForUser 分支
- 现象：handleWaiting 无条件转回 Executing → GetNextStep 取到同一个未完成步骤、用**旧 input 重跑**；用户真正的下一条消息到达时 flow 已 Completed，被 handleCompleted 吞掉只重发 Done。循环无上限，模型反复 ask_user 会无限烧 LLM
- 关联缺陷：ErrWaitForUser 分支提前 return，跳过 setPlan(plan)，UserQuestion/Status=Running 快照全丢
- 修复方向：wait 路径终止本轮 flow goroutine（记忆已保留），下轮 Invoke 从 Waiting 恢复并把新消息作为 input 继续

---

## P1（生产路径实质缺陷）

### LLM 调用链
| # | 问题 | 位置 |
|---|---|---|
| 1 | Anthropic 工厂分支不透传 RequestPolicy/CostPolicy/ToolCallTimeout → DB 配的 Claude 丢 reasoning 模式、成本恒 0 | `external/llm.go` defaultLLMClientFactory |
| 2 | routed plan() 丢弃 env 兜底配置 → DB 模型失败后永不 fallback；ctx 指定 model_id 被健康分更高的模型顶掉，"临时切模型"失效 | `external/routed_llm.go` plan() |
| 3 | OpenAI client 无超时，无 tools 的 Invoke 完全依赖调用方 ctx（task ctx 无 deadline）→ 上游挂起挂住整个会话 | `external/openai_llm.go` |
| 4 | Streaming 半成品：`Stream/LLMDelta/MergeDeltas` 生产零消费者；`toolCallTimeout`(15s) 套整条流；MergeDeltas 只认 `tool_calls` 而 Anthropic 是 `tool_use`；流式 usage 漏 `message_start`(input_tokens)；Anthropic SSE `event: error` 未处理 | `external/openai_llm.go`、`anthropic_llm.go`、`llmcore/normalize.go` |
| 5 | Anthropic extended thinking 的 thinking/signature 块不回传 → 启用思考 + 多轮 tool call 直接 400 | `external/anthropic_llm.go` |

### Agent / 记忆 / 任务
| # | 问题 | 位置 |
|---|---|---|
| 6 | CompactMemory 固定保留 10 条，可能切断 assistant(tool_calls) 与配对 tool 结果，adapter 无孤儿清洗 → 下一轮被上游 400 | `agent/base.go` + `agent/memory.go` |
| 7 | task 完成即 registry.Unregister → 留的 30 分钟 SSE 续读窗口 API 层不可达（GetTaskEvents 报 task not found） | `agent/task_redis.go` destroyRunner vs `agent/service.go` |
| 8 | 只有 PlanEvent=completed 更新 session 状态；flow ErrorEvent / pop 重试超限退出 → session 永远卡 Running | `agent/task_runner.go` |
| 9 | Invoke 重试注入的 `assistant("")+user("请继续")` 成功后随 mergeMemory 永久入记忆 | `agent/base.go` |
| 10 | 无 consumer group/XACK、游标在进程内存 → 重启后在途消息全丢、session 永久 Running（至少需要启动期僵尸清理） | `agent/task_redis.go` |
| 11 | 单条毒消息让游标不推进 → 10 次重读后 runner 整个退出（游标推进应放在数据转换之前） | `agent/task_redis.go` Pop |
| 12 | PutInput 失败时用户事件已入库 → 前端重试造成 DB 事件重复 | `agent/service.go` Chat |

### 基础设施 / 外部进程
| # | 问题 | 位置 |
|---|---|---|
| 13 | MCP client：裸 nextID + 共享 bufio.Reader 无锁（并发数据竞争/JSON 分帧破坏）；readResponse 超时弃 goroutine 继续吃行、不按 ID 匹配 → 一次超时后响应整体错位 | `external/mcp_client.go` |
| 14 | BrowserClient 用 fmt.Sprintf 拼 Playwright JS 再塞 shell：URL/输入未转义（注入 + 引号必炸） | `external/sandbox.go` BrowserClient |
| 15 | a2a Invoke 全程持 RLock（最长 10 分钟）阻塞 Initialize/Cleanup | `external/a2a_client.go` |
| 16 | 事务回滚复用业务 ctx：fn 失败常因 ctx 已取消 → Rollback 立即失败、事务悬挂 | `repository/transaction.go` |

### 错误处理 / 清理链
| # | 问题 | 位置 |
|---|---|---|
| 17 | ErrSessionNotFound 全仓无映射 → 删除竞态/软删后 AppendEvent 一律 500 | `repository/session_repository.go` + service 层 |
| 18 | GetVNCURL 不查 session==nil（repo 对不存在返回 nil,nil），"校验存在性"失效 | `service/session_service.go` |
| 19 | DSN 用 Sprintf 拼 user/password，密码含 `@ : / % #` 即解析错乱 | `config/config.go` DSN() |
| 20 | 对象存储删除成功但 DB 删除失败 → 僵尸记录永久 404 | `service/file_cleanup_service.go` |

---

## P2（主题归纳，约 30 项）

- **假互斥/读改写**：llm_model_service 的 defaultMu 只护 SetDefault 一条路径；MCP/A2A/LLM 配置合并无锁无事务互相覆盖且 map 迭代序漂移；file_cleanup 临时实例绕过 cleanupMu
- **死代码（生产零调用）**：NormalizeResponse、MergeDeltas（流式接线前）、SubscribeOutput、DynamicLLM 两构造器、四个 NewXxxRepositoryWithTx、PlanSchema 全家、doneChans、DefaultTaskRegistry.Clear/CleanupCompleted、planner 的 AddMemory（写入永不读取）、retryInterval 常量、ValidateExpireDuration、runtimeHealthToJSON、file_cleanup 的 CleanSessionFiles/GetCleanupStats 等
- **性能**：planSnapshot 每次状态转换 marshal 深拷贝整个 Plan；persistHealth 每次调用同步写 DB；GetAllSessions 无 LIMIT；GetOutput 每次 XREAD 只取 1 条；事件三重编解码；logger.WithContext 每条日志新建子 logger
- **装配/运维**：Shutdown stopHooks 无超时上限、二次 SIGINT 无法强退；WriteTimeoutSec 非 0 静默打断 SSE；viper AutomaticEnv 对 yaml 中不存在的键不生效；EnablePostgres=false 时 initServices 仍建仓储（nil panic）；healthCheck 注释过期
- **正确性小项**：apperr.Is 仅按 Kind（哨兵误判）、ToInternal 用断言非 errors.As（双重包装）、/health 透出底层错误细节、X-Request-ID 不校验、temperature 越界静默改 0.7、GetDefaultForAgent 吞 DB 错误、Update 冗余 SetDefault、runtime health 先读后写非原子、DownloadFile nil reader 无错误信号、SaveMemory 的 text[] 拼接对特殊字符报错、file cleanup "7d" 文档与 ParseDuration 打架、孤儿文件（session 硬删）永不清理、anthropic Temperature omitempty 吞显式 0、流式未带 stream_options.include_usage、usage-only delta 被跳过、Stream 非 200 未包 ProviderError、JSON-RPC Code 类型错误、Screenshot 返回错误字段、RecordFailure 不看 ErrorKind、routed_llm fallback 只允许触发一次、llm_model List 未收敛 collectRows、事务抽象无单测

---

## 修复顺序建议

1. P0-1 / P0-2（状态机，多轮对话可用性）
2. LLM 链 #1/#2/#3（每次调用受益）
3. Streaming #4：**接线或移除二选一**（移除最省事，git 可恢复；本报告默认先修其内部三颗雷、保留代码待决策）
4. #16 事务 ctx、#13 MCP 并发、#14 BrowserClient 下线/参数化
5. #17/#18/#7/#8 错误映射与清理链
6. #6/#9 记忆正确性，P2 批量清扫

## 本轮已修复记录

（修复过程中回填）
