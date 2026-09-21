package sandbox

import (
	"context"
	"net/http"
	"time"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/model"
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
	// DownloadFile 下载沙箱文件，返回原始字节（二进制安全，不走 JSON 信封）
	DownloadFile(ctx context.Context, filepath string) ([]byte, error)
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

func (c *SandboxClient) ExecCommand(ctx context.Context, sessionID, execDir, command string) (*model.ToolResult, error) {
	return c.postTool(ctx, "shell/exec-command", &shellExecRequest{
		SessionID: sessionID,
		ExecDir:   execDir,
		Command:   command,
	})
}

func (c *SandboxClient) ReadShellOutput(ctx context.Context, sessionID string, console bool) (*model.ToolResult, error) {
	return c.postTool(ctx, "shell/read-shell-output", &shellReadRequest{
		SessionID: sessionID,
		Console:   console,
	})
}

func (c *SandboxClient) WaitProcess(ctx context.Context, sessionID string, seconds *int) (*model.ToolResult, error) {
	return c.postTool(ctx, "shell/wait-process", &shellWaitRequest{
		SessionID: sessionID,
		Seconds:   seconds,
	})
}

func (c *SandboxClient) WriteShellInput(ctx context.Context, sessionID, inputText string, pressEnter bool) (*model.ToolResult, error) {
	return c.postTool(ctx, "shell/write-shell-input", &shellWriteRequest{
		SessionID:  sessionID,
		InputText:  inputText,
		PressEnter: pressEnter,
	})
}

func (c *SandboxClient) KillProcess(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return c.postTool(ctx, "shell/kill-process", &shellKillRequest{SessionID: sessionID})
}

func (c *SandboxClient) WriteFile(ctx context.Context, filepath, content string, append, leadingNewline, trailingNewline, sudo bool) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/write-file", &fileWriteRequest{
		Filepath:   filepath,
		Content:    content,
		Append:     append,
		LeadingNL:  leadingNewline,
		TrailingNL: trailingNewline,
		Sudo:       sudo,
	})
}

func (c *SandboxClient) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	req := &fileReadRequest{
		Filepath:  filepath,
		StartLine: startLine,
		EndLine:   endLine,
		Sudo:      sudo,
	}
	// maxLength<=0 表示由沙箱侧使用默认上限
	if maxLength > 0 {
		req.MaxLength = &maxLength
	}
	return c.postTool(ctx, "file/read-file", req)
}

func (c *SandboxClient) CheckFileExists(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/check-file-exists", &filePathRequest{Filepath: filepath})
}

func (c *SandboxClient) DeleteFile(ctx context.Context, filepath string) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/delete-file", &fileDeleteRequest{Filepath: filepath})
}

func (c *SandboxClient) ListFiles(ctx context.Context, dirPath string) (*model.ToolResult, error) {
	return c.FindFiles(ctx, dirPath, "")
}

func (c *SandboxClient) ReplaceInFile(ctx context.Context, filepath, oldStr, newStr string, sudo bool) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/replace-in-file", &fileReplaceRequest{
		Filepath: filepath,
		OldStr:   oldStr,
		NewStr:   newStr,
		Sudo:     sudo,
	})
}

func (c *SandboxClient) SearchInFile(ctx context.Context, filepath, regex string, sudo bool) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/search-in-file", &fileSearchRequest{
		Filepath: filepath,
		Regex:    regex,
		Sudo:     sudo,
	})
}

func (c *SandboxClient) FindFiles(ctx context.Context, dirPath, globPattern string) (*model.ToolResult, error) {
	return c.postTool(ctx, "file/find-files", &fileFindRequest{
		DirPath:     dirPath,
		GlobPattern: globPattern,
	})
}

func (c *SandboxClient) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	return c.uploadFile(ctx, filepath, filename, fileData)
}

func (c *SandboxClient) DownloadFile(ctx context.Context, filepath string) ([]byte, error) {
	return c.downloadBytes(ctx, filepath)
}

func (c *SandboxClient) HealthCheck(ctx context.Context) error {
	if err := c.requireAddress(); err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
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
