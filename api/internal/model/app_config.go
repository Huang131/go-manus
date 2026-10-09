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
