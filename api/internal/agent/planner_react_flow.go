package agent

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"

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

// LoadMemory 从数据库恢复 planner/react 两个 Agent 的记忆（flow 生命周期内只应调用一次）
func (f *PlannerReActFlow) LoadMemory(ctx context.Context) error {
	if err := f.planner.LoadMemory(ctx); err != nil {
		return fmt.Errorf("加载 planner 记忆失败: %w", err)
	}
	if err := f.react.LoadMemory(ctx); err != nil {
		return fmt.Errorf("加载 react 记忆失败: %w", err)
	}
	return nil
}

// Invoke 运行流，返回事件流
func (f *PlannerReActFlow) Invoke(ctx context.Context, input *TaskInput) <-chan model.BaseEvent {
	ch := make(chan model.BaseEvent, 100)

	go func() {
		defer close(ch)

		// 兜底 recover：任何 panic 不能杀死整个进程，转为 error 事件让前端正常结束
		defer func() {
			if r := recover(); r != nil {
				logger.Error("PlannerReActFlow panic，已恢复",
					logger.String("session_id", f.sessionID),
					logger.Any("panic", r),
					logger.String("stack", string(debug.Stack())))
				ch <- model.NewErrorEvent(fmt.Sprintf("内部错误: %v", r))
				f.mu.Lock()
				f.status = FlowStatusCompleted
				f.mu.Unlock()
			}
		}()

		f.mu.Lock()
		f.status = FlowStatusPlanning
		f.mu.Unlock()

		logger.Info("PlannerReActFlow 开始执行",
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
				f.mu.Lock()
				f.status = FlowStatusPlanning
				f.mu.Unlock()

			case FlowStatusPlanning:
				// 规划状态 -> 调用 Planner 创建计划
				plan, planMsg, err := f.planner.CreatePlan(ctx, input)
				if err != nil {
					logger.Error("Planner 创建计划失败", logger.Err(err))
					ch <- model.NewErrorEvent(err.Error())
					f.mu.Lock()
					f.status = FlowStatusCompleted
					f.mu.Unlock()
					break
				}

				f.plan = plan

				// 发送计划事件
				ch <- model.NewTitleEvent(plan.Title)
				// 与原项目对齐：把 plan.message 作为 assistant 消息发出。
				// 配合 json_object 模式 + 中文 prompt，planner 输出的 planMsg 已经是结构化中文，
				// 不会再泄露英文 CoT。
				ch <- model.NewMessageEvent("assistant", planMsg)
				ch <- model.NewPlanEvent(plan, model.PlanEventStatusCreated)

				logger.Info("Planner 创建计划成功",
					logger.String("session_id", f.sessionID),
					logger.Int("steps", len(plan.Steps)))

				f.mu.Lock()
				f.status = FlowStatusExecuting
				f.mu.Unlock()

			case FlowStatusExecuting:
				// 执行状态 -> 获取下一个步骤并执行
				if f.plan == nil || len(f.plan.Steps) == 0 {
					logger.Warn("计划为空，无法进入执行阶段",
						logger.String("session_id", f.sessionID))
					ch <- model.NewErrorEvent("计划为空，无法执行")
					f.mu.Lock()
					f.status = FlowStatusCompleted
					f.mu.Unlock()
					break
				}

				step := f.plan.GetNextStep()
				if step == nil {
					logger.Info("所有步骤已执行完毕，进入总结阶段")
					f.mu.Lock()
					f.status = FlowStatusSummarizing
					f.mu.Unlock()
					break
				}

				// 更新计划状态
				f.plan.Status = model.ExecutionStatusRunning
				ch <- model.NewStepEvent(step, model.StepEventStatusStarted)

				// 执行步骤
				logger.Info("ReActAgent 开始执行步骤",
					logger.String("step_id", step.ID),
					logger.String("description", step.Description))

				if err := f.react.ExecuteStep(ctx, f.plan, step, input); err != nil {
					if err == ErrWaitForUser {
						// 需要等待用户输入
						logger.Info("ReActAgent 等待用户输入",
							logger.String("question", step.UserQuestion))
						ch <- model.NewMessageEvent("assistant", step.UserQuestion)
						ch <- model.NewWaitEvent()
						f.mu.Lock()
						f.status = FlowStatusWaiting
						f.mu.Unlock()
						// 不压缩记忆，保留上下文
						continue
					}
					logger.Error("ReActAgent 执行步骤失败", logger.Err(err))
					ch <- model.NewStepEvent(step, model.StepEventStatusFailed)
					step.Status = model.ExecutionStatusFailed
					step.Error = err.Error()
				} else {
					ch <- model.NewStepEvent(step, model.StepEventStatusCompleted)

					// 发送步骤结果消息
					if step.Result != "" {
						ch <- model.NewMessageEvent("assistant", step.Result)
					}
				}

				// 压缩记忆
				_ = f.react.CompactMemory()

				f.mu.Lock()
				f.status = FlowStatusUpdating
				f.mu.Unlock()

			case FlowStatusWaiting:
				// 等待状态 -> 等待用户输入后继续执行
				// 用户输入新消息后会再次触发 Invoke，继续执行当前计划
				f.mu.Lock()
				f.status = FlowStatusExecuting
				f.mu.Unlock()
				// 继续到 FlowStatusExecuting 状态

			case FlowStatusUpdating:
				// 更新状态 -> 调用 Planner 更新计划
				step := f.plan.GetNextStep()
				if step == nil {
					// 没有未完成的步骤，进入总结阶段
					f.mu.Lock()
					f.status = FlowStatusSummarizing
					f.mu.Unlock()
					break
				}

				// 找到最近完成的步骤
				var completedStep *model.PlanStep
				for i := len(f.plan.Steps) - 1; i >= 0; i-- {
					if f.plan.Steps[i].Done() {
						completedStep = &f.plan.Steps[i]
						break
					}
				}

				if completedStep != nil {
					updatedPlan, err := f.planner.UpdatePlan(ctx, f.plan, completedStep)
					if err != nil {
						logger.Warn("Planner 更新计划失败", logger.Err(err))
					} else {
						f.plan = updatedPlan
						ch <- model.NewPlanEvent(updatedPlan, model.PlanEventStatusUpdated)
					}
				}

				f.mu.Lock()
				f.status = FlowStatusExecuting
				f.mu.Unlock()

			case FlowStatusSummarizing:
				// 只有真正执行过步骤的计划才进入总结。
				// 空步骤计划如果流转到这里，说明上游已经出了结构化输出问题。
				if len(f.plan.Steps) > 0 {
					summary, attachments, err := f.react.Summarize(ctx)
					if err != nil {
						logger.Warn("ReActAgent 总结任务失败", logger.Err(err))
					} else {
						ch <- model.NewMessageEvent("assistant", summary)
						for _, att := range attachments {
							logger.Info("任务生成附件", logger.String("filepath", att))
						}
					}
				} else {
					logger.Warn("跳过空步骤计划的总结阶段",
						logger.String("session_id", f.sessionID))
				}

				f.mu.Lock()
				f.status = FlowStatusCompleted
				f.mu.Unlock()

			case FlowStatusCompleted:
				// 完成状态 -> 发送完成事件
				if f.plan != nil {
					f.plan.Status = model.ExecutionStatusCompleted
					ch <- model.NewPlanEvent(f.plan, model.PlanEventStatusCompleted)
				}
				ch <- model.NewDoneEvent()
				logger.Info("PlannerReActFlow 执行完成",
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
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.plan
}
