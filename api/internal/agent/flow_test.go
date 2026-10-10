package agent

import "testing"

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
		{FlowStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("FlowStatus %v: expected %s, got %s", tt.status, tt.expected, string(tt.status))
		}
	}
}
