package sandbox

import (
	"fmt"

	"github.com/Huang131/go-manus/api/internal/model"
)

// ==================== 请求结构体 ====================
//
// 每个端点使用独立结构体而非一个通用大结构体：
// 1. 字段与 Python schema 一一对应，漏传/错拼会被编译期或校验期拦下；
// 2. 不会再出现"通用结构体带了不该传的字段被服务端忽略"的隐性契约。
//
// 关于 bool 字段：一律不带 omitempty。false 是有语义的显式取值
// （如 press_enter=false），省略会被 Python 侧默认值覆盖导致意图反转。

// Shell 相关请求

type shellExecRequest struct {
	SessionID string `json:"session_id,omitempty"` // 为空时由沙箱侧生成新会话
	ExecDir   string `json:"exec_dir,omitempty"`
	Command   string `json:"command"`
}

type shellReadRequest struct {
	SessionID string `json:"session_id"`
	Console   bool   `json:"console"`
}

type shellWaitRequest struct {
	SessionID string `json:"session_id"`
	Seconds   *int   `json:"seconds,omitempty"`
}

type shellWriteRequest struct {
	SessionID  string `json:"session_id"`
	InputText  string `json:"input_text"`
	PressEnter bool   `json:"press_enter"`
}

type shellKillRequest struct {
	SessionID string `json:"session_id"`
}

// 文件相关请求

type fileWriteRequest struct {
	Filepath   string `json:"filepath"`
	Content    string `json:"content"`
	Append     bool   `json:"append"`
	LeadingNL  bool   `json:"leading_newline"`
	TrailingNL bool   `json:"trailing_newline"`
	Sudo       bool   `json:"sudo"`
}

type fileReadRequest struct {
	Filepath  string `json:"filepath"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
	Sudo      bool   `json:"sudo"`
	// 省略时由沙箱侧按默认上限（10000 字符）截断
	MaxLength *int `json:"max_length,omitempty"`
}

type filePathRequest struct {
	Filepath string `json:"filepath"`
}

type fileDeleteRequest struct {
	Filepath string `json:"filepath"`
	Sudo     bool   `json:"sudo"`
}

type fileReplaceRequest struct {
	Filepath string `json:"filepath"`
	OldStr   string `json:"old_str"`
	NewStr   string `json:"new_str"`
	Sudo     bool   `json:"sudo"`
}

type fileSearchRequest struct {
	Filepath string `json:"filepath"`
	Regex    string `json:"regex"`
	Sudo     bool   `json:"sudo"`
}

type fileFindRequest struct {
	DirPath     string `json:"dir_path"`
	GlobPattern string `json:"glob_pattern,omitempty"`
}

// 浏览器相关请求

type browserNavigateRequest struct {
	URL string `json:"url"`
}

type browserScreenshotRequest struct {
	FullPage bool `json:"full_page"`
}

// browserTargetRequest 元素定位目标：编号、选择器、坐标三选一
type browserTargetRequest struct {
	Index    *int     `json:"index,omitempty"`
	Selector string   `json:"selector,omitempty"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
}

type browserInputRequest struct {
	Index      *int     `json:"index,omitempty"`
	Selector   string   `json:"selector,omitempty"`
	X          *float64 `json:"x,omitempty"`
	Y          *float64 `json:"y,omitempty"`
	Text       string   `json:"text"`
	PressEnter bool     `json:"press_enter"`
}

type browserPressKeyRequest struct {
	Key string `json:"key"`
}

type browserScrollRequest struct {
	Direction string `json:"direction"`
	ToEnd     bool   `json:"to_end"`
}

type browserConsoleExecRequest struct {
	Javascript string `json:"javascript"`
}

type browserConsoleViewRequest struct {
	MaxLines *int `json:"max_lines,omitempty"`
}

// browserScreenshotData 截图端点返回：PNG 落在沙箱文件系统，需再下载二进制
type browserScreenshotData struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

// ==================== 错误与结果 ====================

// SandboxAPIError 沙箱 API 错误。
//
// StatusCode 是 HTTP 状态码，Code 是响应信封里的业务码；两者都保留，
// 便于上层按状态码分类（4xx 动作错误、5xx 环境错误）。
// RetryAfter 只在沙箱返回"繁忙"（503 + Retry-After）时非零，供上层/模型
// 决定何时重试，而非盲目立刻重发。
type SandboxAPIError struct {
	StatusCode int
	Code       int
	Message    string
	RetryAfter int
}

func (e *SandboxAPIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("sandbox API error: status=%d", e.StatusCode)
	}
	return fmt.Sprintf("sandbox API error: status=%d, message=%s", e.StatusCode, e.Message)
}

// toolResultErr 把底层错误包装为 ToolResult 错误，供工具层直接使用
func toolResultErr(err error) (*model.ToolResult, error) {
	return model.NewToolError(err.Error()), err
}
