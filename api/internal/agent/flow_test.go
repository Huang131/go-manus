package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestFlowStatus_Values(t *testing.T) {
	tests := []struct {
		status   FlowStatus
		expected string
	}{
		{FlowStatusIdle, "idle"},
		{FlowStatusPlanning, "planning"},
		{FlowStatusExecuting, "executing"},
		{FlowStatusWaiting, "waiting"},
		{FlowStatusUpdating, "updating"},
		{FlowStatusSummarizing, "summarizing"},
		{FlowStatusCompleted, "completed"},
		{FlowStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("FlowStatus %v: expected %s, got %s", tt.status, tt.expected, string(tt.status))
		}
	}
}

// TestFlowStatus_ToSessionStatus 验证三套状态中 Flow -> Session 的投影关系，
// 确保 waiting/failed/completed（含聚合）都被正确映射。
func TestFlowStatus_ToSessionStatus(t *testing.T) {
	failedPlan := &model.Plan{Steps: []model.PlanStep{{Status: model.ExecutionStatusFailed}}}
	okPlan := &model.Plan{Steps: []model.PlanStep{{Status: model.ExecutionStatusCompleted}}}

	tests := []struct {
		name   string
		status FlowStatus
		plan   *model.Plan
		want   model.SessionStatus
	}{
		{"idle -> running", FlowStatusIdle, nil, model.SessionStatusRunning},
		{"planning -> running", FlowStatusPlanning, nil, model.SessionStatusRunning},
		{"executing -> running", FlowStatusExecuting, nil, model.SessionStatusRunning},
		{"waiting -> waiting", FlowStatusWaiting, nil, model.SessionStatusWaiting},
		{"updating -> running", FlowStatusUpdating, nil, model.SessionStatusRunning},
		{"summarizing -> running", FlowStatusSummarizing, nil, model.SessionStatusRunning},
		{"failed -> failed", FlowStatusFailed, nil, model.SessionStatusFailed},
		{"completed+failed-step -> failed", FlowStatusCompleted, failedPlan, model.SessionStatusFailed},
		{"completed+ok -> completed", FlowStatusCompleted, okPlan, model.SessionStatusCompleted},
		{"completed+nil-plan -> completed", FlowStatusCompleted, nil, model.SessionStatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.ToSessionStatus(tt.plan); got != tt.want {
				t.Errorf("ToSessionStatus(%s) = %s, want %s", tt.status, got, tt.want)
			}
		})
	}
}
