package service

import (
	"fmt"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// BuildResumeMessages 从持久化的等待输入事实重建继续当前步骤所需的 LLM 上下文。
//
// 它不读取数据库、不获取工具，也不改变 Run 状态，因此同一输入总会得到同一角色和顺序。
func BuildResumeMessages(initialInput model.RunMessage, snapshot model.RunExecutionSnapshot, question model.RunMessage, answer model.RunMessage) ([]llmcore.Message, error) {
	if err := snapshot.ValidateWaitingInput(); err != nil {
		return nil, fmt.Errorf("invalid execution snapshot: %w", err)
	}
	if initialInput.Role != model.RoleUser {
		return nil, fmt.Errorf("initial input role must be user")
	}
	if question.ID != snapshot.WaitingCheckpoint.QuestionMessageID {
		return nil, fmt.Errorf("question message %q does not match checkpoint %q", question.ID, snapshot.WaitingCheckpoint.QuestionMessageID)
	}
	if question.Role != model.RoleAssistant {
		return nil, fmt.Errorf("waiting question role must be assistant")
	}
	if answer.Role != model.RoleUser {
		return nil, fmt.Errorf("waiting answer role must be user")
	}
	if answer.ReplyToMessageID != question.ID {
		return nil, fmt.Errorf("answer reply_to_message_id %q does not match question %q", answer.ReplyToMessageID, question.ID)
	}

	messages := make([]llmcore.Message, 0, len(snapshot.Steps)+3)
	messages = append(messages, llmcore.Message{
		Role:        model.RoleUser,
		ContentText: initialInput.Content,
		Attachments: append([]string(nil), initialInput.Attachments...),
	})
	for _, step := range snapshot.Steps {
		if step.Status != model.RunStepStatusCompleted {
			continue
		}
		messages = append(messages, llmcore.Message{
			Role:        model.RoleAssistant,
			ContentText: fmt.Sprintf("已完成步骤 %s：%s", step.ID, step.ResultSummary),
			Attachments: append([]string(nil), step.ArtifactRefs...),
		})
	}
	messages = append(messages,
		llmcore.Message{Role: model.RoleAssistant, ContentText: question.Content, Attachments: append([]string(nil), question.Attachments...)},
		llmcore.Message{Role: model.RoleUser, ContentText: answer.Content, Attachments: append([]string(nil), answer.Attachments...)},
	)
	return messages, nil
}
