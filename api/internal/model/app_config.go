package model

import "time"

// AppConfig 应用配置，采用 Key-Value 存储，支持多类型配置。
// ConfigType 区分配置领域（LLM/Agent/MCP/A2A），ConfigKey 区分同一领域下的多个配置。
type AppConfig struct {
	ID          string        `json:"id"`           // 唯一标识
	ConfigType  AppConfigType `json:"config_type"`  // 配置类型：llm/agent/mcp/a2a
	ConfigKey   string        `json:"config_key"`   // 配置键，如 "default" 表示默认配置
	ConfigValue interface{}   `json:"config_value"` // 配置值，JSON 格式存储具体配置结构
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// HealthStatus 应用健康检查结果
type HealthStatus struct {
	Status    HealthState                   `json:"status"`    // 整体健康状态
	Timestamp int64                         `json:"timestamp"` // 检查时间戳（Unix ms）
	Services  map[ServiceName]ServiceStatus `json:"services"`  // 各基础设施服务的健康状态
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name   ServiceName `json:"name"`
	Status HealthState `json:"status"`
	Error  string      `json:"error,omitempty"`
}

// LLMConfig LLM 配置。
// 该类型用于服务内部读取和持久化，包含敏感的 APIKey。
type LLMConfig struct {
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key,omitempty"`
	ModelName   string  `json:"model_name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// LLMConfigRequest LLM 配置写入请求。
// APIKey 允许为空，服务层会按更新语义保留已有密钥。
type LLMConfigRequest struct {
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key,omitempty"`
	ModelName   string  `json:"model_name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// LLMConfigResponse LLM 配置读取响应。
// APIKey 永远不通过 HTTP 返回，避免配置接口泄露密钥。
type LLMConfigResponse struct {
	BaseURL          string  `json:"base_url"`
	ModelName        string  `json:"model_name"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	APIKeyConfigured bool    `json:"api_key_configured"`
}

// NewLLMConfig 将请求 DTO 转换为内部配置模型。
func (r LLMConfigRequest) NewLLMConfig() *LLMConfig {
	return &LLMConfig{
		BaseURL:     r.BaseURL,
		APIKey:      r.APIKey,
		ModelName:   r.ModelName,
		Temperature: r.Temperature,
		MaxTokens:   r.MaxTokens,
	}
}

// NewLLMConfigResponse 将内部配置转换为脱敏响应 DTO。
func NewLLMConfigResponse(cfg *LLMConfig) *LLMConfigResponse {
	if cfg == nil {
		return nil
	}
	return &LLMConfigResponse{
		BaseURL:          cfg.BaseURL,
		ModelName:        cfg.ModelName,
		Temperature:      cfg.Temperature,
		MaxTokens:        cfg.MaxTokens,
		APIKeyConfigured: cfg.APIKey != "",
	}
}

// AgentConfig Agent 行为参数配置
type AgentConfig struct {
	MaxIterations    int `json:"max_iterations"`     // 单次任务最大迭代次数，防止无限循环
	MaxRetries       int `json:"max_retries"`        // 工具调用失败时的最大重试次数
	MaxSearchResults int `json:"max_search_results"` // 搜索工具返回的最大结果数
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Servers []MCPServer `json:"servers"`
}

// MCPServer MCP (Model Context Protocol) 服务器配置。
// MCP 是连接 AI 模型与外部数据源/工具的标准协议。
type MCPServer struct {
	ServerName string   `json:"server_name"` // MCP 服务器名称，用于工具调用时的标识
	Enabled    bool     `json:"enabled"`     // 是否启用该服务器
	Transport  string   `json:"transport"`   // 传输协议：stdio、sse、http
	Tools      []string `json:"tools"`       // 该服务器暴露的工具名称列表
}

// A2AConfig A2A 配置
type A2AConfig struct {
	Servers []A2AServer `json:"servers"`
}

// A2AServer A2A 协议服务器节点。
// A2A (Agent-to-Agent) 协议允许不同 Agent 之间直接通信和协作。
type A2AServer struct {
	ID                string   `json:"id"`                 // 唯一标识，Agent 注册时的实例 ID
	Name              string   `json:"name"`               // 服务名称，如 "Claude Agent"
	Description       string   `json:"description"`        // Agent 能力描述，用于服务发现
	InputModes        []string `json:"input_modes"`        // 支持的输入模式：text, image, audio, video
	OutputModes       []string `json:"output_modes"`       // 支持的输出模式：text, image, audio
	Streaming         bool     `json:"streaming"`          // 是否支持 Server-Sent Events 流式响应
	PushNotifications bool     `json:"push_notifications"` // 是否支持主动推送通知
	Enabled           bool     `json:"enabled"`            // 是否启用，未启用的 Agent 不会被路由到
}
