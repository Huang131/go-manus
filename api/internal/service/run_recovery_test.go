package service

import (
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestBuildResumeMessagesRebuildsStableConversation(t *testing.T) {
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 2,
		PlanID:           "plan-1",
		PlanRevision:     1,
		CurrentStepID:    "step-2",
		Steps: []model.RunStepSnapshot{
			{ID: "step-1", Status: model.RunStepStatusCompleted, ResultSummary: "已找到三家供应商", ArtifactRefs: []string{"file-1"}},
			{ID: "step-2", Status: model.RunStepStatusRunning},
		},
		WaitingCheckpoint: &model.WaitingCheckpoint{
			QuestionMessageID: "question-1",
			StepID:            "step-2",
			ResumeMode:        model.ResumeModeContinueStep,
		},
	}

	messages, err := BuildResumeMessages(
		model.RunMessage{ID: "input-1", Role: model.RoleUser, Content: "帮我比较供应商", Attachments: []string{"input-file"}},
		snapshot,
		model.RunMessage{ID: "question-1", Role: model.RoleAssistant, Content: "你更看重价格还是交付期？"},
		model.RunMessage{ID: "answer-1", Role: model.RoleUser, ReplyToMessageID: "question-1", Content: "优先交付期", Attachments: []string{"answer-file"}},
	)
	if err != nil {
		t.Fatalf("BuildResumeMessages() error = %v", err)
	}

	if len(messages) != 4 {
		t.Fatalf("message count = %d, want 4", len(messages))
	}
	if messages[0].Role != model.RoleUser || messages[0].ContentText != "帮我比较供应商" {
		t.Fatalf("initial message = %#v", messages[0])
	}
	if got := messages[0].Attachments; len(got) != 1 || got[0] != "input-file" {
		t.Fatalf("initial attachments = %#v", got)
	}
	if messages[1].Role != model.RoleAssistant || !strings.Contains(messages[1].ContentText, "step-1") || !strings.Contains(messages[1].ContentText, "已找到三家供应商") {
		t.Fatalf("completed step summary = %#v", messages[1])
	}
	if messages[2].Role != model.RoleAssistant || messages[2].ContentText != "你更看重价格还是交付期？" {
		t.Fatalf("question message = %#v", messages[2])
	}
	if messages[3].Role != model.RoleUser || messages[3].ContentText != "优先交付期" {
		t.Fatalf("answer message = %#v", messages[3])
	}
	if got := messages[3].Attachments; len(got) != 1 || got[0] != "answer-file" {
		t.Fatalf("answer attachments = %#v", got)
	}
}

func TestBuildResumeMessagesRejectsBrokenWaitingPair(t *testing.T) {
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 1,
		PlanID:           "plan-1",
		CurrentStepID:    "step-1",
		Steps:            []model.RunStepSnapshot{{ID: "step-1", Status: model.RunStepStatusRunning}},
		WaitingCheckpoint: &model.WaitingCheckpoint{
			QuestionMessageID: "question-1",
			StepID:            "step-1",
			ResumeMode:        model.ResumeModeContinueStep,
		},
	}

	_, err := BuildResumeMessages(
		model.RunMessage{Role: model.RoleUser, Content: "任务"},
		snapshot,
		model.RunMessage{ID: "question-1", Role: model.RoleAssistant, Content: "请确认"},
		model.RunMessage{Role: model.RoleUser, ReplyToMessageID: "wrong-question", Content: "确认"},
	)
	if err == nil {
		t.Fatal("BuildResumeMessages() error = nil, want broken reply-to pair error")
	}
}
