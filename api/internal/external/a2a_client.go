package external

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// A2AAgentCard A2A Agent 卡片信息
// 对应 Python 版本的 agent_card.json
type A2AAgentCard struct {
	// Name Agent 名称
	Name string `json:"name"`
	// Description Agent 描述
	Description string `json:"description,omitempty"`
	// URL Agent 调用端点
	URL string `json:"url,omitempty"`
	// Version Agent 版本
	Version string `json:"version,omitempty"`
	// Capabilities Agent 能力
	Capabilities *A2AAgentCapabilities `json:"capabilities,omitempty"`
	// Skills Agent 技能列表
	Skills []A2AAgentSkill `json:"skills,omitempty"`
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Enabled 是否启用
	Enabled bool `json:"enabled,omitempty"`
}

// A2AAgentCapabilities Agent 能力
type A2AAgentCapabilities struct {
	// Streaming 是否支持流式
	Streaming bool `json:"streaming,omitempty"`
	// PushNotifications 是否支持推送通知
	PushNotifications bool `json:"pushNotifications,omitempty"`
}

// A2AAgentSkill Agent 技能
type A2AAgentSkill struct {
	// ID 技能 ID
	ID string `json:"id,omitempty"`
	// Name 技能名称
	Name string `json:"name,omitempty"`
	// Description 技能描述
	Description string `json:"description,omitempty"`
}

// A2AClientManager A2A 客户端管理器
// 参考 Python 版本的 A2AClientManager
type A2AClientManager struct {
	mu          sync.RWMutex
	httpClient  *http.Client
	agentCards  map[string]*A2AAgentCard
	initialized bool
	baseURLs    map[string]string // agent id -> base URL
}

// A2AClientManagerConfig A2A 客户端管理器配置
type A2AClientManagerConfig struct {
	// Servers A2A 服务器列表
	Servers []A2AServerConfig
	// Timeout 超时时间
	Timeout time.Duration
}

// A2AServerConfig A2A 服务器配置
type A2AServerConfig struct {
	// ID 唯一标识
	ID string
	// BaseURL 服务器基础 URL
	BaseURL string
	// Enabled 是否启用
	Enabled bool
}

// NewA2AClientManager 创建 A2A 客户端管理器
func NewA2AClientManager() *A2AClientManager {
	return &A2AClientManager{
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // 与 Python 版本一致，10 分钟超时
		},
		agentCards: make(map[string]*A2AAgentCard),
		baseURLs:   make(map[string]string),
	}
}

// Initialize 初始化 A2A 客户端，获取所有远程 Agent 的卡片信息
func (m *A2AClientManager) Initialize(ctx context.Context, config *A2AClientManagerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	if config == nil {
		return nil
	}

	logger.Info(fmt.Sprintf("加载 %d 个 A2A 服务", len(config.Servers)))

	// 获取所有 A2A 服务器的 AgentCard
	for _, server := range config.Servers {
		card, err := m.fetchAgentCard(ctx, server.BaseURL)
		if err != nil {
			logger.Warn(fmt.Sprintf("加载 A2A 服务 [%s] 失败: %v", server.ID, err))
			continue
		}

		card.Enabled = server.Enabled
		m.agentCards[server.ID] = card
		m.baseURLs[server.ID] = server.BaseURL
	}

	m.initialized = true
	logger.Info("A2A 客户端加载成功")
	return nil
}

// fetchAgentCard 从远程服务器获取 AgentCard
func (m *A2AClientManager) fetchAgentCard(ctx context.Context, baseURL string) (*A2AAgentCard, error) {
	url := baseURL + "/.well-known/agent-card.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var card A2AAgentCard
	if err := sonic.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("解析 AgentCard 失败: %w", err)
	}

	return &card, nil
}

// GetAgentCards 获取所有 Agent 卡片信息
func (m *A2AClientManager) GetAgentCards() map[string]*A2AAgentCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*A2AAgentCard)
	for id, card := range m.agentCards {
		result[id] = card
	}
	return result
}

// Invoke 调用远程 Agent
// 参考 Python 版本的 A2AClientManager.invoke()
func (m *A2AClientManager) Invoke(ctx context.Context, agentID string, query string) (*A2AInvokeResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 检查 agent 是否存在
	card, ok := m.agentCards[agentID]
	if !ok {
		return nil, fmt.Errorf("该远程 Agent 不存在")
	}

	// 2. 检查端点是否存在
	url := card.URL
	if url == "" {
		return nil, fmt.Errorf("该远程 Agent 调用端点不存在")
	}

	// 3. 构建 A2A JSON-RPC 2.0 请求
	request := A2AJSONRPCRequest{
		ID:      uuid.New().String(),
		JSONRPC: "2.0",
		Method:  "message/send",
		Params: A2AMessageSendParams{
			Message: A2AMessage{
				MessageID: uuid.New().String(),
				Role:      "user",
				Parts: []A2AMessagePart{
					{
						Kind: "text",
						Text: query,
					},
				},
			},
		},
	}

	// 序列化请求
	reqBody, err := sonic.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	logger.Info("调用远程 Agent",
		zap.String("agent_id", agentID),
		zap.String("url", url))

	// 4. 发送请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用远程 Agent 失败: %w", err)
	}
	defer resp.Body.Close()

	// 5. 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("调用远程 Agent 失败",
			zap.String("agent_id", agentID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return nil, fmt.Errorf("调用远程 Agent [%s:%s] 出错: HTTP %d", agentID, url, resp.StatusCode)
	}

	// 6. 解析响应
	var rpcResp A2AJSONRPCResponse
	if err := sonic.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 7. 提取结果
	var result *A2AInvokeResult
	if rpcResp.Result != nil {
		result = &A2AInvokeResult{
			Success: true,
			Data:    rpcResp.Result,
		}
	} else if rpcResp.Error != nil {
		result = &A2AInvokeResult{
			Success: false,
			Message: fmt.Sprintf("A2A 错误: %s - %s", rpcResp.Error.Code, rpcResp.Error.Message),
		}
	}

	return result, nil
}

// Cleanup 清理资源
func (m *A2AClientManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agentCards = make(map[string]*A2AAgentCard)
	m.baseURLs = make(map[string]string)
	m.initialized = false

	logger.Info("清除 A2A 客户端管理器成功")
	return nil
}

// A2AJSONRPCRequest A2A JSON-RPC 2.0 请求
type A2AJSONRPCRequest struct {
	ID      string               `json:"id"`
	JSONRPC string               `json:"jsonrpc"`
	Method  string               `json:"method"`
	Params  A2AMessageSendParams `json:"params"`
}

// A2AMessageSendParams 消息发送参数
type A2AMessageSendParams struct {
	Message A2AMessage `json:"message"`
}

// A2AMessage A2A 消息
type A2AMessage struct {
	MessageID string           `json:"messageId"`
	Role      string           `json:"role"`
	Parts     []A2AMessagePart `json:"parts"`
}

// A2AMessagePart 消息部分
type A2AMessagePart struct {
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
}

// A2AJSONRPCResponse A2A JSON-RPC 2.0 响应
type A2AJSONRPCResponse struct {
	ID      string           `json:"id"`
	JSONRPC string           `json:"jsonrpc"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *A2AJSONRPCError `json:"error,omitempty"`
}

// A2AJSONRPCError JSON-RPC 错误
type A2AJSONRPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// A2AInvokeResult A2A 调用结果
type A2AInvokeResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
