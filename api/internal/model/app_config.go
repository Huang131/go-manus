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

// TimeFunc 时间函数，用于测试
var TimeFunc = time.Now

// LLMConfig LLM 配置
type LLMConfig struct {
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key,omitempty"`
	ModelName   string  `json:"model_name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
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
