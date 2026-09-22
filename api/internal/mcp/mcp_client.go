package mcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/bytedance/sonic"
)

// MCP 客户端错误
var (
	ErrClientClosed         = errors.New("MCP client is closed")
	ErrClientNotInitialized = errors.New("MCP client not initialized")
	ErrInvalidResponse      = errors.New("invalid response format")
	// 响应 channel 已关闭
	ErrResponseChannelClosed = errors.New("response channel closed")
)

// MCPServerError 服务器返回的错误
type MCPServerError struct {
	Code    int
	Message string
}

func (e *MCPServerError) Error() string {
	return fmt.Sprintf("MCP server error [%d]: %s", e.Code, e.Message)
}

// MCPClient MCP 客户端接口
type MCPClient interface {
	// 连接到 MCP 服务器
	Connect(ctx context.Context) error

	// 获取工具列表
	ListTools(ctx context.Context) ([]MCPToolInfo, error)

	// 调用工具
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*MCPToolResult, error)

	// 关闭连接
	Close() error
}

// MCP 工具信息
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// MCP 工具调用结果
type MCPToolResult struct {
	Content []MCPContent `json:"content,omitempty"`
	IsError bool         `json:"isError,omitempty"`
}

// 将工具调用错误转换为标准错误
func (r *MCPToolResult) MCPToolError() error {
	if !r.IsError || len(r.Content) == 0 {
		return nil
	}
	return errors.New(r.Content[0].Text)
}

// MCPContent MCP 内容项
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// StdioMCPClient 基于 stdio 的 MCP 客户端
//
// 并发模型：
//   - closed 用 atomic.Bool 管理，Close 幂等自治，不持锁
//   - cmd/reader/stdin 在 Connect 成功后只读，Close 后资源不可用
//   - 不需要 sync.Mutex，所有并发安全由 atomic + 资源回收顺序保证
type StdioMCPClient struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	reader     *bufio.Reader
	nextID     int
	mu         sync.Mutex // 串行化 ID 分配与请求-响应读取（共享 bufio 不允许并发）
	closed     atomic.Bool
	serverName string
	env        map[string]string
}

// stdio MCP 客户端配置
type StdioMCPClientConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// 创建 stdio MCP 客户端
func NewStdioMCPClient(serverName string, cfg StdioMCPClientConfig) *StdioMCPClient {
	return &StdioMCPClient{
		serverName: serverName,
		nextID:     1,
		cmd:        exec.Command(cfg.Command, cfg.Args...),
		env:        cfg.Env,
	}
}

// 连接到 MCP 服务器
func (c *StdioMCPClient) Connect(ctx context.Context) error {
	if c.cmd == nil {
		return ErrClientNotInitialized
	}

	// 设置环境变量
	if c.cmd.Env == nil {
		c.cmd.Env = os.Environ()
	}
	for k, v := range c.getEnv() {
		c.cmd.Env = append(c.cmd.Env, k+"="+v)
	}

	// 创建管道用于 stdin
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// 创建管道用于 stdout
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		stdinReader.Close()
		stdinWriter.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// 设置进程输入输出
	c.cmd.Stdin = stdinReader
	c.cmd.Stdout = stdoutWriter
	c.cmd.Stderr = os.Stderr

	// 启动进程
	if err := c.cmd.Start(); err != nil {
		stdinReader.Close()
		stdinWriter.Close()
		stdoutReader.Close()
		stdoutWriter.Close()
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// 关闭读取端
	stdinReader.Close()
	stdoutWriter.Close()

	c.stdin = stdinWriter
	c.stdout = stdoutReader
	c.reader = bufio.NewReader(stdoutReader)

	// 发送 initialize 请求
	if err := c.sendInitialize(ctx); err != nil {
		// 错误路径：标记 closed + 回收进程。不调 c.Close()，避免重复清理
		c.closed.Store(true)
		c.cleanup()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	logger.Info("MCP 服务器已启动",
		logger.String("server", c.serverName),
		logger.String("command", c.cmd.Path))

	return nil
}

// cleanup 清理所有资源（内部使用，不检查 closed 状态）
func (c *StdioMCPClient) cleanup() {
	// 关闭 stdin
	if c.stdin != nil {
		c.stdin.Close()
		c.stdin = nil
	}

	// 关闭 stdout
	if c.stdout != nil {
		c.stdout.Close()
		c.stdout = nil
	}

	// 终止进程
	c.terminateProcess()
}

// terminateProcess 强制终止子进程并等待回收。
// 设计为无锁调用，供错误路径使用。
// 不要在持锁状态下调用，否则 cmd.Wait() 可能在持锁状态下阻塞。
func (c *StdioMCPClient) terminateProcess() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Kill()
	// Wait 不能漏：否则子进程会变成 zombie
	_ = c.cmd.Wait()
	c.cmd = nil
}

// 返回配置中的环境变量
func (c *StdioMCPClient) getEnv() map[string]string {
	if c.env == nil {
		return map[string]string{}
	}
	return c.env
}

// Close 关闭连接
//
// 幂等设计：用 atomic.Bool.CompareAndSwap 保证只清理一次。
// 不持锁，可从任何 goroutine 调用（包括持锁代码路径），不会自递归死锁。
func (c *StdioMCPClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // 已关闭，幂等返回
	}

	// cmd.Wait() 在 CAS 之后无锁调用，可以安全阻塞
	c.cleanup()

	logger.Info("MCP 服务器已关闭", logger.String("server", c.serverName))
	return nil
}

// 在串行锁内分配请求 ID、发送一条 JSON-RPC 请求并读取一条响应，
// 供 initialize / ListTools / CallTool 复用，消除重复的请求-响应样板。
func (c *StdioMCPClient) call(ctx context.Context, method string, params map[string]interface{}) (*MCPResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := MCPRequest{
		JSONRPC: mcpJSONRPCVersion,
		ID:      c.nextID,
		Method:  method,
		Params:  params,
	}
	c.nextID++

	if err := c.sendRequest(req); err != nil {
		return nil, err
	}
	return c.readResponse(ctx)
}

// 发送请求
func (c *StdioMCPClient) sendRequest(req MCPRequest) error {
	data, err := sonic.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// readResponse 读取响应。
// - 使用 goroutineDone channel 通知读取 goroutine 退出
// - ctx 取消时，关闭 goroutineDone，读取 goroutine 可安全退出
// - 避免 ctx 取消后，读取 goroutine 永久阻塞
func (c *StdioMCPClient) readResponse(ctx context.Context) (*MCPResponse, error) {
	type readResult struct {
		line []byte
		err  error
	}

	done := make(chan readResult, 1)
	goroutineDone := make(chan struct{}) // 用于通知读取 goroutine 退出

	go func() {
		line, err := c.reader.ReadBytes('\n')
		select {
		case <-goroutineDone:
			// 已被通知退出，不发送结果
			return
		case done <- readResult{line: line, err: err}:
			// 结果已发送，等待 goroutineDone 关闭后退出
			<-goroutineDone
		}
	}()

	select {
	case <-ctx.Done():
		close(goroutineDone) // 通知读取 goroutine 退出
		return nil, fmt.Errorf("read response: %w", ctx.Err())

	case r, ok := <-done:
		close(goroutineDone) // 通知读取 goroutine 可以安全退出
		if !ok {
			return nil, ErrResponseChannelClosed
		}
		if r.err != nil {
			return nil, fmt.Errorf("failed to read response: %w", r.err)
		}

		var resp MCPResponse
		if err := sonic.Unmarshal(r.line, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return &resp, nil
	}
}

// sendInitialize 发送初始化请求
func (c *StdioMCPClient) sendInitialize(ctx context.Context) error {
	_, err := c.call(ctx, mcpMethodInitialize, map[string]interface{}{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": struct{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    mcpClientName,
			"version": mcpClientVersion,
		},
	})
	return err
}

// ListTools 获取工具列表
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	resp, err := c.call(ctx, mcpMethodListTools, nil)
	if err != nil {
		return nil, err
	}

	// 统一错误处理：MCP 服务器错误返回 error
	if resp.Error != nil {
		return nil, &MCPServerError{
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
		}
	}

	// 解析工具列表
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidResponse
	}

	tools, err := parseToolInfos(result)
	if err != nil {
		return nil, err
	}

	logger.Info("获取 MCP 工具列表成功",
		logger.String("server", c.serverName),
		logger.Int("count", len(tools)))

	return tools, nil
}

// 调用工具
func (c *StdioMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*MCPToolResult, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	resp, err := c.call(ctx, mcpMethodCallTool, map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}

	// 统一错误处理：MCP 服务器错误返回 error
	if resp.Error != nil {
		return nil, &MCPServerError{
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
		}
	}

	// 解析结果
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidResponse
	}

	return parseToolResult(result), nil
}

// 解析工具列表
func parseToolInfos(result map[string]interface{}) ([]MCPToolInfo, error) {
	toolsRaw, ok := result["tools"]
	if !ok {
		return []MCPToolInfo{}, nil
	}

	toolsArr, ok := toolsRaw.([]interface{})
	if !ok {
		return nil, ErrInvalidResponse
	}

	tools := make([]MCPToolInfo, 0, len(toolsArr))
	for _, t := range toolsArr {
		tool, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		info := MCPToolInfo{}
		if name, ok := tool["name"].(string); ok {
			info.Name = name
		}
		if desc, ok := tool["description"].(string); ok {
			info.Description = desc
		}
		if schema, ok := tool["inputSchema"].(map[string]interface{}); ok {
			info.InputSchema = schema
		}
		tools = append(tools, info)
	}

	return tools, nil
}

// 解析工具调用结果
func parseToolResult(result map[string]interface{}) *MCPToolResult {
	toolResult := &MCPToolResult{
		Content: []MCPContent{}, // 确保始终返回空 slice 而非 nil
	}

	contentRaw, ok := result["content"]
	if !ok {
		return toolResult
	}

	contentArr, ok := contentRaw.([]interface{})
	if !ok {
		return toolResult
	}

	for _, item := range contentArr {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		content := MCPContent{}
		if t, ok := itemMap["type"].(string); ok {
			content.Type = t
		}
		if text, ok := itemMap["text"].(string); ok {
			content.Text = text
		}
		toolResult.Content = append(toolResult.Content, content)
	}

	return toolResult
}

// ============================================================================
// JSON-RPC 类型
// ============================================================================

// MCPRequest MCP JSON-RPC 请求
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse MCP JSON-RPC 响应
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError MCP 错误
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ============================================================================
// MCPClientManager 实现
// ============================================================================

// MCPClientManager MCP 客户端管理器
type MCPClientManager struct {
	mu      sync.RWMutex
	clients map[string]MCPClient
	config  *model.MCPConfig
}

// NewMCPClientManager 创建 MCP 客户端管理器
func NewMCPClientManager(cfg *model.MCPConfig) *MCPClientManager {
	return &MCPClientManager{
		clients: make(map[string]MCPClient),
		config:  cfg,
	}
}

// Initialize 初始化所有 MCP 客户端
func (m *MCPClientManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return nil
	}

	for _, server := range m.config.Servers {
		client := NewStdioMCPClient(server.Name, StdioMCPClientConfig{
			Command: server.Command,
			Args:    server.Args,
			Env:     server.Env,
		})

		if err := client.Connect(ctx); err != nil {
			logger.Warn("连接 MCP 服务器失败，跳过",
				logger.String("server", server.Name),
				logger.Err(err))
			continue
		}

		m.clients[server.Name] = client
		logger.Info("MCP 服务器连接成功", logger.String("server", server.Name))
	}

	return nil
}

// GetClient 获取 MCP 客户端
func (m *MCPClientManager) GetClient(name string) (MCPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[name]
	return client, ok
}

// ListAllTools 获取所有 MCP 服务器的工具
func (m *MCPClientManager) ListAllTools(ctx context.Context) (map[string][]MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]MCPToolInfo)

	for name, client := range m.clients {
		tools, err := client.ListTools(ctx)
		if err != nil {
			logger.Warn("获取 MCP 工具列表失败",
				logger.String("server", name),
				logger.Err(err))
			continue
		}
		result[name] = tools
	}

	return result, nil
}

// Close 关闭所有 MCP 客户端
func (m *MCPClientManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			logger.Warn("关闭 MCP 客户端失败",
				logger.String("server", name),
				logger.Err(err))
		}
	}

	m.clients = make(map[string]MCPClient)
	return nil
}
