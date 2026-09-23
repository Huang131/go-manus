package tools

import (
	"context"
	"strings"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// MessageTool 消息工具
type MessageTool struct {
}

// NewMessageTool 创建消息工具
func NewMessageTool() *MessageTool {
	return &MessageTool{}
}

// Name 返回工具名称
func (t *MessageTool) Name() string {
	return ToolNameMessage
}

// Description 返回工具描述
func (t *MessageTool) Description() string {
	return "向用户发送消息或提问。"
}

// ReadOnly message 工具会向用户发消息或提问，保守视为有副作用。
func (t *MessageTool) ReadOnly() bool {
	return false
}

// GetTools 返回工具列表
func (t *MessageTool) GetTools() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type":        llmcore.ToolTypeFunction,
			"name":        MessageFunctionNotifyUser,
			"description": "向用户发送消息（无需回复）。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "消息文本",
					},
				},
				"required": []string{"text"},
			},
		},
		{
			"type":        llmcore.ToolTypeFunction,
			"name":        MessageFunctionAskUser,
			"description": "向用户提问并等待回复。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "问题文本",
					},
					"attachments": map[string]interface{}{
						"anyOf": []map[string]interface{}{
							{"type": "string"},
							{"type": "array", "items": map[string]interface{}{"type": "string"}},
						},
						"description": "相关文件或参考资料（可选）",
					},
					"suggest_user_takeover": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"none", "browser"},
						"description": "建议用户接管",
					},
				},
				"required": []string{"text"},
			},
		},
	}
}

// Parameters 返回工具参数定义
func (t *MessageTool) Parameters() map[string]interface{} {
	tools := t.GetTools()
	if len(tools) > 0 {
		if parameters, ok := tools[0]["parameters"].(map[string]interface{}); ok {
			return parameters
		}
	}
	return nil
}

// Invoke 调用工具
func (t *MessageTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	text := ""
	if v, ok := params["text"].(string); ok {
		text = v
	} else if v, ok := params["message"].(string); ok {
		text = v
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"message_sent": text,
	}), nil
}

// InvokeWithName 根据函数名调用工具
func (t *MessageTool) InvokeWithName(functionName string, ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	switch functionName {
	case MessageFunctionNotifyUser:
		return t.invokeMessageSend(params)
	case MessageFunctionAskUser:
		return t.invokeMessageAskUser(params)
	default:
		return model.NewToolError("未知函数: " + functionName), nil
	}
}

// invokeMessageSend 发送消息
func (t *MessageTool) invokeMessageSend(params map[string]interface{}) (*model.ToolResult, error) {
	text := ""
	if v, ok := params["text"].(string); ok {
		text = v
	} else if v, ok := params["message"].(string); ok {
		text = v
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"message_sent": text,
	}), nil
}

// invokeMessageAskUser 向用户提问
// 该方法返回一个成功结果，实际的等待逻辑由 ReActAgent 处理
func (t *MessageTool) invokeMessageAskUser(params map[string]interface{}) (*model.ToolResult, error) {
	text := ""
	if v, ok := params["text"].(string); ok {
		text = v
	}

	var attachments interface{}
	if v, ok := params["attachments"]; ok {
		attachments = v
	}

	var suggestTakeover string
	if v, ok := params["suggest_user_takeover"].(string); ok {
		suggestTakeover = v
	}

	// 返回工具结果，success=true 表示工具调用成功
	// 实际的等待用户输入逻辑由 ReActAgent 在检测到 message_ask_user 时处理
	return model.NewToolResult(map[string]interface{}{
		"waiting_for_user":      true,
		"text":                  text,
		"attachments":           attachments,
		"suggest_user_takeover": suggestTakeover,
	}), nil
}

// HasTool 检查是否包含指定工具
func (t *MessageTool) HasTool(toolName string) bool {
	return strings.HasPrefix(toolName, MessageFunctionPrefix)
}
