package model

import "github.com/Huang131/go-manus/api/internal/llmcore"

// Message 用户消息模型
// 存储用户发送的消息和附件信息
type Message struct {
	Role        string             `json:"role"`        // 消息角色: user, assistant
	Message     string             `json:"message"`     // 用户发送的消息内容
	Attachments []string           `json:"attachments"` // 用户发送的附件列表 (file_id)
	ToolCalls   []llmcore.ToolCall `json:"tool_calls"`  // 工具调用信息 (仅 assistant 角色)
	// AttachmentContexts 附件已加载内容（运行期由 agent.AttachmentLoader 注入）
	AttachmentContexts interface{} `json:"attachment_contexts,omitempty"`
}

// NewUserMessage 创建用户消息
func NewUserMessage(message string) *Message {
	return &Message{
		Role:    "user",
		Message: message,
	}
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(message string, toolCalls []llmcore.ToolCall) *Message {
	return &Message{
		Role:      "assistant",
		Message:   message,
		ToolCalls: toolCalls,
	}
}

// NewToolMessage 创建工具消息
func NewToolMessage(toolCallID, functionName, content string) *Message {
	return &Message{
		Role:    "tool",
		Message: content,
		ToolCalls: []llmcore.ToolCall{
			{
				ID:   toolCallID,
				Type: "function",
				Function: llmcore.ToolCallFunction{
					Name: functionName,
				},
			},
		},
	}
}
