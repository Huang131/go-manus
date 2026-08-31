package model

// Memory 记忆模型
// 用于存储 Agent 的对话历史记忆
type Memory struct {
	Messages []map[string]interface{} `json:"messages"`
}

// NewMemory 创建新的记忆
func NewMemory() *Memory {
	return &Memory{
		Messages: []map[string]interface{}{},
	}
}

// AddMessage 添加消息到记忆
func (m *Memory) AddMessage(role, content string) {
	m.Messages = append(m.Messages, map[string]interface{}{
		"role":    role,
		"content": content,
		"created": TimeFunc().Unix(),
	})
}

// AddMessageWithContext 添加带上下文的消息
func (m *Memory) AddMessageWithContext(message map[string]interface{}) {
	m.Messages = append(m.Messages, message)
}

// Clear 清空记忆
func (m *Memory) Clear() {
	m.Messages = []map[string]interface{}{}
}

// GetMessageRole 获取消息角色
func (m *Memory) GetMessageRole(message map[string]interface{}) string {
	if role, ok := message["role"].(string); ok {
		return role
	}
	return ""
}
