package model

import "github.com/bytedance/sonic"

// ToolResult 工具执行结果的统一封装。
// 用于解耦工具实现与事件系统，所有工具返回都统一为此格式。
type ToolResult struct {
	Success bool        `json:"success"` // 工具是否执行成功
	Message string      `json:"message"` // 状态描述或错误信息
	Data    interface{} `json:"data"`    // 工具返回的原始数据，格式由具体工具决定
	// Display 只承载 UI 展示所需的轻量数据。它不进入 LLM 上下文：
	// 拼装 tool 消息时用 LLMJSON() 序列化，会剔除该字段。
	// 大体积产物（如浏览器截图）不要直接放这里，改用 Artifacts 挂载。
	Display map[string]interface{} `json:"display,omitempty"`
	// Artifacts 承载工具产出的二进制展示产物（如浏览器截图的 PNG 字节）。
	//
	// 它既不进 LLM 上下文，也不进事件流：json:"-" 保证它不会被 JSON()/LLMJSON()
	// 序列化出去。运行期会把每份产物落对象存储，只在 Display 里留下文件引用，
	// 从而避免 base64 把 SSE 事件和事件库撑大。
	Artifacts map[string]ToolArtifact `json:"-"`
	// StatusCode 是外部工具返回的 HTTP/业务状态码，仅供边界层映射错误，
	// 不暴露给 Agent 事件和 API 响应，避免把传输细节泄漏到业务数据。
	StatusCode int `json:"-"`
}

// ToolArtifact 工具产出的二进制展示产物。
// Filename/MimeType 必须给出：落对象存储时据此推导扩展名与内容类型。
type ToolArtifact struct {
	Filename string
	MimeType string
	Data     []byte
}

// WithDisplay 追加一项仅用于 UI 的展示数据，返回自身便于链式调用。
func (r *ToolResult) WithDisplay(key string, value interface{}) *ToolResult {
	if r.Display == nil {
		r.Display = make(map[string]interface{}, 1)
	}
	r.Display[key] = value
	return r
}

// WithArtifact 挂载一份二进制展示产物，返回自身便于链式调用。
// 产物不会直接出现在结果里，需由运行期落存储后转为 Display 中的文件引用。
func (r *ToolResult) WithArtifact(key string, artifact ToolArtifact) *ToolResult {
	if r.Artifacts == nil {
		r.Artifacts = make(map[string]ToolArtifact, 1)
	}
	r.Artifacts[key] = artifact
	return r
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

// JSON 将工具结果转换为 JSON 字符串，包含 Display（供事件推送给 UI）。
func (r *ToolResult) JSON() string {
	data, err := sonic.Marshal(r)
	if err != nil {
		return `{"success": false, "message": "failed to marshal result"}`
	}
	return string(data)
}

// LLMJSON 将工具结果序列化为 LLM 可见的 JSON：剔除仅用于 UI 的 Display。
// tool 消息必须走这个方法，否则截图等重数据会进入对话历史并持续消耗 token。
func (r *ToolResult) LLMJSON() string {
	if r == nil {
		return `{"success": false, "message": "empty tool result"}`
	}
	if len(r.Display) == 0 {
		return r.JSON()
	}
	clone := *r
	clone.Display = nil
	data, err := sonic.Marshal(&clone)
	if err != nil {
		return `{"success": false, "message": "failed to marshal result"}`
	}
	return string(data)
}
