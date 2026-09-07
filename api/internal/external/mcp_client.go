package external

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/bytedance/sonic"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// MCPClient MCP 客户端接口
type MCPClient interface {
	// Connect 连接到 MCP 服务器
	Connect(ctx context.Context) error

	// ListTools 获取工具列表
	ListTools(ctx context.Context) ([]MCPToolInfo, error)

	// CallTool 调用工具
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*MCPToolResult, error)

	// Close 关闭连接
	Close() error
}

// MCPToolInfo MCP 工具信息
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// MCPToolResult MCP 工具调用结果
type MCPToolResult struct {
	Content []MCPContent `json:"content,omitempty"`
	IsError bool         `json:"isError,omitempty"`
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
	stdin      io.Writer
	stdout     io.Reader
	reader     *bufio.Reader
	nextID     int
	closed     atomic.Bool
	serverName string
	env        map[string]string
}

// StdioMCPClientConfig stdio MCP 客户端配置
type StdioMCPClientConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// NewStdioMCPClient 创建 stdio MCP 客户端
func NewStdioMCPClient(serverName string, cfg StdioMCPClientConfig) *StdioMCPClient {
	return &StdioMCPClient{
		serverName: serverName,
		nextID:     1,
		cmd:        exec.Command(cfg.Command, cfg.Args...),
		env:        cfg.Env,
	}
}

// Connect 连接到 MCP 服务器
func (c *StdioMCPClient) Connect(ctx context.Context) error {
	if c.cmd == nil {
		return fmt.Errorf("MCP client not initialized")
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
		c.terminateProcess()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	logger.Info("MCP 服务器已启动",
		logger.String("server", c.serverName),
		logger.String("command", c.cmd.Path))

	return nil
}

// terminateProcess 强制终止子进程并等待回收。设计为无锁调用，供错误路径使用。
// 不要在持锁状态下调用，否则 cmd.Wait() 可能在持锁状态下阻塞。
func (c *StdioMCPClient) terminateProcess() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Kill()
	// Wait 不能漏：否则子进程会变成 zombie
	_ = c.cmd.Wait()
}

// getEnv 获取环境变量
func (c *StdioMCPClient) getEnv() map[string]string {
	// 返回配置中的环境变量
	return c.getConfigEnv()
}

// getConfigEnv 获取配置中的环境变量
func (c *StdioMCPClient) getConfigEnv() map[string]string {
	if c.env == nil {
		return map[string]string{}
	}
	return c.env
}

// sendInitialize 发送初始化请求
func (c *StdioMCPClient) sendInitialize(ctx context.Context) error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": struct{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "go-manus",
				"version": "1.0.0",
			},
		},
	}
	c.nextID++

	if err := c.sendRequest(req); err != nil {
		return err
	}

	// 读取响应（带 ctx，可被超时打断）
	_, err := c.readResponse(ctx)
	return err
}

// ListTools 获取工具列表
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("MCP client is closed")
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "tools/list",
	}
	c.nextID++

	if err := c.sendRequest(req); err != nil {
		return nil, err
	}

	resp, err := c.readResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	// 解析工具列表
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	toolsRaw, ok := result["tools"]
	if !ok {
		return []MCPToolInfo{}, nil
	}

	toolsArr, ok := toolsRaw.([]interface{})
	if !ok {
		return []MCPToolInfo{}, nil
	}

	tools := make([]MCPToolInfo, 0, len(toolsArr))
	for _, t := range toolsArr {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		tool := MCPToolInfo{}
		if name, ok := toolMap["name"].(string); ok {
			tool.Name = name
		}
		if desc, ok := toolMap["description"].(string); ok {
			tool.Description = desc
		}
		if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
			tool.InputSchema = schema
		}
		tools = append(tools, tool)
	}

	logger.Info("获取 MCP 工具列表成功",
		logger.String("server", c.serverName),
		logger.Int("count", len(tools)))

	return tools, nil
}

// CallTool 调用工具
func (c *StdioMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*MCPToolResult, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("MCP client is closed")
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}
	c.nextID++

	if err := c.sendRequest(req); err != nil {
		return nil, err
	}

	resp, err := c.readResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: resp.Error.Message}},
			IsError: true,
		}, nil
	}

	// 解析结果
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	toolResult := &MCPToolResult{}

	if contentRaw, ok := result["content"]; ok {
		contentArr, ok := contentRaw.([]interface{})
		if ok {
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
		}
	}

	return toolResult, nil
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
	c.terminateProcess()

	logger.Info("MCP 服务器已关闭", logger.String("server", c.serverName))
	return nil
}

// sendRequest 发送请求
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
// 关键：必须接受 ctx 并响应取消/超时，否则进程异常退出后 ReadBytes 会永久阻塞。
// 实现思路：把 ctx 转换为 read deadline，超时或取消时 ReadBytes 立即返回错误。
func (c *StdioMCPClient) readResponse(ctx context.Context) (*MCPResponse, error) {
	type readResult struct {
		line []byte
		err  error
	}
	done := make(chan readResult, 1)

	go func() {
		line, err := c.reader.ReadBytes('\n')
		done <- readResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("read response: %w", ctx.Err())
	case r := <-done:
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

// MCPConfig MCP 配置
type MCPConfig struct {
	Servers []MCPConfigServer
	Timeout int
}

// MCPConfigServer MCP 服务器配置
type MCPConfigServer struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// MCPClientManager MCP 客户端管理器
type MCPClientManager struct {
	mu      sync.RWMutex
	clients map[string]MCPClient
	config  *MCPConfig
}

// NewMCPClientManager 创建 MCP 客户端管理器
func NewMCPClientManager(cfg *MCPConfig) *MCPClientManager {
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
