package sandbox

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

// TestSandboxInterface 测试 Sandbox 接口定义
func TestSandboxInterface(t *testing.T) {
	var _ Sandbox = (*MockSandbox)(nil)
}

// MockSandbox 用于测试的 Sandbox Mock 实现
type MockSandbox struct {
	execResult  *model.ToolResult
	execErr     error
	lastCommand string
}

func (m *MockSandbox) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
	m.lastCommand = command
	if m.execErr != nil {
		return nil, m.execErr
	}
	if m.execResult != nil {
		return m.execResult, nil
	}
	return &model.ToolResult{
		Success: true,
		Message: "Command executed",
	}, nil
}

func TestBrowserClientScreenshotExtractsSandboxOutput(t *testing.T) {
	want := []byte("png-bytes")
	sandbox := &MockSandbox{execResult: &model.ToolResult{Success: true, Data: map[string]interface{}{
		"output": base64.StdEncoding.EncodeToString(want),
	}}}
	client := NewBrowserClient(sandbox)
	got, err := client.Screenshot(context.Background(), "session-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("Screenshot() = %q, want %q", got, want)
	}
	if !strings.HasPrefix(sandbox.lastCommand, "node -e '") {
		t.Fatalf("Screenshot command = %q, want node wrapper", sandbox.lastCommand)
	}
}

func (m *MockSandbox) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Shell output",
	}, nil
}

func (m *MockSandbox) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Process waited",
	}, nil
}

func (m *MockSandbox) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Input written",
	}, nil
}

func (m *MockSandbox) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Process killed",
	}, nil
}

func (m *MockSandbox) WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File written",
	}, nil
}

func (m *MockSandbox) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File content",
	}, nil
}

func (m *MockSandbox) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File exists",
	}, nil
}

func (m *MockSandbox) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File deleted",
	}, nil
}

func (m *MockSandbox) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Files listed",
	}, nil
}

func (m *MockSandbox) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File replaced",
	}, nil
}

func (m *MockSandbox) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Search completed",
	}, nil
}

func (m *MockSandbox) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Files found",
	}, nil
}

func (m *MockSandbox) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File uploaded",
	}, nil
}

func (m *MockSandbox) DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "File downloaded",
	}, nil
}

func (m *MockSandbox) HealthCheck(ctx context.Context) error {
	return nil
}

// TestSandbox_ExecCommand 测试沙箱命令执行
func TestSandbox_ExecCommand(t *testing.T) {
	sandbox := &MockSandbox{}

	result, err := sandbox.ExecCommand(context.Background(), "session-1", "/app", "ls -la")
	if err != nil {
		t.Errorf("ExecCommand() error = %v", err)
	}

	if result == nil {
		t.Error("ExecCommand() should return a result")
	}

	if !result.Success {
		t.Error("ExecCommand() should succeed")
	}
}

// TestSandbox_ExecCommand_WithError 测试沙箱命令执行错误
func TestSandbox_ExecCommand_WithError(t *testing.T) {
	sandbox := &MockSandbox{
		execErr: context.DeadlineExceeded,
	}

	result, err := sandbox.ExecCommand(context.Background(), "session-1", "/app", "sleep 100")
	if err == nil {
		t.Error("ExecCommand() should return an error")
	}

	if result != nil {
		t.Error("ExecCommand() with error should return nil result")
	}
}

// TestSandbox_ReadFile 测试读取文件
func TestSandbox_ReadFile(t *testing.T) {
	sandbox := &MockSandbox{}

	result, err := sandbox.ReadFile(context.Background(), "/app/test.txt", nil, nil, false, 0)
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}

	if result == nil {
		t.Error("ReadFile() should return a result")
	}

	if !result.Success {
		t.Error("ReadFile() should succeed")
	}
}

// TestSandbox_HealthCheck 测试沙箱健康检查
func TestSandbox_HealthCheck(t *testing.T) {
	sandbox := &MockSandbox{}

	err := sandbox.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

// TestBrowserInterface 测试 Browser 接口定义
func TestBrowserInterface(t *testing.T) {
	var _ Browser = (*MockBrowser)(nil)
}

// MockBrowser 用于测试的 Browser Mock 实现
type MockBrowser struct{}

func (m *MockBrowser) ViewPage(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Page viewed",
	}, nil
}

func (m *MockBrowser) Navigate(ctx context.Context, sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Navigated to " + url,
	}, nil
}

func (m *MockBrowser) Restart(ctx context.Context, sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Browser restarted at " + url,
	}, nil
}

func (m *MockBrowser) Click(ctx context.Context, sessionID string, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Clicked element",
	}, nil
}

func (m *MockBrowser) Input(ctx context.Context, sessionID, text string, pressEnter bool, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Input text",
	}, nil
}

func (m *MockBrowser) MoveMouse(ctx context.Context, sessionID string, coordinateX, coordinateY float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Mouse moved",
	}, nil
}

func (m *MockBrowser) PressKey(ctx context.Context, sessionID, key string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Key pressed: " + key,
	}, nil
}

func (m *MockBrowser) SelectOption(ctx context.Context, sessionID string, index, option int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Option selected",
	}, nil
}

func (m *MockBrowser) ScrollUp(ctx context.Context, sessionID string, toTop *bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Scrolled up",
	}, nil
}

func (m *MockBrowser) ScrollDown(ctx context.Context, sessionID string, toDown *bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Scrolled down",
	}, nil
}

func (m *MockBrowser) Screenshot(ctx context.Context, sessionID string, fullPage *bool) ([]byte, error) {
	return []byte("screenshot data"), nil
}

func (m *MockBrowser) ConsoleExec(ctx context.Context, sessionID, javascript string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Console executed",
	}, nil
}

func (m *MockBrowser) ConsoleView(ctx context.Context, sessionID string, maxLines *int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Console output",
	}, nil
}

// TestBrowser_Navigate 测试浏览器导航
func TestBrowser_Navigate(t *testing.T) {
	browser := &MockBrowser{}

	result, err := browser.Navigate(context.Background(), "session-1", "https://example.com")
	if err != nil {
		t.Errorf("Navigate() error = %v", err)
	}

	if result == nil {
		t.Error("Navigate() should return a result")
	}

	if !result.Success {
		t.Error("Navigate() should succeed")
	}
}

// TestBrowser_Screenshot 测试截图
func TestBrowser_Screenshot(t *testing.T) {
	browser := &MockBrowser{}

	data, err := browser.Screenshot(context.Background(), "session-1", nil)
	if err != nil {
		t.Errorf("Screenshot() error = %v", err)
	}

	if data == nil {
		t.Error("Screenshot() should return data")
	}

	if len(data) == 0 {
		t.Error("Screenshot() should return non-empty data")
	}
}

func TestBrowserScriptUsesNodeAndCDP(t *testing.T) {
	script := browserScript(`await page.goto("https://example.com");`)
	if !strings.HasPrefix(script, "node -e '") {
		t.Fatalf("browserScript() = %q, want node command", script)
	}
	if !strings.Contains(script, "connectOverCDP") || !strings.Contains(script, "127.0.0.1:9222") {
		t.Fatalf("browserScript() = %q, want sandbox CDP connection", script)
	}
	if strings.Contains(script, "chromium.launch") || strings.Contains(script, "browser.close") {
		t.Fatalf("browserScript() must reuse Supervisor Chrome: %q", script)
	}
}
