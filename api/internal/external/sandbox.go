package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// sandboxToolResult 创建沙箱响应对应的 ToolResult
// 对齐 sandbox 服务的 Response 模型：code=200 表示成功，msg 是结果描述，data 是业务负载
func sandboxToolResult(resp *sandboxResponse) *model.ToolResult {
	if resp.Code == 0 || resp.Code == 200 {
		return model.NewToolResultWithMessage(resp.Msg, resp.Data)
	}
	return model.NewToolError(resp.Msg)
}

// Sandbox 沙箱服务接口
type Sandbox interface {
	// ExecCommand 执行 Shell 命令
	ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error)

	// ReadShellOutput 读取 Shell 输出
	ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error)

	// WaitProcess 等待进程执行
	WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error)

	// WriteShellInput 写入 Shell 输入
	WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error)

	// KillProcess 杀死进程
	KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error)

	// WriteFile 写入文件内容
	WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error)

	// ReadFile 读取文件内容
	ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error)

	// CheckFileExists 检查文件是否存在
	CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error)

	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error)

	// ListFiles 列出目录文件
	ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error)

	// ReplaceInFile 替换文件内容
	ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error)

	// SearchInFile 在文件中搜索
	SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error)

	// FindFiles 查找文件
	FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error)

	// UploadFile 上传文件到沙箱
	UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error)

	// DownloadFile 下载沙箱文件
	DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// SandboxClient 沙箱服务 HTTP 客户端
type SandboxClient struct {
	address    string
	httpClient *http.Client
}

// NewSandboxClient 创建沙箱客户端
func NewSandboxClient(cfg *config.SandboxConfig) *SandboxClient {
	return &SandboxClient{
		address: cfg.Address,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// sandboxRequest 沙箱请求
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

// sandboxResponse 沙箱响应（对齐 sandbox 服务实际返回的 {code, msg, data} 格式）
type sandboxResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// sandboxErrorResponse 沙箱错误响应
type sandboxErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// doRequest 发送请求到沙箱服务
func (c *SandboxClient) doRequest(ctx context.Context, method, action string, req *sandboxRequest) (*sandboxResponse, error) {
	if c.address == "" {
		return nil, fmt.Errorf("sandbox address not configured")
	}

	url := fmt.Sprintf("%s/api/%s", c.address, action)

	var body io.Reader
	if req != nil {
		data, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sandbox API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var sandboxResp sandboxResponse
	if err := json.Unmarshal(respBody, &sandboxResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &sandboxResp, nil
}

// ExecCommand 执行 Shell 命令
func (c *SandboxClient) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/exec-command", &sandboxRequest{
		SessionID: sessionID,
		ExecDir:   execDir,
		Command:   command,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// ReadShellOutput 读取 Shell 输出
func (c *SandboxClient) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/read-shell-output", &sandboxRequest{
		SessionID: sessionID,
		Console:   console,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// WaitProcess 等待进程执行
func (c *SandboxClient) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	req := &sandboxRequest{SessionID: sessionID}
	if seconds != nil {
		req.Seconds = *seconds
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/wait-process", req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// WriteShellInput 写入 Shell 输入
func (c *SandboxClient) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/write-shell-input", &sandboxRequest{
		SessionID:  sessionID,
		InputText:  inputText,
		PressEnter: pressEnter,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// KillProcess 杀死进程
func (c *SandboxClient) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/kill-process", &sandboxRequest{
		SessionID: sessionID,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// WriteFile 写入文件内容
func (c *SandboxClient) WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/write-file", &sandboxRequest{
		Filepath:   filepath,
		Content:    content,
		Append:     append,
		LeadingNL:  leadingNewline,
		TrailingNL: trailingNewline,
		Sudo:       sudo,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// ReadFile 读取文件内容
func (c *SandboxClient) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	req := &sandboxRequest{
		Filepath:  filepath,
		Sudo:      sudo,
		MaxLength: maxLength,
	}
	if startLine != nil {
		req.StartLine = *startLine
	}
	if endLine != nil {
		req.EndLine = *endLine
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "file/read-file", req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// CheckFileExists 检查文件是否存在
func (c *SandboxClient) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/check-file-exists", &sandboxRequest{
		Filepath: filepath,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// DeleteFile 删除文件
func (c *SandboxClient) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodDelete, "file/delete-file", &sandboxRequest{
		Filepath: filepath,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// ListFiles 列出目录文件
func (c *SandboxClient) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "file/find-files", &sandboxRequest{
		DirPath: dirPath,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// ReplaceInFile 替换文件内容
func (c *SandboxClient) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/replace-in-file", &sandboxRequest{
		Filepath: filepath,
		OldStr:   oldStr,
		NewStr:   newStr,
		Sudo:     sudo,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// SearchInFile 在文件中搜索
func (c *SandboxClient) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/search-in-file", &sandboxRequest{
		Filepath: filepath,
		Regex:    regex,
		Sudo:     sudo,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// FindFiles 查找文件
func (c *SandboxClient) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "file/find-files", &sandboxRequest{
		DirPath: dirPath,
		GlobPtn: globPattern,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// UploadFile 上传文件到沙箱
func (c *SandboxClient) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	if c.address == "" {
		return model.NewToolError("sandbox address not configured"), fmt.Errorf("sandbox address not configured")
	}

	url := fmt.Sprintf("%s/api/file/upload-file", c.address)

	// 创建 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	if _, err := part.Write(fileData); err != nil {
		return model.NewToolError(err.Error()), err
	}

	// 添加 filepath 字段
	if err := writer.WriteField("filepath", filepath); err != nil {
		return model.NewToolError(err.Error()), err
	}

	if err := writer.Close(); err != nil {
		return model.NewToolError(err.Error()), err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("upload failed: %s", string(respBody))), fmt.Errorf("upload failed: %s", respBody)
	}

	var sandboxResp sandboxResponse
	if err := json.Unmarshal(respBody, &sandboxResp); err != nil {
		return model.NewToolError(err.Error()), err
	}

	return sandboxToolResult(&sandboxResp), nil
}

// DownloadFile 下载沙箱文件
func (c *SandboxClient) DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "file/download-file", &sandboxRequest{
		Filepath: filepath,
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	return sandboxToolResult(resp), nil
}

// HealthCheck 健康检查
func (c *SandboxClient) HealthCheck(ctx context.Context) error {
	if c.address == "" {
		return fmt.Errorf("sandbox address not configured")
	}

	url := fmt.Sprintf("%s/health", c.address)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sandbox health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sandbox health check: status=%d", resp.StatusCode)
	}

	return nil
}

// BrowserClient 浏览器客户端 (基于 Sandbox 实现)
type BrowserClient struct {
	sandbox    Sandbox
	httpClient *http.Client
}

// NewBrowserClient 创建浏览器客户端
func NewBrowserClient(sandbox Sandbox) *BrowserClient {
	return &BrowserClient{
		sandbox: sandbox,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ViewPage 获取当前浏览器的页面内容
func (c *BrowserClient) ViewPage(sessionID string) (*model.ToolResult, error) {
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", "cat /tmp/browser/page.html")
}

// Navigate 使用浏览器导航到指定 URL
func (c *BrowserClient) Navigate(sessionID, url string) (*model.ToolResult, error) {
	// 通过执行 Playwright 脚本实现导航
	script := fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.goto('%s'); await browser.close(); })();`, url)
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// Restart 重启浏览器并访问指定 URL
func (c *BrowserClient) Restart(sessionID, url string) (*model.ToolResult, error) {
	// 先关闭现有浏览器，再启动新的
	killScript := `pkill -f chromium || true`
	if _, err := c.sandbox.ExecCommand(context.Background(), sessionID, "", killScript); err != nil {
		logger.Warn("failed to kill existing browser", zap.Error(err))
	}
	return c.Navigate(sessionID, url)
}

// Click 通过索引或坐标点击元素
func (c *BrowserClient) Click(sessionID string, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	var script string
	if index != nil {
		script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.locator('a').nth(%d).click(); await browser.close(); })();`, *index)
	} else if coordinateX != nil && coordinateY != nil {
		script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.mouse.click(%f, %f); await browser.close(); })();`, *coordinateX, *coordinateY)
	} else {
		return model.NewToolError("either index or coordinates required"), fmt.Errorf("either index or coordinates required")
	}
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// Input 在输入框中输入文本
func (c *BrowserClient) Input(sessionID, text string, pressEnter bool, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error) {
	var script string
	if index != nil {
		if pressEnter {
			script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.locator('input').nth(%d).fill('%s'); await page.keyboard.press('Enter'); await browser.close(); })();`, *index, text)
		} else {
			script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.locator('input').nth(%d).fill('%s'); await browser.close(); })();`, *index, text)
		}
	} else if coordinateX != nil && coordinateY != nil {
		if pressEnter {
			script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.mouse.click(%f, %f); await page.keyboard.type('%s'); await page.keyboard.press('Enter'); await browser.close(); })();`, *coordinateX, *coordinateY, text)
		} else {
			script = fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.mouse.click(%f, %f); await page.keyboard.type('%s'); await browser.close(); })();`, *coordinateX, *coordinateY, text)
		}
	} else {
		return model.NewToolError("either index or coordinates required"), fmt.Errorf("either index or coordinates required")
	}
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// MoveMouse 移动鼠标到指定坐标
func (c *BrowserClient) MoveMouse(sessionID string, coordinateX, coordinateY float64) (*model.ToolResult, error) {
	script := fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.mouse.move(%f, %f); await browser.close(); })();`, coordinateX, coordinateY)
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// PressKey 模拟按键
func (c *BrowserClient) PressKey(sessionID, key string) (*model.ToolResult, error) {
	script := fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.keyboard.press('%s'); await browser.close(); })();`, key)
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// SelectOption 在下拉菜单中选择选项
func (c *BrowserClient) SelectOption(sessionID string, index, option int) (*model.ToolResult, error) {
	script := fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.locator('select').nth(%d).selectOption({ index: %d }); await browser.close(); })();`, index, option)
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// ScrollUp 向上滚动浏览器
func (c *BrowserClient) ScrollUp(sessionID string, toTop *bool) (*model.ToolResult, error) {
	var script string
	if toTop != nil && *toTop {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.evaluate(() => window.scrollTo(0, 0)); await browser.close(); })();`
	} else {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.evaluate(() => window.scrollBy(0, -window.innerHeight)); await browser.close(); })();`
	}
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// ScrollDown 向下滚动浏览器
func (c *BrowserClient) ScrollDown(sessionID string, toDown *bool) (*model.ToolResult, error) {
	var script string
	if toDown != nil && *toDown {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight)); await browser.close(); })();`
	} else {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.evaluate(() => window.scrollBy(0, window.innerHeight)); await browser.close(); })();`
	}
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// Screenshot 对当前页面截图
func (c *BrowserClient) Screenshot(sessionID string, fullPage *bool) ([]byte, error) {
	var script string
	if fullPage != nil && *fullPage {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); const screenshot = await page.screenshot({ fullPage: true }); console.log(Buffer.from(screenshot).toString('base64')); await browser.close(); })();`
	} else {
		script = `const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); const screenshot = await page.screenshot(); console.log(Buffer.from(screenshot).toString('base64')); await browser.close(); })();`
	}
	result, err := c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("screenshot failed: %s", result.Message)
	}
	return []byte(result.Message), nil
}

// ConsoleExec 在浏览器控制台执行 JavaScript
func (c *BrowserClient) ConsoleExec(sessionID, javascript string) (*model.ToolResult, error) {
	script := fmt.Sprintf(`const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); const page = await browser.newPage(); await page.evaluate(() => { %s }); await browser.close(); })();`, javascript)
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", script)
}

// ConsoleView 获取控制台输出
func (c *BrowserClient) ConsoleView(sessionID string, maxLines *int) (*model.ToolResult, error) {
	// 通过读取 console 日志文件实现
	cmd := "cat /tmp/browser/console.log"
	if maxLines != nil {
		cmd = fmt.Sprintf("head -n %d /tmp/browser/console.log", *maxLines)
	}
	return c.sandbox.ExecCommand(context.Background(), sessionID, "", cmd)
}
