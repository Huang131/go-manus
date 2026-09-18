package agent

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
	// Timeout 超时时间 (秒)
	Timeout int `json:"timeout"`
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
	// Timeout 超时时间 (秒)
	Timeout int `json:"timeout"`
}

// A2AAgent A2A Agent 配置
type A2AAgent struct {
	// Name Agent 名称
	Name string `json:"name"`
	// URL Agent 服务地址
	URL string `json:"url"`
	// Metadata 元数据
	Metadata map[string]string `json:"metadata"`
}
