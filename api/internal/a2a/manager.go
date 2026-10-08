package a2a

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// ErrA2AInitializationSuperseded 表示初始化结果已被后续清理操作废弃。
var ErrA2AInitializationSuperseded = errors.New("A2A 初始化已被清理操作废弃")

// A2AServerConfig A2A 服务器配置。
type A2AServerConfig struct {
	ID      string // 唯一标识
	BaseURL string // 服务器基础 URL
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
	initMu      sync.Mutex
	client      *A2AClient
	agents      map[string]*A2ARemoteAgent
	initialized bool
	generation  uint64
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

	m.mu.RLock()
	startGeneration := m.generation
	m.mu.RUnlock()

	m.initMu.Lock()
	defer m.initMu.Unlock()

	m.mu.RLock()
	if m.initialized {
		m.mu.RUnlock()
		return nil
	}
	if m.generation != startGeneration {
		m.mu.RUnlock()
		return ErrA2AInitializationSuperseded
	}
	client := m.client
	m.mu.RUnlock()

	if config.Timeout > 0 {
		client = NewA2AClient(config.Timeout)
	}

	logger.Info(fmt.Sprintf("加载 %d 个 A2A 服务", len(config.Servers)))

	agents := make(map[string]*A2ARemoteAgent, len(config.Servers))
	for _, server := range config.Servers {
		remote, err := m.fetchAndResolve(ctx, client, server)
		if err != nil {
			return fmt.Errorf("加载 A2A 服务 [%s] 失败: %w", server.ID, err)
		}
		agents[server.ID] = remote
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generation != startGeneration {
		return ErrA2AInitializationSuperseded
	}
	m.client = client
	m.agents = agents
	m.initialized = true

	logger.Info("A2A 客户端加载成功")
	return nil
}

// fetchAndResolve 拉取单个 Agent 卡片并解析端点。
func (m *A2AClientManager) fetchAndResolve(ctx context.Context, client *A2AClient, server A2AServerConfig) (*A2ARemoteAgent, error) {
	baseURL := trimTrailingSlash(server.BaseURL)
	card, err := m.fetchAgentCard(ctx, client, baseURL)
	if err != nil {
		return nil, err
	}

	endpoint, _, err := card.ResolveEndpoint()
	if err != nil {
		return nil, err
	}

	return &A2ARemoteAgent{Card: card, Endpoint: endpoint}, nil
}

// fetchAgentCard 从远程服务器获取 Agent Card。
func (m *A2AClientManager) fetchAgentCard(ctx context.Context, client *A2AClient, baseURL string) (*A2AAgentCard, error) {
	url := baseURL + a2aAgentCardPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.httpClient.Do(req)
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
		result[id] = cloneAgentCard(remote.Card)
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
	client := m.client
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

	result, err := client.SendMessage(ctx, endpoint, message)
	if err != nil {
		return nil, err
	}

	// 无状态直接消息（kind=message）无需轮询。
	if result.Task == nil {
		return result, nil
	}

	// 有任务则轮询直到终态。
	task, err := m.pollUntilSettled(ctx, client, endpoint, result.Task)
	if err != nil {
		return nil, err
	}
	return &A2AResult{Task: task}, nil
}

// pollUntilSettled 轮询 tasks/get 直到任务进入终态或需要人工输入。
func (m *A2AClientManager) pollUntilSettled(ctx context.Context, client *A2AClient, endpoint string, task *A2ATask) (*A2ATask, error) {
	pollCtx, cancel := context.WithTimeout(ctx, a2aPollTimeout)
	defer cancel()

	current := task
	for !isTerminalA2AState(current.Status.State) {
		if current.Status.State == a2aTaskStateInputRequired || current.Status.State == a2aTaskStateAuthRequired {
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

		next, err := client.GetTask(pollCtx, endpoint, current.ID)
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
	client := m.client
	if ok {
		endpoint = remote.Endpoint
	}
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("该远程 Agent 不存在")
	}
	return client.CancelTask(ctx, endpoint, taskID)
}

// Cleanup 清理资源。
func (m *A2AClientManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents = make(map[string]*A2ARemoteAgent)
	m.initialized = false
	m.generation++

	logger.Info("清除 A2A 客户端管理器成功")
	return nil
}

func cloneAgentCard(card *A2AAgentCard) *A2AAgentCard {
	if card == nil {
		return nil
	}
	cloned := *card
	cloned.SupportedInterfaces = append([]A2AInterface(nil), card.SupportedInterfaces...)
	cloned.Skills = append([]A2AAgentSkill(nil), card.Skills...)
	if card.Capabilities != nil {
		capabilities := *card.Capabilities
		cloned.Capabilities = &capabilities
	}
	cloned.Metadata = cloneJSONMap(card.Metadata)
	return &cloned
}

func cloneJSONMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = cloneJSONValue(value)
	}
	return cloned
}

func cloneJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return cloneJSONMap(typed)
	case []interface{}:
		cloned := make([]interface{}, len(typed))
		for i, item := range typed {
			cloned[i] = cloneJSONValue(item)
		}
		return cloned
	default:
		return value
	}
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
