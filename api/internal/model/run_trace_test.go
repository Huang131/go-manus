package model

import "testing"

// TestIsPersistableRunTraceEvent 约束长期回放只保存有限的 UI 追踪事件，
// 逐 token 文本与高频实时事件必须继续停留在短期 Redis Stream 中。
func TestIsPersistableRunTraceEvent(t *testing.T) {
	tests := []struct {
		name  string
		event EventType
		want  bool
	}{
		{name: "plan", event: EventTypePlan, want: true},
		{name: "step", event: EventTypeStep, want: true},
		{name: "tool calling", event: EventTypeToolCalling, want: true},
		{name: "tool called", event: EventTypeToolCalled, want: true},
		{name: "wait", event: EventTypeWait, want: true},
		{name: "error", event: EventTypeError, want: true},
		{name: "message delta", event: EventTypeMessageDelta, want: false},
		{name: "shell output", event: EventTypeShellOutput, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPersistableRunTraceEvent(tt.event); got != tt.want {
				t.Fatalf("IsPersistableRunTraceEvent(%q) = %t, want %t", tt.event, got, tt.want)
			}
		})
	}
}
