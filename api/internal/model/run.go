package model

import (
	"fmt"
	"time"

	"github.com/Huang131/go-manus/api/internal/settings"
)

// RunStatus 表示一次用户执行请求的唯一生命周期状态。
//
// Session 是长期会话容器；Run 是一次可并发控制、可恢复的执行事实。
type RunStatus string

const (
	RunStatusPending      RunStatus = "pending"
	RunStatusRunning      RunStatus = "running"
	RunStatusWaitingInput RunStatus = "waiting_input"
	RunStatusCancelling   RunStatus = "cancelling"
	RunStatusSucceeded    RunStatus = "succeeded"
	RunStatusFailed       RunStatus = "failed"
	RunStatusCancelled    RunStatus = "cancelled"
	RunStatusInterrupted  RunStatus = "interrupted"
)

// IsActive 报告该状态是否占用所属 Session 的唯一活跃执行槽位。
func (s RunStatus) IsActive() bool {
	switch s {
	case RunStatusPending, RunStatusRunning, RunStatusWaitingInput, RunStatusCancelling:
		return true
	default:
		return false
	}
}

// IsTerminal 报告该状态是否不允许继续迁移。
func (s RunStatus) IsTerminal() bool {
	switch s {
	case RunStatusSucceeded, RunStatusFailed, RunStatusCancelled, RunStatusInterrupted:
		return true
	default:
		return false
	}
}

// CanTransitionTo 是 Run 生命周期唯一的状态迁移矩阵。
func (s RunStatus) CanTransitionTo(next RunStatus) bool {
	switch s {
	case RunStatusPending:
		return next == RunStatusRunning || next == RunStatusCancelling || next == RunStatusInterrupted
	case RunStatusRunning:
		return next == RunStatusWaitingInput || next == RunStatusSucceeded || next == RunStatusFailed || next == RunStatusCancelling || next == RunStatusInterrupted
	case RunStatusWaitingInput:
		return next == RunStatusRunning || next == RunStatusCancelling
	case RunStatusCancelling:
		return next == RunStatusCancelled
	default:
		return false
	}
}

// RunStepStatus 是执行快照中步骤的持久化状态。
type RunStepStatus string

const (
	RunStepStatusPending   RunStepStatus = "pending"
	RunStepStatusRunning   RunStepStatus = "running"
	RunStepStatusCompleted RunStepStatus = "completed"
	RunStepStatusFailed    RunStepStatus = "failed"
)

const ResumeModeContinueStep = "continue_step"

// WaitingCheckpoint 描述 waiting_input 恢复到哪一步以及由哪条问题消息触发。
type WaitingCheckpoint struct {
	QuestionMessageID string `json:"question_message_id"`
	StepID            string `json:"step_id"`
	ResumeMode        string `json:"resume_mode"`
}

// RunStepSnapshot 是恢复执行所需的稳定步骤摘要，不保存运行期句柄或完整工具结果。
type RunStepSnapshot struct {
	ID            string        `json:"id"`
	Status        RunStepStatus `json:"status"`
	ResultSummary string        `json:"result_summary,omitempty"`
	ArtifactRefs  []string      `json:"artifact_refs,omitempty"`
}

// RunExecutionSnapshot 是 waiting_input 恢复所需的最小、可持久化执行事实。
type RunExecutionSnapshot struct {
	SnapshotRevision  int                `json:"snapshot_revision"`
	PlanID            string             `json:"plan_id"`
	PlanRevision      int                `json:"plan_revision"`
	CurrentStepID     string             `json:"current_step_id"`
	Steps             []RunStepSnapshot  `json:"steps"`
	WaitingCheckpoint *WaitingCheckpoint `json:"waiting_checkpoint,omitempty"`
}

// Clone 返回独立快照，防止调用方的可变 slice 影响已验证的执行事实。
func (s RunExecutionSnapshot) Clone() RunExecutionSnapshot {
	cloned := s
	if s.Steps != nil {
		cloned.Steps = make([]RunStepSnapshot, len(s.Steps))
		copy(cloned.Steps, s.Steps)
		for i := range cloned.Steps {
			cloned.Steps[i].ArtifactRefs = append([]string(nil), s.Steps[i].ArtifactRefs...)
		}
	}
	if s.WaitingCheckpoint != nil {
		checkpoint := *s.WaitingCheckpoint
		cloned.WaitingCheckpoint = &checkpoint
	}
	return cloned
}

// ValidateWaitingInput 验证快照是否能在进程重启后安全恢复当前步骤。
func (s RunExecutionSnapshot) ValidateWaitingInput() error {
	if s.SnapshotRevision < 1 {
		return fmt.Errorf("snapshot_revision must be positive")
	}
	if s.PlanID == "" {
		return fmt.Errorf("plan_id is required")
	}
	if s.CurrentStepID == "" {
		return fmt.Errorf("current_step_id is required")
	}
	if s.WaitingCheckpoint == nil {
		return fmt.Errorf("waiting_checkpoint is required")
	}
	if s.WaitingCheckpoint.QuestionMessageID == "" {
		return fmt.Errorf("waiting_checkpoint.question_message_id is required")
	}
	if s.WaitingCheckpoint.StepID != s.CurrentStepID {
		return fmt.Errorf("waiting checkpoint step %q does not match current step %q", s.WaitingCheckpoint.StepID, s.CurrentStepID)
	}
	if s.WaitingCheckpoint.ResumeMode != ResumeModeContinueStep {
		return fmt.Errorf("unsupported resume mode %q", s.WaitingCheckpoint.ResumeMode)
	}

	seenCurrentStep := false
	seenStepIDs := make(map[string]struct{}, len(s.Steps))
	for _, step := range s.Steps {
		if step.ID == "" {
			return fmt.Errorf("step id is required")
		}
		if _, exists := seenStepIDs[step.ID]; exists {
			return fmt.Errorf("duplicate step id %q", step.ID)
		}
		seenStepIDs[step.ID] = struct{}{}
		if step.ID == s.CurrentStepID {
			seenCurrentStep = true
		}
		if step.Status == RunStepStatusCompleted && step.ResultSummary == "" {
			return fmt.Errorf("completed step %q requires result_summary", step.ID)
		}
	}
	if !seenCurrentStep {
		return fmt.Errorf("current step %q is absent from snapshot", s.CurrentStepID)
	}
	return nil
}

// Run 是一次用户执行请求的持久化领域事实。存储实现将在下一检查点接入。
type Run struct {
	ID                string                 `json:"id"`
	SessionID         string                 `json:"session_id"`
	Status            RunStatus              `json:"status"`
	IdempotencyKey    string                 `json:"idempotency_key"`
	SettingsSnapshot  settings.AgentSettings `json:"settings_snapshot"`
	PromptHash        string                 `json:"prompt_hash"`
	ExecutionSnapshot RunExecutionSnapshot   `json:"execution_snapshot"`
	SnapshotRevision  int                    `json:"snapshot_revision"`
	WaitingMessageID  string                 `json:"waiting_message_id,omitempty"`
	ErrorCode         string                 `json:"error_code,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	FinishedAt        *time.Time             `json:"finished_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// RunMessage 是后续 messages 持久化表的领域模型。
type RunMessage struct {
	ID               string      `json:"id"`
	SessionID        string      `json:"session_id"`
	RunID            string      `json:"run_id"`
	IdempotencyKey   string      `json:"idempotency_key,omitempty"`
	ReplyToMessageID string      `json:"reply_to_message_id,omitempty"`
	Role             MessageRole `json:"role"`
	Content          string      `json:"content"`
	Attachments      []string    `json:"attachments,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
}
