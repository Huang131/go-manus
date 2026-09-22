package model

import "time"

// MCPConfig MCP (Model Context Protocol) 配置
// 统一配置定义，用于 YAML 配置、数据库持久化和运行时
type MCPConfig struct {
	Servers []MCPServer `json:"servers" mapstructure:"servers"`
}

// MCPServer MCP 服务器配置
// MCP 是连接 AI 模型与外部数据源/工具的标准协议
type MCPServer struct {
	Name       string            `json:"name" mapstructure:"name"`                         // 服务器名称，用于工具调用时的标识
	Enabled    bool              `json:"enabled" mapstructure:"enabled"`                   // 是否启用该服务器
	Command    string            `json:"command,omitempty" mapstructure:"command"`         // 启动命令，如 npx、uvx、python
	Args       []string          `json:"args,omitempty" mapstructure:"args"`               // 命令参数
	Env        map[string]string `json:"env,omitempty" mapstructure:"env"`                 // 环境变量
	Timeout    int               `json:"timeout,omitempty" mapstructure:"timeout"`         // 超时时间（秒），默认 30
	MaxRetries int               `json:"max_retries,omitempty" mapstructure:"max_retries"` // 最大重试次数，默认 2
}

// MCPServerStatus MCP 服务器状态
type MCPServerStatus struct {
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Status    string    `json:"status"` // running, stopped, error
	Error     string    `json:"error,omitempty"`
	ToolCount int       `json:"tool_count"`
	StartedAt time.Time `json:"started_at,omitempty"`
}

// ToRuntime 转换为运行时配置（过滤禁用服务器）
func (c *MCPConfig) ToRuntime() *MCPConfig {
	if c == nil {
		return nil
	}
	runtime := &MCPConfig{
		Servers: make([]MCPServer, 0, len(c.Servers)),
	}
	for _, server := range c.Servers {
		if server.Enabled {
			runtime.Servers = append(runtime.Servers, server)
		}
	}
	return runtime
}

// GetServer 获取指定名称的服务器配置
func (c *MCPConfig) GetServer(name string) *MCPServer {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i]
		}
	}
	return nil
}
