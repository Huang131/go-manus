package sandbox

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestSandboxInterface(t *testing.T) {
	var _ Sandbox = (*MockSandbox)(nil)
}

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
	return &model.ToolResult{Success: true, Message: "Command executed"}, nil
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
	// 下发给沙箱的命令应包含截图脚本（browserScript 生成 heredoc 包裹）
	if !strings.Contains(sandbox.lastCommand, "page.screenshot") {
		t.Fatalf("Screenshot command = %q, want screenshot script", sandbox.lastCommand)
	}
	// 通过 browserScript 验证 heredoc 格式（独立测试）
	script := browserScript(`await page.screenshot();`)
	if !strings.HasPrefix(script, "cat << 'PLAYWRIGHT_EOF'") {
		t.Fatalf("browserScript() = %q, want heredoc wrapper", script)
	}
}

func (m *MockSandbox) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Shell output"}, nil
}

func (m *MockSandbox) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Process waited"}, nil
}

func (m *MockSandbox) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Input written"}, nil
}

func (m *MockSandbox) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Process killed"}, nil
}

func (m *MockSandbox) WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File written"}, nil
}

func (m *MockSandbox) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File content"}, nil
}

func (m *MockSandbox) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File exists"}, nil
}

func (m *MockSandbox) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File deleted"}, nil
}

func (m *MockSandbox) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Files listed"}, nil
}

func (m *MockSandbox) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File replaced"}, nil
}

func (m *MockSandbox) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Search completed"}, nil
}

func (m *MockSandbox) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Files found"}, nil
}

func (m *MockSandbox) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File uploaded"}, nil
}

func (m *MockSandbox) DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "File downloaded"}, nil
}

func (m *MockSandbox) HealthCheck(ctx context.Context) error {
	return nil
}

func TestSandbox_ExecCommand(t *testing.T) {
	sandbox := &MockSandbox{}
	result, err := sandbox.ExecCommand(context.Background(), "session-1", "/app", "ls -la")
	if err != nil {
		t.Errorf("ExecCommand() error = %v", err)
	}
	if result == nil || !result.Success {
		t.Error("ExecCommand() should succeed")
	}
}

func TestSandbox_ExecCommand_WithError(t *testing.T) {
	sandbox := &MockSandbox{execErr: context.DeadlineExceeded}
	result, err := sandbox.ExecCommand(context.Background(), "session-1", "/app", "sleep 100")
	if err == nil {
		t.Error("ExecCommand() should return an error")
	}
	if result != nil {
		t.Error("ExecCommand() with error should return nil result")
	}
}

func TestSandbox_ReadFile(t *testing.T) {
	sandbox := &MockSandbox{}
	result, err := sandbox.ReadFile(context.Background(), "/app/test.txt", nil, nil, false, 0)
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}
	if result == nil || !result.Success {
		t.Error("ReadFile() should succeed")
	}
}

func TestSandbox_HealthCheck(t *testing.T) {
	sandbox := &MockSandbox{}
	if err := sandbox.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestBrowserInterface(t *testing.T) {
	var _ Browser = (*MockBrowser)(nil)
}

type MockBrowser struct{}

func (m *MockBrowser) ViewPage(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Page viewed"}, nil
}

func (m *MockBrowser) Navigate(ctx context.Context, sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Navigated to " + url}, nil
}

func (m *MockBrowser) Restart(ctx context.Context, sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Browser restarted at " + url}, nil
}

func (m *MockBrowser) Click(ctx context.Context, sessionID string, index *int, x, y *float64) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Clicked element"}, nil
}

func (m *MockBrowser) Input(ctx context.Context, sessionID, text string, pressEnter bool, index *int, x, y *float64) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Input text"}, nil
}

func (m *MockBrowser) MoveMouse(ctx context.Context, sessionID string, x, y float64) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Mouse moved"}, nil
}

func (m *MockBrowser) PressKey(ctx context.Context, sessionID, key string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Key pressed: " + key}, nil
}

func (m *MockBrowser) SelectOption(ctx context.Context, sessionID string, index, option int) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Option selected"}, nil
}

func (m *MockBrowser) ScrollUp(ctx context.Context, sessionID string, toTop *bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Scrolled up"}, nil
}

func (m *MockBrowser) ScrollDown(ctx context.Context, sessionID string, toBottom *bool) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Scrolled down"}, nil
}

func (m *MockBrowser) Screenshot(ctx context.Context, sessionID string, fullPage *bool) ([]byte, error) {
	return []byte("screenshot data"), nil
}

func (m *MockBrowser) ConsoleExec(ctx context.Context, sessionID, javascript string) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Console executed"}, nil
}

func (m *MockBrowser) ConsoleView(ctx context.Context, sessionID string, maxLines *int) (*model.ToolResult, error) {
	return &model.ToolResult{Success: true, Message: "Console output"}, nil
}

func TestBrowser_Navigate(t *testing.T) {
	browser := &MockBrowser{}
	result, err := browser.Navigate(context.Background(), "session-1", "https://example.com")
	if err != nil {
		t.Errorf("Navigate() error = %v", err)
	}
	if result == nil || !result.Success {
		t.Error("Navigate() should succeed")
	}
}

func TestBrowser_Screenshot(t *testing.T) {
	browser := &MockBrowser{}
	data, err := browser.Screenshot(context.Background(), "session-1", nil)
	if err != nil {
		t.Errorf("Screenshot() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("Screenshot() should return non-empty data")
	}
}

func TestBrowserScriptUsesNodeAndCDP(t *testing.T) {
	script := browserScript(`await page.goto("https://example.com");`)
	if !strings.HasPrefix(script, "cat << 'PLAYWRIGHT_EOF'") {
		t.Fatalf("browserScript() = %q, want heredoc command", script)
	}
	if !strings.Contains(script, "connectOverCDP") || !strings.Contains(script, cdpAddress) {
		t.Fatalf("browserScript() = %q, want sandbox CDP connection", script)
	}
	if strings.Contains(script, "chromium.launch") || strings.Contains(script, "browser.close") {
		t.Fatalf("browserScript() must reuse Supervisor Chrome: %q", script)
	}
}

func TestBrowserScript_MultiLineOperation(t *testing.T) {
	multiLineOp := `await page.goto("https://example.com");
await page.waitForSelector('input');
await page.fill('input', 'test');`
	script := browserScript(multiLineOp)
	if !strings.Contains(script, multiLineOp) {
		t.Fatalf("browserScript() should preserve multi-line operation")
	}
}

func TestJsString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", `"hello"`},
		{`he said "hi"`, `"he said \"hi\""`},
		// 使用 raw string 表示真正的换行符，sonic 会编码为 \n（两个字面量字符）
		{"line\nbreak", `"line\nbreak"`},
		{"it's", `"it's"`},
		{"with\\backslash", `"with\\backslash"`},
		{"tabs\there", `"tabs\there"`},
	}
	for _, tt := range tests {
		got := jsString(tt.input)
		if got != tt.expected {
			t.Errorf("jsString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSandboxAPIError(t *testing.T) {
	err := &SandboxAPIError{
		StatusCode: 500,
		Code:       500,
		Message:    "internal error",
	}
	if err.Error() != "sandbox API error: status=500, message=internal error" {
		t.Errorf("SandboxAPIError.Error() = %q", err.Error())
	}
}

func TestSandboxErrorStatus(t *testing.T) {
	err := &SandboxAPIError{StatusCode: 404, Code: 404, Message: "not found"}
	if SandboxErrorStatus(err) != 404 {
		t.Errorf("SandboxErrorStatus() = %d, want 404", SandboxErrorStatus(err))
	}

	if SandboxErrorStatus(context.DeadlineExceeded) != 0 {
		t.Errorf("SandboxErrorStatus() for non-SandboxAPIError = %d, want 0", SandboxErrorStatus(context.DeadlineExceeded))
	}
}
