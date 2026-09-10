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

		// 状态机循环
		for {
			f.mu.Lock()
			status := f.status
			f.mu.Unlock()

			switch status {
			case FlowStatusIdle:
				// 空闲状态 -> 规划状态
				f.setStatus(FlowStatusPlanning)

			case FlowStatusPlanning:
				// 规划状态 -> 调用 Planner 创建计划
				plan, planMsg, err := f.planner.CreatePlan(ctx, input)
				if err != nil {
					logger.ErrorContext(ctx, "Planner 创建计划失败", logger.Err(err))
					if !f.emitEvent(ctx, ch, model.NewErrorEvent(err.Error())) {
						return
					}
					f.setStatus(FlowStatusCompleted)
					break
				}

				f.setPlan(plan)

				// 发送计划事件
				if !f.emitEvent(ctx, ch, model.NewTitleEvent(plan.Title)) {
					return
				}
				// 与原项目对齐：把 plan.message 作为 assistant 消息发出。
				// 配合 json_object 模式 + 中文 prompt，planner 输出的 planMsg 已经是结构化中文，
				// 不会再泄露英文 CoT。
				if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", planMsg)) ||
					!f.emitEvent(ctx, ch, model.NewPlanEvent(*plan, model.PlanEventStatusCreated)) {
					return
				}

				logger.InfoContext(ctx, "Planner 创建计划成功",
					logger.String("session_id", f.sessionID),
					logger.Int("steps", len(plan.Steps)))

				f.setStatus(FlowStatusExecuting)

			case FlowStatusExecuting:
				// 执行状态 -> 获取下一个步骤并执行
				plan := f.planSnapshot()
				if plan == nil || len(plan.Steps) == 0 {
					logger.WarnContext(ctx, "计划为空，无法进入执行阶段",
						logger.String("session_id", f.sessionID))
					if !f.emitEvent(ctx, ch, model.NewErrorEvent("计划为空，无法执行")) {
						return
					}
					f.setStatus(FlowStatusCompleted)
					break
				}

				step := plan.GetNextStep()
				if step == nil {
					logger.InfoContext(ctx, "所有步骤已执行完毕，进入总结阶段")
					f.setStatus(FlowStatusSummarizing)
					break
				}

				// 更新计划状态
				plan.Status = model.ExecutionStatusRunning
				if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusStarted)) {
					return
				}

				// 执行步骤
				logger.InfoContext(ctx, "ReActAgent 开始执行步骤",
					logger.String("step_id", step.ID),
					logger.String("description", step.Description))

				if err := f.react.ExecuteStep(ctx, plan, step, input); err != nil {
					if err == ErrWaitForUser {
						// 需要等待用户输入
						logger.InfoContext(ctx, "ReActAgent 等待用户输入",
							logger.String("question", step.UserQuestion))
						if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", step.UserQuestion)) ||
							!f.emitEvent(ctx, ch, model.NewWaitEvent()) {
							return
						}
						f.setStatus(FlowStatusWaiting)
						// 不压缩记忆，保留上下文
						continue
					}
					logger.ErrorContext(ctx, "ReActAgent 执行步骤失败", logger.Err(err))
					if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusFailed)) {
						return
					}
					step.Status = model.ExecutionStatusFailed
					step.Error = err.Error()
				} else {
					if !f.emitEvent(ctx, ch, model.NewStepEvent(*step, model.StepEventStatusCompleted)) {
						return
					}

					// 发送步骤结果消息
					if step.Result != "" {
						if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", step.Result)) {
							return
						}
					}
				}
				// ExecuteStep 操作的是本轮快照，执行结果需要写回 flow，供后续更新计划和外部查询使用。
				f.setPlan(plan)

				// 压缩记忆
				_ = f.react.CompactMemory()

				f.setStatus(FlowStatusUpdating)

			case FlowStatusWaiting:
				// 等待状态 -> 等待用户输入后继续执行
				// 用户输入新消息后会再次触发 Invoke，继续执行当前计划
				f.setStatus(FlowStatusExecuting)
				// 继续到 FlowStatusExecuting 状态

			case FlowStatusUpdating:
				// 更新状态 -> 调用 Planner 更新计划
				plan := f.planSnapshot()
				if plan == nil {
					f.setStatus(FlowStatusCompleted)
					break
				}
				step := plan.GetNextStep()
				if step == nil {
					// 没有未完成的步骤，进入总结阶段
					f.setStatus(FlowStatusSummarizing)
					break
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
							return
						}
					}
				}

				f.setStatus(FlowStatusExecuting)

			case FlowStatusSummarizing:
				// 只有真正执行过步骤的计划才进入总结。
				// 空步骤计划如果流转到这里，说明上游已经出了结构化输出问题。
				plan := f.planSnapshot()
				if plan != nil && len(plan.Steps) > 0 {
					summary, attachments, err := f.react.Summarize(ctx)
					if err != nil {
						logger.WarnContext(ctx, "ReActAgent 总结任务失败", logger.Err(err))
					} else {
						if !f.emitEvent(ctx, ch, model.NewMessageEvent("assistant", summary)) {
							return
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

			case FlowStatusCompleted:
				// 完成状态 -> 发送完成事件
				if plan := f.planSnapshot(); plan != nil {
					plan.Status = model.ExecutionStatusCompleted
					f.setPlan(plan)
					if !f.emitEvent(ctx, ch, model.NewPlanEvent(*plan, model.PlanEventStatusCompleted)) {
						return
					}
				}
				if !f.emitEvent(ctx, ch, model.NewDoneEvent()) {
					return
				}
				logger.InfoContext(ctx, "PlannerReActFlow 执行完成",
					logger.String("session_id", f.sessionID))
				return
			}
		}
	}()

	return ch
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
