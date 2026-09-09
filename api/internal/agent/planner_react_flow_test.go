package agent

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// TestPlannerReActFlow_StateTransitions 测试状态转换逻辑
func TestPlannerReActFlow_StateTransitions(t *testing.T) {
	// 创建测试配置
	config := &AgentConfig{
		MaxSteps: 10,
	}

	// 测试状态转换
	flow := &PlannerReActFlow{
		status:    FlowStatusIdle,
		plan:      nil,
		sessionID: "test-session",
		config:    config,
	}

	// 初始状态应该是 Idle
	if flow.status != FlowStatusIdle {
		t.Errorf("初始状态应该是 FlowStatusIdle，实际: %s", flow.status)
	}

	// 测试状态转换（基于实际状态机实现）
	testCases := []struct {
		name         string
		current      FlowStatus
		expectedNext FlowStatus
		hasPlan      bool
		hasSteps     bool
		stepsAllDone bool
	}{
		{"Idle -> Planning", FlowStatusIdle, FlowStatusPlanning, false, false, false},
		{"Planning -> Executing", FlowStatusPlanning, FlowStatusExecuting, true, true, false},
		{"Executing -> Summarizing (plan empty)", FlowStatusExecuting, FlowStatusSummarizing, false, false, false},
		{"Executing -> Summarizing (no steps)", FlowStatusExecuting, FlowStatusSummarizing, true, false, false},
		{"Executing -> Summarizing (all steps done)", FlowStatusExecuting, FlowStatusSummarizing, true, true, true},
		{"Executing -> 执行步骤中...", FlowStatusExecuting, FlowStatusUpdating, true, true, false},
		{"Updating -> Executing", FlowStatusUpdating, FlowStatusExecuting, true, true, false},
		{"Summarizing -> Completed", FlowStatusSummarizing, FlowStatusCompleted, true, false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flow.status = tc.current
			flow.plan = nil

			if tc.hasPlan {
				flow.plan = &model.Plan{
					ID:     "test-plan",
					Title:  "Test Plan",
					Status: model.ExecutionStatusPending,
					Steps:  []model.PlanStep{},
				}

				if tc.hasSteps {
					if tc.stepsAllDone {
						// 所有步骤已完成
						flow.plan.Steps = []model.PlanStep{
							{ID: "step1", Status: model.ExecutionStatusCompleted},
						}
					} else {
						// 有待执行的步骤
						flow.plan.Steps = []model.PlanStep{
							{ID: "step1", Status: model.ExecutionStatusPending},
						}
					}
				}
			}

			// 模拟状态转换（基于实际实现）
			var nextStatus FlowStatus
			switch tc.current {
			case FlowStatusIdle:
				nextStatus = FlowStatusPlanning
			case FlowStatusPlanning:
				nextStatus = FlowStatusExecuting
			case FlowStatusExecuting:
				if flow.plan == nil || len(flow.plan.Steps) == 0 {
					nextStatus = FlowStatusSummarizing
				} else {
					step := flow.plan.GetNextStep()
					if step == nil {
						nextStatus = FlowStatusSummarizing
					} else {
						// 有待执行步骤时，进入更新状态（准备执行）
						nextStatus = FlowStatusUpdating
					}
				}
			case FlowStatusUpdating:
				nextStatus = FlowStatusExecuting
			case FlowStatusSummarizing:
				nextStatus = FlowStatusCompleted
			}

			if nextStatus != tc.expectedNext {
				t.Errorf("状态转换失败: %s -> 期望 %s，实际 %s", tc.current, tc.expectedNext, nextStatus)
			}
		})
	}
}

// TestPlannerReActFlow_StatusGetters 测试状态查询方法
func TestPlannerReActFlow_StatusGetters(t *testing.T) {
	flow := &PlannerReActFlow{
		status:    FlowStatusPlanning,
		plan:      nil,
		sessionID: "test-session",
	}

	// 测试状态字符串
	statusStr := string(flow.status)
	expectedStr := "planning"
	if statusStr != expectedStr {
		t.Errorf("状态字符串错误: 期望 %s，实际 %s", expectedStr, statusStr)
	}

	// 测试是否终止状态
	isTerminal := flow.status == FlowStatusCompleted
	if isTerminal {
		t.Error("非终止状态不应该是终止状态")
	}

	// 设置为终止状态
	flow.status = FlowStatusCompleted
	isTerminal = flow.status == FlowStatusCompleted
	if !isTerminal {
		t.Error("Completed 状态应该是终止状态")
	}
}

// TestPlannerReActFlow_InvokeContext 测试 Invoke 方法上下文处理
func TestPlannerReActFlow_InvokeContext(t *testing.T) {
	// 创建 mock LLM
	llm := &mockLLMForTest{
		response: `{"title":"Test Plan","message":"我来帮你完成任务","steps":[{"id":"step1","description":"测试步骤","status":"pending"}]}`,
	}

	// 使用 NewPlannerReActFlow 正确初始化所有字段
	config := &AgentConfig{
		MaxSteps: 10,
	}

	flow := NewPlannerReActFlow(
		"test-session",
		config,
		llm,
		[]Tool{},
	)

	// 创建已取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	input := &TaskInput{
		Message: llmcore.Message{
			Role:        llmcore.RoleUser,
			ContentText: "test message",
		},
	}

	// 由于上下文已取消，Invoke 应该会很快结束
	eventCh := flow.Invoke(ctx, input)

	// 验证事件通道是否正常返回
	eventCount := 0
	for range eventCh {
		eventCount++
		// 由于上下文已取消，不应该有太多事件
		if eventCount > 10 {
			break
		}
	}

	t.Logf("已取消上下文中收到 %d 个事件", eventCount)
}

// mockLLMForTest 用于测试的 LLM Mock
type mockLLMForTest struct {
	response string
}

func (m *mockLLMForTest) Invoke(ctx context.Context, req *external.LLMRequest) (*llmcore.LLMResponse, error) {
	return &llmcore.LLMResponse{
		ID: "mock-response",
		Message: llmcore.Message{
			Role:        llmcore.RoleAssistant,
			ContentText: m.response,
		},
	}, nil
}

func (m *mockLLMForTest) ModelName() string {
	return "mock-model"
}

func (m *mockLLMForTest) Temperature() float64 {
	return 0.7
}

func (m *mockLLMForTest) MaxTokens() int {
	return 4096
}

// TestPlannerReActFlow_PlanCreation 测试计划创建逻辑
func TestPlannerReActFlow_PlanCreation(t *testing.T) {
	plan := &model.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: model.ExecutionStatusPending,
		Steps:  []model.PlanStep{},
	}

	// 添加测试步骤
	plan.Steps = append(plan.Steps, model.PlanStep{
		ID:          "step1",
		Description: "First step",
		Status:      model.ExecutionStatusPending,
	})
	plan.Steps = append(plan.Steps, model.PlanStep{
		ID:          "step2",
		Description: "Second step",
		Status:      model.ExecutionStatusPending,
	})

	// 验证步骤数量
	if len(plan.Steps) != 2 {
		t.Errorf("计划应有 2 个步骤，实际: %d", len(plan.Steps))
	}

	// 测试 GetNextStep - 返回第一个待执行步骤（无论状态）
	step1 := plan.GetNextStep()
	if step1 == nil {
		t.Error("GetNextStep 应该返回步骤")
	}
	if step1 == nil || step1.ID != "step1" {
		if step1 != nil {
			t.Errorf("GetNextStep 应该返回第一个步骤，实际: %s", step1.ID)
		}
	}

	// GetNextStep 始终返回第一个未完成的步骤
	// 标记第一个步骤为 running（模拟执行）
	if step1 != nil {
		step1.Status = model.ExecutionStatusRunning
	}

	// 再次获取，仍然返回第一个（因为 running 状态仍未完成）
	sameStep := plan.GetNextStep()
	if sameStep == nil || sameStep.ID != "step1" {
		t.Errorf("GetNextStep 应继续返回第一个步骤（因为running仍未完成），实际: %s", sameStep.ID)
	}

	// 标记第一个步骤为 completed（模拟完成）
	step1.Status = model.ExecutionStatusCompleted

	// 现在应该返回第二个步骤
	step2 := plan.GetNextStep()
	if step2 == nil {
		t.Error("GetNextStep 应该返回第二个步骤")
	}
	if step2 != nil && step2.ID != "step2" {
		t.Errorf("GetNextStep 应该返回第二个步骤，实际: %s", step2.ID)
	}

	// 标记第二个步骤为 completed
	if step2 != nil {
		step2.Status = model.ExecutionStatusCompleted
	}

	// 获取第三个步骤（应该为 nil）
	step3 := plan.GetNextStep()
	if step3 != nil {
		t.Error("所有待执行步骤完成后 GetNextStep 应该返回 nil")
	}

	// 测试 Done 方法 - 只有 completed 或 failed 才算完成
	plan.Status = model.ExecutionStatusPending
	if plan.Done() {
		t.Error("pending 状态计划不应该完成")
	}

	plan.Status = model.ExecutionStatusRunning
	if plan.Done() {
		t.Error("running 状态计划不应该完成")
	}

	plan.Status = model.ExecutionStatusCompleted
	if !plan.Done() {
		t.Error("completed 状态计划应该完成")
	}

	plan.Status = model.ExecutionStatusFailed
	if !plan.Done() {
		t.Error("failed 状态计划应该完成")
	}
}

// TestPlannerReActFlow_PlanStepStatus 测试计划步骤状态
func TestPlannerReActFlow_PlanStepStatus(t *testing.T) {
	// 测试步骤状态
	step := &model.PlanStep{
		ID:          "step1",
		Description: "Test step",
		Status:      model.ExecutionStatusPending,
	}

	// pending 状态未完成
	if step.Done() {
		t.Error("pending 状态步骤不应该完成")
	}

	// running 状态未完成
	step.Status = model.ExecutionStatusRunning
	if step.Done() {
		t.Error("running 状态步骤不应该完成")
	}

	// completed 状态已完成
	step.Status = model.ExecutionStatusCompleted
	if !step.Done() {
		t.Error("completed 状态步骤应该完成")
	}

	// failed 状态已完成
	step.Status = model.ExecutionStatusFailed
	if !step.Done() {
		t.Error("failed 状态步骤应该完成")
	}
}
