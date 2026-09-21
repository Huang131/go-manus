package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

// mockSandbox 模拟 Sandbox，记录各操作收到的入参，用于验证
// ShellTool 的参数透传是否正确（而非仅验证方法被调用）。
type mockSandbox struct {
	// 记录的入参
	execSessionID  string
	execDir        string
	execCommand    string
	readSessionID  string
	readConsole    bool
	writeSessionID string
	writeInput     string
	writeEnter     bool
	waitSessionID  string
	waitSeconds    *int
	killSessionID  string

	// 可注入的错误与返回
	execError   error
	readError   error
	writeError  error
	execResult  *model.ToolResult
	readResult  *model.ToolResult
	writeResult *model.ToolResult
	waitResult  *model.ToolResult
	killResult  *model.ToolResult
}

func (m *mockSandbox) CreateSession(ctx context.Context) (string, error) {
	return "test-session", nil
}

func (m *mockSandbox) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
	m.execSessionID, m.execDir, m.execCommand = sessionID, execDir, command
	if m.execError != nil {
		return nil, m.execError
	}
	return m.execResult, nil
}

func (m *mockSandbox) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	m.readSessionID, m.readConsole = sessionID, console
	if m.readError != nil {
		return nil, m.readError
	}
	return m.readResult, nil
}

func (m *mockSandbox) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	m.writeSessionID, m.writeInput, m.writeEnter = sessionID, inputText, pressEnter
	if m.writeError != nil {
		return nil, m.writeError
	}
	return m.writeResult, nil
}

func (m *mockSandbox) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	m.waitSessionID, m.waitSeconds = sessionID, seconds
	return m.waitResult, nil
}

func (m *mockSandbox) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	m.killSessionID = sessionID
	return m.killResult, nil
}

func (m *mockSandbox) CloseSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSandbox) WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) DownloadFile(ctx context.Context, filepath string) ([]byte, error) {
	return nil, nil
}

func (m *mockSandbox) HealthCheck(ctx context.Context) error {
	return nil
}

// TestToolMetadata 以表格驱动验证各工具的元数据（名称/描述/参数 schema），
// 替代原先分散在多个测试函数中的重复断言。
func TestToolMetadata(t *testing.T) {
	tests := []struct {
		tool     Tool
		wantName string
	}{
		{tool: NewShellTool(&mockSandbox{}), wantName: ToolNameShell},
		{tool: NewFileTool(&mockSandbox{}), wantName: ToolNameFile},
		{tool: NewMessageTool(), wantName: ToolNameMessage},
		{tool: NewMCPTool(), wantName: ToolNameMCP},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			if got := tt.tool.Name(); got != tt.wantName {
				t.Errorf("Name() = %s, want %s", got, tt.wantName)
			}
			if tt.tool.Description() == "" {
				t.Error("Description() should not be empty")
			}
			params := tt.tool.Parameters()
			if params["type"] != "object" {
				t.Errorf("Parameters() type = %v, want object", params["type"])
			}
			if _, ok := params["properties"].(map[string]interface{}); !ok {
				t.Error("Parameters() should have properties")
			}
		})
	}
}

// TestShellTool_Invoke_PassesParamsToSandbox 验证 Invoke 把参数正确路由/透传给
// sandbox（行为断言），而非仅验证方法被调用。
func TestShellTool_Invoke_PassesParamsToSandbox(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		result *model.ToolResult
		verify func(t *testing.T, sb *mockSandbox)
	}{
		{
			name: "exec",
			params: map[string]interface{}{
				"action": ShellActionExec, "session_id": "s1",
				"command": "ls -la", "exec_dir": "/tmp",
			},
			verify: func(t *testing.T, sb *mockSandbox) {
				if sb.execSessionID != "s1" || sb.execCommand != "ls -la" || sb.execDir != "/tmp" {
					t.Errorf("ExecCommand got (%q, %q, %q), want (s1, ls -la, /tmp)",
						sb.execSessionID, sb.execDir, sb.execCommand)
				}
			},
		},
		{
			name: "read",
			params: map[string]interface{}{
				"action": ShellActionRead, "session_id": "s2", "console": true,
			},
			verify: func(t *testing.T, sb *mockSandbox) {
				if sb.readSessionID != "s2" || !sb.readConsole {
					t.Errorf("ReadShellOutput got (%q, %v), want (s2, true)", sb.readSessionID, sb.readConsole)
				}
			},
		},
		{
			name: "write",
			params: map[string]interface{}{
				"action": ShellActionWrite, "session_id": "s3",
				"input_text": "hello", "press_enter": false,
			},
			verify: func(t *testing.T, sb *mockSandbox) {
				if sb.writeSessionID != "s3" || sb.writeInput != "hello" || sb.writeEnter {
					t.Errorf("WriteShellInput got (%q, %q, %v), want (s3, hello, false)",
						sb.writeSessionID, sb.writeInput, sb.writeEnter)
				}
			},
		},
		{
			name: "wait",
			params: map[string]interface{}{
				"action": ShellActionWait, "session_id": "s4", "seconds": 3.0,
			},
			verify: func(t *testing.T, sb *mockSandbox) {
				if sb.waitSessionID != "s4" || sb.waitSeconds == nil || *sb.waitSeconds != 3 {
					t.Errorf("WaitProcess got (%q, %v), want (s4, 3)", sb.waitSessionID, sb.waitSeconds)
				}
			},
		},
		{
			name: "kill",
			params: map[string]interface{}{
				"action": ShellActionKill, "session_id": "s5",
			},
			verify: func(t *testing.T, sb *mockSandbox) {
				if sb.killSessionID != "s5" {
					t.Errorf("KillProcess got %q, want s5", sb.killSessionID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := &mockSandbox{
				execResult:  model.NewToolResult("ok"),
				readResult:  model.NewToolResult("ok"),
				writeResult: model.NewToolResult("ok"),
				waitResult:  model.NewToolResult("ok"),
				killResult:  model.NewToolResult("ok"),
			}
			result, err := NewShellTool(sb).Invoke(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			if result == nil {
				t.Fatal("Invoke() should return result")
			}
			tt.verify(t, sb)
		})
	}
}

// TestShellTool_Invoke_PropagatesSandboxError 验证 sandbox 错误被原样返回。
func TestShellTool_Invoke_PropagatesSandboxError(t *testing.T) {
	sb := &mockSandbox{execError: errors.New("exec error")}
	tool := NewShellTool(sb)

	_, err := tool.Invoke(context.Background(), map[string]interface{}{
		"action": ShellActionExec, "session_id": "s1", "command": "bad",
	})
	if err == nil {
		t.Error("Invoke() should return error when sandbox returns error")
	}
}

// TestShellTool_Invoke_UnknownAction 验证未知 action 返回错误结果而非 panic。
func TestShellTool_Invoke_UnknownAction(t *testing.T) {
	result, err := NewShellTool(&mockSandbox{}).Invoke(context.Background(), map[string]interface{}{
		"action": "unknown", "session_id": "s1",
	})
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}
	if result == nil || result.Success {
		t.Errorf("Invoke() = %+v, want error result for unknown action", result)
	}
}

// TestMessageTool_Invoke_PassesTextThrough 验证消息文本被透传到结果数据中。
func TestMessageTool_Invoke_PassesTextThrough(t *testing.T) {
	result, err := NewMessageTool().Invoke(context.Background(), map[string]interface{}{
		"message": "hello user",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Invoke() data type = %T, want map", result.Data)
	}
	if data["message_sent"] != "hello user" {
		t.Errorf("message_sent = %v, want hello user", data["message_sent"])
	}
}

// TestMCPTool_InitializeWithoutConfig 验证无配置时初始化为空且不暴露工具。
func TestMCPTool_InitializeWithoutConfig(t *testing.T) {
	tool := NewMCPTool()
	if err := tool.Initialize(context.Background(), nil); err != nil {
		t.Errorf("Initialize(nil) error = %v", err)
	}
	if tools := tool.GetToolsForLLM(); len(tools) != 0 {
		t.Errorf("GetToolsForLLM() = %d tools, want 0", len(tools))
	}
}
