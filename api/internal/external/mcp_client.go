package external

import (
	"bufio"
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"os"
	"os/exec"
	"sync"

	"go.uber.org/zap"

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
type StdioMCPClient struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.Writer
	stdout     io.Reader
	reader     *bufio.Reader
	nextID     int
	closed     bool
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
	c.mu.Lock()
	defer c.mu.Unlock()

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

	logger.Info("MCP 服务器已启动",
		zap.String("server", c.serverName),
		zap.String("command", c.cmd.Path))

	// 发送 initialize 请求
	if err := c.sendInitialize(ctx); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	return nil
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

	// 读取响应
	_, err := c.readResponse()
	return err
}

// ListTools 获取工具列表
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
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

	resp, err := c.readResponse()
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
		zap.String("server", c.serverName),
		zap.Int("count", len(tools)))

	return tools, nil
}

// CallTool 调用工具
func (c *StdioMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*MCPToolResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
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

	resp, err := c.readResponse()
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
func (c *StdioMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	logger.Info("MCP 服务器已关闭", zap.String("server", c.serverName))
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

// readResponse 读取响应
func (c *StdioMCPClient) readResponse() (*MCPResponse, error) {
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp MCPResponse
	if err := sonic.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
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
				zap.String("server", server.Name),
				zap.Error(err))
			continue
		}

		m.clients[server.Name] = client
		logger.Info("MCP 服务器连接成功", zap.String("server", server.Name))
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
				zap.String("server", name),
				zap.Error(err))
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
				zap.String("server", name),
				zap.Error(err))
		}
	}

	m.clients = make(map[string]MCPClient)
	return nil
}
