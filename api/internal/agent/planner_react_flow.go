package agent

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// PlannerReActFlow 规划与执行流
type PlannerReActFlow struct {
	mu        sync.Mutex
	status    FlowStatus
	plan      *model.Plan
	sessionID string
	config    *AgentConfig
	llm       external.LLM
	tools     []Tool

	planner *PlannerAgent
	react   *ReActAgent
}

// NewPlannerReActFlow 创建规划与执行流
func NewPlannerReActFlow(
	sessionID string,
	config *AgentConfig,
	llm external.LLM,
	tools []Tool,
) *PlannerReActFlow {
	flow := &PlannerReActFlow{
		status:    FlowStatusIdle,
		plan:      nil,
		sessionID: sessionID,
		config:    config,
		llm:       llm,
		tools:     tools,
	}

	// 创建 Planner 和 ReAct Agent
	flow.planner = NewPlannerAgent(sessionID, config, llm, tools)
	flow.react = NewReActAgent(sessionID, config, llm, tools)

	return flow
}

// setStatus 在加锁状态下迁移流状态
func (f *PlannerReActFlow) setStatus(s FlowStatus) {
	f.mu.Lock()
	f.status = s
	f.mu.Unlock()
}

// emitEvent 将事件发送给下游；请求取消后立即停止，避免消费者退出时阻塞 flow goroutine。
func (f *PlannerReActFlow) emitEvent(ctx context.Context, ch chan<- model.BaseEvent, event model.BaseEvent) bool {
	select {
	case ch <- event:
		return true
	case <-ctx.Done():
		f.setStatus(FlowStatusCompleted)
		return false
	}
}

func (f *PlannerReActFlow) setPlan(plan *model.Plan) {
	f.mu.Lock()
	f.plan = plan
	f.mu.Unlock()
}

func (f *PlannerReActFlow) planSnapshot() *model.Plan {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.plan == nil {
		return nil
	}
	encoded, err := sonic.Marshal(f.plan)
	if err != nil {
		return nil
	}
	var snapshot model.Plan
	if err := sonic.Unmarshal(encoded, &snapshot); err != nil {
		return nil
	}
	return &snapshot
}

// Invoke 运行流，返回事件流
func (f *PlannerReActFlow) Invoke(ctx context.Context, input *TaskInput) <-chan model.BaseEvent {
	ch := make(chan model.BaseEvent, 100)

	// 注入事件通道，让 agent 层工具调用事件（tool_calling/tool_called）进入事件流。
	// planner 不调用工具，注入仅为统一；react 的工具调用发生在 Invoke 内的 handleToolCall。
	f.planner.SetEventCh(ch)
	f.react.SetEventCh(ch)

	go func() {
		defer close(ch)

		// 兜底 recover：任何 panic 不能杀死整个进程，转为 error 事件让前端正常结束
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "PlannerReActFlow panic，已恢复",
					logger.String("session_id", f.sessionID),
					logger.Any("panic", r),
					logger.String("stack", string(debug.Stack())))
				f.emitEvent(ctx, ch, model.NewErrorEvent(fmt.Sprintf("内部错误: %v", r)))
				f.setStatus(FlowStatusCompleted)
			}
		}()

		f.setStatus(FlowStatusPlanning)

		logger.InfoContext(ctx, "PlannerReActFlow 开始执行",
			logger.String("session_id", f.sessionID),
			logger.String("message", input.Message.ContentText))

		// 状态机循环：主循环只做调度，每个状态的处理逻辑在对应的 handleXxx 方法中
		for {
			status := f.currentStatus()
			var stop bool
			switch status {
			case FlowStatusIdle:
				stop = f.handleIdle()
			case FlowStatusPlanning:
				stop = f.handlePlanning(ctx, input, ch)
			case FlowStatusExecuting:
				stop = f.handleExecuting(ctx, input, ch)
			case FlowStatusWaiting:
				stop = f.handleWaiting()
			case FlowStatusUpdating:
				stop = f.handleUpdating(ctx, ch)
			case FlowStatusSummarizing:
				stop = f.handleSummarizing(ctx, ch)
			case FlowStatusCompleted:
				stop = f.handleCompleted(ctx, ch)
			default:
				logger.ErrorContext(ctx, "PlannerReActFlow 遇到未知状态，终止执行",
					logger.String("status", string(status)))
				_ = f.emitEvent(ctx, ch, model.NewErrorEvent("内部错误: 未知流状态"))
				return
			}
			if stop {
				return
			}
		}
	}()

	return ch
}

// currentStatus 在加锁状态下读取流状态
func (f *PlannerReActFlow) currentStatus() FlowStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

// 各状态处理器返回 true 表示流应终止（goroutine 退出）。

// handleIdle 空闲 -> 规划
func (f *PlannerReActFlow) handleIdle() bool {
	f.setStatus(FlowStatusPlanning)
	return false
}

// handlePlanning 调用 Planner 创建计划并发出 title/message/plan 事件 -> 执行
func (f *PlannerReActFlow) handlePlanning(ctx context.Context, input *TaskInput, ch chan<- model.BaseEvent) bool {
	plan, planMsg, err := f.planner.CreatePlan(ctx, input)
	if err != nil {
		logger.ErrorContext(ctx, "Planner 创建计划失败", logger.Err(err))
		if !f.emitEvent(ctx, ch, model.NewErrorEvent(err.Error())) {
			return true
		}
		f.setStatus(FlowStatusCompleted)
		return false
	}

	f.setPlan(plan)

	// 发送计划事件
	if !f.emitEvent(ctx, ch, model.NewTitleEvent(plan.Title)) {
		return true
	}
	// 与原项目对齐：把 plan.message 作为 assistant 消息发出。
	// 配合 json_object 模式 + 中文 prompt，planner 输出的 planMsg 已经是结构化中文，
	// 不会再泄露英文 CoT。
	if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", planMsg)) ||
		!f.emitEvent(ctx, ch, model.NewPlanEvent(*plan, model.PlanEventStatusCreated)) {
		return true
	}

	logger.InfoContext(ctx, "Planner 创建计划成功",
		logger.String("session_id", f.sessionID),
		logger.Int("steps", len(plan.Steps)))

	f.setStatus(FlowStatusExecuting)
	return false
}

// handleExecuting 取出下一个步骤交给 ReActAgent 执行 -> 更新（或等待用户输入）
func (f *PlannerReActFlow) handleExecuting(ctx context.Context, input *TaskInput, ch chan<- model.BaseEvent) bool {
	plan := f.planSnapshot()
	if plan == nil || len(plan.Steps) == 0 {
		logger.WarnContext(ctx, "计划为空，无法进入执行阶段",
			logger.String("session_id", f.sessionID))
		if !f.emitEvent(ctx, ch, model.NewErrorEvent("计划为空，无法执行")) {
			return true
		}
		f.setStatus(FlowStatusCompleted)
		return false
	}

	step := plan.GetNextStep()
	if step == nil {
		logger.InfoContext(ctx, "所有步骤已执行完毕，进入总结阶段")
		f.setStatus(FlowStatusSummarizing)
		return false
	}

	// 更新计划状态
	plan.Status = model.ExecutionStatusRunning
	if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusStarted)) {
		return true
	}

	// 执行步骤
	logger.InfoContext(ctx, "ReActAgent 开始执行步骤",
		logger.String("step_id", step.ID),
		logger.String("description", step.Description))

	if err := f.react.ExecuteStep(ctx, plan, step, input); err != nil {
		if err == ErrWaitForUser {
			// 需要等待用户输入：下一轮 Invoke 会从 waiting -> executing 继续当前计划
			logger.InfoContext(ctx, "ReActAgent 等待用户输入",
				logger.String("question", step.UserQuestion))
			if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", step.UserQuestion)) ||
				!f.emitEvent(ctx, ch, model.NewWaitEvent()) {
				return true
			}
			f.setStatus(FlowStatusWaiting)
			// 不压缩记忆，保留上下文
			return false
		}
		logger.ErrorContext(ctx, "ReActAgent 执行步骤失败", logger.Err(err))
		if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusFailed)) {
			return true
		}
		step.Status = model.ExecutionStatusFailed
		step.Error = err.Error()
	} else {
		if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusCompleted)) {
			return true
		}

		// 发送步骤结果消息
		if step.Result != "" {
			if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", step.Result)) {
				return true
			}
		}
	}
	// ExecuteStep 操作的是本轮快照，执行结果需要写回 flow，供后续更新计划和外部查询使用。
	f.setPlan(plan)

	// 压缩记忆
	if err := f.react.CompactMemory(); err != nil {
		logger.WarnContext(ctx, "压缩 Agent 记忆失败",
			logger.String("session_id", f.sessionID),
			logger.Err(err))
	}

	f.setStatus(FlowStatusUpdating)
	return false
}

// handleWaiting 用户新消息触发下一轮 Invoke 后，等待 -> 继续执行
func (f *PlannerReActFlow) handleWaiting() bool {
	f.setStatus(FlowStatusExecuting)
	return false
}

// handleUpdating 调用 Planner 根据最近完成的步骤更新计划 -> 执行
func (f *PlannerReActFlow) handleUpdating(ctx context.Context, ch chan<- model.BaseEvent) bool {
	plan := f.planSnapshot()
	if plan == nil {
		f.setStatus(FlowStatusCompleted)
		return false
	}
	step := plan.GetNextStep()
	if step == nil {
		// 没有未完成的步骤，进入总结阶段
		f.setStatus(FlowStatusSummarizing)
		return false
	}

	// 找到最近完成的步骤
	var completedStep *model.PlanStep
	for i := len(plan.Steps) - 1; i >= 0; i-- {
		if plan.Steps[i].Done() {
			completedStep = &plan.Steps[i]
			break
		}
	}

	if completedStep != nil {
		updatedPlan, err := f.planner.UpdatePlan(ctx, plan, completedStep)
		if err != nil {
			logger.WarnContext(ctx, "Planner 更新计划失败", logger.Err(err))
		} else {
			f.setPlan(updatedPlan)
			if !f.emitEvent(ctx, ch, model.NewPlanEvent(*updatedPlan, model.PlanEventStatusUpdated)) {
				return true
			}
		}
	}

	f.setStatus(FlowStatusExecuting)
	return false
}

// handleSummarizing 所有步骤完成后由 ReActAgent 产出总结 -> 完成
func (f *PlannerReActFlow) handleSummarizing(ctx context.Context, ch chan<- model.BaseEvent) bool {
	// 只有真正执行过步骤的计划才进入总结。
	// 空步骤计划如果流转到这里，说明上游已经出了结构化输出问题。
	plan := f.planSnapshot()
	if plan != nil && len(plan.Steps) > 0 {
		summary, attachments, err := f.react.Summarize(ctx)
		if err != nil {
			logger.WarnContext(ctx, "ReActAgent 总结任务失败", logger.Err(err))
		} else {
			if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", summary)) {
				return true
			}
			for _, att := range attachments {
				logger.InfoContext(ctx, "任务生成附件", logger.String("filepath", att))
			}
		}
	} else {
		logger.WarnContext(ctx, "跳过空步骤计划的总结阶段",
			logger.String("session_id", f.sessionID))
	}

	f.setStatus(FlowStatusCompleted)
	return false
}

// handleCompleted 标记计划完成并发出终态事件；该状态必定终止流
func (f *PlannerReActFlow) handleCompleted(ctx context.Context, ch chan<- model.BaseEvent) bool {
	if plan := f.planSnapshot(); plan != nil {
		plan.Status = model.ExecutionStatusCompleted
		f.setPlan(plan)
		if !f.emitEvent(ctx, ch, model.NewPlanEvent(*plan, model.PlanEventStatusCompleted)) {
			return true
		}
	}
	if !f.emitEvent(ctx, ch, model.NewDoneEvent()) {
		return true
	}
	logger.InfoContext(ctx, "PlannerReActFlow 执行完成",
		logger.String("session_id", f.sessionID))
	return true
}

// Done 返回流是否结束
func (f *PlannerReActFlow) Done() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status == FlowStatusCompleted || f.status == FlowStatusIdle
}

// GetStatus 返回当前状态
func (f *PlannerReActFlow) GetStatus() FlowStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

// GetPlan 返回当前计划
func (f *PlannerReActFlow) GetPlan() *model.Plan {
	return f.planSnapshot()
}
