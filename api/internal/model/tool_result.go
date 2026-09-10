package model

import "github.com/bytedance/sonic"

// ToolResult 工具执行结果的统一封装。
// 用于解耦工具实现与事件系统，所有工具返回都统一为此格式。
type ToolResult struct {
	Success bool        `json:"success"` // 工具是否执行成功
	Message string      `json:"message"` // 状态描述或错误信息
	Data    interface{} `json:"data"`    // 工具返回的原始数据，格式由具体工具决定
}

// NewToolResult 创建成功结果
func NewToolResult(data interface{}) *ToolResult {
	return &ToolResult{
		Success: true,
		Message: "",
		Data:    data,
	}
}

// NewToolResultWithMessage 创建带消息的成功结果
func NewToolResultWithMessage(message string, data interface{}) *ToolResult {
	return &ToolResult{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewToolError 创建错误结果
func NewToolError(message string) *ToolResult {
	return &ToolResult{
		Success: false,
		Message: message,
		Data:    nil,
	}
}

// FromSandbox 从沙箱返回数据构建工具结果
func (r *ToolResult) FromSandbox(code int, msg string, data interface{}) *ToolResult {
	r.Success = code < 300
	r.Message = msg
	r.Data = data
	return r
}

// JSON 将工具结果转换为 JSON 字符串
func (r *ToolResult) JSON() string {
	data, err := sonic.Marshal(r)
	if err != nil {
		return `{"success": false, "message": "failed to marshal result"}`
	}
	return string(data)
}
