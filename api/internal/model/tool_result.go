package model

import "encoding/json"

// ToolResult 工具执行结果
// Generic 类型用于承载不同工具的返回数据
type ToolResult struct {
	Success bool        `json:"success"` // 是否成功调用
	Message string      `json:"message"` // 额外的信息提示
	Data    interface{} `json:"data"`    // 工具的执行结果/数据
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
	data, err := json.Marshal(r)
	if err != nil {
		return `{"success": false, "message": "failed to marshal result"}`
	}
	return string(data)
}
