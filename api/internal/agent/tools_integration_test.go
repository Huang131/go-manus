package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

// mockSandbox 模拟 Sandbox 用于测试
type mockSandbox struct {
	execCalled  bool
	readCalled  bool
	writeCalled bool
	waitCalled  bool
	killCalled  bool
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
	m.execCalled = true
	if m.execError != nil {
		return nil, m.execError
	}
	return m.execResult, nil
}

func (m *mockSandbox) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	m.readCalled = true
	if m.readError != nil {
		return nil, m.readError
	}
	return m.readResult, nil
}

func (m *mockSandbox) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	m.writeCalled = true
	if m.writeError != nil {
		return nil, m.writeError
	}
	return m.writeResult, nil
}

func (m *mockSandbox) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	m.waitCalled = true
	return m.waitResult, nil
}

func (m *mockSandbox) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	m.killCalled = true
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

func (m *mockSandbox) DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return nil, nil
}

func (m *mockSandbox) HealthCheck(ctx context.Context) error {
	return nil
}

// TestShellTool_Name 测试 ShellTool 名称
func TestShellTool_Name(t *testing.T) {
	tool := NewShellTool(&mockSandbox{})
	if tool.Name() != "shell" {
		t.Errorf("Name() = %s, want shell", tool.Name())
	}
}

// TestShellTool_Description 测试 ShellTool 描述
func TestShellTool_Description(t *testing.T) {
	tool := NewShellTool(&mockSandbox{})
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
}

// TestShellTool_Parameters 测试 ShellTool 参数定义
func TestShellTool_Parameters(t *testing.T) {
	tool := NewShellTool(&mockSandbox{})
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("Parameters() should have properties")
	}

	if _, ok := props["action"]; !ok {
		t.Error("Parameters() should have action property")
	}
}

// TestShellTool_Invoke_Exec 测试 ShellTool exec 操作
func TestShellTool_Invoke_Exec(t *testing.T) {
	sandbox := &mockSandbox{
		execResult: model.NewToolResult("Hello, World!"),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "exec",
		"session_id": "test-session",
		"command":    "echo 'Hello, World!'",
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !sandbox.execCalled {
		t.Error("Invoke() should call sandbox.ExecCommand")
	}

	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestShellTool_Invoke_Read 测试 ShellTool read 操作
func TestShellTool_Invoke_Read(t *testing.T) {
	sandbox := &mockSandbox{
		readResult: model.NewToolResult("output"),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "read",
		"session_id": "test-session",
		"console":    true,
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !sandbox.readCalled {
		t.Error("Invoke() should call sandbox.ReadOutput")
	}

	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestShellTool_Invoke_Write 测试 ShellTool write 操作
func TestShellTool_Invoke_Write(t *testing.T) {
	sandbox := &mockSandbox{
		writeResult: model.NewToolResult(""),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":      "write",
		"session_id":  "test-session",
		"input_text":  "test input",
		"press_enter": true,
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !sandbox.writeCalled {
		t.Error("Invoke() should call sandbox.WriteInput")
	}

	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestShellTool_Invoke_Wait 测试 ShellTool wait 操作
func TestShellTool_Invoke_Wait(t *testing.T) {
	sandbox := &mockSandbox{
		waitResult: model.NewToolResult(""),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "wait",
		"session_id": "test-session",
		"seconds":    1.0,
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !sandbox.waitCalled {
		t.Error("Invoke() should call sandbox.WaitProcess")
	}

	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestShellTool_Invoke_Kill 测试 ShellTool kill 操作
func TestShellTool_Invoke_Kill(t *testing.T) {
	sandbox := &mockSandbox{
		killResult: model.NewToolResult(""),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "kill",
		"session_id": "test-session",
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !sandbox.killCalled {
		t.Error("Invoke() should call sandbox.KillProcess")
	}

	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestShellTool_Invoke_Error 测试 ShellTool 错误处理
func TestShellTool_Invoke_Error(t *testing.T) {
	sandbox := &mockSandbox{
		execError: errors.New("exec error"),
	}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "exec",
		"session_id": "test-session",
		"command":    "invalid command",
	}

	_, err := tool.Invoke(context.Background(), params)
	if err == nil {
		t.Error("Invoke() should return error when sandbox returns error")
	}
}

// TestShellTool_Invoke_UnknownAction 测试 ShellTool 未知操作
func TestShellTool_Invoke_UnknownAction(t *testing.T) {
	sandbox := &mockSandbox{}
	tool := NewShellTool(sandbox)

	params := map[string]interface{}{
		"action":     "unknown",
		"session_id": "test-session",
	}

	result, err := tool.Invoke(context.Background(), params)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if result == nil {
		t.Error("Invoke() should return error result for unknown action")
	}
}

// TestMessageTool 测试消息工具
func TestMessageTool(t *testing.T) {
	tool := NewMessageTool()

	// 测试名称
	if tool.Name() != "message" {
		t.Errorf("Name() = %s, want message", tool.Name())
	}

	// 测试描述
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	// 测试参数
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}

	// 测试调用
	result, err := tool.Invoke(context.Background(), map[string]interface{}{
		"message": "test message",
	})
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}
	if result == nil {
		t.Error("Invoke() should return result")
	}
}

// TestFileTool 测试文件工具
func TestFileTool(t *testing.T) {
	sandbox := &mockSandbox{}
	tool := NewFileTool(sandbox)

	// 测试名称
	if tool.Name() != "file" {
		t.Errorf("Name() = %s, want file", tool.Name())
	}

	// 测试描述
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	// 测试参数
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

// TestMCPTool 测试 MCP 工具初始化
func TestMCPTool(t *testing.T) {
	tool := NewMCPTool()

	// 测试名称
	if tool.Name() != "mcp" {
		t.Errorf("Name() = %s, want mcp", tool.Name())
	}

	// 测试描述
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	// 测试参数
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}

	// 测试无配置初始化
	err := tool.Initialize(nil)
	if err != nil {
		t.Errorf("Initialize(nil) error = %v", err)
	}

	// 测试获取工具列表（无配置时应为空）
	tools := tool.GetToolsForLLM()
	if len(tools) != 0 {
		t.Errorf("GetToolsForLLM() = %d tools, want 0", len(tools))
	}
}

// TestMCPTool_WithConfig 测试 MCP 工具配置
func TestMCPTool_WithConfig(t *testing.T) {
	tool := NewMCPTool()

	config := &MCPConfig{
		Servers: []MCPServer{
			{
				Name:    "test-server",
				Command: "echo",
				Args:    []string{"test"},
			},
		},
		Timeout: 30,
	}

	err := tool.Initialize(config)
	if err != nil {
		t.Errorf("Initialize(config) error = %v", err)
	}

	// 清理
	tool.Cleanup()
}

// TestA2ATool 测试 A2A 工具
func TestA2ATool(t *testing.T) {
	tool := NewA2ATool()

	// 测试名称
	if tool.Name() != "a2a" {
		t.Errorf("Name() = %s, want a2a", tool.Name())
	}

	// 测试描述
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	// 测试无配置初始化
	err := tool.Initialize(nil)
	if err != nil {
		t.Errorf("Initialize(nil) error = %v", err)
	}
}
