package agent

import "github.com/Huang131/go-manus/api/internal/model"

// AgentConfig Agent 配置
type AgentConfig struct {
	// MaxIterations 最大迭代次数
	MaxIterations int `json:"max_iterations"`
	// MaxRetries LLM 调用失败时的最大重试次数
	MaxRetries int `json:"max_retries"`
	// MaxSearchResults 搜索工具返回的最大结果数
	MaxSearchResults int `json:"max_search_results"`
}

// DefaultAgentConfig 返回默认 Agent 配置
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxIterations:    10,
		MaxRetries:       3,
		MaxSearchResults: 10,
	}
}

// NormalizeAgentConfig fills missing runtime values from the defaults.
// It returns a copy so callers can safely retain or mutate their input.
func NormalizeAgentConfig(cfg *AgentConfig) *AgentConfig {
	defaults := DefaultAgentConfig()
	if cfg == nil {
		return defaults
	}
	normalized := *cfg
	if normalized.MaxIterations <= 0 {
		normalized.MaxIterations = defaults.MaxIterations
	}
	if normalized.MaxRetries <= 0 {
		normalized.MaxRetries = defaults.MaxRetries
	}
	if normalized.MaxSearchResults <= 0 {
		normalized.MaxSearchResults = defaults.MaxSearchResults
	}
	return &normalized
}

// MCPConfig MCP 配置
type MCPConfig struct {
	// Servers MCP 服务器列表
	Servers []MCPServer `json:"servers"`
}

// MCPServer MCP 服务器配置
type MCPServer struct {
	// Name 服务器名称
	Name string `json:"name"`
	// Command 启动命令
	Command string `json:"command"`
	// Args 命令参数
	Args []string `json:"args"`
	// Env 环境变量
	Env map[string]string `json:"env"`
}

// A2AConfig A2A 配置
type A2AConfig struct {
	// Agents A2A Agent 列表
	Agents []A2AAgent `json:"agents"`
}

// A2AAgent A2A Agent 配置
type A2AAgent struct {
	// Name Agent 名称
	Name string `json:"name"`
	// URL Agent 服务地址
	URL string `json:"url"`
}

// RuntimeMCPConfig converts the persisted control-plane model into an isolated
// runtime snapshot. Disabled servers never reach the client manager.
func RuntimeMCPConfig(cfg *model.MCPConfig) *MCPConfig {
	if cfg == nil {
		return nil
	}
	runtimeCfg := &MCPConfig{Servers: make([]MCPServer, 0, len(cfg.Servers))}
	for _, server := range cfg.Servers {
		if !server.Enabled {
			continue
		}
		runtimeCfg.Servers = append(runtimeCfg.Servers, MCPServer{
			Name:    server.ServerName,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     cloneStringMap(server.Env),
		})
	}
	return runtimeCfg
}

// RuntimeA2AConfig converts persisted A2A settings into an isolated runtime snapshot.
func RuntimeA2AConfig(cfg *model.A2AConfig) *A2AConfig {
	if cfg == nil {
		return nil
	}
	runtimeCfg := &A2AConfig{Agents: make([]A2AAgent, 0, len(cfg.Servers))}
	for _, server := range cfg.Servers {
		if !server.Enabled {
			continue
		}
		runtimeCfg.Agents = append(runtimeCfg.Agents, A2AAgent{
			Name: server.ID,
			URL:  server.URL,
		})
	}
	return runtimeCfg
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
