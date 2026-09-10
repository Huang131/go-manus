package model

import "time"

// AppConfig 应用配置
type AppConfig struct {
	ID          string        `json:"id"`
	ConfigType  AppConfigType `json:"config_type"`
	ConfigKey   string        `json:"config_key"`
	ConfigValue interface{}   `json:"config_value"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    HealthState                   `json:"status"`
	Timestamp int64                         `json:"timestamp"`
	Services  map[ServiceName]ServiceStatus `json:"services"`
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

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxIterations    int `json:"max_iterations"`
	MaxRetries       int `json:"max_retries"`
	MaxSearchResults int `json:"max_search_results"`
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Servers []MCPServer `json:"servers"`
}

// MCPServer MCP 服务器
type MCPServer struct {
	ServerName string   `json:"server_name"`
	Enabled    bool     `json:"enabled"`
	Transport  string   `json:"transport"`
	Tools      []string `json:"tools"`
}

// A2AConfig A2A 配置
type A2AConfig struct {
	Servers []A2AServer `json:"servers"`
}

// A2AServer A2A 服务器
type A2AServer struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	InputModes        []string `json:"input_modes"`
	OutputModes       []string `json:"output_modes"`
	Streaming         bool     `json:"streaming"`
	PushNotifications bool     `json:"push_notifications"`
	Enabled           bool     `json:"enabled"`
}
