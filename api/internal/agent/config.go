package agent

// AgentConfig Agent 配置
type AgentConfig struct {
	// MaxSteps 最大执行步骤数
	MaxSteps int `json:"max_steps"`
	// MaxIterations 最大迭代次数
	MaxIterations int `json:"max_iterations"`
	// MaxRetries LLM 调用失败时的最大重试次数
	MaxRetries int `json:"max_retries"`
	// MaxMemorySize 最大记忆大小 (token)
	MaxMemorySize int `json:"max_memory_size"`
	// CompactThreshold 压缩阈值 (记忆超过此大小时触发压缩)
	CompactThreshold int `json:"compact_threshold"`
	// PlanningPrompt 规划提示词
	PlanningPrompt string `json:"planning_prompt"`
	// ExecutionPrompt 执行提示词
	ExecutionPrompt string `json:"execution_prompt"`
	// SummarizationPrompt 总结提示词
	SummarizationPrompt string `json:"summarization_prompt"`
	// SummaryMaxLength 总结最大长度
	SummaryMaxLength int `json:"summary_max_length"`
}

// DefaultAgentConfig 返回默认 Agent 配置
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxSteps:            50,
		MaxIterations:       10,
		MaxRetries:          3,
		MaxMemorySize:       128000,
		CompactThreshold:    100000,
		PlanningPrompt:      "你是一个任务规划助手。根据用户的需求，创建一个分步骤的执行计划。",
		ExecutionPrompt:     "你是一个任务执行助手。按照计划步骤逐步执行任务。",
		SummarizationPrompt: "你是一个总结助手。请总结整个任务的执行结果。",
		SummaryMaxLength:    2000,
	}
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
