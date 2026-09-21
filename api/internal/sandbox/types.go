package sandbox

import (
	"fmt"

	"github.com/Huang131/go-manus/api/internal/model"
)

// sandboxRequest 沙箱服务请求参数
type sandboxRequest struct {
	SessionID  string `json:"session_id,omitempty"`
	ExecDir    string `json:"exec_dir,omitempty"`
	Command    string `json:"command,omitempty"`
	Console    bool   `json:"console,omitempty"`
	Seconds    int    `json:"seconds,omitempty"`
	InputText  string `json:"input_text,omitempty"`
	PressEnter bool   `json:"press_enter,omitempty"`
	Filepath   string `json:"filepath,omitempty"`
	Content    string `json:"content,omitempty"`
	Append     bool   `json:"append,omitempty"`
	LeadingNL  bool   `json:"leading_newline,omitempty"`
	TrailingNL bool   `json:"trailing_newline,omitempty"`
	Sudo       bool   `json:"sudo,omitempty"`
	StartLine  int    `json:"start_line,omitempty"`
	EndLine    int    `json:"end_line,omitempty"`
	MaxLength  int    `json:"max_length,omitempty"`
	OldStr     string `json:"old_str,omitempty"`
	NewStr     string `json:"new_str,omitempty"`
	Regex      string `json:"regex,omitempty"`
	DirPath    string `json:"dir_path,omitempty"`
	GlobPtn    string `json:"glob_pattern,omitempty"`
}

// sandboxResponse 沙箱响应 {code, msg, data}
type sandboxResponse struct {
	Code int                    `json:"code"` // 0/200=成功，其他=失败
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// SandboxAPIError 沙箱 API 错误
type SandboxAPIError struct {
	StatusCode int
	Code       int
	Message    string
	Data       map[string]interface{}
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

// sandboxToolResult 沙箱响应转换为 ToolResult（code=0/200 表示成功）
func sandboxToolResult(resp *sandboxResponse) *model.ToolResult {
	if resp.Code == 0 || resp.Code == 200 {
		return model.NewToolResultWithMessage(resp.Msg, resp.Data)
	}
	return &model.ToolResult{Message: resp.Msg, Data: resp.Data, StatusCode: resp.Code}
}

// toolResultErr 把底层错误包装为 ToolResult 错误
func toolResultErr(err error) (*model.ToolResult, error) {
	return model.NewToolError(err.Error()), err
}
