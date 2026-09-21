package a2a

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// A2AServerConfig A2A 服务器配置。
type A2AServerConfig struct {
	ID      string // 唯一标识
	BaseURL string // 服务器基础 URL
	Enabled bool   // 是否启用
}

// A2AClientManagerConfig A2A 客户端管理器配置。
type A2AClientManagerConfig struct {
	Servers []A2AServerConfig // A2A 服务器列表
	Timeout time.Duration     // 单个请求超时上限（可选，默认 defaultA2AHTTPTimeout）
}

// A2ARemoteAgent 已解析的远程 Agent。
type A2ARemoteAgent struct {
	Card     *A2AAgentCard // Agent 卡片
	Endpoint string        // 解析后的 JSON-RPC 调用端点
}

// A2AClientManager A2A 客户端管理器：负责卡片发现、缓存与 Agent 路由。
// 协议编码与 HTTP 交互下沉到 A2AClient。
type A2AClientManager struct {
	mu          sync.RWMutex
	client      *A2AClient
	agents      map[string]*A2ARemoteAgent
	initialized bool
}

// NewA2AClientManager 创建管理器。
func NewA2AClientManager() *A2AClientManager {
	return &A2AClientManager{
		client: NewA2AClient(defaultA2AHTTPTimeout),
		agents: make(map[string]*A2ARemoteAgent),
	}
}

// Initialize 初始化：拉取所有远程 Agent 卡片并解析调用端点。
//
// 与旧实现不同：单个 Agent 加载失败会返回错误，让调用方感知（而非静默继续）。
func (m *A2AClientManager) Initialize(ctx context.Context, config *A2AClientManagerConfig) error {
	if config == nil {
		return fmt.Errorf("A2A 配置为空")
	}

	m.mu.Lock()
	if m.initialized {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	logger.Info(fmt.Sprintf("加载 %d 个 A2A 服务", len(config.Servers)))

	agents := make(map[string]*A2ARemoteAgent, len(config.Servers))
	for _, server := range config.Servers {
		remote, err := m.fetchAndResolve(ctx, server)
		if err != nil {
			return fmt.Errorf("加载 A2A 服务 [%s] 失败: %w", server.ID, err)
		}
		agents[server.ID] = remote
	}

	m.mu.Lock()
	m.agents = agents
	m.initialized = true
	m.mu.Unlock()

	logger.Info("A2A 客户端加载成功")
	return nil
}

// fetchAndResolve 拉取单个 Agent 卡片并解析端点。
func (m *A2AClientManager) fetchAndResolve(ctx context.Context, server A2AServerConfig) (*A2ARemoteAgent, error) {
	baseURL := trimTrailingSlash(server.BaseURL)
	card, err := m.fetchAgentCard(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	endpoint, _, err := card.ResolveEndpoint()
	if err != nil {
		return nil, err
	}

	card.Enabled = server.Enabled
	return &A2ARemoteAgent{Card: card, Endpoint: endpoint}, nil
}

// fetchAgentCard 从远程服务器获取 Agent Card。
func (m *A2AClientManager) fetchAgentCard(ctx context.Context, baseURL string) (*A2AAgentCard, error) {
	url := baseURL + a2aAgentCardPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := m.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 Agent Card 失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxA2AResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var card A2AAgentCard
	if err := sonic.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("解析 Agent Card 失败: %w", err)
	}

	return &card, nil
}

// GetAgentCards 返回所有 Agent 卡片快照（不含端点，仅用于展示）。
// 返回的是引用快照，卡片内部字段为只读使用，调用方不应修改。
func (m *A2AClientManager) GetAgentCards() map[string]*A2AAgentCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*A2AAgentCard, len(m.agents))
	for id, remote := range m.agents {
		result[id] = remote.Card
	}
	return result
}

// GetAgents 返回所有远程 Agent（含解析后的调用端点）。
func (m *A2AClientManager) GetAgents() map[string]*A2ARemoteAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*A2ARemoteAgent, len(m.agents))
	for id, remote := range m.agents {
		result[id] = remote
	}
	return result
}

// Invoke 同步调用远程 Agent：先 message/send，若返回长任务则轮询 tasks/get 直到终态。
//
// 返回 *A2AResult；传输/协议错误以 error 返回，Agent 业务结果在 result 中。
func (m *A2AClientManager) Invoke(ctx context.Context, agentID string, query string) (*A2AResult, error) {
	if query == "" {
		return nil, fmt.Errorf("任务内容不能为空")
	}
	if len(query) > maxA2AMessageBytes {
		return nil, fmt.Errorf("任务内容过长（超过 %d 字节）", maxA2AMessageBytes)
	}

	// 在锁内取端点副本后立即释放锁，避免与 Cleanup 竞态（修复 A8）。
	m.mu.RLock()
	remote, ok := m.agents[agentID]
	endpoint := ""
	if ok {
		endpoint = remote.Endpoint
	}
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("该远程 Agent 不存在")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("该远程 Agent 调用端点不存在")
	}

	logger.Info("调用远程 Agent",
		logger.String("agent_id", agentID),
		logger.String("url", endpoint))

	message := A2AMessage{
		MessageID: uuid.New().String(),
		Role:      a2aRoleUser,
		Parts:     []A2APart{{Kind: a2aPartKindText, Text: query}},
	}

	result, err := m.client.SendMessage(ctx, endpoint, message)
	if err != nil {
		return nil, err
	}

	// 无状态直接消息（kind=message）无需轮询。
	if result.Task == nil {
		return result, nil
	}

	// 有任务则轮询直到终态。
	task, err := m.pollUntilSettled(ctx, endpoint, result.Task)
	if err != nil {
		return nil, err
	}
	return &A2AResult{Task: task}, nil
}

// pollUntilSettled 轮询 tasks/get 直到任务进入终态或需要人工输入。
func (m *A2AClientManager) pollUntilSettled(ctx context.Context, endpoint string, task *A2ATask) (*A2ATask, error) {
	pollCtx, cancel := context.WithTimeout(ctx, a2aPollTimeout)
	defer cancel()

	current := task
	for !isTerminalA2AState(current.Status.State) {
		if current.Status.State == a2aTaskStateInputRequired {
			break // 需要额外输入，工具无法自动补全，返回当前状态。
		}
		if current.ID == "" {
			return nil, fmt.Errorf("远程 Agent 返回的任务缺少 ID，无法轮询")
		}

		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("等待远程 Agent 完成超时: %w", pollCtx.Err())
		case <-time.After(a2aPollInterval):
		}

		next, err := m.client.GetTask(pollCtx, endpoint, current.ID)
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

// CancelTask 取消远程 Agent 任务。
func (m *A2AClientManager) CancelTask(ctx context.Context, agentID, taskID string) (*A2ATask, error) {
	m.mu.RLock()
	remote, ok := m.agents[agentID]
	endpoint := ""
	if ok {
		endpoint = remote.Endpoint
	}
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("该远程 Agent 不存在")
	}
	return m.client.CancelTask(ctx, endpoint, taskID)
}

// Cleanup 清理资源。
func (m *A2AClientManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents = make(map[string]*A2ARemoteAgent)
	m.initialized = false

	logger.Info("清除 A2A 客户端管理器成功")
	return nil
}

// isTerminalA2AState 判断任务是否处于终态。
func isTerminalA2AState(state string) bool {
	switch state {
	case a2aTaskStateCompleted, a2aTaskStateFailed, a2aTaskStateCanceled, a2aTaskStateRejected:
		return true
	default:
		return false
	}
}

// trimTrailingSlash 去除末尾斜杠。
func trimTrailingSlash(s string) string {
	if s == "" {
		return s
	}
	// 去除连续的末尾斜杠。
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
