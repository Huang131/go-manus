package agent

import (
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/bytedance/sonic"
)

func conversationMessages(events []model.Event) []llmcore.Message {
	messages := make([]llmcore.Message, 0, len(events))
	for _, event := range events {
		switch event.Type {
		case model.EventTypeMessage:
			var message model.MessageEvent
			if sonic.Unmarshal(event.Data, &message) == nil && message.Message != "" {
				messages = append(messages, llmcore.Message{Role: message.Role, ContentText: message.Message})
			}
		case model.EventTypeMessageDone:
			var message model.MessageDoneEvent
			if sonic.Unmarshal(event.Data, &message) == nil && message.Content != "" {
				messages = append(messages, llmcore.Message{Role: model.RoleAssistant, ContentText: message.Content})
			}
		}
	}
	return messages
}
