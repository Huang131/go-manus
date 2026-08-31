package agent

import (
	"testing"
)

func TestFlowStatus_Values(t *testing.T) {
	tests := []struct {
		status   FlowStatus
		expected string
	}{
		{FlowStatusIdle, "idle"},
		{FlowStatusPlanning, "planning"},
		{FlowStatusExecuting, "executing"},
		{FlowStatusUpdating, "updating"},
		{FlowStatusSummarizing, "summarizing"},
		{FlowStatusCompleted, "completed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("FlowStatus %v: expected %s, got %s", tt.status, tt.expected, string(tt.status))
		}
	}
}
