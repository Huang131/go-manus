package model

import "testing"

func TestRunStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from RunStatus
		to   RunStatus
		want bool
	}{
		{name: "pending starts", from: RunStatusPending, to: RunStatusRunning, want: true},
		{name: "running waits for input", from: RunStatusRunning, to: RunStatusWaitingInput, want: true},
		{name: "waiting resumes same run", from: RunStatusWaitingInput, to: RunStatusRunning, want: true},
		{name: "pending can be cancelled", from: RunStatusPending, to: RunStatusCancelling, want: true},
		{name: "cancelling settles only after executor exits", from: RunStatusCancelling, to: RunStatusCancelled, want: true},
		{name: "terminal cannot resume", from: RunStatusSucceeded, to: RunStatusRunning, want: false},
		{name: "running cannot skip cancelling", from: RunStatusRunning, to: RunStatusCancelled, want: false},
		{name: "waiting cannot succeed without resume", from: RunStatusWaitingInput, to: RunStatusSucceeded, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Fatalf("CanTransitionTo(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestRunStatusClassifiesActiveAndTerminalStates(t *testing.T) {
	active := []RunStatus{RunStatusPending, RunStatusRunning, RunStatusWaitingInput, RunStatusCancelling}
	for _, status := range active {
		if !status.IsActive() {
			t.Fatalf("%s must be active", status)
		}
		if status.IsTerminal() {
			t.Fatalf("%s must not be terminal", status)
		}
	}

	terminal := []RunStatus{RunStatusSucceeded, RunStatusFailed, RunStatusCancelled, RunStatusInterrupted}
	for _, status := range terminal {
		if !status.IsTerminal() {
			t.Fatalf("%s must be terminal", status)
		}
		if status.IsActive() {
			t.Fatalf("%s must not be active", status)
		}
	}
}

func TestRunExecutionSnapshotValidateWaitingInput(t *testing.T) {
	valid := RunExecutionSnapshot{
		SnapshotRevision: 3,
		PlanID:           "plan-1",
		PlanRevision:     2,
		CurrentStepID:    "step-2",
		Steps: []RunStepSnapshot{
			{ID: "step-1", Description: "research facts", Status: RunStepStatusCompleted, ResultSummary: "researched facts", ArtifactRefs: []string{"file-1"}},
			{ID: "step-2", Description: "continue current task", Status: RunStepStatusRunning},
		},
		WaitingCheckpoint: &WaitingCheckpoint{
			QuestionMessageID: "message-q",
			StepID:            "step-2",
			ResumeMode:        ResumeModeContinueStep,
		},
	}

	if err := valid.ValidateWaitingInput(); err != nil {
		t.Fatalf("ValidateWaitingInput() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*RunExecutionSnapshot)
	}{
		{name: "missing current step", mutate: func(snapshot *RunExecutionSnapshot) { snapshot.CurrentStepID = "" }},
		{name: "checkpoint references other step", mutate: func(snapshot *RunExecutionSnapshot) { snapshot.WaitingCheckpoint.StepID = "step-1" }},
		{name: "completed step loses summary", mutate: func(snapshot *RunExecutionSnapshot) { snapshot.Steps[0].ResultSummary = "" }},
		{name: "unsupported resume mode", mutate: func(snapshot *RunExecutionSnapshot) { snapshot.WaitingCheckpoint.ResumeMode = "restart_plan" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := valid.Clone()
			tt.mutate(&snapshot)
			if err := snapshot.ValidateWaitingInput(); err == nil {
				t.Fatal("ValidateWaitingInput() error = nil, want invalid snapshot error")
			}
		})
	}
}
