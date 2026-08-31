package model

import (
	"encoding/json"
	"testing"
)

func TestExecutionStatus_Values(t *testing.T) {
	tests := []struct {
		status   ExecutionStatus
		expected string
	}{
		{ExecutionStatusPending, "pending"},
		{ExecutionStatusRunning, "running"},
		{ExecutionStatusCompleted, "completed"},
		{ExecutionStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("ExecutionStatus %v: expected %s, got %s", tt.status, tt.expected, string(tt.status))
		}
	}
}

func TestPlan_Done(t *testing.T) {
	tests := []struct {
		name   string
		status ExecutionStatus
		want   bool
	}{
		{"pending not done", ExecutionStatusPending, false},
		{"running not done", ExecutionStatusRunning, false},
		{"completed is done", ExecutionStatusCompleted, true},
		{"failed is done", ExecutionStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &Plan{Status: tt.status}
			if got := plan.Done(); got != tt.want {
				t.Errorf("Plan.Done() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_GetNextStep(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: "1", Status: ExecutionStatusCompleted},
			{ID: "2", Status: ExecutionStatusRunning},
			{ID: "3", Status: ExecutionStatusPending},
		},
	}

	next := plan.GetNextStep()
	if next == nil {
		t.Fatal("GetNextStep() returned nil, want step 2")
	}
	if next.ID != "2" {
		t.Errorf("GetNextStep() = %s, want 2", next.ID)
	}
}

func TestPlan_GetNextStep_AllDone(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: "1", Status: ExecutionStatusCompleted},
			{ID: "2", Status: ExecutionStatusCompleted},
		},
	}

	next := plan.GetNextStep()
	if next != nil {
		t.Errorf("GetNextStep() = %v, want nil (all steps done)", next)
	}
}

func TestPlanStep_Done(t *testing.T) {
	tests := []struct {
		name   string
		status ExecutionStatus
		want   bool
	}{
		{"pending not done", ExecutionStatusPending, false},
		{"running not done", ExecutionStatusRunning, false},
		{"completed is done", ExecutionStatusCompleted, true},
		{"failed is done", ExecutionStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &PlanStep{Status: tt.status}
			if got := step.Done(); got != tt.want {
				t.Errorf("PlanStep.Done() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewErrorEvent(t *testing.T) {
	event := NewErrorEvent("test error")
	if event.Message != "test error" {
		t.Errorf("NewErrorEvent() message = %s, want test error", event.Message)
	}
	if event.GetType() != EventTypeError {
		t.Errorf("NewErrorEvent() type = %v, want EventTypeError", event.GetType())
	}
}

func TestNewTitleEvent(t *testing.T) {
	event := NewTitleEvent("Test Title")
	if event.Title != "Test Title" {
		t.Errorf("NewTitleEvent() title = %s, want Test Title", event.Title)
	}
	if event.GetType() != EventTypeTitle {
		t.Errorf("NewTitleEvent() type = %v, want EventTypeTitle", event.GetType())
	}
}

func TestNewMessageEvent(t *testing.T) {
	event := NewMessageEvent("assistant", "Hello!")
	if event.Content != "Hello!" {
		t.Errorf("NewMessageEvent() content = %s, want Hello!", event.Content)
	}
	if event.IsUser {
		t.Error("NewMessageEvent() IsUser should be false for assistant")
	}
	if event.GetType() != EventTypeMessage {
		t.Errorf("NewMessageEvent() type = %v, want EventTypeMessage", event.GetType())
	}
}

func TestNewDoneEvent(t *testing.T) {
	event := NewDoneEvent()
	if event.GetType() != EventTypeDone {
		t.Errorf("NewDoneEvent() type = %v, want EventTypeDone", event.GetType())
	}
}

func TestNewPlanEvent(t *testing.T) {
	plan := &Plan{ID: "plan-1", Title: "Test Plan"}
	event := NewPlanEvent(plan, PlanEventStatusCreated)

	if event.Plan != plan {
		t.Error("NewPlanEvent() plan mismatch")
	}
	if event.Status != PlanEventStatusCreated {
		t.Errorf("NewPlanEvent() status = %v, want PlanEventStatusCreated", event.Status)
	}
	if event.GetType() != EventTypePlan {
		t.Errorf("NewPlanEvent() type = %v, want EventTypePlan", event.GetType())
	}
}

func TestEvent_ToJSON(t *testing.T) {
	event := &ErrorEvent{Message: "test"}
	jsonStr := event.ToJSON()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("ToJSON() is not valid JSON: %v", err)
	}

	if parsed["message"] != "test" {
		t.Errorf("ToJSON() message = %v, want test", parsed["message"])
	}
}
