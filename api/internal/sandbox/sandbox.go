package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/httpconst"
)

// Sandbox 沙箱服务接口
type Sandbox interface {
	// ExecCommand 执行 Shell 命令
	ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error)
	// ReadShellOutput 读取 Shell 输出
	ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error)
	// WaitProcess 等待进程执行完成
	WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error)
	// WriteShellInput 向 Shell 写入输入
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
	// FindFiles 按 glob 查找文件
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

func NewSandboxClient(cfg *config.SandboxConfig) *SandboxClient {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(config.DefaultSandboxHTTPTimeoutSec) * time.Second
	}
	return &SandboxClient{
		address: cfg.Address,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// doRequest 发送 JSON 信封请求到沙箱服务
func (c *SandboxClient) doRequest(ctx context.Context, method, action string, req *sandboxRequest) (*sandboxResponse, error) {
	if c.address == "" {
		return nil, fmt.Errorf("sandbox address not configured")
	}

	endpoint := fmt.Sprintf("%s/api/%s", c.address, action)

	var body io.Reader
	if req != nil {
		data, err := sonic.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)

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
		var errorResp sandboxResponse
		if err := sonic.Unmarshal(respBody, &errorResp); err == nil {
			code := errorResp.Code
			if code == 0 {
				code = resp.StatusCode
			}
			return nil, &SandboxAPIError{
				StatusCode: resp.StatusCode,
				Code:       code,
				Message:    errorResp.Msg,
				Data:       errorResp.Data,
			}
		}
		return nil, &SandboxAPIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var sandboxResp sandboxResponse
	if err := sonic.Unmarshal(respBody, &sandboxResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &sandboxResp, nil
}

// SandboxErrorStatus 从错误中提取 HTTP 状态码，便于 handler 映射响应
func SandboxErrorStatus(err error) int {
	var apiErr *SandboxAPIError
	if errors.As(err, &apiErr) && apiErr != nil {
		if apiErr.StatusCode != 0 {
			return apiErr.StatusCode
		}
		return apiErr.Code
	}
	return 0
}

func (c *SandboxClient) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/exec-command", &sandboxRequest{
		SessionID: sessionID,
		ExecDir:   execDir,
		Command:   command,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/read-shell-output", &sandboxRequest{
		SessionID: sessionID,
		Console:   console,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	req := &sandboxRequest{SessionID: sessionID}
	if seconds != nil {
		req.Seconds = *seconds
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/wait-process", req)
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/write-shell-input", &sandboxRequest{
		SessionID:  sessionID,
		InputText:  inputText,
		PressEnter: pressEnter,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "shell/kill-process", &sandboxRequest{
		SessionID: sessionID,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

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
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

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
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/check-file-exists", &sandboxRequest{
		Filepath: filepath,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodDelete, "file/delete-file", &sandboxRequest{
		Filepath: filepath,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/find-files", &sandboxRequest{
		DirPath: dirPath,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/replace-in-file", &sandboxRequest{
		Filepath: filepath,
		OldStr:   oldStr,
		NewStr:   newStr,
		Sudo:     sudo,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "file/search-in-file", &sandboxRequest{
		Filepath: filepath,
		Regex:    regex,
		Sudo:     sudo,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

func (c *SandboxClient) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	glob := globPattern
	if glob == "" {
		glob = "*"
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "file/find-files", &sandboxRequest{
		DirPath: dirPath,
		GlobPtn: glob,
	})
	if err != nil {
		return toolResultErr(err)
	}
	return sandboxToolResult(resp), nil
}

// UploadFile 上传文件到沙箱（multipart/form-data）
func (c *SandboxClient) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	if c.address == "" {
		return toolResultErr(fmt.Errorf("sandbox address not configured"))
	}

	endpoint := fmt.Sprintf("%s/api/file/upload-file", c.address)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return toolResultErr(err)
	}
	if _, err := part.Write(fileData); err != nil {
		return toolResultErr(err)
	}

	if err := writer.WriteField("filepath", filepath); err != nil {
		return toolResultErr(err)
	}

	if err := writer.Close(); err != nil {
		return toolResultErr(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return toolResultErr(err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return toolResultErr(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolResultErr(err)
	}

	if resp.StatusCode != http.StatusOK {
		return toolResultErr(&SandboxAPIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    string(respBody),
		})
	}

	var sandboxResp sandboxResponse
	if err := sonic.Unmarshal(respBody, &sandboxResp); err != nil {
		return toolResultErr(err)
	}

	return sandboxToolResult(&sandboxResp), nil
}

// DownloadFile 下载沙箱文件（直接返回二进制流，非 JSON 信封）
func (c *SandboxClient) DownloadFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	if c.address == "" {
		return toolResultErr(fmt.Errorf("sandbox address not configured"))
	}
	escapedPath := url.QueryEscape(filepath)
	endpoint := fmt.Sprintf("%s/api/file/download-file?filepath=%s", c.address, escapedPath)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return toolResultErr(err)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return toolResultErr(fmt.Errorf("download file failed: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return toolResultErr(&SandboxAPIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    fmt.Sprintf("download file failed: status=%d body=%s", resp.StatusCode, string(body)),
		})
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return toolResultErr(fmt.Errorf("read download body: %w", err))
	}
	if len(data) > maxDownloadBytes {
		return toolResultErr(fmt.Errorf("download file exceeds 64MB limit"))
	}
	return &model.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"filepath": filepath,
			"filename": filepath,
			"size":     len(data),
			"content":  string(data),
		},
	}, nil
}

// HealthCheck 健康检查
func (c *SandboxClient) HealthCheck(ctx context.Context) error {
	if c.address == "" {
		return fmt.Errorf("sandbox address not configured")
	}

	endpoint := fmt.Sprintf("%s/health", c.address)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sandbox health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &SandboxAPIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    "sandbox health check failed",
		}
	}

	return nil
}
