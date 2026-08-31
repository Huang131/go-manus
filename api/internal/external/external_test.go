package external

import (
	"context"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/model"
)

// TestLLMInterface 测试 LLM 接口定义
func TestLLMInterface(t *testing.T) {
	// 验证 LLM 接口存在
	var _ LLM = (*MockLLM)(nil)
}

// MockLLM 用于测试的 LLM Mock 实现
type MockLLM struct {
	invokeResult string
	invokeErr    error
}

func (m *MockLLM) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if m.invokeErr != nil {
		return nil, m.invokeErr
	}
	return &LLMResponse{
		ID:      "mock-response",
		Content: m.invokeResult,
	}, nil
}

func (m *MockLLM) ModelName() string {
	return "mock-model"
}

func (m *MockLLM) Temperature() float64 {
	return 0.7
}

func (m *MockLLM) MaxTokens() int {
	return 4096
}

// TestLLM_Invoke 测试 LLM 调用
func TestLLM_Invoke(t *testing.T) {
	llm := &MockLLM{
		invokeResult: "This is a mock response",
	}

	req := &LLMRequest{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello, how are you?"},
		},
	}

	result, err := llm.Invoke(context.Background(), req)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if result == nil {
		t.Error("Invoke() should return a result")
	}

	if result.Content != "This is a mock response" {
		t.Errorf("Invoke() Content = %s, want This is a mock response", result.Content)
	}
}

// TestLLM_Invoke_WithError 测试带错误的 LLM 调用
func TestLLM_Invoke_WithError(t *testing.T) {
	llm := &MockLLM{
		invokeErr: context.DeadlineExceeded,
	}

	req := &LLMRequest{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	result, err := llm.Invoke(context.Background(), req)
	if err == nil {
		t.Error("Invoke() should return an error")
	}

	if result != nil {
		t.Error("Invoke() with error should return nil result")
	}
}

// TestLLM_ModelInfo 测试 LLM 模型信息
func TestLLM_ModelInfo(t *testing.T) {
	llm := &MockLLM{}

	if llm.ModelName() != "mock-model" {
		t.Errorf("ModelName() = %s, want mock-model", llm.ModelName())
	}

	if llm.Temperature() != 0.7 {
		t.Errorf("Temperature() = %f, want 0.7", llm.Temperature())
	}

	if llm.MaxTokens() != 4096 {
		t.Errorf("MaxTokens() = %d, want 4096", llm.MaxTokens())
	}
}

// TestSandboxInterface 测试 Sandbox 接口定义
func TestSandboxInterface(t *testing.T) {
	var _ Sandbox = (*MockSandbox)(nil)
}

// MockSandbox 用于测试的 Sandbox Mock 实现
type MockSandbox struct {
	execResult *model.ToolResult
	execErr    error
}

func (m *MockSandbox) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
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

func (m *MockBrowser) ViewPage(sessionID string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Page viewed",
	}, nil
}

func (m *MockBrowser) Navigate(sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Navigated to " + url,
	}, nil
}

func (m *MockBrowser) Restart(sessionID, url string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Browser restarted at " + url,
	}, nil
}

func (m *MockBrowser) Click(sessionID string, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Clicked element",
	}, nil
}

func (m *MockBrowser) Input(sessionID, text string, pressEnter bool, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Input text",
	}, nil
}

func (m *MockBrowser) MoveMouse(sessionID string, coordinateX, coordinateY float64) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Mouse moved",
	}, nil
}

func (m *MockBrowser) PressKey(sessionID, key string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Key pressed: " + key,
	}, nil
}

func (m *MockBrowser) SelectOption(sessionID string, index, option int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Option selected",
	}, nil
}

func (m *MockBrowser) ScrollUp(sessionID string, toTop *bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Scrolled up",
	}, nil
}

func (m *MockBrowser) ScrollDown(sessionID string, toDown *bool) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Scrolled down",
	}, nil
}

func (m *MockBrowser) Screenshot(sessionID string, fullPage *bool) ([]byte, error) {
	return []byte("screenshot data"), nil
}

func (m *MockBrowser) ConsoleExec(sessionID, javascript string) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Console executed",
	}, nil
}

func (m *MockBrowser) ConsoleView(sessionID string, maxLines *int) (*model.ToolResult, error) {
	return &model.ToolResult{
		Success: true,
		Message: "Console output",
	}, nil
}

// TestBrowser_Navigate 测试浏览器导航
func TestBrowser_Navigate(t *testing.T) {
	browser := &MockBrowser{}

	result, err := browser.Navigate("session-1", "https://example.com")
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

	data, err := browser.Screenshot("session-1", nil)
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

// TestSearchEngineInterface 测试 SearchEngine 接口定义
func TestSearchEngineInterface(t *testing.T) {
	var _ SearchEngine = (*MockSearchEngine)(nil)
}

// MockSearchEngine 用于测试的 SearchEngine Mock 实现
type MockSearchEngine struct {
	searchResult *model.ToolResult
	searchErr    error
}

func (m *MockSearchEngine) Invoke(ctx context.Context, query string, dateRange *string) (*model.ToolResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return &model.ToolResult{
		Success: true,
		Message: "Search results for: " + query,
	}, nil
}

// TestSearchEngine_Invoke 测试搜索引擎调用
func TestSearchEngine_Invoke(t *testing.T) {
	search := &MockSearchEngine{}

	result, err := search.Invoke(context.Background(), "test query", nil)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if result == nil {
		t.Error("Invoke() should return a result")
	}

	if !result.Success {
		t.Error("Invoke() should succeed")
	}
}

// TestSearchEngine_Invoke_WithError 测试搜索引擎错误
func TestSearchEngine_Invoke_WithError(t *testing.T) {
	search := &MockSearchEngine{
		searchErr: context.DeadlineExceeded,
	}

	result, err := search.Invoke(context.Background(), "test query", nil)
	if err == nil {
		t.Error("Invoke() should return an error")
	}

	if result != nil {
		t.Error("Invoke() with error should return nil result")
	}
}
