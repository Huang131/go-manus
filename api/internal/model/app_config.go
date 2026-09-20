package model

import (
	"encoding/json"
	"time"
)

// AppConfig 应用配置，采用 Key-Value 存储，支持多类型配置。
// ConfigType 区分配置领域（Agent/MCP/A2A），ConfigKey 区分同一领域下的多个配置。
type AppConfig struct {
	ID          string          `json:"id"`           // 唯一标识
	ConfigType  AppConfigType   `json:"config_type"`  // 配置类型：agent/mcp/a2a
	ConfigKey   string          `json:"config_key"`   // 配置键，如 "default" 表示默认配置
	ConfigValue json.RawMessage `json:"config_value"` // 配置值，原始 JSON 字节，对应具体配置结构
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// HealthStatus 应用健康检查结果
type HealthStatus struct {
	Status    HealthState                   `json:"status"`    // 整体健康状态
	Timestamp int64                         `json:"timestamp"` // 检查时间戳（Unix 秒）
	Services  map[ServiceName]ServiceStatus `json:"services"`  // 各基础设施服务的健康状态
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name   ServiceName `json:"name"`
	Status HealthState `json:"status"`
	Error  string      `json:"error,omitempty"`
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
	ServerName string            `json:"server_name"` // MCP 服务器名称，用于工具调用时的标识
	Enabled    bool              `json:"enabled"`     // 是否启用该服务器
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// A2AConfig A2A 配置
type A2AConfig struct {
	Servers []A2AServer `json:"servers"`
}

// A2AServer A2A 协议服务器节点。
// A2A (Agent-to-Agent) 协议允许不同 Agent 之间直接通信和协作。
type A2AServer struct {
	ID      string `json:"id"`      // 运行时使用的唯一标识
	Enabled bool   `json:"enabled"` // 是否启用，未启用的 Agent 不会被路由到
	URL     string `json:"url"`     // 远程 Agent 基础地址
}
