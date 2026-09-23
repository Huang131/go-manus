package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/google/uuid"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

const (
	// Pop 重试配置
	popRetryBaseDelay = 100 * time.Millisecond // 基础重试延迟
	popRetryMaxDelay  = 1 * time.Second        // 最大重试延迟
	popRetryMaxCount  = 10                     // 最大连续错误次数
)

// AgentTaskRunner 基于 Agent 智能体的任务运行器
// 对齐 Python 版本的 AgentTaskRunner。
//
// 运行器只关心"驱动 Flow 的事件循环"，会话持久化与文件/附件协作交由 SessionRuntime 承担，
// 避免把仓储、沙箱、对象存储等依赖平铺在结构体里。
type AgentTaskRunner struct {
	mu        sync.Mutex
	sessionID string
	config    *AgentConfig
	llm       llm.LLM
	tools     []toolspkg.Tool
	flow      *PlannerReActFlow
	runtime   *SessionRuntime
}

// AgentTaskRunnerConfig AgentTaskRunner 配置
type AgentTaskRunnerConfig struct {
	SessionID       string
	AgentConfig     *AgentConfig
	InitialMessages []llmcore.Message
	LLM             llm.LLM
	Tools           []toolspkg.Tool
	Runtime         *SessionRuntime
}

// NewAgentTaskRunner 创建任务运行器
func NewAgentTaskRunner(cfg *AgentTaskRunnerConfig) *AgentTaskRunner {
	runner := &AgentTaskRunner{
		sessionID: cfg.SessionID,
		config:    cfg.AgentConfig,
		llm:       cfg.LLM,
		tools:     cfg.Tools,
		runtime:   cfg.Runtime,
	}

	// 创建流程
	runner.flow = NewPlannerReActFlow(cfg.SessionID, cfg.AgentConfig, cfg.LLM, cfg.Tools)
	if len(cfg.InitialMessages) > 0 {
		runner.flow.planner.mergeMemory(context.Background(), cfg.InitialMessages)
		runner.flow.react.mergeMemory(context.Background(), cfg.InitialMessages)
	}

	return runner
}

// Invoke 实现 TaskRunner 接口
// 从 task.input_stream 获取事件，执行 Flow，结果写入 task.output_stream
// 对齐 Python: await self._task_runner.invoke(self)
func (r *AgentTaskRunner) Invoke(ctx context.Context, task *RedisStreamTask) error {
	r.mu.Lock()
	if r.flow == nil {
		r.flow = NewPlannerReActFlow(r.sessionID, r.config, r.llm, r.tools)
	}
	r.mu.Unlock()

	// 首次运行，更新会话状态为运行中
	if r.flow.GetPlan() == nil {
		if err := r.runtime.UpdateStatus(ctx, model.SessionStatusRunning); err != nil {
			logger.WarnContext(ctx, "更新会话运行状态失败",
				logger.String("session_id", r.sessionID),
				logger.Err(err))
		}
	}

	logger.InfoContext(ctx, "AgentTaskRunner 开始执行",
		logger.String("session_id", r.sessionID),
		logger.String("task_id", task.ID()))

	// 死循环修复：添加重试延迟和最大错误次数限制
	var popRetryCount int
	var popRetryDelay = popRetryBaseDelay

	// 从 input_stream 循环获取事件并处理
	for {
		// 检查任务是否已取消或完成
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "AgentTaskRunner 上下文取消，退出执行",
				logger.String("task_id", task.ID()))
			return ctx.Err()
		case <-task.DoneChan():
			logger.InfoContext(ctx, "AgentTaskRunner 任务完成，退出执行",
				logger.String("task_id", task.ID()))
			return nil
		default:
		}

		// 阻塞获取输入消息
		_, data, err := task.InputStream().Pop(ctx)
		if err != nil {
			// 检查是否是 context 取消
			if ctx.Err() != nil {
				logger.InfoContext(ctx, "AgentTaskRunner 上下文取消，退出执行",
					logger.String("task_id", task.ID()))
				return ctx.Err()
			}
			// 检查是否是任务完成信号
			if task.Done() {
				logger.InfoContext(ctx, "AgentTaskRunner 任务完成，退出执行",
					logger.String("task_id", task.ID()))
				return nil
			}

			// 死循环修复：累计错误次数，超过阈值则退出
			popRetryCount++
			if popRetryCount >= popRetryMaxCount {
				logger.ErrorContext(ctx, "获取输入消息连续失败次数过多，退出执行",
					logger.String("task_id", task.ID()),
					logger.Int("retry_count", popRetryCount),
					logger.Err(err))
				return fmt.Errorf("pop retry exceeded max count: %d", popRetryMaxCount)
			}

			// 指数退避延迟
			logger.WarnContext(ctx, "获取输入消息失败，等待重试",
				logger.String("task_id", task.ID()),
				logger.Int("retry_count", popRetryCount),
				logger.Dur("retry_delay", popRetryDelay),
				logger.Err(err))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(popRetryDelay):
				// 指数退避，最大延迟限制
				popRetryDelay = popRetryDelay * 2
				if popRetryDelay > popRetryMaxDelay {
					popRetryDelay = popRetryMaxDelay
				}
			}
			continue
		}

		// 重置错误计数
		popRetryCount = 0
		popRetryDelay = popRetryBaseDelay

		if data == "" {
			// 无消息，继续等待
			continue
		}

		// 解析事件
		var inputEvent model.MessageEvent
		if err := sonic.UnmarshalString(data, &inputEvent); err != nil {
			logger.WarnContext(ctx, "解析输入事件失败",
				logger.String("data", data),
				logger.Err(err))
			continue
		}

		attachments, err := r.runtime.SyncUserAttachmentsToSandbox(ctx, inputEvent.Attachments)
		if err != nil {
			logger.WarnContext(ctx, "同步用户附件失败",
				logger.String("session_id", r.sessionID),
				logger.Err(err))
		}

		// 加载附件内容到 LLM 上下文（解决"只列计划"问题）
		attachmentContexts := r.runtime.LoadAttachments(ctx, inputEvent.Attachments, inputEvent.Message)
		if len(attachmentContexts) > 0 {
			logger.InfoContext(ctx, "已加载附件内容到 LLM 上下文",
				logger.String("session_id", r.sessionID),
				logger.Int("count", len(attachmentContexts)))
		}

		// 转换为 Flow 需要的任务输入
		input := &TaskInput{
			Message: llmcore.Message{
				Role:        inputEvent.Role,
				ContentText: inputEvent.Message,
				Attachments: attachments,
			},
			AttachmentContexts: attachmentContexts,
		}

		// 运行 Flow
		eventChan := r.flow.Invoke(ctx, input)

		// 处理 Flow 输出事件
		for event := range eventChan {
			if ctx.Err() != nil {
				break
			}
			// 工具产出的二进制展示数据（如浏览器截图）必须先落存储：
			// 这里是 SSE 与事件库的唯一汇合点，序列化前把字节换成文件引用，
			// 事件里才不会出现撑爆体积的 base64。
			if called, ok := event.(*model.ToolCalledEvent); ok && r.runtime != nil {
				r.runtime.StoreToolArtifacts(ctx, called.Result)
			}

			// 业务事件先序列化为 payload
			eventJSON, err := sonic.Marshal(event)
			if err != nil {
				logger.ErrorContext(ctx, "序列化事件失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
				continue
			}

			// 生成事件元数据：event_id 与 created_at 在创建时确定，
			// 平铺进业务 payload（对齐 Python 版 BaseEventData 结构），
			// Redis 流与 DB 各存一份，SSE 直传无需再解析重组。
			eventID := uuid.New().String()
			eventCreatedAt := time.Now()

			// 业务事件序列化为 payload 并注入元数据
			var payload map[string]interface{}
			if err := sonic.Unmarshal(eventJSON, &payload); err == nil {
				payload[model.EventMetadataKeyEventID] = eventID
				payload[model.EventMetadataKeyCreatedAt] = eventCreatedAt.Unix()
				eventJSON, err = sonic.Marshal(payload)
				if err != nil {
					logger.ErrorContext(ctx, "序列化事件 payload 失败",
						logger.String("task_id", task.ID()),
						logger.Err(err))
					continue
				}
			}
			// payload 不是对象时（罕见），保留原始 payload，仅元数据落在信封上

			// 同时包一层 model.Event（与 DB 一致），并写入 Redis Stream，
			// 否则 GetOutput 反序列化得到的是业务 payload，event.Type 永远是空，
			// SSE 推送时没有 event:xxx 业务类型行，前端 lastEventIdRef 也拿不到。
			baseEvent := &model.Event{
				Type:      event.GetType(),
				CreatedAt: eventCreatedAt,
				Data:      eventJSON,
			}
			wrappedJSON, err := sonic.Marshal(baseEvent)
			if err != nil {
				logger.ErrorContext(ctx, "序列化 model.Event 失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
				continue
			}

			outputID, err := task.OutputStream().Put(ctx, string(wrappedJSON))
			if err != nil {
				logger.WarnContext(ctx, "写入 output_stream 失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
			} else if outputID != "" {
				// Redis Stream ID 是 SSE 续读游标；业务 UUID 已保留在 payload.event_id。
				// DB 历史事件也保存同一游标，页面刷新后可直接从该位置继续读取。
				baseEvent.ID = outputID
			}

			// 同步到会话数据库。
			// 逐 token 的 delta 事件只进 Redis（保 SSE 续读），不落库：
			// 长回复每秒数百个增量会让 DB 写入放大，且最终内容已由
			// MessageDoneEvent 作为权威记录落库。
			if event.GetType() == model.EventTypeMessageDelta || event.GetType() == model.EventTypeShellOutput {
				logger.DebugContext(ctx, "跳过 delta 事件落库",
					logger.String("task_id", task.ID()),
					logger.String("event_id", eventID))
			} else if err := r.runtime.AppendEvent(ctx, baseEvent); err != nil {
				logger.WarnContext(ctx, "添加事件到会话失败",
					logger.String("session_id", r.sessionID),
					logger.Err(err))
			}

			// 处理不同类型的事件
			switch e := event.(type) {
			case *model.PlanEvent, *model.DoneEvent:
				// 会话状态统一在事件循环结束后的收尾 switch 里，由 flow 状态投影回写，
				// 不在 PlanEvent/DoneEvent 两处重复判断 success/failed 语义。
			case *model.ErrorEvent:
				logger.ErrorContext(ctx, "Agent 运行出错", logger.String("error", e.Message))
			case *model.StepEvent:
				if e.Status == model.StepEventStatusCompleted && e.Step.Success {
					// 步骤完成，同步附件文件
					for _, filePath := range e.Step.Attachments {
						_ = r.runtime.SyncFileToStorage(ctx, filePath)
					}
				}
			}

			logger.DebugContext(ctx, "AgentTaskRunner 输出事件",
				logger.String("task_id", task.ID()),
				logger.String("event_id", outputID),
				logger.String("event_type", string(event.GetType())))
		}

		// Flow goroutine 已退出（事件通道关闭）。按流状态决定 runner 去向：
		//   - Waiting：上一轮在等用户输入，继续留在循环里 Pop 下一条消息（task 保持活跃）
		//   - Completed/Failed/Idle：本轮任务已到终态，退出 runner 触发 task 完成清理链
		//     （onDone -> destroy -> registry 摘除 -> 短 TTL），否则 task 永远不算完成
		status := r.flow.GetStatus()
		if ctx.Err() != nil || status == FlowStatusCancelled {
			logger.InfoContext(ctx, "Flow 已取消，runner 退出",
				logger.String("task_id", task.ID()))
			return ctx.Err()
		}
		switch status {
		case FlowStatusWaiting:
			logger.InfoContext(ctx, "Flow 等待用户输入，runner 继续监听输入流",
				logger.String("task_id", task.ID()))
			if err := r.runtime.UpdateStatus(ctx, model.SessionStatusWaiting); err != nil {
				logger.WarnContext(ctx, "更新会话等待状态失败",
					logger.String("session_id", r.sessionID),
					logger.Err(err))
			}
		default:
			// 按流状态与计划结果投影会话终态（completed/failed），这是会话状态的唯一回写点。
			if err := r.runtime.UpdateStatus(ctx, status.ToSessionStatus(r.flow.GetPlan())); err != nil {
				logger.WarnContext(ctx, "更新会话终态失败",
					logger.String("session_id", r.sessionID),
					logger.Err(err))
			}
			logger.InfoContext(ctx, "Flow 已到终态，runner 退出",
				logger.String("task_id", task.ID()),
				logger.String("flow_status", string(status)))
			return nil
		}
	}
}

// Destroy 实现 TaskRunner 接口
// 销毁运行器，释放资源
func (r *AgentTaskRunner) Destroy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.flow != nil {
		// 清理 flow 资源
		r.flow = nil
	}

	logger.Info("AgentTaskRunner 已销毁",
		logger.String("session_id", r.sessionID))

	return nil
}

// OnDone 实现 TaskRunner 接口
// 任务完成时的回调
func (r *AgentTaskRunner) OnDone(task *RedisStreamTask) {
	logger.Info("AgentTaskRunner 任务完成回调",
		logger.String("task_id", task.ID()),
		logger.String("session_id", r.sessionID))
}

// Done 返回任务是否完成
func (r *AgentTaskRunner) Done() bool {
	if r.flow == nil {
		return true
	}
	return r.flow.Done()
}

// GetStatus 返回当前状态
func (r *AgentTaskRunner) GetStatus() FlowStatus {
	if r.flow == nil {
		return FlowStatusIdle
	}
	return r.flow.GetStatus()
}

// GetPlan 返回当前计划
func (r *AgentTaskRunner) GetPlan() *model.Plan {
	if r.flow == nil {
		return nil
	}
	return r.flow.GetPlan()
}
